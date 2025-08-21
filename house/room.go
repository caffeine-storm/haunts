package house

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/house/perspective"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/MobRulesGames/mathgl"
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/debug"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
)

func GetAllRoomNames() []string {
	return base.GetAllNamesInRegistry("rooms")
}

func LoadAllRoomsInDir(dir string) {
	base.RemoveRegistry("rooms")
	base.RegisterRegistry("rooms", make(map[string]*RoomDef))
	base.RegisterAllObjectsInDir("rooms", dir, ".room", "json")
}

var (
	datadir string
	tags    Tags
)

// Sets the directory to/from which all data will be saved/loaded
// Also immediately loads/reloads all Tags
func SetDatadir(_datadir string) error {
	datadir = _datadir
	return loadTags()
}

func loadTags() error {
	return base.LoadJson(filepath.Join(datadir, "tags.json"), &tags)
}

type RoomSize struct {
	Name   string
	Dx, Dy BoardSpaceUnit
}

func (r *RoomSize) GetDx() BoardSpaceUnit {
	return r.Dx
}

func (r *RoomSize) GetDy() BoardSpaceUnit {
	return r.Dy
}

func (r RoomSize) String() string {
	return fmt.Sprintf(r.format(), r.Name, r.Dx, r.Dy)
}
func (r *RoomSize) Scan(str string) {
	fmt.Sscanf(str, r.format(), &r.Name, &r.Dx, &r.Dy)
}
func (r *RoomSize) format() string {
	return "%s (%d, %d)"
}

type Tags struct {
	Themes     []string
	RoomSizes  []RoomSize
	HouseSizes []string
	Decor      []string
}

type RoomDef struct {
	Name string
	Size RoomSize

	Furniture []*Furniture `registry:"loadfrom-furniture"`

	Decals []*Decal `registry:"loadfrom-decals"`

	Floor texture.Object
	Wall  texture.Object

	// What house themes this room is appropriate for
	Themes map[string]bool

	// What house sizes this room is appropriate for
	Sizes map[string]bool

	// What kinds of decorations are appropriate in this room
	Decor map[string]bool
}

type Room struct {
	Defname string
	*RoomDef

	// The placement of doors in this room
	Doors []*Door `registry:"loadfrom-doors"`

	// The offset of this room on this floor
	X, Y BoardSpaceUnit

	temporary, invalid bool

	// whether or not to draw the walls transparent
	far_left struct {
		wall_alpha byte
	}
	far_right struct {
		wall_alpha byte
	}

	glData struct {
		// Vertex buffer storing the vertices of the room as well as the texture
		// coordinates for the los texture.
		vBuffer gl.Buffer

		// Buffers of indices referencing the data in vBuffer.
		leftWallIBuffer  gl.Buffer // assumed to be 6 indices long
		rightWallIBuffer gl.Buffer // assumed to be 6 indices long

		floorIBuffer gl.Buffer
		// TODO(tmckee): we shouldn't need to store floorICount; it's always 9
		// quads, a.k.a. 54 vertices.
		floorICount int // holds how many indices are in 'floorIBuffer'
	}

	// We only need to rebuild 'glData' if there was a change to one of the
	// relevant inputs.
	glDataInputs struct {
		x, y, dx, dy       BoardSpaceUnit
		decal_dx, decal_dy BoardSpaceUnit
	}

	decal_gl_map    map[*Decal]decalGlIDs
	decal_state_map map[*Decal]decalState
}

func BlankRoomSize() *RoomSize {
	return &RoomSize{
		Name: "Blank(Small)",
		Dx:   10,
		Dy:   10,
	}
}

func BlankRoom() *Room {
	return &Room{
		Defname: "blank",
		RoomDef: &RoomDef{
			Name: "blank",
			Size: *BlankRoomSize(),
		},
	}
}

func (room *Room) Color() (r, g, b, a byte) {
	if room.temporary {
		if room.invalid {
			return 255, 127, 127, 200
		} else {
			return 127, 127, 255, 200
		}
	}
	return 255, 255, 255, 255
}

// Data for an individual vertex that is used to render rooms.
type roomVertex struct {
	// World space co-ordinates of this vertex.
	x, y, z float32

	// Texture space co-ordinates for shading fragments.
	u, v float32

	// Texture coordinates for the los texture.
	los_u, los_v float32
}

type plane struct {
	iBuffer gl.Buffer
	texture texture.Object
	mat     *mathgl.Mat4
}

func visibilityOfObject(xoff, yoff BoardSpaceUnit, ro RectObject, los_tex *LosTexture) byte {
	if los_tex == nil {
		return 255
	}
	x, y := ro.FloorPos()
	x += xoff
	y += yoff
	dx, dy := ro.Dims()
	count := 0
	pix := los_tex.Pix()
	for i := x; i < x+dx; i++ { // foreach column
		// check for visibilty below the rect
		if y-1 >= 0 && pix[i][y-1] > LosVisibilityThreshold {
			count++
		}
		// check for visibilty above the rect
		if y+dy+1 < LosTextureSize && pix[i][y+dy+1] > LosVisibilityThreshold {
			count++
		}
	}
	for j := y; j < y+dy; j++ { // foreach row
		// check for visibilty left of the rect
		if x-1 > 0 && pix[x-1][j] > LosVisibilityThreshold {
			count++
		}
		// check for visibilty right of the rect
		if x+dx+1 < LosTextureSize && pix[x+dx+1][j] > LosVisibilityThreshold {
			count++
		}
	}

	// If half or more of the 'perimiter' is visible, the whole thing is
	// fully visible(?).
	// TODO(tmckee): isn't this case redundant?
	if count >= int(dx+dy) {
		return 255
	}

	// Scale the visibility of the object by how much of its 'perimiter' is
	// visible.
	v := 256 * float64(count) / float64(dx+dy)
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return byte(v)
}

