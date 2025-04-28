package house

import (
	"fmt"
	"unsafe"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/go-gl-legacy/gl"
)

type WallFacing int

const (
	NearLeft WallFacing = iota
	NearRight
	FarLeft
	FarRight
)

func (f WallFacing) String() string {
	switch f {
	case NearLeft:
		return "NearLeft"
	case NearRight:
		return "NearRight"
	case FarLeft:
		return "FarLeft"
	case FarRight:
		return "FarRight"
	}
	return fmt.Sprintf("invalid facing (%d)", int(f))
}

func MakeDoor(name string) *Door {
	d := Door{Defname: name}
	base.GetObject("doors", &d)
	return &d
}

func GetAllDoorNames() []string {
	return base.GetAllNamesInRegistry("doors")
}

func LoadAllDoorsInDir(dir string) {
	base.RemoveRegistry("doors")
	base.RegisterRegistry("doors", make(map[string]*DoorDef))
	base.RegisterAllObjectsInDir("doors", dir, ".json", "json")
}

func (d *Door) Load() {
	base.GetObject("doors", d)
}

type DoorDef struct {
	// Name of this texture as it appears in the editor, should be unique among
	// all Doors
	Name string

	// Number of cells wide the door is
	Width int

	// If true then this door is always open, cannot be interacted with, and
	// never draws a threshold.
	Always_open bool

	Opened_texture texture.Object
	Closed_texture texture.Object

	Open_sound base.Path
	Shut_sound base.Path
}

type Door struct {
	Defname string
	*DoorDef

	// Which wall the door is on
	Facing WallFacing

	// How far along this wall the door is located
	Pos int

	// Whether or not the door is opened - determines what texture to use
	Opened bool

	temporary, invalid bool

	highlight_threshold bool

	// gl stuff for drawing the threshold on the ground
	thresholdIds doorGlIds
	doorGlIds    doorGlIds
	state        doorState
}

func (d *Door) AlwaysOpen() bool {
	return d.DoorDef.Always_open
}

func (d *Door) IsOpened() bool {
	return d.DoorDef.Always_open || d.Opened
}

func (d *Door) SetOpened(opened bool) {
	d.Opened = opened
}

func (d *Door) HighlightThreshold(v bool) {
	d.highlight_threshold = v
}

type doorState struct {
	// for tracking whether the buffers are dirty
	facing WallFacing
	pos    int
	room   struct {
		x, y, dx, dy int
	}
}

type doorGlIds struct {
	vBuffer gl.Buffer

	iBuffer gl.Buffer
	// TODO(tmckee): we ought not need to store this, it's always 6.
	iCount gl.GLsizei
}

func (ids *doorGlIds) Reset() {
	if ids.vBuffer == 0 {
		return
	}

	ids.vBuffer.Delete()
	ids.vBuffer = 0
	ids.iBuffer.Delete()
	ids.iBuffer = 0
	ids.iCount = 0
}

