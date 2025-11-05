package leveltests

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system/systemtest"
	"github.com/smartystreets/goconvey/convey"
)

type LevelChoice int
type ModeChoice  int

const (
	LevelNone LevelChoice = iota
	Level1
)

const (
	ModePassNPlay ModeChoice = 1
)

func testdataDir(lvl LevelChoice) string {
	return map[LevelChoice]string{
		LevelNone: "",
		Level1:    "level1",
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

type unAnchoredBox struct {
	*gui.AnchorBox
}

func (uab *unAnchoredBox) AddChild(w gui.Widget) {
	uab.AnchorBox.AddChild(w, gui.Anchor{})
}

type Plan struct{}

type Step struct {
	do func()
}

type Planner interface {
	PlanAndRun(...Step)

	StartApplication() Step
	ChooseVersusMode() Step
	ChooseLevel(LevelChoice) Step
	ChooseDenizens() Step
	PlaceRoster() Step
}

type rootWidget interface {
	gui.Widget
	gui.WidgetParent
}

type testPlanner struct {
	window    systemtest.Window
	rootGui   *gui.Gui
	timepoint uint64

	windowRoot        rootWidget
	startUiMenuConfig game.MenuConfig
}

func (tp *testPlanner) validateExpectations(lvl LevelChoice, testcase string) {
	expectedFile := rendertest.NewTestdataReference(path.Join(testdataDir(lvl), testcase))

	convey.So(tp.window.GetQueue(), rendertest.ShouldLookLikeFile, expectedFile, rendertest.Threshold(6))
}

func (tp *testPlanner) redrawRootGui() {
	tp.window.GetQueue().Queue(func(render.RenderQueueState) {
		rendertest.ClearScreen()
		tp.rootGui.Draw()
	})
	tp.window.GetQueue().Purge()
}

func (tp *testPlanner) thinkSeconds(seconds uint64) {
	rendertest.AdvanceTimeMillis(tp.window.GetSystemInterface(), seconds*1000)
	tp.timepoint += seconds * 1000
	tp.rootGui.Think(int64(tp.timepoint))
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
			gameScreenRegion := gui.Region{}
			gameScreenRegion.Dims = tp.window.GetDims()

			tp.windowRoot = &unAnchoredBox{
				AnchorBox: gui.MakeAnchorBox(gameScreenRegion.Dims),
			}
			tp.rootGui.AddChild(tp.windowRoot)

			// TODO(tmckee:clean): lifted from cmd/main.go; DRY it out
			layout, err := game.LoadStartLayoutFromDatadir(base.GetDataDir())
			if err != nil {
				panic(fmt.Errorf("loading start layout failed: %w", err))
			}

			err = game.InsertStartMenu(tp.windowRoot, *layout)
			if err != nil {
				panic(fmt.Errorf("couldn't insert start menu: %w", err))
			}

			startUiMenu := tp.windowRoot.GetChildren()[0].(*game.StartMenu)
			tp.startUiMenuConfig = startUiMenu.Layout.Menu
			startUiMenu.SetOpacity(0.6)
			tp.thinkSeconds(5)
			tp.redrawRootGui()

			err = texture.BlockWithTimeboxUntilIdle(time.Second * 5)
			if err != nil {
				panic(fmt.Errorf("couldn't wait for textures to load: %w", err))
			}

			tp.redrawRootGui()
			tp.validateExpectations(LevelNone, "startui")
		},
	}
}

func (tp *testPlanner) ChooseVersusMode() Step {
	return Step{
		do: func() {
			versusButton := tp.startUiMenuConfig.Versus
			bx, by := versusButton.X, versusButton.Y
			drv := tp.window.NewDriver()
			drv.Click(bx, by)
			drv.ProcessFrame()

			tp.redrawRootGui()

			// At this point, the root widget will contain a map-select screen; we
			// need to simulate some time passing so that it knows to be faded in.
			tp.thinkSeconds(5)

			if err := texture.BlockWithTimeboxUntilIdle(time.Second * 5); err != nil {
				panic(fmt.Errorf("texture loading failed: %w", err))
			}

			tp.redrawRootGui()

			tp.validateExpectations(LevelNone, "level-select")
		},
	}
}

func (*testPlanner) ChooseLevel(LevelChoice) Step {
	return Step{
		do: func() {
			fmt.Println("testPlanner.ChooseLevel is stubbed!")
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

		inputObj := gin.MakeLogged(logging.DebugLogger())
		rootGui, err := gui.MakeLogged(region.Dims, inputObj, logging.DebugLogger())
		syswindow.AddInputListener(rootGui)
		if err != nil {
			panic(fmt.Errorf("gui.MakeLogged failed: %w", err))
		}
		registry.LoadAllRegistries()
		base.InitDictionaries(rootGui)
		texture.Init(renderQueue)
		base.InitShaders(renderQueue)
		// TODO-end

		convey.Convey(testname, t, func(conveyContext convey.C) {
			planner := &testPlanner{
				window:  syswindow,
				rootGui: rootGui,
			}

			testCase(planner)
		})
	})
}
