package leveltests

import (
	"fmt"
	"path"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game/gametest"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/render/rendertest/testbuilder"
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

func testLabel(lvl LevelChoice) string {
	return map[LevelChoice]string{
		Level1: "Level 01",
	}[lvl]
}

type Tester interface {
	ValidateExpectations(testcase string)
}

type renderingTester struct {
	lvl            LevelChoice
	renderQueue    render.RenderQueueInterface
	region         gui.Region
	drawingContext gui.UpdateableDrawingContext
}

func (rt *renderingTester) ValidateExpectations(testcase string) {
	expectedFile := rendertest.NewTestdataReference(path.Join(testdataDir(rt.lvl), testcase))

	convey.So(rt.renderQueue, rendertest.ShouldLookLikeFile, expectedFile)
}

func (rt *renderingTester) Start() {
	// Build a game.Game and game.GamePanel
	gamePanel := gametest.GivenAGamePanel()

	// TODO: Load the level

	// TODO: Place units from the 'roster'

	// Draw the UI
	rt.renderQueue.Queue(func(render.RenderQueueState) {
		gamePanel.Draw(rt.region, rt.drawingContext)
	})
}

func (rt *renderingTester) End() {
}

func IntegrationTest(t *testing.T, level LevelChoice, mode ModeChoice, fn func(Tester)) {
	testname := fmt.Sprintf("%s end-to-end test", testLabel(level))
	region := gui.MakeRegion(0, 0, 1024, 750)
	testbuilder.WithSize(region.Dx, region.Dy, func(renderQueue render.RenderQueueInterface) {

		// TODO(tmckee:clean): this was lifted from gametest; DRY it out
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
			tst := &renderingTester{
				lvl:            level,
				renderQueue:    renderQueue,
				region:         region,
				drawingContext: ctx,
			}

			tst.Start()

			fn(tst)

			tst.End()
		})
	})
}
