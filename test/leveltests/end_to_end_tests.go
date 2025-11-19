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
	"github.com/caffeine-storm/glop/gin"
	"github.com/caffeine-storm/glop/gui"
	"github.com/caffeine-storm/glop/render"
	"github.com/caffeine-storm/glop/render/rendertest"
	"github.com/caffeine-storm/glop/system/systemtest"
	"github.com/smartystreets/goconvey/convey"
)

type (
	LevelChoice int
	ModeChoice  int
)

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

func (tp *testPlanner) getLevelChooser() *game.Chooser {
	tp.windowRoot.GetChildren()
	unanchored := tp.windowRoot.(*unAnchoredBox)
	return unanchored.GetChildren()[0].(*game.Chooser)
}

func scenarioForLevel(lvl LevelChoice) game.Scenario {
	switch lvl {
	case Level1:
		return game.Scenario{
			Script:    "Lvl01.lua",
			HouseName: "Lvl_01_Haunted_House",
		}
	default:
		panic(fmt.Errorf("couldn't get scenario for level choice %v", lvl))
	}
}

func (tp *testPlanner) scrollChooserToTarget(driver systemtest.Driver, chooser *game.Chooser, lvl LevelChoice) (game.Option, game.ForEachOptionData) {
	// Get a reference to the Option, 'ret' for 'lvl'
	targetScenario := scenarioForLevel(lvl)

	// TODO: make a better API for what we need
	var targetData game.ForEachOptionData
	var ret game.Option

	getTargetData := func() {
		chooser.ForEachOption(func(_ int, o game.Option, data game.ForEachOptionData) {
			if o.Scenario() != targetScenario {
				return
			}
			ret = o
			targetData = data
		})
		if ret == nil {
			panic(fmt.Errorf("couldn't find option for scenario %v", targetScenario))
		}
	}
	getTargetData()

	downCount := 0

	// It's counter-intuitive but clicking the 'down' button scrolls each item
	// 'up', thereby increasing the y-offset.
	_, downX, downY := chooser.FindButton("down_arrow.png")
	for targetData.Y < 0 {
		driver.Click(downX, downY)
		driver.ProcessFrame()
		getTargetData()
		downCount++
	}

	maxHeight := tp.window.GetDims().Dy
	_, upX, upY := chooser.FindButton("up_arrow.png")
	for targetData.Y > maxHeight {
		if downCount > 0 {
			panic(fmt.Errorf("shouldn't have to go back up after having gone down"))
		}
		driver.Click(upX, upY)
		driver.ProcessFrame()
		getTargetData()
	}

	// Return a reference to 'ret'
	return ret, targetData
}

func (tp *testPlanner) getNextButton(chooser *game.Chooser) (game.ButtonLike, int, int) {
	return chooser.FindButton("arrow_rf.png")
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

func (tp *testPlanner) redrawRootGuiWithNewTextures() {
	tp.redrawRootGui()
	err := texture.BlockWithTimeboxUntilIdle(time.Second * 5)
	if err != nil {
		panic(fmt.Errorf("texture.BlockWithTimeboxUntilIdle(5s) failed: %w", err))
	}
	tp.redrawRootGui()
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
			// TODO(till-here)

			startUiMenu := tp.windowRoot.GetChildren()[0].(*game.StartMenu)
			tp.startUiMenuConfig = startUiMenu.Layout.Menu
			startUiMenu.SetOpacity(0.6)
			tp.thinkSeconds(5)

			tp.redrawRootGuiWithNewTextures()

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

			tp.redrawRootGuiWithNewTextures()

			tp.validateExpectations(LevelNone, "level-select")
		},
	}
}

func (tp *testPlanner) ChooseLevel(lvl LevelChoice) Step {
	return Step{
		do: func() {
			// Scroll if needed
			levelChooser := tp.getLevelChooser()
			driver := tp.window.NewDriver()
			_, optionData := tp.scrollChooserToTarget(driver, levelChooser, lvl)

			// Hover over LevelChoice's button
			driver.MoveMouse(optionData.X, optionData.Y)
			driver.ProcessFrame()
			tp.thinkSeconds(5)
			tp.redrawRootGuiWithNewTextures()
			tp.validateExpectations(lvl, "map-chooser-hover-choice")

			// Click
			driver.Click(optionData.X, optionData.Y)
			driver.ProcessFrame()
			tp.thinkSeconds(5)
			tp.redrawRootGuiWithNewTextures()
			tp.validateExpectations(lvl, "map-chooser-clicked-choice")

			// Hover Next
			_, nextX, nextY := tp.getNextButton(levelChooser)
			driver.MoveMouse(nextX, nextY)
			driver.ProcessFrame()
			tp.thinkSeconds(5)
			tp.redrawRootGuiWithNewTextures()
			tp.validateExpectations(lvl, "map-chooser-hover-next")

			// Click
			driver.Click(nextX, nextY)
			driver.ProcessFrame()
			tp.thinkSeconds(5)
			tp.redrawRootGuiWithNewTextures()
			tp.thinkSeconds(5)
			tp.redrawRootGuiWithNewTextures()
			// right now, there's a game.Chooser that gets added to the
			// GamePanel during the lua Script (side_choices =
			// Script.ChooserFromFile("ui/start/versus/side.json")).
			// it doesn't seem to be drawing right now :-/
			tp.validateExpectations(lvl, "side-choice-start")
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
