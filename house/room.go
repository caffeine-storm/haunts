package house

import (
	"fmt"
	"image"
	"math"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/MobRulesGames/mathgl"
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/debug"
	"github.com/runningwild/glop/glog"
	"github.com/runningwild/glop/gui"
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
	Dx, Dy int
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

	WallTextures []*WallTexture `registry:"loadfrom-wall_textures"`

	Floor texture.Object
	Wall  texture.Object

	// What house themes this room is appropriate for
	Themes map[string]bool

	// What house sizes this room is appropriate for
	Sizes map[string]bool

	// What kinds of decorations are appropriate in this room
	Decor map[string]bool
}

type roomVertex struct {
	x, y, z float32
	u, v    float32

	// Texture coordinates for the los texture
	los_u, los_v float32
}

type plane struct {
	index_buffer uint32
	texture      texture.Object
	mat          *mathgl.Mat4
}

func visibilityOfObject(xoff, yoff int, ro RectObject, los_tex *LosTexture) byte {
	if los_tex == nil {
		return 255
	}
	x, y := ro.Pos()
	x += xoff
	y += yoff
	dx, dy := ro.Dims()
	count := 0
	pix := los_tex.Pix()
	for i := x; i < x+dx; i++ {
		if y-1 >= 0 && pix[i][y-1] > LosVisibilityThreshold {
			count++
		}
		if y+dy+1 < LosTextureSize && pix[i][y+dy+1] > LosVisibilityThreshold {
			count++
		}
	}
	for j := y; j < y+dy; j++ {
		if x-1 > 0 && pix[x-1][j] > LosVisibilityThreshold {
			count++
		}
		if x+dx+1 < LosTextureSize && pix[x+dx+1][j] > LosVisibilityThreshold {
			count++
		}
	}
	if count >= dx+dy {
		return 255
	}
	v := 256 * float64(count) / float64(dx+dy)
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return byte(v)
}

