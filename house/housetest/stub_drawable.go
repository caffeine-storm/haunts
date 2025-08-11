package housetest

import (
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/mathgl"
	"github.com/go-gl-legacy/gl"
)

type StubDraw struct {
	X, Y   int
	Dx, Dy float64
}

var _ house.Drawable = (*StubDraw)(nil)

func (s *StubDraw) Dims() (int, int) {
	return int(s.Dx), int(s.Dy)
}

func (s *StubDraw) Pos() (int, int) {
	return s.X, s.Y
}

func (s *StubDraw) FPos() (float64, float64) {
	return float64(s.X), float64(s.Y)
}

func (s *StubDraw) Render(pos mathgl.Vec2, width float32) {
	logging.Debug("StubDraw.Render", "pos", pos, "width", width)

	gl.Begin(gl.TRIANGLES)
	gl.Vertex3d(-(s.Dx * 0.5), -(s.Dy * 0.5), 0)
	gl.Vertex3d(-(s.Dx * 0.5), +(s.Dy * 0.5), 0)
	gl.Vertex3d(+(s.Dx * 0.5), +(s.Dx * 0.5), 0)
	gl.End()
}

func (*StubDraw) Color() (r, g, b, a byte) {
	r, g, b, a = 255, 255, 0, 255
	return
}
