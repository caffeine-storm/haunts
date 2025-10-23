package leveltests

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/gui/guitest"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system/systemtest"
	"github.com/smartystreets/goconvey/convey"
)

type LevelChoice int
type ModeChoice int

var (
	Level1 LevelChoice = 1

	ModePassNPlay ModeChoice = 1
)

func testdataDir(lvl LevelChoice) string {
	return map[LevelChoice]string{
		Level1: "level1",
	}[lvl]
}

func testScenario(lvl LevelChoice) game.Scenario {
	return map[LevelChoice]game.Scenario{
		// TODO(tmckee): hardcoding filesystem path and house name is sad-making
		Level1: {
			Script:    "Lvl01.lua",
			HouseName: "Lvl_01_Haunted_House",
		},
	}[lvl]
}

type renderingTester struct {
	lvl            LevelChoice
	renderQueue    render.RenderQueueInterface
	region         gui.Region
	drawingContext gui.UpdateableDrawingContext
}

func (rt *renderingTester) ValidateExpectations(testcase string) {
	expectedFile := rendertest.NewTestdataReference(path.Join(testdataDir(rt.lvl), testcase))

	convey.So(rt.renderQueue, rendertest.ShouldLookLikeFile, expectedFile, rendertest.Threshold(6))
}

func (rt *renderingTester) Start() {
	// Build a game.Game and game.GamePanel
	scenario := testScenario(rt.lvl)
	gamePanel := gametest.GivenAGamePanelForScenario(scenario)

	// Draw the UI
	rt.renderQueue.Queue(func(render.RenderQueueState) {
		gamePanel.Draw(rt.region, rt.drawingContext)
	})
	rt.renderQueue.Purge()

	// Wait for textures to load
	err := texture.BlockWithTimeboxUntilIdle(time.Second * 5)
	if err != nil {
		panic(fmt.Errorf("texture loading failed: %w", err))
	}

	// Blank the screen and draw the UI again now that textures are loaded.
	rt.renderQueue.Queue(func(render.RenderQueueState) {
		rendertest.ClearScreen()
		gamePanel.Draw(rt.region, rt.drawingContext)
	})
}

func (rt *renderingTester) End() {
}

type Thener interface {
	Then(any) Thener
}

type stubThener struct {
}

func (st *stubThener) Then(any) Thener {
	fmt.Println("stub thener done did the then-en-ing")
	return st
}

type StartUiTester interface {
	VersusMode(func(VersusUiTester)) Thener
}

type VersusUiTester interface {
	SelectLevel(LevelChoice) Thener
}

type SideUiTester interface {
	SelectDenizens() Thener
}

type DeployUiTester interface {
	DefaultDeployment() Thener
}

type RenderTester interface {
	ValidateExpectations(testcase string)
}

type startUiTester struct {
	window  systemtest.Window
	stubGui *gui.Gui
}

type unAnchoredBox struct {
	*gui.AnchorBox
}

func (uab *unAnchoredBox) AddChild(w gui.Widget) {
	uab.AnchorBox.AddChild(w, gui.Anchor{})
}

func (tst *startUiTester) Start() {
}

func (tst *startUiTester) RenderQueue() render.RenderQueueInterface {
	return tst.window.GetQueue()
}

// Verifies startup ui against expectation file
func (tst *startUiTester) ValidateStartupExpectations() {
}

func (*startUiTester) End() {}
func (*startUiTester) VersusMode(nextStep func(VersusUiTester)) Thener {
	// TODO: click the 'Versus' button
	return &stubThener{}
}

type TestStarter interface{}
type testStarter struct{}

type Plan struct{}

type Step struct {
	do func()
}

type Runner interface {
	Run()
}
type Planner interface {
	PlanAndRun(...Step)

	StartApplication() Step
	ChooseVersusMode() Step
	ChooseLevel(LevelChoice) Step
	ChooseDenizens() Step
	PlaceRoster() Step
}
type testPlanner struct {
	window  systemtest.Window
	stubGui *gui.Gui
}