func (room *Room) renderDrawables(base_alpha byte, drawables []Drawable, los_tex *LosTexture) {
	logging.Debug("renderDrawables called", "drawables", drawables)

	var all []Drawable
	for _, d := range drawables {
		x, y := d.FloorPos()
		logging.Debug("cull-check", "floorpos", []any{x, y}, "room", []any{room.X, room.Y, room.Size.Dx, room.Size.Dy})
		if x < room.X {
			continue
		}
		if y < room.Y {
			continue
		}
		if x >= room.X+room.Size.Dx {
			continue
		}
		if y >= room.Y+room.Size.Dy {
			continue
		}
		all = append(all, offsetDrawable{d, -room.X, -room.Y})
	}

	logging.Debug("after culling by room dims", "all-rect-objects", all)

	// Do not include temporary objects in the ordering, since they will likely
	// overlap with other objects and make it difficult to determine the proper
	// ordering.  Just draw the temporary ones last.
	var temps []Drawable
	for _, f := range room.Furniture {
		if f.temporary {
			temps = append(temps, f)
		} else {
			all = append(all, f)
		}
	}
	logging.Debug("after collecting furniture", "all", all, "temps", temps)
	all = OrderRectObjects(all)
	for i := range all {
		temps = append(temps, all[i])
	}

	logging.Debug("after reordering", "all", all, "temps", temps, "glstate", debug.GetGlState())

	for i := len(temps) - 1; i >= 0; i-- {
		d := temps[i]
		fx, fy := d.FPos()
		near_x, near_y := float32(fx), float32(fy)
		vis := visibilityOfObject(room.X, room.Y, d, los_tex)
		r, g, b, a := d.Color()
		r = alphaMult(r, vis)
		g = alphaMult(g, vis)
		b = alphaMult(b, vis)
		a = alphaMult(a, vis)
		a = alphaMult(a, base_alpha)
		gl.Color4ub(r, g, b, a)
		dx, _ := d.Dims()

		logging.Debug("going to render", "near_x,near_y,dims", []any{near_x, near_y, dx})
		d.Render(mathgl.Vec2{X: near_x, Y: near_y}, float32(dx))
	}
}

func (room *Room) getNearWallAlpha(los_tex *LosTexture) (left, right byte) {
	if los_tex == nil {
		return 255, 255
	}
	pix := los_tex.Pix()
	var v1, v2 BoardSpaceUnit
	for y := room.Y; y < room.Y+room.Size.Dy; y++ {
		if pix[room.X][y] > LosVisibilityThreshold {
			v1++
		}
		if pix[room.X-1][y] > LosVisibilityThreshold {
			v2++
		}
	}
	if v1 < v2 {
		v1 = v2
	}
	right = byte((v1 * 255) / room.Size.Dy)
	v1, v2 = 0, 0
	for x := room.X; x < room.X+room.Size.Dx; x++ {
		if pix[x][room.Y] > LosVisibilityThreshold {
			v1++
		}
		if pix[x][room.Y-1] > LosVisibilityThreshold {
			v2++
		}
	}
	if v1 < v2 {
		v1 = v2
	}
	left = byte((v1 * 255) / room.Size.Dy)
	return
}

func (room *Room) getMaxLosAlpha(los_tex *LosTexture) byte {
	if los_tex == nil {
		return 255
	}
	var max_room_alpha byte = 0
	pix := los_tex.Pix()
	for x := room.X; x < room.X+room.Size.Dx; x++ {
		for y := room.Y; y < room.Y+room.Size.Dy; y++ {
			if pix[x][y] > max_room_alpha {
				max_room_alpha = pix[x][y]
			}
		}
	}
	max_room_alpha = byte(255 * (float64(max_room_alpha-LosMinVisibility) / float64(255-LosMinVisibility)))
	return max_room_alpha
}

func alphaMult(a, b byte) byte {
	return byte((int(a) * int(b)) >> 8)
}

var Num_rows float32 = 1150
var Noise_rate float32 = 60
var Num_steps float32 = 3
var Foo int = 0

func (room *Room) getDecalState(decal *Decal) decalGlIDs {
	result := room.decal_gl_map[decal]
	cachedState := room.decal_state_map[decal]
	var new_state decalState
	new_state.flip = decal.Flip
	new_state.rot = decal.Rot
	new_state.x = decal.X
	new_state.y = decal.Y
	new_state.room.x = room.X
	new_state.room.y = room.Y
	new_state.room.dx = room.Size.Dx
	new_state.room.dy = room.Size.Dy
	if new_state != cachedState {
		decal.setupGlStuff(room.X, room.Y, room.Size.Dx, room.Size.Dy, &result)
		room.decal_gl_map[decal] = result
		room.decal_state_map[decal] = new_state
	}

	return result
}