func (room *Room) renderFurniture(floor mathgl.Mat4, base_alpha byte, drawables []Drawable, los_tex *LosTexture) {
	board_to_window := func(mx, my float32) (x, y float32) {
		v := mathgl.Vec4{X: mx, Y: my, W: 1}
		v.Transform(&floor)
		x, y = v.X, v.Y
		return
	}

	var all []RectObject
	for _, d := range drawables {
		x, y := d.Pos()
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

	// Do not include temporary objects in the ordering, since they will likely
	// overlap with other objects and make it difficult to determine the proper
	// ordering.  Just draw the temporary ones last.
	var temps []RectObject
	for _, f := range room.Furniture {
		if f.temporary {
			temps = append(temps, f)
		} else {
			all = append(all, f)
		}
	}
	all = OrderRectObjects(all)
	for i := range all {
		temps = append(temps, all[i])
	}

	for i := len(temps) - 1; i >= 0; i-- {
		d := temps[i].(Drawable)
		fx, fy := d.FPos()
		near_x, near_y := float32(fx), float32(fy)
		idx, idy := d.Dims()
		dx, dy := float32(idx), float32(idy)
		leftx, _ := board_to_window(near_x, near_y+dy)
		rightx, _ := board_to_window(near_x+dx, near_y)
		_, boty := board_to_window(near_x, near_y)
		vis := visibilityOfObject(room.X, room.Y, d, los_tex)
		r, g, b, a := d.Color()
		r = alphaMult(r, vis)
		g = alphaMult(g, vis)
		b = alphaMult(b, vis)
		a = alphaMult(a, vis)
		a = alphaMult(a, base_alpha)
		gl.Color4ub(r, g, b, a)
		d.Render(mathgl.Vec2{leftx, boty}, rightx-leftx)
	}
}

func (room *Room) getNearWallAlpha(los_tex *LosTexture) (left, right byte) {
	if los_tex == nil {
		return 255, 255
	}
	pix := los_tex.Pix()
	v1, v2 := 0, 0
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

func (room *Room) RenderWalls(floor *mathgl.Mat4, base_alpha byte) {
	var vert roomVertex
	for _, wt := range room.WallTextures {
		if room.wall_texture_gl_map == nil {
			room.wall_texture_gl_map = make(map[*WallTexture]wallTextureGlIds)
			room.wall_texture_state_map = make(map[*WallTexture]wallTextureState)
		}
		ids := room.wall_texture_gl_map[wt]
		state := room.wall_texture_state_map[wt]
		var new_state wallTextureState
		new_state.flip = wt.Flip
		new_state.rot = wt.Rot
		new_state.x = wt.X
		new_state.y = wt.Y
		new_state.room.x = room.X
		new_state.room.y = room.Y
		new_state.room.dx = room.Size.Dx
		new_state.room.dy = room.Size.Dy
		if new_state != state {
			wt.setupGlStuff(room.X, room.Y, room.Size.Dx, room.Size.Dy, &ids)
			room.wall_texture_gl_map[wt] = ids
			room.wall_texture_state_map[wt] = new_state
		}
		gl.LoadMatrixf((*[16]float32)(floor))
		if ids.vbuffer != 0 {
			wt.Texture.Data().Bind()
			R, G, B, A := wt.Color()

			gl.ClientActiveTexture(gl.TEXTURE0)
			gl.Buffer(ids.vbuffer).Bind(gl.ARRAY_BUFFER)
			gl.VertexPointer(3, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.x)
			gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.u)
			gl.ClientActiveTexture(gl.TEXTURE1)
			gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.los_u)
			gl.ClientActiveTexture(gl.TEXTURE0)
			if ids.floor_buffer != 0 {
				gl.StencilFunc(gl.ALWAYS, 2, 2)
				gl.Buffer(ids.floor_buffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
				gl.Color4ub(R, G, B, A)
				gl.DrawElements(gl.TRIANGLES, int(ids.floor_count), gl.UNSIGNED_SHORT, nil)
			}
			if ids.left_buffer != 0 {
				gl.StencilFunc(gl.ALWAYS, 1, 1)
				gl.Buffer(ids.left_buffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
				doColour(room, R, G, B, alphaMult(A, room.far_left.wall_alpha), base_alpha)
				gl.DrawElements(gl.TRIANGLES, int(ids.left_count), gl.UNSIGNED_SHORT, nil)
			}
			if ids.right_buffer != 0 {
				gl.StencilFunc(gl.ALWAYS, 1, 1)
				gl.Buffer(ids.right_buffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
				doColour(room, R, G, B, alphaMult(A, room.far_right.wall_alpha), base_alpha)
				gl.DrawElements(gl.TRIANGLES, int(ids.right_count), gl.UNSIGNED_SHORT, nil)
			}
		}
	}
}

func doColour(room *Room, r, g, b, a, base_alpha byte) {
	R, G, B, A := room.Color()
	A = alphaMult(A, base_alpha)
	gl.Color4ub(alphaMult(R, r), alphaMult(G, g), alphaMult(B, b), alphaMult(A, a))
}

// Need floor, right wall, and left wall matrices to draw the details
func (room *Room) Render(floor, left, right mathgl.Mat4, zoom float32, base_alpha byte, drawables []Drawable, los_tex *LosTexture, floor_drawers []RenderOnFloorer) {
	debug.LogAndClearGlErrors(glog.InfoLogger())

	do_color := func(r, g, b, a byte) {
		doColour(room, r, g, b, a, base_alpha)
	}
	gl.Enable(gl.TEXTURE_2D)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.STENCIL_TEST)
	gl.ClearStencil(0)
	gl.Clear(gl.STENCIL_BUFFER_BIT)

	gl.EnableClientState(gl.VERTEX_ARRAY)
	gl.EnableClientState(gl.TEXTURE_COORD_ARRAY)
	defer gl.DisableClientState(gl.VERTEX_ARRAY)
	defer gl.DisableClientState(gl.TEXTURE_COORD_ARRAY)

	var vert roomVertex

	planes := []plane{
		{room.left_buffer, room.Wall, &left},
		{room.right_buffer, room.Wall, &right},
		{room.floor_buffer, room.Floor, &floor},
	}

	gl.MatrixMode(gl.MODELVIEW)
	gl.PushMatrix()
	defer gl.PopMatrix()

	if los_tex != nil {
		gl.LoadMatrixf((*[16]float32)(&floor))
		gl.ClientActiveTexture(gl.TEXTURE1)
		gl.ActiveTexture(gl.TEXTURE1)
		gl.Enable(gl.TEXTURE_2D)
		gl.EnableClientState(gl.TEXTURE_COORD_ARRAY)
		los_tex.Bind()
		gl.Buffer(room.vbuffer).Bind(gl.ARRAY_BUFFER)
		gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.los_u)
		gl.ClientActiveTexture(gl.TEXTURE0)
		gl.ActiveTexture(gl.TEXTURE0)
		base.EnableShader("los")
		base.SetUniformI("los", "tex2", 1)
	}

	var mul, run mathgl.Mat4
	for _, plane := range planes {
		gl.LoadMatrixf((*[16]float32)(&floor))
		run.Assign(&floor)

		// Render the doors and cut out the stencil buffer so we leave them empty
		// if they're open
		switch plane.mat {
		case &left:
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
				if door.door_glids.floor_buffer != 0 {
					gl.Buffer(door.threshold_glids.vbuffer).Bind(gl.ARRAY_BUFFER)
					gl.VertexPointer(3, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.x)
					gl.Buffer(door.door_glids.floor_buffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
					gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.u)
					gl.ClientActiveTexture(gl.TEXTURE1)
					gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.los_u)
					gl.DrawElements(gl.TRIANGLES, int(door.door_glids.floor_count), gl.UNSIGNED_SHORT, nil)
				}
			}
			gl.StencilFunc(gl.NOTEQUAL, 1, 1)
			gl.StencilOp(gl.KEEP, gl.KEEP, gl.KEEP)
			do_color(255, 255, 255, room.far_left.wall_alpha)

		case &right:
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
				if door.door_glids.floor_buffer != 0 {
					gl.Buffer(door.threshold_glids.vbuffer).Bind(gl.ARRAY_BUFFER)
					gl.VertexPointer(3, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.x)
					gl.Buffer(door.door_glids.floor_buffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
					gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.u)
					gl.ClientActiveTexture(gl.TEXTURE1)
					gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.los_u)
					gl.DrawElements(gl.TRIANGLES, int(door.door_glids.floor_count), gl.UNSIGNED_SHORT, nil)
				}
			}
			gl.StencilFunc(gl.NOTEQUAL, 1, 1)
			gl.StencilOp(gl.KEEP, gl.KEEP, gl.KEEP)
			do_color(255, 255, 255, room.far_right.wall_alpha)

		case &floor:
			gl.StencilFunc(gl.ALWAYS, 2, 2)
			gl.StencilOp(gl.REPLACE, gl.REPLACE, gl.REPLACE)
			do_color(255, 255, 255, 255)
		}

		debug.LogAndClearGlErrors(glog.InfoLogger())

		gl.ClientActiveTexture(gl.TEXTURE0)
		gl.Buffer(room.vbuffer).Bind(gl.ARRAY_BUFFER)
		gl.VertexPointer(3, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.x)
		gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.u)
		gl.ClientActiveTexture(gl.TEXTURE1)
		if los_tex != nil {
			los_tex.Bind()
		}
		gl.Buffer(room.vbuffer).Bind(gl.ARRAY_BUFFER)
		gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.los_u)
		// Now draw the walls
		gl.LoadMatrixf((*[16]float32)(&floor))
		plane.texture.Data().Bind()
		gl.Buffer(plane.index_buffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
		if (plane.mat == &left || plane.mat == &right) && strings.Contains(string(room.Wall.Path), "gradient.png") {
			base.EnableShader("gorey")
			base.SetUniformI("gorey", "tex", 0)
			base.SetUniformI("gorey", "foo", Foo)
			base.SetUniformF("gorey", "num_rows", Num_rows)
			base.SetUniformF("gorey", "noise_rate", Noise_rate)
			base.SetUniformF("gorey", "num_steps", Num_steps)
		}
		if plane.mat == &floor && strings.Contains(string(room.Floor.Path), "gradient.png") {
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
		if plane.mat == &floor {
			R, G, B, _ := room.Color()
			gl.Color4ub(R, G, B, 255)
		}
		gl.DrawElements(gl.TRIANGLES, int(room.floor_count), gl.UNSIGNED_SHORT, nil)
		if los_tex != nil {
			base.EnableShader("los")
		} else {
			base.EnableShader("")
		}
	}

	room.RenderWalls(&floor, base_alpha)

	base.EnableShader("marble")
	base.SetUniformI("marble", "tex2", 1)
	base.SetUniformF("marble", "room_x", float32(room.X))
	base.SetUniformF("marble", "room_y", float32(room.Y))
	for _, door := range room.Doors {
		door.setupGlStuff(room)
		if door.threshold_glids.vbuffer == 0 {
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
		gl.Buffer(door.threshold_glids.vbuffer).Bind(gl.ARRAY_BUFFER)
		gl.VertexPointer(3, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.x)
		gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.u)
		gl.ClientActiveTexture(gl.TEXTURE1)
		gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(vert)), &vert.los_u)
		gl.ClientActiveTexture(gl.TEXTURE0)
		gl.Buffer(door.threshold_glids.floor_buffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
		gl.DrawElements(gl.TRIANGLES, int(door.threshold_glids.floor_count), gl.UNSIGNED_SHORT, nil)
	}
	base.EnableShader("")
	if los_tex != nil {
		base.EnableShader("")
		gl.ActiveTexture(gl.TEXTURE1)
		gl.Disable(gl.TEXTURE_2D)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.ClientActiveTexture(gl.TEXTURE1)
		gl.DisableClientState(gl.TEXTURE_COORD_ARRAY)
		gl.ClientActiveTexture(gl.TEXTURE0)
	}

	run.Assign(&floor)
	mul.Translation(float32(-room.X), float32(-room.Y), 0)
	run.Multiply(&mul)
	gl.LoadMatrixf((*[16]float32)(&run))
	gl.StencilFunc(gl.EQUAL, 2, 3)
	gl.StencilOp(gl.KEEP, gl.KEEP, gl.KEEP)
	room_rect := image.Rect(room.X, room.Y, room.X+room.Size.Dx, room.Y+room.Size.Dy)
	for _, fd := range floor_drawers {
		x, y := fd.Pos()
		dx, dy := fd.Dims()
		if room_rect.Overlaps(image.Rect(x, y, x+dx, y+dy)) {
			fd.RenderOnFloor()
		}
	}

	do_color(255, 255, 255, 255)
	gl.LoadIdentity()
	gl.Disable(gl.STENCIL_TEST)
	room.renderFurniture(floor, 255, drawables, los_tex)

	gl.ClientActiveTexture(gl.TEXTURE1)
	gl.Disable(gl.TEXTURE_2D)
	gl.ClientActiveTexture(gl.TEXTURE0)
	base.EnableShader("")
}

