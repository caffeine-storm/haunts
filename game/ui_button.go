package game

import (
	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/sound"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/MobRulesGames/opengl/gl"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
)

type ButtonLike interface {
	handleClick(x, y int, data interface{}) bool
	Respond(group gui.EventGroup, data interface{}) bool
	Think(x, y, mx, my int, dt int64)
	RenderAt(x, y int)
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
	// be clicked.  Will only be called during Think.
	valid_func func() bool
	valid      bool

	// Key that can be bound to have the same effect as clicking this button
	key gin.KeyId

	// TODO(tmckee): clean: this should just be a gui.Region, no?
	bounds struct {
		x, y, dx, dy int
	}
}

// If x,y is inside the button's region then it will run its function and
// return true, otherwise it does nothing and returns false.
func (b *Button) handleClick(x, y int, data interface{}) bool {
	in := pointInsideRect(x, y, b.bounds.x, b.bounds.y, b.bounds.dx, b.bounds.dy)
	if in && b.valid {
		b.f(data)
		sound.PlaySound("Haunts/SFX/UI/Select", 0.75)
	}
	return in
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
	if group.Events[0].Key.Id() == b.key && group.Events[0].Type == gin.Press {
		if b.valid {
			b.f(data)
		}
		return true
	}
	return false
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
	in := b.valid && pointInsideRect(mx, my, b.bounds.x, b.bounds.y, b.bounds.dx, b.bounds.dy)
	if in && !b.was_in {
		sound.PlaySound("Haunts/SFX/UI/Tick", 0.75)
	}
	b.was_in = in
	b.opacity = computeOpacity(b.opacity, in, dt)
}

func (b *Button) RenderAt(x, y int) {
	// TODO(tmckee): clean: why not
	// gl.Color4d(1, 1, 1, b.shade))
	gl.Color4ub(255, 255, 255, byte(b.opacity*255))
	base.Log().Trace("Button.RenderAt", "tex-path", b.Texture.Path)
	if b.Texture.Path != "" {
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
		base.Log().Trace("button.RenderAt", "b.Text.String", b.Text.String, "b.X", b.X, "b.Y", b.Y, "x", x, "y", y)
		shaderBank := globals.RenderQueueState().Shaders()
		d.RenderString(b.Text.String, gui.Point{X: b.X + x, Y: b.Y + y}, d.MaxHeight(), just, shaderBank)
	}
}