func (room *Room) SetWallTransparency(transparent bool) {
	alphavalue := byte(255)
	if transparent {
		alphavalue = 0
	}

	room.far_right.wall_alpha = alphavalue
	room.far_left.wall_alpha = alphavalue
}

func showpaths(parts []string) string {
	dup := make([]string, len(parts))
	for i, part := range parts {
		dup[i] = fmt.Sprintf("%q", part)
	}
	return strings.Join(dup, ", ")
}

func (room *Room) LoadAndWaitForTexturesForTest() {
	paths := []string{}
	for _, decal := range room.Decals {
		paths = append(paths, decal.Texture.GetPath())
	}
	paths = append(paths, room.Floor.GetPath())
	paths = append(paths, room.Wall.GetPath())

	logging.Warn("going to block on textures", "paths", showpaths(paths), "wall", room.Wall.GetPath(), "floor", room.Floor.GetPath())
	for i, path := range paths {
		_, err := texture.LoadFromPath(path)
		if err != nil {
			panic(fmt.Errorf("couldn't load texture %d, %q: %w", i, path, err))
		}
	}

	texture.BlockUntilLoaded(context.Background(), paths...)
}

func (room *Room) RenderDecals(worldToView *mathgl.Mat4, base_alpha byte) {
	// TODO(#13): once the registry supports it, initialize Room instances while
	// loading them from the registry so we don't have to guard/deal with
	// half-initialized rooms here.
	if room.decal_gl_map == nil {
		room.decal_gl_map = make(map[*Decal]decalGlIDs)
		room.decal_state_map = make(map[*Decal]decalState)
	}

	defer func() {
		gl.Buffer(0).Unbind(gl.ARRAY_BUFFER)
		gl.Buffer(0).Unbind(gl.ELEMENT_ARRAY_BUFFER)
		gl.Texture(0).Unbind(gl.TEXTURE_2D)
	}()

	var vert roomVertex
	for _, decal := range room.Decals {
		var ids decalGlIDs = room.getDecalState(decal)
		if ids.vBuffer == 0 {
			logging.ErrorBracket(func() {
				// TODO(tmckee:#34): do warn about this; it happens if the decal is
				// positioned such that its entirely clipped out when intersected with
				// what will be drawn. It might happen in other cases too?
				logging.Warn("decal state had zeroed vBuffer", "name", decal.Defname)
			})
			continue
		}

		// Bind the the texture for the decal.
		decal.Texture.Data().Bind()
		R, G, B, A := decal.Color()

		gl.ClientActiveTexture(gl.TEXTURE0)
		ids.vBuffer.Bind(gl.ARRAY_BUFFER)
		gl.VertexPointer(3, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.x))
		gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.u))
		gl.ClientActiveTexture(gl.TEXTURE1)
		gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.los_u))
		gl.ClientActiveTexture(gl.TEXTURE0)
		if ids.floorIBuffer != 0 {
			gl.StencilFunc(gl.ALWAYS, 2, 2)
			ids.floorIBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
			gl.Color4ub(R, G, B, A)
			gl.DrawElements(gl.TRIANGLES, int(ids.floorICount), gl.UNSIGNED_SHORT, nil)
		}
		if ids.leftIBuffer != 0 {
			gl.StencilFunc(gl.ALWAYS, 1, 1)
			ids.leftIBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
			doColour(room, R, G, B, alphaMult(A, room.far_left.wall_alpha), base_alpha)
			gl.DrawElements(gl.TRIANGLES, int(ids.leftICount), gl.UNSIGNED_SHORT, nil)
		}
		if ids.rightIBuffer != 0 {
			gl.StencilFunc(gl.ALWAYS, 1, 1)
			ids.rightIBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
			doColour(room, R, G, B, alphaMult(A, room.far_right.wall_alpha), base_alpha)

			gl.DrawElements(gl.TRIANGLES, int(ids.rightICount), gl.UNSIGNED_SHORT, nil)
		}
	}
}

func doColour(room *Room, r, g, b, a, base_alpha byte) {
	R, G, B, A := room.Color()
	A = alphaMult(A, base_alpha)
	gl.Color4ub(alphaMult(R, r), alphaMult(G, g), alphaMult(B, b), alphaMult(A, a))
}

func WithRoomRenderGlSettings(modelView mathgl.Mat4, fn func()) {
	render.MustBeOnRenderThread()

	gl.Enable(gl.TEXTURE_2D)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.STENCIL_TEST)
	defer gl.Disable(gl.STENCIL_TEST)
	gl.ClearStencil(0)
	gl.Clear(gl.STENCIL_BUFFER_BIT)

	gl.EnableClientState(gl.VERTEX_ARRAY)
	gl.EnableClientState(gl.TEXTURE_COORD_ARRAY)
	defer gl.DisableClientState(gl.VERTEX_ARRAY)
	defer gl.DisableClientState(gl.TEXTURE_COORD_ARRAY)

	render.WithMultMatrixInMode(&modelView, render.MatrixModeModelView, func() {
		render.WithColour(1, 1, 1, 1, func() {
			fn()
		})
	})
}

