package house

import (
	"unsafe"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/MobRulesGames/mathgl"
	"github.com/go-gl-legacy/gl"
)

func LoadWallTexture(name string) *WallTexture {
	wt := WallTexture{Defname: name}
	wt.Load()
	return &wt
}

func GetAllWallTextureNames() []string {
	return base.GetAllNamesInRegistry("wall_textures")
}

func LoadAllWallTexturesInDir(dir string) {
	base.RemoveRegistry("wall_textures")
	base.RegisterRegistry("wall_textures", make(map[string]*WallTextureDef))
	base.RegisterAllObjectsInDir("wall_textures", dir, ".json", "json")
}

func (wt *WallTexture) Load() {
	base.GetObject("wall_textures", wt)
}

type wallTextureGlIDs struct {
	vBuffer uint32

	leftBuffer  uint32
	leftCount   gl.GLsizei
	rightBuffer uint32
	rightCount  gl.GLsizei
	floorBuffer uint32
	floorCount  gl.GLsizei
}

type wallTextureState struct {
	// for tracking whether the buffers are dirty
	x, y, rot float32
	flip      bool
	room      struct {
		x, y, dx, dy int
	}
}

type WallTexture struct {
	Defname string
	*WallTextureDef

	// Position of the texture in floor coordinates.  If these coordinates exceed
	// either the dx or dy of the room, then this texture will be drawn, at least
	// partially, on the wall.  The coordinates should not both exceed the
	// dimensions of the room.
	X, Y float32
	Rot  float32

	// Whether or not to flip the texture about one of its axes
	Flip bool

	// If this is currently being dragged around it will be marked as temporary
	// so that it will be drawn differently
	temporary bool
}

type WallTextureDef struct {
	// Name of this texture as it appears in the editor, should be unique among
	// all WallTextures
	Name string

	Texture texture.Object
}

func (wt *WallTexture) Color() (r, g, b, a byte) {
	if wt.temporary {
		return 127, 127, 255, 200
	}
	return 255, 255, 255, 255
}

func (wt *WallTexture) Render() {
	data := wt.Texture.Data()
	dx, dy := data.Dx(), data.Dy()
	wt.Texture.Data().RenderAdvanced(float64(wt.X), float64(wt.Y), float64(dx), float64(dy), float64(wt.Rot), wt.Flip)
}

