package game_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
	. "github.com/smartystreets/goconvey/convey"
)

func givenAStartMenu() *game.StartMenu {
	return &game.StartMenu{}
}

type stubContext struct{}

func (*stubContext) GetDictionary(fontname string) *gui.Dictionary {
	return nil
}

func (*stubContext) GetShaders(fontname string) *render.ShaderBank {
	return nil
}

func givenADrawingContext(render.RenderQueueInterface) gui.DrawingContext {
	return &stubContext{}
}

func RunStartupSpecs() {
	base.SetupLogger("../testdata")
	windowRegion := gui.Region{
		Point: gui.Point{X: 0, Y: 0},
		Dims:  gui.Dims{Dx: 1024, Dy: 750},
	}
	menu := givenAStartMenu()

	rendertest.WithGlForTest(windowRegion.Dx, windowRegion.Dy, func(sys system.System, queue render.RenderQueueInterface) {
		texture.Init(queue)
		ctx := givenADrawingContext(queue)
		menu.Draw(windowRegion, ctx)

		So(queue, rendertest.ShouldLookLikeFile, "startup")
	})
}

func TestDrawStartupUi(t *testing.T) {
	Convey("Startup UI", t, RunStartupSpecs)
}