// Need floor, right wall, and left wall matrices to draw the details
func (room *Room) Render(roomMats perspective.RoomMats, zoom float32, base_alpha byte, drawables []Drawable, los_tex *LosTexture, floor_drawers []RenderOnFloorer) {
	render.LogAndClearGlErrors(logging.InfoLogger())

	logging.Trace("Room.Render called", "base_alpha", base_alpha, "glstate", debug.GetGlState(), "floor", roomMats.Floor)

	defer func() {
		gl.Buffer(0).Unbind(gl.ARRAY_BUFFER)
		gl.Buffer(0).Unbind(gl.ELEMENT_ARRAY_BUFFER)
		gl.Texture(0).Unbind(gl.TEXTURE_2D)
	}()

	do_color := func(r, g, b, a byte) {
		doColour(room, r, g, b, a, base_alpha)
	}

	WithRoomRenderGlSettings(roomMats.Floor, func() {
		var vert roomVertex

		planes := []plane{
			{room.glData.leftWallIBuffer, room.Wall, &roomMats.Left},
			{room.glData.rightWallIBuffer, room.Wall, &roomMats.Right},
			{room.glData.floorIBuffer, room.Floor, &roomMats.Floor},
		}

		if los_tex != nil {
			logging.Trace("los_tex not nil")
			gl.ClientActiveTexture(gl.TEXTURE1)
			gl.ActiveTexture(gl.TEXTURE1)
			gl.Enable(gl.TEXTURE_2D)
			gl.EnableClientState(gl.TEXTURE_COORD_ARRAY)
			los_tex.Bind()
			room.glData.vBuffer.Bind(gl.ARRAY_BUFFER)
			gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.los_u))
			gl.ClientActiveTexture(gl.TEXTURE0)
			gl.ActiveTexture(gl.TEXTURE0)
			base.EnableShader("los")
			// TODO(tmckee): 🤔🤔🤔 shouldn't we be setting which texture unit the
			// 'tex1' sampler is set to? Maybe we're just lucky that 0 is the
			// default?
			// base.SetUniformI("los", "tex1", 0)
			base.SetUniformI("los", "tex2", 1)
		}

		for planeIdx, plane := range planes {
			// Render the doors and cut out the stencil buffer so we leave them empty
			// if they're open
			switch plane.mat {
			case &roomMats.Left:
				gl.StencilFunc(gl.ALWAYS, 1, 1)
				gl.StencilOp(gl.REPLACE, gl.REPLACE, gl.REPLACE)
				for _, door := range room.Doors {
					if door.Facing != FarLeft {
						continue
					}
					door.TextureData().Bind()
					R, G, B, A := door.Color()
					do_color(R, G, B, alphaMult(A, room.far_left.wall_alpha))
					gl.ClientActiveTexture(gl.TEXTURE0)
					door.TextureData().Bind()
					if door.doorGlIds.iBuffer != 0 {
						door.thresholdIds.vBuffer.Bind(gl.ARRAY_BUFFER)
						gl.VertexPointer(3, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.x))
						door.doorGlIds.iBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
						gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.u))
						gl.ClientActiveTexture(gl.TEXTURE1)
						gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.los_u))
						gl.DrawElements(gl.TRIANGLES, int(door.doorGlIds.iCount), gl.UNSIGNED_SHORT, nil)
					}
				}
				gl.StencilFunc(gl.NOTEQUAL, 1, 1)
				gl.StencilOp(gl.KEEP, gl.KEEP, gl.KEEP)
				do_color(255, 255, 255, room.far_left.wall_alpha)

			case &roomMats.Right:
				gl.StencilFunc(gl.ALWAYS, 1, 1)
				gl.StencilOp(gl.REPLACE, gl.REPLACE, gl.REPLACE)
				for _, door := range room.Doors {
					if door.Facing != FarRight {
						continue
					}
					door.TextureData().Bind()
					R, G, B, A := door.Color()
					do_color(R, G, B, alphaMult(A, room.far_right.wall_alpha))
					gl.ClientActiveTexture(gl.TEXTURE0)
					door.TextureData().Bind()
					if door.doorGlIds.iBuffer != 0 {
						door.thresholdIds.vBuffer.Bind(gl.ARRAY_BUFFER)
						gl.VertexPointer(3, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.x))
						door.doorGlIds.iBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
						gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.u))
						gl.ClientActiveTexture(gl.TEXTURE1)
						gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.los_u))
						gl.DrawElements(gl.TRIANGLES, int(door.doorGlIds.iCount), gl.UNSIGNED_SHORT, nil)
					}
				}
				gl.StencilFunc(gl.NOTEQUAL, 1, 1)
				gl.StencilOp(gl.KEEP, gl.KEEP, gl.KEEP)
				do_color(255, 255, 255, room.far_right.wall_alpha)

			case &roomMats.Floor:
				// Write '0b00000010' to the stencil buffer when stencil testing.
				gl.StencilFunc(gl.ALWAYS, 2, 2)
				gl.StencilOp(gl.REPLACE, gl.REPLACE, gl.REPLACE)

				do_color(255, 255, 255, 255)
			}

			render.LogAndClearGlErrors(logging.InfoLogger())

			gl.ClientActiveTexture(gl.TEXTURE0)
			room.glData.vBuffer.Bind(gl.ARRAY_BUFFER)
			gl.VertexPointer(3, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.x))
			gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.u))
			gl.ClientActiveTexture(gl.TEXTURE1)
			if los_tex != nil {
				los_tex.Bind()
			}
			// Rebind vbuffer so that it is asociated to texture-unit-1 too (i guess?)
			room.glData.vBuffer.Bind(gl.ARRAY_BUFFER)
			gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.los_u))
			// Now draw the plane
			plane.texture.Data().Bind()
			plane.iBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
			if (plane.mat == &roomMats.Left || plane.mat == &roomMats.Right) && strings.Contains(room.Wall.GetPath(), "gradient.png") {
				logging.Trace("seeing a gradient.png texture; enabling 'gorey' shader", "planeIdx", planeIdx)
				base.EnableShader("gorey")
				base.SetUniformI("gorey", "tex", 0)
				base.SetUniformI("gorey", "foo", Foo)
				base.SetUniformF("gorey", "num_rows", Num_rows)
				base.SetUniformF("gorey", "noise_rate", Noise_rate)
				base.SetUniformF("gorey", "num_steps", Num_steps)
			}
			if plane.mat == &roomMats.Floor && strings.Contains(room.Floor.GetPath(), "gradient.png") {
				logging.Trace("seeing a gradient.png texture; enabling 'gorey' shader", "planeIdx", planeIdx)
				base.EnableShader("gorey")
				base.SetUniformI("gorey", "tex", 0)
				base.SetUniformI("gorey", "foo", Foo)
				base.SetUniformF("gorey", "num_rows", Num_rows)
				base.SetUniformF("gorey", "noise_rate", Noise_rate)
				base.SetUniformF("gorey", "num_steps", Num_steps)
				zexp := math.Log(float64(zoom))
				frac := 1 - 1/zexp
				frac = (frac - 0.6) * 5.0
				switch {
				case frac > 0.7:
					base.SetUniformI("gorey", "range", 1)
				case frac > 0.3:
					base.SetUniformI("gorey", "range", 2)
				default:
					base.SetUniformI("gorey", "range", 3)
				}
			}
			if plane.mat == &roomMats.Floor {
				R, G, B, _ := room.Color()
				gl.Color4ub(R, G, B, 255)
			}

			// Draw triangles using the vertices and indices that we buffered in
			// 'vBuffer' and 'index_buffer', respectively.
			gl.DrawElements(gl.TRIANGLES, int(room.glData.floorICount), gl.UNSIGNED_SHORT, nil)
			if los_tex != nil {
				base.EnableShader("los")
			} else {
				base.EnableShader("")
			}
		}

		room.RenderDecals(&roomMats.Floor, base_alpha)

		base.EnableShader("marble")
		base.SetUniformI("marble", "tex2", 1)
		base.SetUniformF("marble", "room_x", float32(room.X))
		base.SetUniformF("marble", "room_y", float32(room.Y))
		for _, door := range room.Doors {
			door.setupGlStuff(room)
			if door.thresholdIds.vBuffer == 0 {
				continue
			}
			if door.AlwaysOpen() {
				continue
			}
			if door.highlight_threshold {
				gl.Color4ub(255, 255, 255, 255)
			} else {
				gl.Color4ub(128, 128, 128, 255)
			}
			door.thresholdIds.vBuffer.Bind(gl.ARRAY_BUFFER)
			gl.VertexPointer(3, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.x))
			gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.u))
			gl.ClientActiveTexture(gl.TEXTURE1)
			gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), unsafe.Offsetof(vert.los_u))
			gl.ClientActiveTexture(gl.TEXTURE0)
			door.thresholdIds.iBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
			gl.DrawElements(gl.TRIANGLES, int(door.thresholdIds.iCount), gl.UNSIGNED_SHORT, nil)
		}
		base.EnableShader("")
		if los_tex != nil {
			gl.ActiveTexture(gl.TEXTURE1)
			gl.Disable(gl.TEXTURE_2D)
			gl.ActiveTexture(gl.TEXTURE0)
			gl.ClientActiveTexture(gl.TEXTURE1)
			gl.DisableClientState(gl.TEXTURE_COORD_ARRAY)
			gl.ClientActiveTexture(gl.TEXTURE0)
		}

		var mul mathgl.Mat4
		mul.Translation(float32(-room.X), float32(-room.Y), 0)

		render.WithMultMatrixInMode(&mul, render.MatrixModeModelView, func() {
			gl.StencilFunc(gl.EQUAL, 2, 3)
			gl.StencilOp(gl.KEEP, gl.KEEP, gl.KEEP)
			room_rect := ImageRect(room.X, room.Y, room.X+room.Size.Dx, room.Y+room.Size.Dy)
			for _, fd := range floor_drawers {
				x, y := fd.FloorPos()
				dx, dy := fd.Dims()
				if room_rect.Overlaps(ImageRect(x, y, x+dx, y+dy)) {
					fd.RenderOnFloor()
				}
			}
		})
	})

	render.WithMultMatrixInMode(&roomMats.Standup, render.MatrixModeModelView, func() {
		gl.PushAttrib(gl.CURRENT_BIT)
		defer gl.PopAttrib()
		do_color(255, 255, 255, 255)
		gl.Disable(gl.STENCIL_TEST)
		gl.Disable(gl.TEXTURE_2D)
		room.renderDrawables(255, drawables, los_tex)

		gl.ClientActiveTexture(gl.TEXTURE1)
		gl.Disable(gl.TEXTURE_2D)
		gl.ClientActiveTexture(gl.TEXTURE0)
		base.EnableShader("")
	})
}