func (d *Door) setupGlStuff(room *Room) {
	var state doorState
	state.facing = d.Facing
	state.pos = d.Pos
	state.room.x = room.X
	state.room.y = room.Y
	state.room.dx = room.Size.Dx
	state.room.dy = room.Size.Dy
	if state == d.state {
		return
	}
	if d.TextureData().Dy() == 0 {
		// Can't build this data until the texture is loaded, so we'll have to try
		// again later.
		return
	}
	d.state = state
	d.thresholdIds.Reset()
	d.doorGlIds.Reset()

	// far left, near right, do threshold
	// near left, far right, do threshold
	// far left, far right, do door
	var vs []roomVertex
	// TODO(tmckee): DRY out this code; we're really just x-y swapping between
	// the two cases.
	if d.Facing == FarLeft || d.Facing == NearRight {
		x1 := float32(d.Pos)
		x2 := float32(d.Pos + d.Width)
		var y1 float32 = -0.25
		var y2 float32 = 0.25
		if d.Facing == FarLeft {
			y1 = float32(room.Size.Dy)
			y2 = float32(room.Size.Dy) - 0.25
		}
		// los_x1 := (x1 + float32(room.X)) / LosTextureSize
		vs = append(vs, roomVertex{x: x1, y: y1})
		vs = append(vs, roomVertex{x: x1, y: y2})
		vs = append(vs, roomVertex{x: x2, y: y2})
		vs = append(vs, roomVertex{x: x2, y: y1})
		for i := 0; i < 4; i++ {
			vs[i].los_u = (y2 + float32(room.Y)) / LosTextureSize
			vs[i].los_v = (vs[i].x + float32(room.X)) / LosTextureSize
		}
	} else if d.Facing == FarRight || d.Facing == NearLeft {
		y1 := float32(d.Pos)
		y2 := float32(d.Pos + d.Width)
		var x1 float32 = -0.25
		var x2 float32 = 0.25
		if d.Facing == FarRight {
			x1 = float32(room.Size.Dx)
			x2 = float32(room.Size.Dx) - 0.25
		}
		// los_y1 := (y1 + float32(room.Y)) / LosTextureSize
		vs = append(vs, roomVertex{x: x1, y: y1})
		vs = append(vs, roomVertex{x: x1, y: y2})
		vs = append(vs, roomVertex{x: x2, y: y2})
		vs = append(vs, roomVertex{x: x2, y: y1})
		for i := 0; i < 4; i++ {
			vs[i].los_u = (vs[i].y + float32(room.Y)) / LosTextureSize
			vs[i].los_v = (x2 + float32(room.X)) / LosTextureSize
		}
	} else {
		panic(fmt.Errorf("can't orient door by facing: %s", d.Facing))
	}
	dz := -float32(d.Width*d.TextureData().Dy()) / float32(d.TextureData().Dx())
	if d.Facing == FarRight {
		x := float32(room.Size.Dx)
		y1 := float32(d.Pos + d.Width)
		y2 := float32(d.Pos)
		los_v := (float32(room.X) + x - 0.5) / LosTextureSize
		los_u1 := (float32(room.Y) + y1) / LosTextureSize
		los_u2 := (float32(room.Y) + y2) / LosTextureSize
		vs = append(vs, roomVertex{
			x: x, y: y1, z: 0,
			u: 0, v: 1,
			los_u: los_u1,
			los_v: los_v,
		})
		vs = append(vs, roomVertex{
			x: x, y: y1, z: dz,
			u: 0, v: 0,
			los_u: los_u1,
			los_v: los_v,
		})
		vs = append(vs, roomVertex{
			x: x, y: y2, z: dz,
			u: 1, v: 0,
			los_u: los_u2,
			los_v: los_v,
		})
		vs = append(vs, roomVertex{
			x: x, y: y2, z: 0,
			u: 1, v: 1,
			los_u: los_u2,
			los_v: los_v,
		})
	}
	if d.Facing == FarLeft {
		x1 := float32(d.Pos)
		x2 := float32(d.Pos + d.Width)
		y := float32(room.Size.Dy)
		los_v1 := (float32(room.X) + x1) / LosTextureSize
		los_v2 := (float32(room.X) + x2) / LosTextureSize
		los_u := (float32(room.Y) + y - 0.5) / LosTextureSize
		vs = append(vs, roomVertex{
			x: x1, y: y, z: 0,
			u: 0, v: 1,
			los_u: los_u,
			los_v: los_v1,
		})
		vs = append(vs, roomVertex{
			x: x1, y: y, z: dz,
			u: 0, v: 0,
			los_u: los_u,
			los_v: los_v1,
		})
		vs = append(vs, roomVertex{
			x: x2, y: y, z: dz,
			u: 1, v: 0,
			los_u: los_u,
			los_v: los_v2,
		})
		vs = append(vs, roomVertex{
			x: x2, y: y, z: 0,
			u: 1, v: 1,
			los_u: los_u,
			los_v: los_v2,
		})
	}
	d.thresholdIds.vBuffer = gl.GenBuffer()
	d.thresholdIds.vBuffer.Bind(gl.ARRAY_BUFFER)
	size := int(unsafe.Sizeof(roomVertex{}))
	gl.BufferData(gl.ARRAY_BUFFER, size*len(vs), vs, gl.STATIC_DRAW)

	is := []uint16{0, 1, 2, 0, 2, 3}
	d.thresholdIds.iBuffer = gl.GenBuffer()
	d.thresholdIds.iBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(unsafe.Sizeof(is[0]))*len(is), is, gl.STATIC_DRAW)
	d.thresholdIds.iCount = 6

	if d.Facing == FarLeft || d.Facing == FarRight {
		is2 := []uint16{4, 5, 6, 4, 6, 7}
		d.doorGlIds.iBuffer = gl.GenBuffer()
		d.doorGlIds.iBuffer.Bind(gl.ELEMENT_ARRAY_BUFFER)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, int(unsafe.Sizeof(is2[0]))*len(is2), is2, gl.STATIC_DRAW)
		d.doorGlIds.iCount = 6
	}
}

func (d *Door) TextureData() *texture.Data {
	if d.IsOpened() {
		return d.Opened_texture.Data()
	}
	return d.Closed_texture.Data()
}

func (d *Door) Color() (r, g, b, a byte) {
	if d.temporary {
		if d.invalid {
			return 255, 127, 127, 200
		} else {
			return 127, 127, 255, 200
		}
	}
	return 255, 255, 255, 255
}