func (tp *testPlanner) sanitizeSteps(plan []Step) {
	// TODO: validate that the sequence of steps make sense; can't place a roster
	// before choosing a level, for example.
}

func (tp *testPlanner) PlanAndRun(steps ...Step) {
	tp.sanitizeSteps(steps)

	for _, step := range steps {
		step.do()
	}
}

func (tp *testPlanner) StartApplication() Step {
	return Step{
		do: func() {
			queue := tp.window.GetQueue()
			gameScreenRegion := gui.Region{}
			gameScreenRegion.Dims = tp.window.GetDims()

			box := &unAnchoredBox{
				AnchorBox: gui.MakeAnchorBox(gameScreenRegion.Dims),
			}

			queue.Queue(func(render.RenderQueueState) {
				// TODO(tmckee:clean): lifted from cmd/main.go; DRY it out
				layout, err := game.LoadStartLayoutFromDatadir(base.GetDataDir())
				if err != nil {
					panic(fmt.Errorf("loading start layout failed: %w", err))
				}

				err = game.InsertStartMenu(box, *layout)
				if err != nil {
					panic(fmt.Errorf("couldn't insert start menu: %w", err))
				}

				menu := box.GetChildren()[0]
				menu.(*game.StartMenu).SetOpacity(0.6)
				box.Think(tp.stubGui, 12)
				box.Draw(gameScreenRegion, tp.stubGui)
			})
			queue.Purge()

			err := texture.BlockWithTimeboxUntilIdle(time.Second * 5)
			if err != nil {
				panic(fmt.Errorf("couldn't wait for textures to load: %w", err))
			}

			queue.Queue(func(render.RenderQueueState) {
				box.Draw(gameScreenRegion, tp.stubGui)
			})
			queue.Purge()

			startupUi := rendertest.NewTestdataReference("startui")
			convey.So(queue, rendertest.ShouldLookLikeFile, startupUi, rendertest.Threshold(6))
		},
	}
}

func (*testPlanner) ChooseVersusMode() Step {
	return Step{
		do: func() {
			fmt.Println("ChooseVersusMode is stubbed!")
		},
	}
}

func (*testPlanner) ChooseLevel(LevelChoice) Step {
	return Step{
		do: func() {
			fmt.Println("ChooseLevel is stubbed!")
		},
	}
}

func (*testPlanner) ChooseDenizens() Step {
	return Step{
		do: func() {
			fmt.Println("ChooseDenizens is stubbed!")
		},
	}
}

func (*testPlanner) PlaceRoster() Step {
	return Step{
		do: func() {
			fmt.Println("PlaceRoster is stubbed!")
		},
	}
}

func EndToEndTest(t *testing.T, label string, testCase func(Planner)) {
	testname := fmt.Sprintf("%s end-to-end test", label)
	region := gui.MakeRegion(0, 0, 1024, 750)
	systemtest.WithTestWindow(region.Dx, region.Dy, func(syswindow systemtest.Window) {
		// TODO(tmckee:clean): this was lifted from gametest; DRY it out
		renderQueue := syswindow.GetQueue()
		base.SetDatadir("../../data")
		globals.SetRenderQueue(renderQueue)
		renderQueue.Queue(func(st render.RenderQueueState) {
			globals.SetRenderQueueState(st)
		})
		renderQueue.Purge()

		ctx := gametest.GivenADrawingContext(region.Dims)
		registry.LoadAllRegistries()
		base.InitDictionaries(ctx)
		texture.Init(renderQueue)
		base.InitShaders(renderQueue)
		// TODO-end

		convey.Convey(testname, t, func(conveyContext convey.C) {
			planner := &testPlanner{
				window:  syswindow,
				stubGui: guitest.MakeStubbedGui(syswindow.GetDims()),
			}

			testCase(planner)
		})
	})
}