type RoomSetupGlProxy interface {
	GenBuffer() gl.Buffer
	BufferData(target gl.GLenum, size int, data interface{}, usage gl.GLenum)
}

type RoomRealGl struct{}

var _ RoomSetupGlProxy = ((*RoomRealGl)(nil))

func (*RoomRealGl) GenBuffer() gl.Buffer {
	return gl.GenBuffer()
}

func (*RoomRealGl) BufferData(target gl.GLenum, size int, data interface{}, usage gl.GLenum) {
	gl.BufferData(target, size, data, usage)
}

// Check if the current configuration of the room matches the configuration it
// had during the last GL initialization run.
func (room *Room) glDataInputsDiffer() bool {
	return room.X == room.glDataInputs.x &&
		room.Y == room.glDataInputs.y &&
		room.Size.Dx == room.glDataInputs.dx &&
		room.Size.Dy == room.glDataInputs.dy &&
		room.Wall.Data().Dx() == int(room.glDataInputs.decal_dx) &&
		room.Wall.Data().Dy() == int(room.glDataInputs.decal_dy)
}

func (room *Room) resetGlDataInputs() {
	room.glDataInputs.x = room.X
	room.glDataInputs.y = room.Y
	room.glDataInputs.dx = room.Size.Dx
	room.glDataInputs.dy = room.Size.Dy
	room.glDataInputs.decal_dx = BoardSpaceUnit(room.Wall.Data().Dx())
	room.glDataInputs.decal_dy = BoardSpaceUnit(room.Wall.Data().Dy())
}

