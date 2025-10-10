package house

import (
	"math"

	"github.com/MobRulesGames/haunts/house/perspective"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/mathgl"
	"github.com/runningwild/glop/gui"
)

type roomViewer struct {
	gui.Childless
	gui.EmbeddedWidget
	gui.BasicZone
	gui.StubDrawFocuseder
	gui.StubDoThinker

	room *Room

	// In case the size of the room changes we will need to update the matrices
	size RoomSize

	// Focus, in map coordinates
	fx, fy float32

	// The viewing angle, 0 means the map is viewed head-on, 90 means the map is viewed
	// on its edge (i.e. it would not be visible)
	angle float32

	// Zoom factor, 1.0 is standard
	zoom float32

	// The modelview matrices that are sent to opengl. Updated any time focus,
	// zoom, or viewing angle changes.
	roomMats perspective.RoomMats

	Temp struct {
		Furniture *Furniture
		Decal     *Decal
	}

	// This tells us what to highlight based on the mouse position
	edit_mode editMode

	dragging dragState
}

func (rv *roomViewer) SetEditMode(mode editMode) {
	rv.edit_mode = mode
}

func (rv *roomViewer) String() string {
	return "viewer"
}

func MakeRoomViewer(room *Room, angle float32) *roomViewer {
	var rv roomViewer
	rv.EmbeddedWidget = &gui.BasicWidget{CoreWidget: &rv}
	rv.room = room
	rv.angle = angle
	rv.fx = float32(rv.room.Size.Dx / 2)
	rv.fy = float32(rv.room.Size.Dy / 2)
	rv.zoom = 10.0
	rv.size = rv.room.Size
	rv.makeMat()
	rv.Request_dims.Dx = 100
	rv.Request_dims.Dy = 100
	rv.Ex = true
	rv.Ey = true

	return &rv
}

func (rv *roomViewer) AdjAngle(ang float32) {
	rv.angle = ang
	rv.makeMat()
}

func (rv *roomViewer) makeMat() {
	logging.Debug("roomViewer>makeMat", "rv", []any{
		rv.room.Size, rv.Render_region, rv.fx, rv.fy, rv.angle, rv.zoom,
	})
	rv.roomMats = perspective.MakeRoomMats(rv.room.Size.GetDx(), rv.room.Size.GetDy(), rv.Render_region, rv.fx, rv.fy, rv.angle, rv.zoom)
}

// Transforms a cursor position in window coordinates to board coordinates.
// TODO(tmckee): this should be returning BoardSpaceUnits, not float32s.
func (rv *roomViewer) WindowToBoard(wx, wy int) (float32, float32) {
	return rv.WindowToBoardf(float32(wx), float32(wy))
}
func (rv *roomViewer) WindowToBoardf(wx, wy float32) (float32, float32) {
	fx, fy, fdist := rv.modelviewToBoard(wx, wy)
	lbx, lby, ldist := rv.modelviewToLeftWall(wx, wy)
	rbx, rby, rdist := rv.modelviewToRightWall(wx, wy)
	if fdist < ldist && fdist < rdist {
		if fx > float32(rv.room.Size.Dx) {
			fx = float32(rv.room.Size.Dx)
		}
		if fy > float32(rv.room.Size.Dy) {
			fy = float32(rv.room.Size.Dy)
		}
		return fx, fy
	}
	if ldist < rdist {
		return lbx, lby
	}
	return rbx, rby
}

func (rv *roomViewer) BoardToWindow(bx, by float32) (int, int) {
	x, y := rv.BoardToWindowf(bx, by)
	return int(x), int(y)
}
func (rv *roomViewer) BoardToWindowf(bx, by float32) (float32, float32) {
	fx, fy, fz := rv.boardToModelview(float32(bx), float32(by))
	lbx, lby, lz := rv.leftWallToModelview(float32(bx), float32(by))
	rbx, rby, rz := rv.rightWallToModelview(float32(bx), float32(by))
	if fz < lz && fz < rz {
		return fx, fy
	}
	if lz < rz {
		return lbx, lby
	}
	return rbx, rby
}

func (rv *roomViewer) modelviewToLeftWall(mx, my float32) (x, y, dist float32) {
	mz := d2p(rv.roomMats.Left, mathgl.Vec3{X: mx, Y: my, Z: 0}, mathgl.Vec3{X: 0, Y: 0, Z: 1})
	v := mathgl.Vec4{X: mx, Y: my, Z: mz, W: 1}
	v.Transform(&rv.roomMats.ILeft)
	if v.X > float32(rv.room.Size.Dx) {
		v.X = float32(rv.room.Size.Dx)
	}
	return v.X, v.Y + float32(rv.room.Size.Dy), mz
}

