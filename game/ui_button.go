package game

import (
	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/sound"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
)

type ButtonLike interface {
	// TODO(#18): 'handleClick' is a bit redundant; we should just always
	// 'Respond' and call handleClick internally
	handleClick(x, y int, data interface{}) bool
	Respond(group gui.EventGroup, data interface{}) bool
	Think(x, y, mx, my int, dt int64)
	RenderAt(x, y int)
}

type SetOpacityer interface {
	SetOpacity(percent float64)
}

type Button struct {
	X, Y int
	// TODO(tmckee): clean: split button into TextButton and TextureButton
	Texture texture.Object
	Text    struct {
		String        string
		Size          int
		Justification string
	}

	// True if the mouse was over this button on the last frame
	was_in bool

	// Intensity as a percent - buttons will brighten when the mouse is over it
	opacity float64

	// Function to run whenever the button is clicked
	f func(interface{})

	// If not nil this function can return false to indicate that it cannot
	// be clicked.
	valid_func func() bool
	valid      bool

	// Key that can be bound to have the same effect as clicking this button
	key gin.KeyId

	// TODO(tmckee): clean: this should just be a gui.Region, no?
	bounds struct {
		x, y, dx, dy int
	}
}

var _ ButtonLike = (*Button)(nil)
var _ SetOpacityer = (*Button)(nil)

// If x,y is inside the button's region then it will run its function and
// return true, otherwise it does nothing and returns false.
func (b *Button) handleClick(x, y int, data interface{}) bool {
	if b.valid_func != nil {
		b.valid = b.valid_func()
	} else {
		b.valid = true
	}
	in := b.Over(x, y)
	if in && b.valid {
		b.f(data)
		sound.PlaySound("Haunts/SFX/UI/Select", 0.75)
	}
	return in
}

func (b *Button) SetOpacity(pct float64) {
	b.opacity = pct
}

func (b *Button) Over(mx, my int) bool {
	return pointInsideRect(mx, my, b.bounds.x, b.bounds.y, b.bounds.dx, b.bounds.dy)
}

func (b *Button) Respond(group gui.EventGroup, data interface{}) bool {
	if b.valid_func != nil {
		b.valid = b.valid_func()
	} else {
		b.valid = true
	}

	if !group.IsPressed(b.key) {
		return false
	}

	// TODO(tmckee): if the button is invalid, shouldn't we return false?
	// Otherwise, it looks like the button handled the event but we didn't call
	// the wrapped func.
	if b.valid {
		b.f(data)
	}

	return true
}

func computeOpacity(current float64, in bool, dt int64) float64 {
	var target float64
	if in {
		target = 1.0
	} else {
		target = 0.6
	}
	return assymptoticApproach(current, target, dt)
}

func (b *Button) Think(x, y, mx, my int, dt int64) {
	if b.valid_func != nil {
		b.valid = b.valid_func()
	} else {
		b.valid = true
	}

	in := b.valid && b.Over(mx, my)
	if in && !b.was_in {
		sound.PlaySound("Haunts/SFX/UI/Tick", 0.75)
	}
	b.was_in = in
	b.opacity = computeOpacity(b.opacity, in, dt)
}

func (b *Button) RenderAt(x, y int) {
	render.WithColour(1.0, 1.0, 1.0, float32(b.opacity), func() {
		texpath := b.Texture.GetPath()
		logging.Trace("Button.RenderAt", "texpath", texpath)
		if texpath != "" {
			b.Texture.Data().RenderNatural(b.X+x, b.Y+y)
			b.bounds.x = b.X + x
			b.bounds.y = b.Y + y
			b.bounds.dx = b.Texture.Data().Dx()
			b.bounds.dy = b.Texture.Data().Dy()
		} else {
			d := base.GetDictionary(b.Text.Size)
			b.bounds.x = b.X + x
			b.bounds.y = b.Y + y
			b.bounds.dx = int(d.StringPixelWidth(b.Text.String))
			b.bounds.dy = int(d.MaxHeight())
			var just gui.Justification
			switch b.Text.Justification {
			case "center":
				just = gui.Center
				b.bounds.x -= b.bounds.dx / 2
			case "left":
				just = gui.Left
			case "right":
				just = gui.Right
				b.bounds.x -= b.bounds.dx
			default:
				just = gui.Center
				b.bounds.x -= b.bounds.dx / 2
				b.Text.Justification = "center"
			}
			logging.Trace("button.RenderAt", "b.Text.String", b.Text.String, "b.X", b.X, "b.Y", b.Y, "x", x, "y", y)
			shaderBank := globals.RenderQueueState().Shaders()
			d.RenderString(b.Text.String, gui.Point{X: b.X + x, Y: b.Y + y}, d.MaxHeight(), just, shaderBank)
		}
	})
}
