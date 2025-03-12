package house

import (
	"math"
	"unsafe"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/logging"
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
	vBuffer gl.Buffer

	leftIBuffer  gl.Buffer
	leftICount   gl.GLsizei
	rightIBuffer gl.Buffer
	rightICount  gl.GLsizei
	floorIBuffer gl.Buffer
	floorICount  gl.GLsizei
}

func (ids *wallTextureGlIDs) Reset() {
	deleteIfNeeded := func(buffid *gl.Buffer) {
		if *buffid == 0 {
			return
		}
		buffid.Delete()
		*buffid = 0
	}

	deleteIfNeeded(&ids.vBuffer)
	deleteIfNeeded(&ids.leftIBuffer)
	deleteIfNeeded(&ids.rightIBuffer)
	deleteIfNeeded(&ids.floorIBuffer)
}

func (ids *wallTextureGlIDs) setVertexData(verts []roomVertex) {
	ids.vBuffer = gl.GenBuffer()
	ids.vBuffer.Bind(gl.ARRAY_BUFFER)
	size := int(unsafe.Sizeof(roomVertex{}))
	gl.BufferData(gl.ARRAY_BUFFER, size*len(verts), verts, gl.STATIC_DRAW)
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

	// Whether or not to mirror the decal about a vertical midline.
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
	data.RenderAdvanced(float64(wt.X), float64(wt.Y), float64(dx), float64(dy), float64(wt.Rot), wt.Flip)
}

