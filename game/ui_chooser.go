package game

import (
	"fmt"
	"math"
	"path/filepath"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/sound"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/caffeine-storm/gl"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/util/algorithm"
)

type Option interface {
	// index is the index of this option into the layout's array of options,
	// and is also the index into the map selected.  hovered indicates whether
	// or not the mouse is over this particular option.  selected is a map from
	// index to hether or not that option is selected right now.
	Scenario() Scenario
	Draw(x, y, dx int)
	DrawInfo(x, y, dx, dy int)
	Height() int
	Think(hovered, selected, selectable bool, dt int64)
}

type colorOption struct {
	r, g, b, a byte
}

func (co *colorOption) String() string {
	return fmt.Sprintf("ColorOption(%d, %d, %d, %d)", co.r, co.g, co.b, co.a)
}

func (co *colorOption) Think(hovered, selected, selectable bool, dt int64) {
	var target byte
	switch {
	case selected:
		target = 255
	case selectable && hovered:
		target = 200
	case selectable && !hovered:
		target = 150
	default:
		target = 50
	}
	co.a = target
}

func (co *colorOption) Height() int {
	return 125
}

func (co *colorOption) Draw(x, y, dx int) {
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4ub(co.r, co.g, co.b, co.a)
	gl.Begin(gl.QUADS)
	gl.Vertex2i(x, y)
	gl.Vertex2i(x, y+co.Height())
	gl.Vertex2i(x+dx, y+co.Height())
	gl.Vertex2i(x+dx, y)
	gl.End()
}

func (co *colorOption) DrawInfo(x, y, dx, dy int) {
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4ub(co.r, co.g, co.b, co.a)
	gl.Begin(gl.QUADS)
	gl.Vertex2i(x, y)
	gl.Vertex2i(x, y+dy)
	gl.Vertex2i(x+dx, y+dy)
	gl.Vertex2i(x+dx, y)
	gl.End()
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
		for k := range selected {
			other = k
		}
		delete(selected, other)
		selected[index] = true
	}
	return
}

type chooseLayout struct {
	Background texture.Object

	Scroller struct {
		X, Y    int
		Texture texture.Object
	}

	Options ScrollingRegion

	Up, Down, Back, Next Button

	Info struct {
		X, Y, Dx, Dy int
	}
}

type OptionBasic struct {
	// TODO(tmckee:25): rename 'Id' to ScriptPath or smth
	Id           string
	HouseName    string
	Small        texture.Object
	Large        texture.Object
	Text         string
	Size         int
	alpha        byte
	was_over     bool
	was_selected bool
}

func (ob *OptionBasic) Scenario() Scenario {
	return Scenario{
		ob.Id,
		ob.HouseName,
	}
}

func (ob *OptionBasic) Draw(x, y, dx int) {
	render.WithColour(1, 1, 1, float32(ob.alpha)/255.0, func() {
		ob.Small.Data().RenderNatural(x, y)
	})
}

func (ob *OptionBasic) DrawInfo(x, y, dx, dy int) {
	shaderBank := globals.RenderQueueState().Shaders()
	render.WithColour(1, 1, 1, 1, func() {
		tx := x + (dx-ob.Large.Data().Dx())/2
		ty := y + dy - ob.Large.Data().Dy()
		ob.Large.Data().RenderNatural(tx, ty)
		d := base.GetDictionary(ob.Size)
		d.RenderParagraph(ob.Text, x, y+dy-ob.Large.Data().Dy()-d.MaxHeight(), dx, d.MaxHeight(), gui.Left, gui.Top, shaderBank)
	})
}

func (ob *OptionBasic) Height() int {
	return ob.Small.Data().Dy()
}

func (ob *OptionBasic) Think(hovered, selected, selectable bool, dt int64) {
	if selectable && hovered && !ob.was_over {
		sound.PlaySound("Haunts/SFX/UI/Tick", 0.75)
	}
	ob.was_over = hovered
	if ob.was_selected != selected {
		sound.PlaySound("Haunts/SFX/UI/Select", 0.75)
	}
	ob.was_selected = selected
	switch {
	case selected:
		ob.alpha = 255
	case selectable && hovered:
		ob.alpha = 200
	case selectable && !hovered:
		ob.alpha = 150
	default:
		ob.alpha = 50
	}
}

type Chooser struct {
	layout             chooseLayout
	region             gui.Region
	buttons            []*Button
	non_scroll_buttons []*Button
	options            []Option
	selected           map[int]bool
	selector           Selector
	min, max           int
	info_region        gui.Region
	mx, my             int

	last_t int64
}