func (room *Room) resetGlData() {
	if room.glData.vBuffer == 0 {
		return
	}

	room.glData.vBuffer.Delete()
	room.glData.leftWallIBuffer.Delete()
	room.glData.rightWallIBuffer.Delete()
	room.glData.floorIBuffer.Delete()
}

func (room *Room) SetupGlStuff(glProxy RoomSetupGlProxy) {
	if room.glDataInputsDiffer() {
		logging.Trace("room.SetupGlStuff: reusing cached values")
		return
	}
	room.resetGlDataInputs()
	room.resetGlData()

	defer func() {
		gl.Buffer(0).Unbind(gl.ARRAY_BUFFER)
		gl.Buffer(0).Unbind(gl.ELEMENT_ARRAY_BUFFER)
	}()

	dx := float32(room.Size.Dx)
	dy := float32(room.Size.Dy)
	var dz float32
	if room.Wall.Data().Dx() > 0 {
		dz = -float32(room.Wall.Data().Dy()*int(room.Size.Dx+room.Size.Dy)) / float32(room.Wall.Data().Dx())
	}

	// Conveniently casted values
	frx := float32(room.X)
	fry := float32(room.Y)
	frdx := float32(room.Size.Dx)
	frdy := float32(room.Size.Dy)

	// c is the u-texcoord of the corner of the room
	c := frdx / (frdx + frdy)

	// lt_llx := frx / LosTextureSize
	// lt_lly := fry / LosTextureSize
	// lt_urx := (frx + frdx) / LosTextureSize
	// lt_ury := (fry + frdy) / LosTextureSize

	lt_llx_ep := (frx + 0.5) / LosTextureSize
	lt_lly_ep := (fry + 0.5) / LosTextureSize
	lt_urx_ep := (frx + frdx - 0.5) / LosTextureSize
	lt_ury_ep := (fry + frdy - 0.5) / LosTextureSize

	vs := []roomVertex{
		// Walls
		{0, dy, 0, 0, 1, lt_ury_ep, lt_llx_ep},
		{dx, dy, 0, c, 1, lt_ury_ep, lt_urx_ep},
		{dx, 0, 0, 1, 1, lt_lly_ep, lt_urx_ep},
		{0, dy, dz, 0, 0, lt_ury_ep, lt_llx_ep},
		{dx, dy, dz, c, 0, lt_ury_ep, lt_urx_ep},
		{dx, 0, dz, 1, 0, lt_lly_ep, lt_urx_ep},

		// Floor
		// This is the bulk of the floor, containing all but the outer edges of
		// the room.  los_tex can map directly onto this so we don't need to do
		// anything weird here.
		{0.5, 0.5, 0, 0.5 / dx, 1 - 0.5/dy, lt_lly_ep, lt_llx_ep},
		{0.5, dy - 0.5, 0, 0.5 / dx, 0.5 / dy, lt_ury_ep, lt_llx_ep},
		{dx - 0.5, dy - 0.5, 0, 1 - 0.5/dx, 0.5 / dy, lt_ury_ep, lt_urx_ep},
		{dx - 0.5, 0.5, 0, 1 - 0.5/dx, 1 - 0.5/dy, lt_lly_ep, lt_urx_ep},

		{0, 0.5, 0, 0, 1 - 0.5/dy, lt_lly_ep, lt_llx_ep},
		{0, dy - 0.5, 0, 0, 0.5 / dy, lt_ury_ep, lt_llx_ep},
		{0.5, dy - 0.5, 0, 0.5 / dx, 0.5 / dy, lt_ury_ep, lt_llx_ep},
		{0.5, 0.5, 0, 0.5 / dx, 1 - 0.5/dy, lt_lly_ep, lt_llx_ep},

		{0.5, 0, 0, 0.5 / dx, 1, lt_lly_ep, lt_llx_ep},
		{0.5, 0.5, 0, 0.5 / dx, 1 - 0.5/dy, lt_lly_ep, lt_llx_ep},
		{dx - 0.5, 0.5, 0, 1 - 0.5/dx, 1 - 0.5/dy, lt_lly_ep, lt_urx_ep},
		{dx - 0.5, 0, 0, 1 - 0.5/dx, 1, lt_lly_ep, lt_urx_ep},

		{dx - 0.5, 0.5, 0, 1 - 0.5/dx, 1 - 0.5/dy, lt_lly_ep, lt_urx_ep},
		{dx - 0.5, dy - 0.5, 0, 1 - 0.5/dx, 0.5 / dy, lt_ury_ep, lt_urx_ep},
		{dx, dy - 0.5, 0, 1, 0.5 / dy, lt_ury_ep, lt_urx_ep},
		{dx, 0.5, 0, 1, 1 - 0.5/dy, lt_lly_ep, lt_urx_ep},

		{0.5, dy - 0.5, 0, 0.5 / dx, 0.5 / dy, lt_ury_ep, lt_llx_ep},
		{0.5, dy, 0, 0.5 / dx, 0, lt_ury_ep, lt_llx_ep},
		{dx - 0.5, dy, 0, 1 - 0.5/dx, 0, lt_ury_ep, lt_urx_ep},
		{dx - 0.5, dy - 0.5, 0, 1 - 0.5/dx, 0.5 / dy, lt_ury_ep, lt_urx_ep},

		{0, 0, 0, 0, 1, lt_lly_ep, lt_llx_ep},
		{0, 0.5, 0, 0, 1 - 0.5/dy, lt_lly_ep, lt_llx_ep},
		{0.5, 0.5, 0, 0.5 / dx, 1 - 0.5/dy, lt_lly_ep, lt_llx_ep},
		{0.5, 0, 0, 0.5 / dx, 1, lt_lly_ep, lt_llx_ep},

		{0, dy - 0.5, 0, 0, 0.5 / dy, lt_ury_ep, lt_llx_ep},
		{0, dy, 0, 0, 0, lt_ury_ep, lt_llx_ep},
		{0.5, dy, 0, 0.5 / dx, 0, lt_ury_ep, lt_llx_ep},
		{0.5, dy - 0.5, 0, 0.5 / dx, 0.5 / dy, lt_ury_ep, lt_llx_ep},

		{dx - 0.5, dy - 0.5, 0, 1 - 0.5/dx, 0.5 / dy, lt_ury_ep, lt_urx_ep},
		{dx - 0.5, dy, 0, 1 - 0.5/dx, 0, lt_ury_ep, lt_urx_ep},
		{dx, dy, 0, 1, 0, lt_ury_ep, lt_urx_ep},
		{dx, dy - 0.5, 0, 1, 0.5 / dy, lt_ury_ep, lt_urx_ep},

		{dx - 0.5, 0, 0, 1 - 0.5/dx, 1, lt_lly_ep, lt_urx_ep},
		{dx - 0.5, 0.5, 0, 1 - 0.5/dx, 1 - 0.5/dy, lt_lly_ep, lt_urx_ep},
		{dx, 0.5, 0, 1, 1 - 0.5/dy, lt_lly_ep, lt_urx_ep},
		{dx, 0, 0, 1, 1, lt_lly_ep, lt_urx_ep},
	}

	logging.Trace("room geometry", "dx,dy,dz", []float32{dx, dy, dz}, "c", c, "vs", vs)

	room.glData.vBuffer = glProxy.GenBuffer()
	room.glData.vBuffer.Bind(gl.ARRAY_BUFFER)
	size := int(unsafe.Sizeof(roomVertex{}))
	glProxy.BufferData(gl.ARRAY_BUFFER, size*len(vs), vs, gl.STATIC_DRAW)

	// left wall indices
	is := []uint16{0, 3, 4, 0, 4, 1}
	room.glData.leftWallIBuffer = glProxy.GenBuffer()
	room.glData.leftWallIBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
	glProxy.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(unsafe.Sizeof(is[0]))*len(is), is, gl.STATIC_DRAW)

	// right wall indices
	is = []uint16{1, 4, 5, 1, 5, 2}
	room.glData.rightWallIBuffer = glProxy.GenBuffer()
	room.glData.rightWallIBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
	glProxy.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(unsafe.Sizeof(is[0]))*len(is), is, gl.STATIC_DRAW)

	// floor indices
	is = []uint16{
		6, 7, 8, 6, 8, 9, // middle
		10, 11, 12, 10, 12, 13, // left side
		14, 15, 16, 14, 16, 17, // bottom side
		18, 19, 20, 18, 20, 21, // right side
		22, 23, 24, 22, 24, 25, // top side
		26, 27, 28, 26, 28, 29, // bottom left corner
		30, 31, 32, 30, 32, 33, // upper left corner
		34, 35, 36, 34, 36, 37, // upper right corner
		38, 39, 40, 38, 40, 41, // lower right corner
	}
	room.glData.floorIBuffer = glProxy.GenBuffer()
	room.glData.floorIBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
	glProxy.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(unsafe.Sizeof(is[0]))*len(is), is, gl.STATIC_DRAW)
	room.glData.floorICount = len(is)
}