func (rv *roomViewer) modelviewToRightWall(mx, my float32) (x, y, dist float32) {
	mz := d2p(rv.roomMats.Right, mathgl.Vec3{X: mx, Y: my, Z: 0}, mathgl.Vec3{X: 0, Y: 0, Z: 1})
	v := mathgl.Vec4{X: mx, Y: my, Z: mz, W: 1}
	v.Transform(&rv.roomMats.IRight)
	if v.Y > float32(rv.room.Size.Dy) {
		v.Y = float32(rv.room.Size.Dy)
	}
	return v.X + float32(rv.room.Size.Dx), v.Y, mz
}

func (rv *roomViewer) leftWallToModelview(bx, by float32) (x, y, z float32) {
	v := mathgl.Vec4{X: bx, Y: by - float32(rv.room.Size.Dy), W: 1}
	v.Transform(&rv.roomMats.Left)
	return v.X, v.Y, v.Z
}

func (rv *roomViewer) rightWallToModelview(bx, by float32) (x, y, z float32) {
	v := mathgl.Vec4{X: bx - float32(rv.room.Size.Dx), Y: by, W: 1}
	v.Transform(&rv.roomMats.Right)
	return v.X, v.Y, v.Z
}

// Distance to Plane(Point?)?  WTF IS THIS!?
// This is used to measure the distance from the given point to the plane
// defined by 'xfrmMtrix' whilst travelling along the given ray.
// e.g. you can find 'z' co-ordinate of the world-space point clicked on by a
// user if you pass the screen-space x/y point clicked and the z unit-vector.
func d2p(xfrmMatrix mathgl.Mat4, point, ray mathgl.Vec3) float32 {
	// Pull out the vector that encodes the 'translation' part of the xfrm.
	worldSpaceOrigin := mathgl.Vec3{
		X: xfrmMatrix[12],
		Y: xfrmMatrix[13],
		Z: xfrmMatrix[14],
	}

	// Move point to be relative to the world-space origin.
	point.Subtract(&worldSpaceOrigin)
	point.Scale(-1)

	var forward mathgl.Vec3
	forward.Assign(xfrmMatrix.GetForwardVec3())

	// What amount of the point's vector is parallel with the 'forward' vector?
	dist := point.Dot(&forward)

	ray.Normalize()
	cos := float64(forward.Dot(&ray))
	return dist / float32(cos)
}

func (rv *roomViewer) modelviewToBoard(mx, my float32) (x, y, dist float32) {
	mz := d2p(rv.roomMats.Floor, mathgl.Vec3{X: mx, Y: my, Z: 0}, mathgl.Vec3{X: 0, Y: 0, Z: 1})
	v := mathgl.Vec4{X: mx, Y: my, Z: mz, W: 1}
	v.Transform(&rv.roomMats.IFloor)
	return v.X, v.Y, mz
}

func (rv *roomViewer) boardToModelview(mx, my float32) (x, y, z float32) {
	v := mathgl.Vec4{X: mx, Y: my, W: 1}
	v.Transform(&rv.roomMats.Floor)
	x, y, z = v.X, v.Y, v.Z
	return
}

func (rv *roomViewer) GetFocus() (float32, float32) {
	return rv.fx, rv.fy
}

func (rv *roomViewer) SetFocusTarget(x, y float32) {
	// TODO(tmckee:clean): also 'approach' (x,y) like in HouseViewer for smoother
	// panning animation.
	rv.fx, rv.fy = x, y
	rv.makeMat()
}

func clamp(f, min, max float32) float32 {
	if f < min {
		return min
	}
	if f > max {
		return max
	}
	return f
}

// TODO(tmckee#40): this is 'AdjustZoom', not 'SetZoom'.
// Changes the current zoom from e^(zoom) to e^(zoom+dz)
func (rv *roomViewer) SetZoom(dz float32) {
	logging.Warn("SetZoom called but this is really AdjustZoom; see #40")
	if dz == 0 {
		logging.Warn("attempted to set zoom to 0")
		return
	}
	exp := math.Log(float64(rv.zoom)) + float64(dz)
	exp = float64(clamp(float32(exp), 2.5, 5.0))
	rv.zoom = float32(math.Exp(exp))
	rv.makeMat()
}

func (rv *roomViewer) GetZoom() float32 {
	return rv.zoom
}

func (rv *roomViewer) Draw(region gui.Region, ctx gui.DrawingContext) {
	region.PushClipPlanes()
	defer region.PopClipPlanes()

	if rv.Render_region != region {
		rv.Render_region = region
		rv.makeMat()
	}

	logging.Trace("roomViewer.Draw", "region", region, "rv", rv)

	rv.room.SetupGlStuff(&RoomRealGl{})
	rv.room.SetWallTransparency(false)
	rv.room.Render(rv.roomMats, rv.zoom, 255, nil, nil, nil)
}

func (rv *roomViewer) Think(*gui.Gui, int64) {
	if rv.size != rv.room.Size {
		rv.size = rv.room.Size
		rv.makeMat()
	}
}

func (rv *roomViewer) DoRespond(gui gui.EventHandlingContext, group gui.EventGroup) (consume, change_focus bool) {
	consume = rv.dragging.HandleDragEventGroup(rv, group)
	return
}