func InsertMapChooser(ui gui.WidgetParent, chosen func(Scenario), resert func(ui gui.WidgetParent) error) error {
	var bops []OptionBasic
	datadir := base.GetDataDir()
	err := base.LoadAndProcessObject(filepath.Join(datadir, "ui", "start", "versus", "map_select.json"), "json", &bops)
	if err != nil {
		logging.Error("InsertMapChooser failed", "LoadAndProcessObject(map_select.json) error", err)
		return err
	}
	var opts []Option
	algorithm.Map(bops, &opts, func(ob OptionBasic) Option { return &ob })
	logging.Debug("InsertMapChooser", "options", opts)

	var ch Chooser
	err = base.LoadAndProcessObject(filepath.Join(datadir, "ui", "chooser", "layout.json"), "json", &ch.layout)
	if err != nil {
		logging.Error("InsertMapChooser failed", "LoadAndProcessObject(chooser/layout.json) error", err)
		return err
	}
	ch.options = opts
	ch.buttons = []*Button{
		&ch.layout.Up,
		&ch.layout.Down,
		&ch.layout.Back,
		&ch.layout.Next,
	}
	ch.non_scroll_buttons = []*Button{
		&ch.layout.Back,
		&ch.layout.Next,
	}
	ch.layout.Up.f = func(interface{}) {
		ch.layout.Options.Up()
	}
	ch.layout.Down.f = func(interface{}) {
		ch.layout.Options.Down()
	}
	ch.selected = make(map[int]bool)
	ch.layout.Back.f = func(interface{}) {
		ui.RemoveChild(&ch)
		err := resert(ui)
		if err != nil {
			logging.Error("InsertMapChooser", "resert failed", err)
			return
		}
	}
	ch.layout.Next.f = func(interface{}) {
		for i := range ch.options {
			if ch.selected[i] {
				ui.RemoveChild(&ch)
				chosen(ch.options[i].Scenario())
			}
		}
	}
	ch.layout.Next.valid_func = func() bool {
		return ch.selector(-1, ch.selected, false)
	}
	ch.min, ch.max = 1, 1
	if ch.min == 1 && ch.max == 1 {
		ch.selector = SelectExactlyOne
	} else {
		ch.selector = SelectInRange(ch.min, ch.max)
	}
	ch.info_region = gui.Region{
		Point: gui.Point{
			X: ch.layout.Info.X,
			Y: ch.layout.Info.Y,
		},
		Dims: gui.Dims{
			Dx: ch.layout.Info.Dx,
			Dy: ch.layout.Info.Dy,
		},
	}
	ui.AddChild(&ch)
	return nil
}

func MakeChooser(opts []Option) (*Chooser, <-chan []Scenario, error) {
	var ch Chooser
	datadir := base.GetDataDir()
	err := base.LoadAndProcessObject(filepath.Join(datadir, "ui", "chooser", "layout.json"), "json", &ch.layout)
	if err != nil {
		return nil, nil, err
	}
	ch.options = opts
	ch.buttons = []*Button{
		&ch.layout.Up,
		&ch.layout.Down,
		&ch.layout.Back,
		&ch.layout.Next,
	}
	ch.non_scroll_buttons = []*Button{
		&ch.layout.Back,
		&ch.layout.Next,
	}
	ch.layout.Up.f = func(interface{}) {
		ch.layout.Options.Up()
	}
	ch.layout.Down.f = func(interface{}) {
		ch.layout.Options.Down()
	}
	done := make(chan []Scenario, 1)
	ch.selected = make(map[int]bool)
	ch.layout.Back.f = func(interface{}) {
		// TODO(tmckee#42): this means clicking the 'back' button will send 'nil'
		// for the scenario. This seems to be causing issue #42.
		done <- nil
		close(done)
	}
	ch.layout.Next.f = func(interface{}) {
		var res []Scenario
		for i := range ch.options {
			if ch.selected[i] {
				res = append(res, ch.options[i].Scenario())
			}
		}
		done <- res
		close(done)
	}
	ch.layout.Next.valid_func = func() bool {
		return ch.selector(-1, ch.selected, false)
	}
	ch.min, ch.max = 1, 1
	if ch.min == 1 && ch.max == 1 {
		ch.selector = SelectExactlyOne
	} else {
		ch.selector = SelectInRange(ch.min, ch.max)
	}
	ch.info_region = gui.Region{
		Point: gui.Point{
			X: ch.layout.Info.X,
			Y: ch.layout.Info.Y,
		},
		Dims: gui.Dims{
			Dx: ch.layout.Info.Dx,
			Dy: ch.layout.Info.Dy,
		},
	}
	return &ch, done, nil
}

func (c *Chooser) optionsHeight() int {
	h := 0
	for _, options := range c.options {
		h += options.Height()
	}
	return h
}

type doOnOptionData struct {
	hovered      bool
	selected     bool
	selectable   bool
	x, y, dx, dy int
}