func (room *RoomDef) Dims() (dx, dy BoardSpaceUnit) {
	return room.Size.Dx, room.Size.Dy
}

func (r *RoomDef) Resize(size RoomSize) {
	r.Size = size
}

func (r *Room) FloorPos() (x, y BoardSpaceUnit) {
	return r.X, r.Y
}

func (room *Room) canAddDoor(door *Door) bool {
	if door.Pos < 0 {
		return false
	}

	// Make sure that the door only occupies valid cells
	if door.Facing == FarLeft || door.Facing == NearRight {
		if door.Pos+door.Width >= room.Size.Dx {
			return false
		}
	}
	if door.Facing == FarRight || door.Facing == NearLeft {
		if door.Pos+door.Width >= room.Size.Dy {
			return false
		}
	}

	// Now make sure that the door doesn't overlap any other doors
	for _, other := range room.Doors {
		if other.Facing != door.Facing {
			continue
		}
		if other.temporary {
			continue
		}
		if other.Pos >= door.Pos && other.Pos-door.Pos < door.Width {
			return false
		}
		if door.Pos >= other.Pos && door.Pos-other.Pos < other.Width {
			return false
		}
	}

	return true
}

func imagePathFilter(path string, isdir bool) bool {
	if isdir {
		return path[0] != '.'
	}
	ext := filepath.Ext(path)
	return ext == ".jpg" || ext == ".png"
}