func (wt *WallTexture) setupGlStuff(x, y, dx, dy int, glIDs *wallTextureGlIDs) {
	if glIDs.vBuffer != 0 {
		gl.Buffer(glIDs.vBuffer).Delete()
		gl.Buffer(glIDs.leftBuffer).Delete()
		gl.Buffer(glIDs.rightBuffer).Delete()
		gl.Buffer(glIDs.floorBuffer).Delete()
		glIDs.vBuffer = 0
		glIDs.leftBuffer = 0
		glIDs.rightBuffer = 0
		glIDs.floorBuffer = 0
	}

	// All vertices for both walls and the floor will go here and get sent to
	// opengl all at once
	var vs []roomVertex

	// Conveniently casted values
	frx := float32(x)
	fry := float32(y)
	frdx := float32(dx)
	frdy := float32(dy)
	tdx := float32(wt.Texture.Data().Dx()) / 100
	tdy := float32(wt.Texture.Data().Dy()) / 100

	wtx := wt.X
	wty := wt.Y
	wtr := wt.Rot

	if wtx > frdx {
		wtr -= 3.1415926535 / 2
	}

	// Floor
	verts := []mathgl.Vec2{
		{X: -tdx / 2, Y: -tdy / 2},
		{X: -tdx / 2, Y: tdy / 2},
		{X: tdx / 2, Y: tdy / 2},
		{X: tdx / 2, Y: -tdy / 2},
	}
	var m, run mathgl.Mat3
	run.Identity()
	m.Translation(wtx, wty)
	run.Multiply(&m)
	m.RotationZ(wtr)
	run.Multiply(&m)
	if wt.Flip {
		m.Scaling(-1, 1)
		run.Multiply(&m)
	}
	for i := range verts {
		verts[i].Transform(&run)
	}
	p := mathgl.Poly(verts)
	p.Clip(&mathgl.Seg2{A: mathgl.Vec2{X: 0, Y: 0}, B: mathgl.Vec2{X: 0, Y: frdy}})
	p.Clip(&mathgl.Seg2{A: mathgl.Vec2{X: 0, Y: frdy}, B: mathgl.Vec2{X: frdx, Y: frdy}})
	p.Clip(&mathgl.Seg2{A: mathgl.Vec2{X: frdx, Y: frdy}, B: mathgl.Vec2{X: frdx, Y: 0}})
	p.Clip(&mathgl.Seg2{A: mathgl.Vec2{X: frdx, Y: 0}, B: mathgl.Vec2{X: 0, Y: 0}})
	if len(p) >= 3 {
		// floor indices
		var is []uint16
		for i := 1; i < len(p)-1; i++ {
			is = append(is, uint16(len(vs)+0))
			is = append(is, uint16(len(vs)+i))
			is = append(is, uint16(len(vs)+i+1))
		}
		// TODO(tmckee): don't store uint32 and cast-to-buffer; just store a
		// gl.Buffer
		glIDs.floorBuffer = uint32(gl.GenBuffer())
		gl.Buffer(glIDs.floorBuffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, (int(unsafe.Sizeof(is[0])) * len(is)), is, gl.STATIC_DRAW)
		glIDs.floorCount = gl.GLsizei(len(is))

		run.Inverse()
		for i := range p {
			v := mathgl.Vec2{X: p[i].X, Y: p[i].Y}
			v.Transform(&run)
			vs = append(vs, roomVertex{
				x:     p[i].X,
				y:     p[i].Y,
				u:     v.X/tdx + 0.5,
				v:     -(v.Y/tdy + 0.5),
				los_u: (fry + p[i].Y) / LosTextureSize,
				los_v: (frx + p[i].X) / LosTextureSize,
			})
		}
	}

	// Left Wall
	verts = []mathgl.Vec2{
		{X: -tdx / 2, Y: -tdy / 2},
		{X: -tdx / 2, Y: tdy / 2},
		{X: tdx / 2, Y: tdy / 2},
		{X: tdx / 2, Y: -tdy / 2},
	}
	run.Identity()
	m.Translation(wtx, wty)
	run.Multiply(&m)
	m.RotationZ(wtr)
	run.Multiply(&m)
	if wt.Flip {
		m.Scaling(-1, 1)
		run.Multiply(&m)
	}
	for i := range verts {
		verts[i].Transform(&run)
	}
	p = mathgl.Poly(verts)
	p.Clip(&mathgl.Seg2{A: mathgl.Vec2{X: 0, Y: 0}, B: mathgl.Vec2{X: 0, Y: frdy}})
	p.Clip(&mathgl.Seg2{B: mathgl.Vec2{X: 0, Y: frdy}, A: mathgl.Vec2{X: frdx, Y: frdy}})
	p.Clip(&mathgl.Seg2{A: mathgl.Vec2{X: frdx, Y: frdy}, B: mathgl.Vec2{X: frdx, Y: 0}})
	if len(p) >= 3 {
		// left wall indices
		var is []uint16
		for i := 1; i < len(p)-1; i++ {
			is = append(is, uint16(len(vs)+0))
			is = append(is, uint16(len(vs)+i))
			is = append(is, uint16(len(vs)+i+1))
		}
		glIDs.leftBuffer = uint32(gl.GenBuffer())
		gl.Buffer(glIDs.leftBuffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(unsafe.Sizeof(is[0]))*len(is), is, gl.STATIC_DRAW)
		glIDs.leftCount = gl.GLsizei(len(is))

		run.Inverse()
		for i := range p {
			v := mathgl.Vec2{X: p[i].X, Y: p[i].Y}
			v.Transform(&run)
			vs = append(vs, roomVertex{
				x:     p[i].X,
				y:     frdy,
				z:     frdy - p[i].Y,
				u:     v.X/tdx + 0.5,
				v:     -(v.Y/tdy + 0.5),
				los_u: (fry + frdy - 0.5) / LosTextureSize,
				los_v: (frx + p[i].X) / LosTextureSize,
			})
		}
	}

	// Right Wall
	verts = []mathgl.Vec2{
		{X: -tdx / 2, Y: -tdy / 2},
		{X: -tdx / 2, Y: tdy / 2},
		{X: tdx / 2, Y: tdy / 2},
		{X: tdx / 2, Y: -tdy / 2},
	}
	run.Identity()
	m.Translation(wtx, wty)
	run.Multiply(&m)
	m.RotationZ(wtr)
	run.Multiply(&m)
	if wt.Flip {
		m.Scaling(-1, 1)
		run.Multiply(&m)
	}
	for i := range verts {
		verts[i].Transform(&run)
	}
	p = mathgl.Poly(verts)
	p.Clip(&mathgl.Seg2{A: mathgl.Vec2{X: 0, Y: frdy}, B: mathgl.Vec2{X: frdx, Y: frdy}})
	p.Clip(&mathgl.Seg2{B: mathgl.Vec2{X: frdx, Y: frdy}, A: mathgl.Vec2{X: frdx, Y: 0}})
	p.Clip(&mathgl.Seg2{A: mathgl.Vec2{X: frdx, Y: 0}, B: mathgl.Vec2{X: 0, Y: 0}})
	if len(p) >= 3 {
		// right wall indices
		var is []uint16
		for i := 1; i < len(p)-1; i++ {
			is = append(is, uint16(len(vs)+0))
			is = append(is, uint16(len(vs)+i))
			is = append(is, uint16(len(vs)+i+1))
		}
		glIDs.rightBuffer = uint32(gl.GenBuffer())
		gl.Buffer(glIDs.rightBuffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(unsafe.Sizeof(is[0]))*len(is), is, gl.STATIC_DRAW)
		glIDs.rightCount = gl.GLsizei(len(is))

		run.Inverse()
		for i := range p {
			v := mathgl.Vec2{X: p[i].X, Y: p[i].Y}
			v.Transform(&run)
			vs = append(vs, roomVertex{
				x:     frdx,
				y:     p[i].Y,
				z:     frdx - p[i].X,
				u:     v.X/tdx + 0.5,
				v:     -(v.Y/tdy + 0.5),
				los_u: (fry + p[i].Y) / LosTextureSize,
				los_v: (frx + frdx - 0.5) / LosTextureSize,
			})
		}
	}

	if len(vs) > 0 {
		glIDs.vBuffer = uint32(gl.GenBuffer())
		gl.Buffer(glIDs.vBuffer).Bind(gl.ARRAY_BUFFER)
		size := int(unsafe.Sizeof(roomVertex{}))
		gl.BufferData(gl.ARRAY_BUFFER, size*len(vs), vs, gl.STATIC_DRAW)
	}
}
