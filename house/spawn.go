package house

import (
	"fmt"
	"hash/fnv"
	"image/color"
	"path"
	"regexp"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/MobRulesGames/mathgl"
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/debug"
)

var spawn_regex []*regexp.Regexp

func PushSpawnRegexp(pattern string) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		panic(fmt.Errorf("bad regexp pattern: %q, err: %w", pattern, err))
	}
	spawn_regex = append(spawn_regex, re)
}

func PopSpawnRegexp() {
	if len(spawn_regex) == 0 {
		panic(fmt.Errorf("tried to pop empty regex stack"))
	}
	spawn_regex = spawn_regex[0 : len(spawn_regex)-1]
}

func topSpawnRegexp() *regexp.Regexp {
	if len(spawn_regex) == 0 {
		panic(fmt.Errorf("tried to peek at empty regex stack"))
	}
	return spawn_regex[len(spawn_regex)-1]
}

type SpawnPoint struct {
	Name   string
	Dx, Dy int
	X, Y   int
	Tex    texture.Object

	// just for the shader
	temporary, invalid bool
}

func (sp *SpawnPoint) Dims() (int, int) {
	return sp.Dx, sp.Dy
}

func (sp *SpawnPoint) Pos() (int, int) {
	return sp.X, sp.Y
}

func (sp *SpawnPoint) FPos() (float64, float64) {
	return float64(sp.X), float64(sp.Y)
}

func (sp *SpawnPoint) Color() (r, g, b, a byte) {
	return 255, 255, 255, 255
}

func (sp *SpawnPoint) Render(pos mathgl.Vec2, width float32) {
	gl.Disable(gl.TEXTURE_2D)
	gl.Color4d(1, 1, 1, 0.1)
	gl.Begin(gl.QUADS)
	gl.Vertex2f(pos.X-width/2, pos.Y)
	gl.Vertex2f(pos.X-width/2, pos.Y+width)
	gl.Vertex2f(pos.X+width/2, pos.Y+width)
	gl.Vertex2f(pos.X+width/2, pos.Y)
	gl.End()
}
func (sp *SpawnPoint) RenderOnFloor() {
	re := topSpawnRegexp()
	if !re.MatchString(sp.Name) {
		logging.Debug("skipping spawn", "re", re, "spawn", sp)
		return
	}

	logging.Trace("SpawnPoint.RenderOnFloor not skipping", "spawn", sp)
	var rgba [4]float64
	gl.GetDoublev(gl.CURRENT_COLOR, rgba[:])
	gl.PushAttrib(gl.CURRENT_BIT)
	gl.Disable(gl.TEXTURE_2D)

	// This just creates a color that is consistent among all spawn points whose
	// names match SpawnName-.*
	prefix := sp.Name
	for i := range prefix {
		if prefix[i] == '-' {
			prefix = prefix[0:i]
			break
		}
	}

	h := fnv.New32()
	h.Write([]byte(prefix))
	hs := h.Sum32()
	colour := color.NRGBA{}
	colour.R = uint8(hs % 256)
	hs = hs >> 8
	colour.G = uint8(hs % 256)
	hs = hs >> 8
	colour.B = uint8(hs % 256)
	hs = hs >> 8
	colour.A = uint8(hs % 256)
	// gl.Color4ub(, uint8((hs/256)%256), uint8((hs/(256*256))%256), uint8(255*rgba[3]))

	gl.Color4ub(255, 0, 0, 255)
	logging.Trace("glstate", "glstate", debug.GetGlState(), "colour", colour, "for now", "red")

	/*
		base.EnableShader("box")
		base.SetUniformF("box", "dx", float32(sp.Dx))
		base.SetUniformF("box", "dy", float32(sp.Dy))
		if !sp.temporary {
			base.SetUniformI("box", "temp_invalid", 0)
		} else if !sp.invalid {
			base.SetUniformI("box", "temp_invalid", 1)
		} else {
			base.SetUniformI("box", "temp_invalid", 2)
		}
	*/
	gl.Enable(gl.TEXTURE_2D)
	sp.Tex.ResetPath(base.Path(path.Join(base.GetDataDir(), "textures/pentagram_04_large_red.png")))
	sp.Tex.Data().Render(float64(sp.X), float64(sp.Y), float64(sp.Dx), float64(sp.Dy))
	//base.EnableShader("")
	gl.PopAttrib()
}