type roomError struct {
	ErrorString string
}

func (re *roomError) Error() string {
	return re.ErrorString
}

type tabWidget interface {
	Respond(*gui.Gui, gui.EventGroup) bool
	Reload()
	Collapse()
	Expand()
}

type RoomEditorPanel struct {
	*gui.HorizontalTable
	tab     *gui.TabFrame
	widgets []tabWidget

	panels struct {
		furniture *FurniturePanel
		wall      *WallPanel
	}

	room   Room
	viewer *roomViewer
}

// Manually pass all events to the tabs, regardless of location, since the tabs
// need to know where the user clicks.
func (w *RoomEditorPanel) Respond(ui *gui.Gui, group gui.EventGroup) bool {
	return w.widgets[w.tab.SelectedTab()].Respond(ui, group)
}

func (w *RoomEditorPanel) SelectTab(n int) {
	if n < 0 || n >= len(w.widgets) {
		panic(fmt.Errorf("bad tab index: %d expected: [0:%d)", n, len(w.widgets)))
	}
	if n != w.tab.SelectedTab() {
		w.widgets[w.tab.SelectedTab()].Collapse()
		w.tab.SelectTab(n)
		w.viewer.SetEditMode(editNothing)
		w.widgets[n].Expand()
	}
}

func (w *RoomEditorPanel) GetViewer() Viewer {
	return w.viewer
}

type Viewer interface {
	gui.Widget
	SetZoom(float32)
	GetZoom() float32
	WindowToBoard(int, int) (float32, float32)
	BoardToWindow(float32, float32) (int, int)
}

type Editor interface {
	gui.Widget

	Save() (string, error)
	Load(path string) error

	// Called when we tab into the editor from another editor.  It's possible that
	// a portion of what is being edited in the new editor was changed in another
	// editor, so we reload everything so we can see the up-to-date version.
	Reload()

	GetViewer() Viewer

	// TODO: Deprecate when tabs handle the switching themselves
	SelectTab(int)
}

func MakeRoomEditorPanel() Editor {
	var rep RoomEditorPanel

	rep.HorizontalTable = gui.MakeHorizontalTable()
	rep.room.RoomDef = new(RoomDef)
	rep.viewer = MakeRoomViewer(&rep.room, 65)
	rep.AddChild(rep.viewer)

	var tabs []gui.Widget

	rep.panels.furniture = makeFurniturePanel(&rep.room, rep.viewer)
	tabs = append(tabs, rep.panels.furniture)
	rep.widgets = append(rep.widgets, rep.panels.furniture)

	rep.panels.wall = MakeWallPanel(&rep.room, rep.viewer)
	tabs = append(tabs, rep.panels.wall)
	rep.widgets = append(rep.widgets, rep.panels.wall)

	rep.tab = gui.MakeTabFrame(tabs)
	rep.AddChild(rep.tab)
	rep.viewer.SetEditMode(editFurniture)

	return &rep
}

func (rep *RoomEditorPanel) Load(path string) error {
	var room Room
	err := base.LoadAndProcessObject(path, "json", &room.RoomDef)
	if err == nil {
		rep.room = room
		for _, tab := range rep.widgets {
			tab.Reload()
		}
	}
	return err
}

func (rep *RoomEditorPanel) Save() (string, error) {
	path := filepath.Join(datadir, "rooms", rep.room.Name+".room")
	err := base.SaveJson(path, rep.room)
	return path, err
}

func (rep *RoomEditorPanel) Reload() {
	for _, tab := range rep.widgets {
		tab.Reload()
	}
}
