package house

import (
	"fmt"
	"math"
	"reflect"

	"github.com/MobRulesGames/haunts/house/perspective"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/mathgl"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/util/algorithm"
)

// This structure is used for temporary doors (that are being dragged around in
// the editor) since the normal Door struct only handles one door out of a pair
// and doesn't know what rooms it connects.
type doorInfo struct {
	Door  *Door
	Valid bool
}

// TODO(tmckee#34): make these float32s into float64s for less recasting
type HouseViewerState struct {
	zoom, angle, fx, fy float32
	floor, ifloor       mathgl.Mat4

	// target[xy] are the values that f[xy] approach, this gives us a nice way
	// to change what the camera is looking at. All such values are in 'board'
	// units. That is, relative to a floor along the X-Y plane at Z=0.
	targetx, targety float32
	target_on        bool

	// as above, but for zooming
	targetzoom     float32
	target_zoom_on bool

	// Need to keep track of time so we can measure time between thinks
	last_timestamp int64

	dragging dragState
}

func (st *HouseViewerState) GetFocus() (float32, float32) {
	return st.fx, st.fy
}

type HouseViewer struct {
	gui.Childless
	gui.BasicZone
	gui.StubDrawFocuseder

	house *HouseDef

	HouseViewerState

	drawables          []Drawable
	Los_tex            *LosTexture
	temp_floor_drawers []RenderOnFloorer
	Edit_mode          bool

	bounds struct {
		on  bool
		min struct{ x, y float32 }
		max struct{ x, y float32 }
	}

	floor_drawers []RenderOnFloorer
}

func (hv *HouseViewer) GetFloors() []*Floor {
	return hv.house.Floors
}

func MakeHouseViewer(house *HouseDef, angle float32) *HouseViewer {
	ret := &HouseViewer{
		house: house,
		BasicZone: gui.BasicZone{
			Request_dims: gui.Dims{
				Dx: 100,
				Dy: 100,
			},
			Ex: true,
			Ey: true,
		},
	}

	ret.SetAngle(angle)
	ret.SetZoom(10)
	ret.SetBounds()
	ret.SetFocusTarget(0, 0)

	ret.floor, ret.ifloor = perspective.MakeFloorTransforms(ret.Render_region, ret.fx, ret.fy, ret.angle, ret.zoom)

	return ret
}

type framePressTotaler interface {
	FramePressTotal() float64
}

func (hv *HouseViewer) Respond(g *gui.Gui, group gui.EventGroup) bool {
	if group.IsPressed(gin.AnyMouseWheelVertical) {
		var wheelKey gin.Key = group.PrimaryEvent().Key
		zoomDelta := wheelKey.CurPressTotal()
		if zoomDelta != 0 {
			// TODO(tmckee): don't change targetzoom linearly; should be
			// percent-change I think?
			hv.HouseViewerState.targetzoom += float32(zoomDelta)
			if hv.HouseViewerState.targetzoom <= 0 {
				hv.HouseViewerState.targetzoom = 0.0001
			}
			hv.HouseViewerState.target_zoom_on = true
		}
		return true
	}
	return hv.HouseViewerState.dragging.HandleEventGroup(hv, group)
}

func (hv *HouseViewer) Think(g *gui.Gui, t int64) {
	dt := t - hv.last_timestamp
	hv.last_timestamp = t

	if dt < 0 {
		panic(fmt.Errorf("time travel!? dt: %v", dt))
	}

	// as 'dt' grows, 'scale' approaches 1
	scale := 1 - float32(math.Pow(0.005, float64(dt)/1000))

	logging.Debug("Think", "hv.target_on", hv.target_on, "scale", scale, "target", []any{hv.targetx, hv.targety}, "pos", []any{hv.fx, hv.fy})

	if hv.target_on {
		f := mathgl.Vec2{X: hv.fx, Y: hv.fy}
		v := mathgl.Vec2{X: hv.targetx, Y: hv.targety}
		v.Subtract(&f)
		v.Scale(scale)
		f.Add(&v)
		hv.fx = f.X
		hv.fy = f.Y
	}

	if hv.fx == hv.targetx && hv.fy == hv.targety {
		hv.target_on = false
	}

	logging.Info("zoomzoomzoom", "zoom", hv.zoom)
	if hv.target_zoom_on {
		exp := math.Log(float64(hv.zoom))
		exp += (math.Log(float64(hv.targetzoom)) - exp) * float64(scale)
		hv.zoom = float32(math.Exp(exp))
		if math.Abs(float64(hv.zoom-hv.targetzoom)) < 0.001 {
			hv.target_zoom_on = false
		}
	}
}

func (hv *HouseViewer) AddDrawable(d Drawable) {
	hv.drawables = append(hv.drawables, d)
}

func (hv *HouseViewer) RemoveDrawable(d Drawable) {
	algorithm.Choose(&hv.drawables, func(t Drawable) bool {
		return t != d
	})
}

