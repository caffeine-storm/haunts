package hui // haunts ui

import (
	"path/filepath"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
)

type Option interface {
	// index is the index of this option into the layout's array of options,
	// and is also the index into the map selected.  hovered indicates whether
	// or not the mouse is over this particular option.  selected is a map from
	// index to hether or not that option is selected right now.
	Draw(hovered, selected, selectable bool, region gui.Region)
	Think(dt int64)
}

type RosterChooserLayout struct {
	Num_options int
	Option      struct {
		Dx, Dy int
	}
	Up, Down texture.Object

	// speed at which the scrolling happens
	// 0.0 doesn't move at all
	// 1.0 is instantaneous
	Speed float64

	// Might want background too?  Mabye textures for the other buttons?
}

type RosterChooser struct {
	gui.BasicZone
	layout  RosterChooserLayout
	options []Option

	// last position of the mouse cursor
	mouse gui.Point

	// So we can give a dt to the options if they want to animate
	last_think int64

	// What option is at the top of the list
	focus int

	// As we move the focus around we gradually move our view to smoothly
	// adjust
	focus_pos float64

	// What options we have currently selected
	selected map[int]bool

	selector Selector

	on_complete func(map[int]bool)

	on_undo func()

	// Render regions - makes it easy to remember where we rendered things so we
	// know where to check for clicks.
	render struct {
		up, down    gui.Region
		options     []gui.Region
		all_options gui.Region
		done, undo  gui.Region
	}
}

// A Selector determines whether a particular index can be clicked to toggle
// whether or not it is selected.
//
// index: The index of the option that the user is trying to select.  If index
// is -1 the function should return whether or not the current selected map is
// valid.  If index is -1 doit will be false.
//
// selected: a map from index to whether or not that index is already selected
// only selected indices should be stored in the map, when an index is
// deselected it should be removed from the map.
//
// doit: if this is true this function should also add/remove index from
// selected.
type Selector func(index int, selected map[int]bool, doit bool) bool

func SelectInRange(min, max int) Selector {
	return func(index int, selected map[int]bool, doit bool) (valid bool) {
		if index == -1 {
			valid = (len(selected) >= min && len(selected) <= max)
		} else {
			if _, ok := selected[index]; ok {
				valid = true
			} else {
				valid = len(selected) < max
			}
		}
		if doit && valid {
			if _, ok := selected[index]; ok {
				delete(selected, index)
			} else {
				selected[index] = true
			}
		}
		return
	}
}

func SelectExactlyOne(index int, selected map[int]bool, doit bool) (valid bool) {
	if index == -1 {
		valid = (len(selected) == 1)
	} else {
		valid = true
	}
	if doit {
		var other int
		for k, _ := range selected {
			other = k
		}
		delete(selected, other)
		selected[index] = true
	}
	return
}

func MakeRosterChooser(options []Option, selector Selector, on_complete func(map[int]bool), on_undo func()) *RosterChooser {
	var rc RosterChooser
	rc.options = options
	err := base.LoadAndProcessObject(filepath.Join(base.GetDataDir(), "ui", "widgets", "roster_chooser.json"), "json", &rc.layout)
	if err != nil {
		base.DeprecatedLog().Error("MakeRosterChooser failed", "err", err)
		return nil
	}

	// TODO(tmckee): do we _need_ to know the font height?
	// fontHeight := int(base.GetDictionary(15).MaxHeight())
	fontHeight := 20
	rc.Request_dims = gui.Dims{
		Dx: rc.layout.Down.Data().Dx() + rc.layout.Option.Dx,
		Dy: rc.layout.Num_options*rc.layout.Option.Dy + 2*fontHeight,
	}
	rc.selected = make(map[int]bool)
	rc.selector = selector
	rc.on_complete = on_complete
	rc.on_undo = on_undo
	rc.render.options = make([]gui.Region, len(rc.options))
	return &rc
}

func (rc *RosterChooser) Think(ui *gui.Gui, t int64) {
	var dt int64
	if rc.last_think != 0 {
		dt = t - rc.last_think
	}
	rc.last_think = t
	for i := range rc.options {
		rc.options[i].Think(dt)
	}

	max := len(rc.options) - rc.layout.Num_options
	if rc.focus > max {
		rc.focus = max
	}
	if rc.focus < 0 {
		rc.focus = 0
	}
	rc.focus_pos = (1-rc.layout.Speed)*rc.focus_pos + rc.layout.Speed*float64(rc.focus)

	// TODO(tmckee): need to query a glop.System object or consult a cache;
	// gui.CursorPos() might be a decent option...
	// rc.mouse.X, rc.mouse.Y = gin.In().GetCursorPos()
	rc.mouse.X, rc.mouse.Y = 0, 0
}