func (wt *WallTexture) setupGlStuff(roomX, roomY, roomDx, roomDy int, glIDs *wallTextureGlIDs) {
	glIDs.Reset()

	// If we panic, don't leak GL resources
	defer func() {
		if e := recover(); e != nil {
			glIDs.Reset()
			panic(e)
		}
	}()

	// All vertices for both walls and the floor will go here and get sent to
	// opengl all at once
	var vs []roomVertex

	// Conveniently casted values
	frx := float32(roomX)
	fry := float32(roomY)
	frdx := float32(roomDx)
	frdy := float32(roomDy)

	// TODO(tmckee): we _were_ dividing these by 100 ... why?
	// Maybe the source images for the textures were much larger than the
	// expected size of the decal?
	tdx := float32(wt.Texture.Data().Dx())
	tdy := float32(wt.Texture.Data().Dy())

	wtx := wt.X
	wty := wt.Y
	wtr := wt.Rot

	// If the wall-texture is positioned to the right of the room's floor, that
	// means it's on the wall to the viewer's right. Rotate it 90 degrees
	// clockwise about the z-axis so that when we 'stand it up', the texture will
	// be right-side up w.r.t. the rest of the scene.
	if wtx > frdx {
		wtr -= math.Pi / 2
	}

	logging.Trace("wall texture>setupGlStuff", "all the stuff", map[string]any{
		"frx":  frx,
		"fry":  fry,
		"frdx": frdx,
		"frdy": frdy,
		"tdx":  tdx,
		"tdy":  tdy,
		"wtx":  wtx,
		"wty":  wty,
		"wtr":  wtr,
	})

	// Build geometry for each of the three surfaces that might have a piece of
	// the decal.

	// Floor
	// TODO(tmckee): readability: we can factor out a "gimme a polygon,
	// transformed by such-and-such a matrix, clipped by such-and-such extents".
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
	p.Clip(&mathgl.Seg2{
		// Clip out things to the left of the floor.
		A: mathgl.Vec2{X: 0, Y: 0},
		B: mathgl.Vec2{X: 0, Y: frdy},
	})
	p.Clip(&mathgl.Seg2{
		// Clip out things above the floor.
		A: mathgl.Vec2{X: 0, Y: frdy},
		B: mathgl.Vec2{X: frdx, Y: frdy},
	})
	p.Clip(&mathgl.Seg2{
		// Clip out things to the right of the floor.
		A: mathgl.Vec2{X: frdx, Y: frdy},
		B: mathgl.Vec2{X: frdx, Y: 0},
	})
	p.Clip(&mathgl.Seg2{
		// Clip out things below the floor.
		A: mathgl.Vec2{X: frdx, Y: 0},
		B: mathgl.Vec2{X: 0, Y: 0},
	})
	// TODO(tmckee): readability: (end-of-refactor noted above)
	if len(p) >= 3 {
		// floor indices
		var is []uint16
		for i := 1; i < len(p)-1; i++ {
			is = append(is, uint16(len(vs)+0))
			is = append(is, uint16(len(vs)+i))
			is = append(is, uint16(len(vs)+i+1))
		}

		glIDs.floorIBuffer = gl.GenBuffer()
		glIDs.floorIBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, (int(unsafe.Sizeof(is[0])) * len(is)), is, gl.STATIC_DRAW)
		glIDs.floorICount = gl.GLsizei(len(is))

		run.Inverse()
		for _, polyVert := range p {
			preimage := mathgl.Vec2{X: polyVert.X, Y: polyVert.Y}
			preimage.Transform(&run)
			vs = append(vs, roomVertex{
				x:     polyVert.X,
				y:     polyVert.Y,
				u:     preimage.X/tdx + 0.5,
				v:     -(preimage.Y/tdy + 0.5),
				los_u: (fry + polyVert.Y) / LosTextureSize,
				los_v: (frx + polyVert.X) / LosTextureSize,
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
	p.Clip(&mathgl.Seg2{
		A: mathgl.Vec2{X: 0, Y: 0},
		B: mathgl.Vec2{X: 0, Y: frdy},
	})
	p.Clip(&mathgl.Seg2{
		B: mathgl.Vec2{X: 0, Y: frdy},
		A: mathgl.Vec2{X: frdx, Y: frdy},
	})
	p.Clip(&mathgl.Seg2{
		A: mathgl.Vec2{X: frdx, Y: frdy},
		B: mathgl.Vec2{X: frdx, Y: 0},
	})
	if len(p) >= 3 {
		// left wall indices
		var is []uint16
		for i := 1; i < len(p)-1; i++ {
			is = append(is, uint16(len(vs)+0))
			is = append(is, uint16(len(vs)+i))
			is = append(is, uint16(len(vs)+i+1))
		}
		glIDs.leftIBuffer = gl.GenBuffer()
		glIDs.leftIBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(unsafe.Sizeof(is[0]))*len(is), is, gl.STATIC_DRAW)
		glIDs.leftICount = gl.GLsizei(len(is))

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

	// Because the texture geometry starts centred about the origin, we can
	// combine any aesthic rotations with the rotation needed to align our decal
	// with the right-hand wall
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
	p.Clip(&mathgl.Seg2{
		// Clip out whatever is north of the top of the floor; anything in that
		// region is outside this room.
		A: mathgl.Vec2{X: 0, Y: frdy},
		B: mathgl.Vec2{X: frdx, Y: frdy},
	})
	p.Clip(&mathgl.Seg2{
		// Clip out whatever is west of the right-most edge of the floor; anything
		// in that region is either on the floor or outside of the room.
		B: mathgl.Vec2{X: frdx, Y: frdy},
		A: mathgl.Vec2{X: frdx, Y: 0},
	})
	p.Clip(&mathgl.Seg2{
		// Clip out whatever is south of the bottom of the floor; anything in that
		// region is outside this room.
		A: mathgl.Vec2{X: frdx, Y: 0},
		B: mathgl.Vec2{X: 0, Y: 0},
	})

	// We may have clipped the entire decal out; skip buffering data if there's
	// nothing left in the polygon.
	if len(p) >= 3 {
		// right wall indices
		var is []uint16
		for i := 1; i < len(p)-1; i++ {
			is = append(is, uint16(len(vs)+0))
			is = append(is, uint16(len(vs)+i))
			is = append(is, uint16(len(vs)+i+1))
		}
		glIDs.rightIBuffer = gl.GenBuffer()
		glIDs.rightIBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(unsafe.Sizeof(is[0]))*len(is), is, gl.STATIC_DRAW)
		glIDs.rightICount = gl.GLsizei(len(is))

		run.Inverse()
		for _, polyVert := range p {
			preImage := mathgl.Vec2{X: polyVert.X, Y: polyVert.Y}
			preImage.Transform(&run)
			vs = append(vs, roomVertex{
				// everything on the right wall exists at the right-edge of the room.
				x: frdx,

				// the left-right co-ordinate along the wall is how far the polygon's
				// vertex is away from the 'right' edge of the wall.
				y: polyVert.Y,

				// the top-bottom co-ordinate on the wall is how far the polygon's
				// vertex is from the 'bottom' edge of the wall.
				z: frdx - polyVert.X,

				// transform from centred at (0,0) to an origin in the bottom-left of
				// the texture image.
				u: preImage.X/tdx + 0.5,
				v: -(preImage.Y/tdy + 0.5),

				// TODO: grok LoS shading and inputs
				los_u: (fry + polyVert.Y) / LosTextureSize,
				los_v: (frx + frdx - 0.5) / LosTextureSize,
			})
		}
	}

	if len(vs) != 0 {
		glIDs.setVertexData(vs)
	}
}