func (room *Room) setupGlStuff() {
	if room.X == room.gl.x &&
		room.Y == room.gl.y &&
		room.Size.Dx == room.gl.dx &&
		room.Size.Dy == room.gl.dy &&
		room.Wall.Data().Dx() == room.gl.wall_tex_dx &&
		room.Wall.Data().Dy() == room.gl.wall_tex_dy {
		return
	}
	room.gl.x = room.X
	room.gl.y = room.Y
	room.gl.dx = room.Size.Dx
	room.gl.dy = room.Size.Dy
	room.gl.wall_tex_dx = room.Wall.Data().Dx()
	room.gl.wall_tex_dy = room.Wall.Data().Dy()
	if room.vbuffer != 0 {
		gl.Buffer(room.vbuffer).Delete()
		gl.Buffer(room.left_buffer).Delete()
		gl.Buffer(room.right_buffer).Delete()
		gl.Buffer(room.floor_buffer).Delete()
	}
	dx := float32(room.Size.Dx)
	dy := float32(room.Size.Dy)
	var dz float32
	if room.Wall.Data().Dx() > 0 {
		dz = -float32(room.Wall.Data().Dy()*(room.Size.Dx+room.Size.Dy)) / float32(room.Wall.Data().Dx())
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
	room.vbuffer = uint32(gl.GenBuffer())
	gl.Buffer(room.vbuffer).Bind(gl.ARRAY_BUFFER)
	size := int(unsafe.Sizeof(roomVertex{}))
	gl.BufferData(gl.ARRAY_BUFFER, size*len(vs), vs, gl.STATIC_DRAW)

	// left wall indices
	is := []uint16{0, 3, 4, 0, 4, 1}
	room.left_buffer = uint32(gl.GenBuffer())
	gl.Buffer(room.left_buffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(unsafe.Sizeof(is[0]))*len(is), is, gl.STATIC_DRAW)

	// right wall indices
	is = []uint16{1, 4, 5, 1, 5, 2}
	room.right_buffer = uint32(gl.GenBuffer())
	gl.Buffer(room.right_buffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(unsafe.Sizeof(is[0]))*len(is), is, gl.STATIC_DRAW)

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
	room.floor_buffer = uint32(gl.GenBuffer())
	gl.Buffer(room.floor_buffer).Bind(gl.ELEMENT_ARRAY_BUFFER)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(unsafe.Sizeof(is[0]))*len(is), is, gl.STATIC_DRAW)
	room.floor_count = len(is)
}

func (room *RoomDef) Dims() (dx, dy int) {
	return room.Size.Dx, room.Size.Dy
}

func (r *RoomDef) Resize(size RoomSize) {
	r.Size = size
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
	Zoom(float64)
	Drag(float64, float64)
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

type selectMode int

const (
	modeNoSelect selectMode = iota
	modeSelect
	modeDeselect
)
