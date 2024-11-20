package base

import (
	"bufio"
	"strings"
	"unicode"

	"github.com/MobRulesGames/opengl/gl"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
)

const maxLines = 25
const maxLineLength = 150

// A simple gui element that will display the last several lines of text from
// a log file (TODO: and also allow you to enter some basic commands).
type Console struct {
	gui.BasicZone
	lines      [maxLines]string
	start, end int
	// TODO(tmckee): clean: use a gui.Point instead of a raw X co-ordinate.
	xscroll int

	input *bufio.Reader
	cmd   []byte
}

func MakeConsole() *Console {
	if log_console == nil {
		panic("Cannot make a console until the logging system has been set up.")
	}
	var c Console
	c.BasicZone.Ex = true
	c.BasicZone.Ey = true
	c.BasicZone.Request_dims = gui.Dims{1000, 1000}
	c.input = bufio.NewReader(log_console)
	return &c
}

func (c *Console) String() string {
	return "console"
}

func (c *Console) Think(ui *gui.Gui, dt int64) {
	for line, _, err := c.input.ReadLine(); err == nil; line, _, err = c.input.ReadLine() {
		c.lines[c.end] = string(line)
		c.end = (c.end + 1) % len(c.lines)
		if c.start == c.end {
			c.start = (c.start + 1) % len(c.lines)
		}
	}
}

func (c *Console) Respond(ui *gui.Gui, group gui.EventGroup) bool {
	if found, event := group.FindEvent(GetDefaultKeyMap()["console"].Id()); found && event.Type == gin.Press {
		if group.Focus {
			ui.DropFocus()
		} else {
			ui.TakeFocus(c)
		}
		return true
	}
	if found, event := group.FindEvent(gin.AnyLeft); found && event.Type == gin.Press {
		c.xscroll += 250
	}
	if found, event := group.FindEvent(gin.AnyRight); found && event.Type == gin.Press {
		c.xscroll -= 250
	}
	if c.xscroll > 0 {
		c.xscroll = 0
	}
	if found, event := group.FindEvent(gin.AnySpace); found && event.Type == gin.Press {
		c.xscroll = 0
	}

	if group.Events[0].Type == gin.Press {
		r := rune(group.Events[0].Key.Id().Index)
		if r < 256 {
			if gin.In().GetKey(gin.AnyLeftShift).IsDown() || gin.In().GetKey(gin.AnyRightShift).IsDown() {
				r = unicode.ToUpper(r)
			}
			c.cmd = append(c.cmd, byte(r))
		}
	}

	return group.Focus
}

func (c *Console) Draw(region gui.Region, ctx gui.DrawingContext) {
}

func (c *Console) DrawFocused(region gui.Region, ctx gui.DrawingContext) {
	// TODO(tmckee): is 'standard_18' correct here?
	dict := ctx.GetDictionary("standard_18")
	gl.Color4d(0.2, 0, 0.3, 0.8)
	gl.Disable(gl.TEXTURE_2D)
	gl.Begin(gl.QUADS)
	gl.Vertex2i(region.X, region.Y)
	gl.Vertex2i(region.X, region.Y+region.Dy)
	gl.Vertex2i(region.X+region.Dx, region.Y+region.Dy)
	gl.Vertex2i(region.X+region.Dx, region.Y)
	gl.End()
	gl.Color4d(1, 1, 1, 1)
	y := region.Y + len(c.lines)*dict.MaxHeight()
	do_color := func(line string) {
		if strings.HasPrefix(line, "LOG") {
			gl.Color4d(1, 1, 1, 1)
		}
		if strings.HasPrefix(line, "WARN") {
			gl.Color4d(1, 1, 0, 1)
		}
		if strings.HasPrefix(line, "ERROR") {
			gl.Color4d(1, 0, 0, 1)
		}
	}
	// TODO(tmckee): expose the glop.font shader id instead of hardcoding here.
	shaderBank := ctx.GetShaders("glop.font")
	if c.start > c.end {
		for i := c.start; i < len(c.lines); i++ {
			do_color(c.lines[i])
			dict.RenderString(c.lines[i], gui.Point{X: c.xscroll, Y: y}, dict.MaxHeight(), gui.Left, shaderBank)
			y -= dict.MaxHeight()
		}
		for i := 0; i < c.end; i++ {
			do_color(c.lines[i])
			dict.RenderString(c.lines[i], gui.Point{X: c.xscroll, Y: y}, dict.MaxHeight(), gui.Left, shaderBank)
			y -= dict.MaxHeight()
		}
	} else {
		for i := c.start; i < c.end && i < len(c.lines); i++ {
			do_color(c.lines[i])
			dict.RenderString(c.lines[i], gui.Point{X: c.xscroll, Y: y}, dict.MaxHeight(), gui.Left, shaderBank)
			y -= dict.MaxHeight()
		}
	}
	dict.RenderString(string(c.cmd), gui.Point{X: c.xscroll, Y: y}, dict.MaxHeight(), gui.Left, shaderBank)
}
