package game

import (
	"fmt"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
)

type Timer interface {
	Now() time.Time
}

type useStdTimer struct{}

func (t *useStdTimer) Now() time.Time {
	return time.Now()
}

type Overlay struct {
	region gui.Region
	game   *Game
	timer  Timer
}

func MakeOverlay(g *Game) gui.Widget {
	return MakeOverlayWithTimer(g, &useStdTimer{})
}

func MakeOverlayWithTimer(g *Game, t Timer) gui.Widget {
	return &Overlay{game: g, timer: t}
}

func (o *Overlay) Requested() gui.Dims {
	return gui.Dims{Dx: 1024, Dy: 768}
}

func (o *Overlay) Expandable() (bool, bool) {
	return false, false
}

func (o *Overlay) Rendered() gui.Region {
	return o.region
}

func (o *Overlay) Respond(g *gui.Gui, group gui.EventGroup) bool {
	return false
}

func (o *Overlay) Think(g *gui.Gui, dt int64) {
	var side Side
	if o.game.viewer.Los_tex == o.game.los.intruders.tex {
		side = SideExplorers
	} else if o.game.viewer.Los_tex == o.game.los.denizens.tex {
		side = SideHaunt
	} else {
		side = SideNone
	}

	for i := range o.game.Waypoints {
		o.game.viewer.RemoveFloorDrawable(&o.game.Waypoints[i])
		o.game.viewer.AddFloorDrawable(&o.game.Waypoints[i])
		o.game.Waypoints[i].Active = o.game.Waypoints[i].Side == side
		o.game.Waypoints[i].drawn = false
	}
}

func (o *Overlay) Draw(region gui.Region, ctx gui.DrawingContext) {
	logging.Trace("Overlay.Draw", "region", region, "o.game.Side", o.game.Side, "o.game.Waypoints", o.game.Waypoints)
	o.region = region
	switch o.game.Side {
	case SideHaunt:
		if o.game.los.denizens.mode == LosModeBlind {
			logging.Trace("skipping Overlay.Draw for blinded denizens")
			return
		}
	case SideExplorers:
		if o.game.los.intruders.mode == LosModeBlind {
			logging.Trace("skipping Overlay.Draw for blinded intruders")
			return
		}
	default:
		panic(fmt.Errorf("unknown side: %v", o.game.Side))
	}
	if len(o.game.Waypoints) == 0 {
		return
	}

	base.EnableShader("waypoint")
	defer base.EnableShader("")
	t := float32(o.timer.Now().UnixNano()%1e15) / 1.0e9
	base.SetUniformF("waypoint", "time", t)

	render.WithColour(1.0, 0.0, 0.0, 0.5, func() {
		for _, wp := range o.game.Waypoints {
			if !wp.Active || wp.drawn {
				continue
			}

			// TODO(tmckee): move this to a Waypoint.Draw function
			cx := float32(wp.X)
			cy := float32(wp.Y)
			r := float32(wp.Radius)
			cx1, cy1 := o.game.viewer.BoardToWindow(cx-r, cy-r)
			cx2, cy2 := o.game.viewer.BoardToWindow(cx-r, cy+r)
			cx3, cy3 := o.game.viewer.BoardToWindow(cx+r, cy+r)
			cx4, cy4 := o.game.viewer.BoardToWindow(cx+r, cy-r)
			logging.Trace("draw waypoint", "wp", wp, "windowcoords", []int{
				cx1, cy1,
				cx2, cy2,
				cx3, cy3,
				cx4, cy4,
			})

			base.SetUniformF("waypoint", "radius", float32(wp.Radius))

			gl.PushAttrib(gl.COLOR_BUFFER_BIT)
			defer gl.PopAttrib()

			gl.Enable(gl.BLEND)
			gl.BlendFuncSeparate(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA, gl.ZERO, gl.ONE)

			gl.Begin(gl.QUADS)
			gl.TexCoord2i(0, 1)
			gl.Vertex2i(cx1, cy1)
			gl.TexCoord2i(0, 0)
			gl.Vertex2i(cx2, cy2)
			gl.TexCoord2i(1, 0)
			gl.Vertex2i(cx3, cy3)
			gl.TexCoord2i(1, 1)
			gl.Vertex2i(cx4, cy4)
			gl.End()
		}
	})
}

func (o *Overlay) DrawFocused(region gui.Region, ctx gui.DrawingContext) {
	o.Draw(region, ctx)
}

func (o *Overlay) String() string {
	return "overlay"
}