func (hv *HouseViewer) AddFloorDrawable(fd RenderOnFloorer) {
	if fd == nil || reflect.ValueOf(fd).IsNil() {
		panic("WTF")
	}
	hv.floor_drawers = append(hv.floor_drawers, fd)
}

func (hv *HouseViewer) RemoveFloorDrawable(fd RenderOnFloorer) {
	algorithm.Choose(&hv.floor_drawers, func(t RenderOnFloorer) bool {
		return t != fd
	})
}

func (hv *HouseViewerState) modelviewToBoard(mx, my float32) (x, y, dist float32) {
	mz := d2p(hv.floor, mathgl.Vec3{X: mx, Y: my, Z: 0}, mathgl.Vec3{X: 0, Y: 0, Z: 1})
	v := mathgl.Vec4{X: mx, Y: my, Z: mz, W: 1}
	v.Transform(&hv.ifloor)
	return v.X, v.Y, mz
}

func (hv *HouseViewerState) boardToModelview(mx, my float32) (x, y, z float32) {
	v := mathgl.Vec4{X: mx, Y: my, Z: 0, W: 1}
	v.Transform(&hv.floor)
	x, y, z = v.X, v.Y, v.Z
	return
}

// TODO(tmckee#47): this and its compatriots ought to be using BoardSpaceUnit
// where appropriate. Introducing a 'ScreenSpaceUnit' would also be helpful.
// Returning a float32 here is also barftastic but nescessary to support
// sub-tile resolution when clicking.
func (hv *HouseViewer) WindowToBoard(wx, wy int) (float32, float32) {
	hv.floor, hv.ifloor = perspective.MakeFloorTransforms(hv.Render_region, hv.fx, hv.fy, hv.angle, hv.zoom)

	fx, fy, _ := hv.modelviewToBoard(float32(wx), float32(wy))
	return fx, fy
}

func (hv *HouseViewer) BoardToWindow(bx, by float32) (int, int) {
	hv.floor, hv.ifloor = perspective.MakeFloorTransforms(hv.Render_region, hv.fx, hv.fy, hv.angle, hv.zoom)

	fx, fy, _ := hv.boardToModelview(bx, by)
	return int(fx), int(fy)
}

func (hv *HouseViewer) GetState() HouseViewerState {
	return hv.HouseViewerState
}

func (hv *HouseViewer) SetState(state HouseViewerState) {
	hv.HouseViewerState = state
}

func (hv *HouseViewer) SetAngle(theta float32) {
	hv.angle = theta
}

func (hv *HouseViewer) GetAngle() float32 {
	return hv.angle
}

func (hv *HouseViewer) SetZoom(dz float32) {
	if dz == 0 {
		panic(fmt.Errorf("you don't want 0 zoom; it means don't draw anything!"))
	}
	hv.zoom = dz
	hv.targetzoom = hv.zoom
	hv.target_zoom_on = false
}

func (hv *HouseViewer) GetZoom() float32 {
	return hv.zoom
}

func (hv *HouseViewer) SetBounds() {
	if hv.house == nil || len(hv.house.Floors[0].Rooms) == 0 {
		return
	}
	hv.bounds.on = true
	hv.bounds.min.x = float32(hv.house.Floors[0].Rooms[0].X)
	hv.bounds.max.x = hv.bounds.min.x
	hv.bounds.min.y = float32(hv.house.Floors[0].Rooms[0].Y)
	hv.bounds.max.y = hv.bounds.min.y
	for _, floor := range hv.house.Floors {
		for _, room := range floor.Rooms {
			if float32(room.X) < hv.bounds.min.x {
				hv.bounds.min.x = float32(room.X)
			}
			if float32(room.Y) < hv.bounds.min.y {
				hv.bounds.min.y = float32(room.Y)
			}
			if float32(room.X+room.Size.Dx) > hv.bounds.max.x {
				hv.bounds.max.x = float32(room.X + room.Size.Dx)
			}
			if float32(room.Y+room.Size.Dy) > hv.bounds.max.y {
				hv.bounds.max.y = float32(room.Y + room.Size.Dy)
			}
		}
	}
}

func (hv *HouseViewer) SetFocusTarget(bx, by float32) {
	hv.targetx = bx
	hv.targety = by
	hv.target_on = true
}

func (hv *HouseViewer) SetZoomTarget(z float64) {
	z = float64(clamp(float32(z), 0, 1))
	max := 4.25759904621048
	min := 2.87130468509059
	z = z*(max-min) + min
	hv.targetzoom = float32(z)
	hv.target_zoom_on = true
}

func (hv *HouseViewer) String() string {
	mp := map[string]any{}
	mp["HouseDef"] = hv.house
	mp["HouseViewerState"] = hv.HouseViewerState
	mp["drawables"] = hv.drawables
	mp["Los_tex"] = hv.Los_tex
	mp["temp_floor_drawers"] = hv.temp_floor_drawers
	mp["Edit_mode"] = hv.Edit_mode
	mp["bounds"] = hv.bounds
	mp["floor_drawers"] = hv.floor_drawers

	mpp := map[string]string{}
	for k, v := range mp {
		mpp[k] = fmt.Sprintf("%+v", v)
	}

	return fmt.Sprintf("%v", mpp)
}