func (c *Chooser) doOnOptions(f func(index int, opt Option, data doOnOptionData)) {
	var data doOnOptionData
	data.x = c.layout.Options.X + c.region.X
	data.y = c.region.Y + c.layout.Options.Top()
	data.dx = c.layout.Options.Dx
	in_box := pointInsideRect(c.mx, c.my, c.layout.Options.X, c.layout.Options.Y, c.layout.Options.Dx, c.layout.Options.Dy)
	for i, option := range c.options {
		data.dy = option.Height()
		data.y -= data.dy
		data.hovered = in_box && pointInsideRect(c.mx, c.my, data.x, data.y, data.dx, data.dy)
		data.selected = c.selected[i]
		data.selectable = c.selector(i, c.selected, false)
		f(i, option, data)
	}
}

func assymptoticApproach(cur, target float64, dt int64) float64 {
	delta := target - cur
	delta *= math.Exp(-20.0 / float64(1+dt))
	cur += delta
	if math.Abs(cur-target) < 1e-2 {
		return target
	}
	return cur
}

func (c *Chooser) Requested() gui.Dims {
	return gui.Dims{
		Dx: 1024,
		Dy: 768,
	}
}

func (c *Chooser) Expandable() (bool, bool) {
	return false, false
}

func (c *Chooser) Rendered() gui.Region {
	return c.region
}

func (c *Chooser) Think(g *gui.Gui, t int64) {
	if c.last_t == 0 {
		c.last_t = t
		return
	}
	dt := t - c.last_t
	c.last_t = t
	c.layout.Options.Height = c.optionsHeight()
	c.layout.Options.Think(dt)
	buttons := c.buttons
	if c.optionsHeight() <= c.layout.Options.Dy {
		buttons = c.non_scroll_buttons
	}
	for _, button := range buttons {
		button.Think(c.region.X, c.region.Y, c.mx, c.my, dt)
	}
	c.doOnOptions(func(index int, opt Option, data doOnOptionData) {
		opt.Think(data.hovered, data.selected, data.selectable, dt)
	})
}

func (c *Chooser) Respond(g *gui.Gui, group gui.EventGroup) bool {
	if mpos, ok := g.UseMousePosition(group); ok {
		c.mx, c.my = mpos.X, mpos.Y
	}
	if group.IsPressed(gin.AnyMouseLButton) {
		buttons := c.buttons
		if c.optionsHeight() <= c.layout.Options.Dy {
			buttons = c.non_scroll_buttons
		}
		for _, button := range buttons {
			if button.handleClick(c.mx, c.my, nil) {
				return true
			}
			clicked := false
			c.doOnOptions(func(index int, opt Option, data doOnOptionData) {
				if clicked {
					return
				}
				if data.hovered {
					c.selector(index, c.selected, true)
					clicked = true
				}
			})
		}
	}
	return false
}

func (c *Chooser) Draw(region gui.Region, ctx gui.DrawingContext) {
	c.region = region
	render.WithColour(1, 1, 1, 1, func() {
		c.layout.Background.Data().RenderNatural(region.X, region.Y)
		// TODO(#54): what do we want to draw here? Perhaps a scroll bar? For now,
		// it's goggles 🙃
		tex := c.layout.Scroller.Texture.Data()
		tex.RenderNatural(region.X+c.layout.Scroller.X, region.Y+c.layout.Scroller.Y)

		buttons := c.buttons
		if c.optionsHeight() <= c.layout.Options.Dy {
			buttons = c.non_scroll_buttons
		}
		for _, button := range buttons {
			button.RenderAt(region.X, region.Y)
		}

		c.layout.Options.Region().PushClipPlanes()
		hovered := -1
		c.doOnOptions(func(index int, opt Option, data doOnOptionData) {
			if data.hovered {
				hovered = index
			}
			opt.Draw(data.x, data.y, data.dx)
		})
		c.layout.Options.Region().PopClipPlanes()
		c.info_region.PushClipPlanes()
		if hovered != -1 {
			c.options[hovered].DrawInfo(c.layout.Info.X, c.layout.Info.Y, c.layout.Info.Dx, c.layout.Info.Dy)
		} else {
			if c.min == 1 && c.max == 1 && len(c.selected) == 1 {
				var index int
				for index = range c.selected {
				}
				c.options[index].DrawInfo(c.layout.Info.X, c.layout.Info.Y, c.layout.Info.Dx, c.layout.Info.Dy)
			}
		}
		c.info_region.PopClipPlanes()
	})
}

func (c *Chooser) DrawFocused(region gui.Region, ctx gui.DrawingContext) {
	c.Draw(region, ctx)
}

func (c *Chooser) String() string {
	return "chooser"
}