func (rc *RosterChooser) Respond(ui *gui.Gui, group gui.EventGroup) bool {
	base.DeprecatedLog().Info("RosterChooser.Respond")
	// TODO(tmckee): upper case 'L' vs lower case 'l'? Originally a rune, 'l'.
	if group.IsPressed(gin.AnyKeyL) {
		rc.focus += rc.layout.Num_options
		return true
	}
	// TODO(tmckee): upper case 'O' vs lower case 'o'? Originally a rune, 'o'.
	if group.IsPressed(gin.AnyKeyO) {
		rc.focus -= rc.layout.Num_options
		return true
	}
	if group.IsPressed(gin.AnyMouseLButton) {
		if gp, ok := ui.UseMousePosition(group); ok {
			if gp.Inside(rc.render.down) {
				rc.focus += rc.layout.Num_options
				return true
			} else if gp.Inside(rc.render.up) {
				rc.focus -= rc.layout.Num_options
				return true
			} else if gp.Inside(rc.render.all_options) {
				for i := range rc.render.options {
					if gp.Inside(rc.render.options[i]) {
						rc.selector(i, rc.selected, true)
						return true
					}
				}
			} else if gp.Inside(rc.render.done) {
				if rc.selector(-1, rc.selected, false) {
					base.DeprecatedLog().Info("calling on-complete")
					rc.on_complete(rc.selected)
				}
				return true
			} else if rc.on_undo != nil && gp.Inside(rc.render.undo) {
				rc.on_undo()
				return true
			}
		}
	}
	return false
}

func (rc *RosterChooser) Draw(r gui.Region, ctx gui.DrawingContext) {
	rc.Render_region = r
	r.PushClipPlanes()
	defer r.PopClipPlanes()
	gl.Enable(gl.TEXTURE_2D)

	{ // Up button
		x := r.X
		y := r.Y + r.Dy - rc.layout.Up.Data().Dy()
		rc.render.up.X = x
		rc.render.up.Y = y
		rc.render.up.Dx = rc.layout.Up.Data().Dx()
		rc.render.up.Dy = rc.layout.Up.Data().Dy()
		if rc.mouse.Inside(rc.render.up) {
			gl.Color4d(1, 1, 1, 1)
		} else {
			gl.Color4d(0.8, 0.8, 0.8, 1)
		}
		rc.layout.Up.Data().RenderNatural(x, y)
	}

	{ // Down button
		x := r.X
		y := r.Y + rc.layout.Down.Data().Dy()
		rc.render.down.X = x
		rc.render.down.Y = y
		rc.render.down.Dx = rc.layout.Down.Data().Dx()
		rc.render.down.Dy = rc.layout.Down.Data().Dy()
		if rc.mouse.Inside(rc.render.down) {
			gl.Color4d(1, 1, 1, 1)
		} else {
			gl.Color4d(0.8, 0.8, 0.8, 1)
		}
		rc.layout.Down.Data().RenderNatural(x, y)
	}

	{ // Options
		rc.render.all_options.X = r.X + rc.layout.Down.Data().Dx()
		rc.render.all_options.Y = r.Y + r.Dy - rc.layout.Num_options*rc.layout.Option.Dy
		rc.render.all_options.Dx = rc.layout.Option.Dx
		rc.render.all_options.Dy = rc.layout.Num_options * rc.layout.Option.Dy
		rc.render.all_options.PushClipPlanes()
		x := rc.render.all_options.X
		y := r.Y + r.Dy - rc.layout.Option.Dy + int(float64(rc.layout.Option.Dy)*rc.focus_pos)
		for i := range rc.options {
			rc.render.options[i] = gui.Region{
				Point: gui.Point{X: x, Y: y},
				Dims:  gui.Dims{Dx: rc.layout.Option.Dx, Dy: rc.layout.Option.Dy},
			}
			hovered := rc.mouse.Inside(rc.render.options[i])
			selected := rc.selected[i]
			selectable := rc.selector(i, rc.selected, false)
			rc.options[i].Draw(hovered, selected, selectable, rc.render.options[i])
			y -= rc.layout.Option.Dy
		}

		rc.render.all_options.PopClipPlanes()
	}

	{ // Text
		// TODO(tmckee): is this the right font name?
		d := ctx.GetDictionary("glop.font")
		x := r.X
		y := r.Y + d.MaxHeight()/2
		x1 := x + r.Dx/3
		x2 := x + 2*r.Dx/3

		rc.render.done = gui.Region{
			Point: gui.Point{X: x, Y: r.Y},
			Dims:  gui.Dims{Dx: r.Dx / 2, Dy: int(d.MaxHeight() * 2)},
		}
		rc.render.undo = gui.Region{
			Point: gui.Point{X: x + r.Dx/2, Y: r.Y},
			Dims:  gui.Dims{Dx: r.Dx / 2, Dy: int(d.MaxHeight() * 2)},
		}

		shaderBank := globals.RenderQueueState().Shaders()

		if rc.mouse.Inside(rc.render.done) {
			gl.Color4d(1, 1, 1, 1)
		} else {
			gl.Color4d(0.6, 0.6, 0.6, 1)
		}
		d.RenderString("Done", gui.Point{X: x1, Y: y}, d.MaxHeight(), gui.Center, shaderBank)

		if rc.on_undo != nil {
			if rc.mouse.Inside(rc.render.undo) {
				gl.Color4d(1, 1, 1, 1)
			} else {
				gl.Color4d(0.6, 0.6, 0.6, 1)
			}
			d.RenderString("Undo", gui.Point{X: x2, Y: y}, d.MaxHeight(), gui.Center, shaderBank)
		}

	}
}

func (rc *RosterChooser) DrawFocused(gui.Region, gui.DrawingContext) {

}

func (rc *RosterChooser) String() string {
	return "roster chooser"
}