func roomOverlapOnce(a, b *Room) bool {
	x1in := a.X+a.Size.Dx > b.X && a.X+a.Size.Dx <= b.X+b.Size.Dx
	x2in := b.X+b.Size.Dx > a.X && b.X+b.Size.Dx <= a.X+a.Size.Dx
	y1in := a.Y+a.Size.Dy > b.Y && a.Y+a.Size.Dy <= b.Y+b.Size.Dy
	y2in := b.Y+b.Size.Dy > a.Y && b.Y+b.Size.Dy <= a.Y+a.Size.Dy
	return (x1in || x2in) && (y1in || y2in)
}

func roomOverlap(a, b *Room) bool {
	return roomOverlapOnce(a, b) || roomOverlapOnce(b, a)
}

func (hv *HouseViewer) FindClosestDoorPos(door *Door, bx, by float32) *Room {
	current_floor := 0
	best := 1.0e9 // If this is unsafe then the house is larger than earth
	var best_room *Room

	clampToBoardSpace := func(nf float32, low, high BoardSpaceUnit) BoardSpaceUnit {
		n := BoardSpaceUnit(nf)
		return max(min(n, high), low)
	}
	for _, room := range hv.house.Floors[current_floor].Rooms {
		fl := math.Abs(float64(by) - float64(room.Y+room.Size.Dy))
		fr := math.Abs(float64(bx) - float64(room.X+room.Size.Dx))
		if bx < float32(room.X) {
			fl += float64(room.X) - float64(bx)
		}
		if bx > float32(room.X+room.Size.Dx) {
			fl += float64(bx) - float64(room.X+room.Size.Dx)
		}
		if by < float32(room.Y) {
			fr += float64(room.Y) - float64(by)
		}
		if by > float32(room.Y+room.Size.Dy) {
			fr += float64(by) - float64(room.Y+room.Size.Dy)
		}
		if best <= fl && best <= fr {
			continue
		}
		best_room = room
		switch {
		case fl < fr:
			best = fl
			door.Facing = FarLeft
			door.Pos = clampToBoardSpace(bx-float32(room.X)-float32(door.Width)/2, 0, room.Size.Dx-door.Width)

			//      case fr < fl:  this case must be true, so we just call it default here
		default:
			best = fr
			door.Facing = FarRight
			door.Pos = clampToBoardSpace(by-float32(room.Y)-float32(door.Width)/2, 0, room.Size.Dy-door.Width)
		}
	}
	return best_room
}

func (hv *HouseViewer) FindClosestExistingDoor(bx, by float32) (*Room, *Door) {
	current_floor := 0
	for _, room := range hv.house.Floors[current_floor].Rooms {
		for _, door := range room.Doors {
			if door.Facing != FarLeft && door.Facing != FarRight {
				continue
			}
			var vx, vy float32
			if door.Facing == FarLeft {
				vx = float32(room.X+door.Pos) + float32(door.Width)/2
				vy = float32(room.Y + room.Size.Dy)
			} else {
				// door.Facing == FarRight
				vx = float32(room.X + room.Size.Dx)
				vy = float32(room.Y+door.Pos) + float32(door.Width)/2
			}
			dsq := (vx-bx)*(vx-bx) + (vy-by)*(vy-by)
			if dsq <= float32(door.Width*door.Width) {
				return room, door
			}
		}
	}
	return nil, nil
}

type offsetDrawable struct {
	Drawable
	dx, dy BoardSpaceUnit
}

func (o offsetDrawable) FPos() (float64, float64) {
	x, y := o.Drawable.FPos()
	return x + float64(o.dx), y + float64(o.dy)
}
func (o offsetDrawable) FloorPos() (BoardSpaceUnit, BoardSpaceUnit) {
	x, y := o.Drawable.FloorPos()
	return x + o.dx, y + o.dy
}

func (hv *HouseViewer) Draw(region gui.Region, ctx gui.DrawingContext) {
	logging.Debug("HouseViewer.Draw", "hv", hv)
	region.PushClipPlanes()
	defer region.PopClipPlanes()

	hv.Render_region = region

	hv.temp_floor_drawers = hv.temp_floor_drawers[0:0]
	if hv.Edit_mode {
		for _, spawn := range hv.house.Floors[0].Spawns {
			hv.temp_floor_drawers = append(hv.temp_floor_drawers, spawn)
		}
	}
	for _, fd := range hv.floor_drawers {
		hv.temp_floor_drawers = append(hv.temp_floor_drawers, fd)
	}

	hv.house.Floors[0].render(region, hv.fx, hv.fy, hv.angle, hv.zoom, hv.drawables, hv.Los_tex, hv.temp_floor_drawers)
}
