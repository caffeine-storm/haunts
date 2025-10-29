package house

import (
	"path/filepath"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/MobRulesGames/mathgl"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/util/algorithm"
)

type RectObject interface {
	// Position in board space relative to the entire floor.
	FloorPos() (BoardSpaceUnit, BoardSpaceUnit)

	// Dimensions in board space.
	Dims() (BoardSpaceUnit, BoardSpaceUnit)
}

type RenderOnFloorer interface {
	// Draws stuff on the floor.  This will be called after the floor and all
	// textures on it have been drawn, but before furniture has been drawn.
	RenderOnFloor()

	RectObject
}

type Drawable interface {
	RectObject
	FPos() (float64, float64)
	Render(pos mathgl.Vec2, width float32)
	Color() (r, g, b, a byte)
}

type HouseDef struct {
	Name string

	Icon texture.Object

	Floors []*Floor
}

func MakeHouseDef() *HouseDef {
	var h HouseDef
	h.Name = "name"
	h.Floors = append(h.Floors, &Floor{})
	return &h
}

// Shifts the rooms in all floors such that the coordinates of all rooms are as
// low on each axis as possible without being zero or negative.
func (h *HouseDef) Normalize() {
	for i := range h.Floors {
		if len(h.Floors[i].Rooms) == 0 {
			continue
		}
		var minx, miny BoardSpaceUnit
		minx, miny = h.Floors[i].Rooms[0].FloorPos()
		for j := range h.Floors[i].Rooms {
			x, y := h.Floors[i].Rooms[j].FloorPos()
			if x < minx {
				minx = x
			}
			if y < miny {
				miny = y
			}
		}
		for j := range h.Floors[i].Rooms {
			h.Floors[i].Rooms[j].X -= minx - 1
			h.Floors[i].Rooms[j].Y -= miny - 1
		}
		for _, sp := range h.Floors[0].Spawns {
			sp.X -= minx - 1
			sp.Y -= miny - 1
		}
	}
}

type HouseEditor struct {
	*gui.HorizontalTable
	tab     *gui.TabFrame
	widgets []tabWidget

	house  HouseDef
	viewer *HouseViewer
}

func (he *HouseEditor) GetViewer() Viewer {
	return he.viewer
}

func (w *HouseEditor) SelectTab(n int) {
	if n < 0 || n >= len(w.widgets) {
		return
	}
	if n != w.tab.SelectedTab() {
		w.widgets[w.tab.SelectedTab()].Collapse()
		w.tab.SelectTab(n)
		// w.viewer.SetEditMode(editNothing)
		w.widgets[n].Expand()
	}
}

type houseDataTab struct {
	*gui.VerticalTable

	name       *gui.TextEditLine
	num_floors *gui.ComboBox
	icon       *gui.FileWidget

	house  *HouseDef
	viewer *HouseViewer

	// Distance from the mouse to the center of the object, in board coordinates
	drag_anchor struct{ x, y float32 }

	// Which floor we are viewing and editing
	current_floor int

	temp_room, prev_room *Room

	temp_spawns []*SpawnPoint
}

// TODO(tmckee): add test coverage for this!
func makeHouseDataTab(house *HouseDef, viewer *HouseViewer) *houseDataTab {
	var hdt houseDataTab
	hdt.VerticalTable = gui.MakeVerticalTable()
	hdt.house = house
	hdt.viewer = viewer

	hdt.name = gui.MakeTextEditLine("standard_18", "name", 300, 1, 1, 1, 1)
	num_floors_options := []string{"1 Floor", "2 Floors", "3 Floors", "4 Floors"}
	hdt.num_floors = gui.MakeComboTextBox(num_floors_options, 300)
	hdt.house.Icon.ResetPath(base.Path(filepath.Join(datadir, "houses", "icons")))
	hdt.icon = gui.MakeFileWidget(hdt.house.Icon.GetPath(), imagePathFilter)

	hdt.VerticalTable.AddChild(hdt.name)
	hdt.VerticalTable.AddChild(hdt.num_floors)
	hdt.VerticalTable.AddChild(hdt.icon)

	names := GetAllRoomNames()
	room_buttons := gui.MakeVerticalTable()
	for _, name := range names {
		n := name
		room_buttons.AddChild(gui.MakeButton("standard_18", name, 300, 1, 1, 1, 1, func(gui.EventHandlingContext, int64) {
			if hdt.temp_room != nil {
				return
			}
			hdt.temp_room = &Room{Defname: n}
			base.GetObject("rooms", hdt.temp_room)
			hdt.temp_room.temporary = true
			hdt.temp_room.invalid = true
			hdt.house.Floors[0].Rooms = append(hdt.house.Floors[0].Rooms, hdt.temp_room)
			hdt.drag_anchor.x = float32(hdt.temp_room.Size.Dx / 2)
			hdt.drag_anchor.y = float32(hdt.temp_room.Size.Dy / 2)
		}))
	}
	scroller := gui.MakeScrollFrame(room_buttons, 300, 700)
	hdt.VerticalTable.AddChild(scroller)
	return &hdt
}
func (hdt *houseDataTab) Think(ui *gui.Gui, t int64) {
	if hdt.temp_room != nil {
		mx, my := ui.GetLastMousePosition().XY()
		bx, by := hdt.viewer.WindowToBoard(mx, my)
		cx, cy := hdt.temp_room.FloorPos()
		hdt.temp_room.X = BoardSpaceUnit(bx - hdt.drag_anchor.x)
		hdt.temp_room.Y = BoardSpaceUnit(by - hdt.drag_anchor.y)
		dx := hdt.temp_room.X - cx
		dy := hdt.temp_room.Y - cy
		for i := range hdt.temp_spawns {
			hdt.temp_spawns[i].X += dx
			hdt.temp_spawns[i].Y += dy
		}
		hdt.temp_room.invalid = !hdt.house.Floors[0].canAddRoom(hdt.temp_room)
	}
	hdt.VerticalTable.Think(ui, t)
	num_floors := hdt.num_floors.GetComboedIndex() + 1
	if len(hdt.house.Floors) != num_floors {
		for len(hdt.house.Floors) < num_floors {
			hdt.house.Floors = append(hdt.house.Floors, &Floor{})
		}
		if len(hdt.house.Floors) > num_floors {
			hdt.house.Floors = hdt.house.Floors[0:num_floors]
		}
	}
	hdt.house.Name = hdt.name.GetText()
	hdt.house.Icon.ResetPath(base.Path(hdt.icon.GetPath()))
}

func (hdt *houseDataTab) onEscape() {
	if hdt.prev_room != nil {
		dx := hdt.prev_room.X - hdt.temp_room.X
		dy := hdt.prev_room.Y - hdt.temp_room.Y
		for i := range hdt.temp_spawns {
			hdt.temp_spawns[i].X += dx
			hdt.temp_spawns[i].Y += dy
		}
		*hdt.temp_room = *hdt.prev_room
		hdt.prev_room = nil
	} else {
		algorithm.Choose(&hdt.house.Floors[0].Rooms, func(r *Room) bool {
			return r != hdt.temp_room
		})
	}
	hdt.temp_room = nil
}

func (hdt *houseDataTab) Respond(ui *gui.Gui, group gui.EventGroup) bool {
	if hdt.VerticalTable.Respond(ui, group) {
		return true
	}

	if group.IsPressed(gin.AnyEscape) {
		hdt.onEscape()
		return true
	}

	if group.IsPressed(gin.AnyBackspace) || group.IsPressed(gin.AnyKeyDelete) {
		if hdt.temp_room != nil {
			spawns := make(map[*SpawnPoint]bool)
			for i := range hdt.temp_spawns {
				spawns[hdt.temp_spawns[i]] = true
			}
			algorithm.Choose(&hdt.house.Floors[0].Spawns, func(s *SpawnPoint) bool {
				return !spawns[s]
			})
			algorithm.Choose(&hdt.house.Floors[0].Rooms, func(r *Room) bool {
				return r != hdt.temp_room
			})
			hdt.temp_room = nil
			hdt.prev_room = nil
			hdt.viewer.SetBounds()
		}
		return true
	}

	floor := hdt.house.Floors[hdt.current_floor]
	if group.IsPressed(gin.AnyMouseLButton) {
		if hdt.temp_room != nil {
			if !hdt.temp_room.invalid {
				hdt.temp_room.temporary = false
				floor.removeInvalidDoors()
				hdt.temp_room = nil
				hdt.prev_room = nil
				hdt.viewer.SetBounds()
			}
		} else {
			if mpos, ok := ui.UseMousePosition(group); ok {
				cx, cy := mpos.X, mpos.Y
				bx, by := hdt.viewer.WindowToBoard(cx, cy)
				for i := range floor.Rooms {
					x, y := floor.Rooms[i].FloorPos()
					dx, dy := floor.Rooms[i].Dims()
					if int(bx) >= int(x) && int(bx) < int(x+dx) && int(by) >= int(y) && int(by) < int(y+dy) {
						hdt.temp_room = floor.Rooms[i]
						hdt.prev_room = new(Room)
						*hdt.prev_room = *hdt.temp_room
						hdt.temp_room.temporary = true
						hdt.drag_anchor.x = bx - float32(x)
						hdt.drag_anchor.y = by - float32(y)
						break
					}
				}
			}
			if hdt.temp_room != nil {
				hdt.temp_spawns = hdt.temp_spawns[0:0]
				for _, sp := range hdt.house.Floors[0].Spawns {
					x, y := sp.FloorPos()
					rx, ry := hdt.temp_room.FloorPos()
					rdx, rdy := hdt.temp_room.Dims()
					if x >= rx && x < rx+rdx && y >= ry && y < ry+rdy {
						hdt.temp_spawns = append(hdt.temp_spawns, sp)
					}
				}
			}
		}
		return true
	}

	return false
}
func (hdt *houseDataTab) Collapse() {}
func (hdt *houseDataTab) Expand()   {}
func (hdt *houseDataTab) Reload() {
	hdt.name.SetText(hdt.house.Name)
	hdt.icon.SetPath(hdt.house.Icon.GetPath())
}

type houseDoorTab struct {
	*gui.VerticalTable

	num_floors *gui.ComboBox

	house  *HouseDef
	viewer *HouseViewer

	// Distance from the mouse to the center of the object, in board coordinates
	drag_anchor struct{ x, y float32 }

	// Which floor we are viewing and editing
	current_floor int

	temp_room, prev_room *Room
	temp_door, prev_door *Door
}

func makeHouseDoorTab(house *HouseDef, viewer *HouseViewer) *houseDoorTab {
	var hdt houseDoorTab
	hdt.VerticalTable = gui.MakeVerticalTable()
	hdt.house = house
	hdt.viewer = viewer

	names := GetAllDoorNames()
	door_buttons := gui.MakeVerticalTable()
	for _, name := range names {
		n := name
		door_buttons.AddChild(gui.MakeButton("standard_18", name, 300, 1, 1, 1, 1, func(gui.EventHandlingContext, int64) {
			if len(hdt.house.Floors[0].Rooms) < 2 || hdt.temp_door != nil {
				return
			}
			hdt.temp_door = MakeDoor(n)
			hdt.temp_door.temporary = true
			hdt.temp_door.invalid = true
			hdt.temp_room = hdt.house.Floors[0].Rooms[0]
		}))
	}
	scroller := gui.MakeScrollFrame(door_buttons, 300, 700)
	hdt.VerticalTable.AddChild(scroller)
	return &hdt
}
func (hdt *houseDoorTab) Think(ui *gui.Gui, t int64) {
	hdt.VerticalTable.Think(ui, t)
}
func (hdt *houseDoorTab) onEscape() {
	if hdt.temp_door != nil {
		if hdt.temp_room != nil {
			algorithm.Choose(&hdt.temp_room.Doors, func(d *Door) bool {
				return d != hdt.temp_door
			})
		}
		if hdt.prev_door != nil {
			hdt.prev_room.Doors = append(hdt.prev_room.Doors, hdt.prev_door)
			hdt.prev_door.state.pos = -1 // forces it to redo its gl data
			hdt.prev_door = nil
			hdt.prev_room = nil
		}
		hdt.temp_door = nil
		hdt.temp_room = nil
	}
}
func (hdt *houseDoorTab) Respond(ui *gui.Gui, group gui.EventGroup) bool {
	if hdt.VerticalTable.Respond(ui, group) {
		return true
	}

	if group.IsPressed(gin.AnyEscape) {
		hdt.onEscape()
		return true
	}

	if group.IsPressed(gin.AnyBackspace) || group.IsPressed(gin.AnyKeyDelete) {
		algorithm.Choose(&hdt.temp_room.Doors, func(d *Door) bool {
			return d != hdt.temp_door
		})
		hdt.temp_room = nil
		hdt.temp_door = nil
		hdt.prev_room = nil
		hdt.prev_door = nil
		return true
	}

	var bx, by float32
	mpos, isMouseEvent := ui.UseMousePosition(group)
	if isMouseEvent {
		bx, by = hdt.viewer.WindowToBoard(mpos.X, mpos.Y)
	}
	if isMouseEvent && hdt.temp_door != nil {
		room := hdt.viewer.FindClosestDoorPos(hdt.temp_door, bx, by)
		if room != hdt.temp_room {
			algorithm.Choose(&hdt.temp_room.Doors, func(d *Door) bool {
				return d != hdt.temp_door
			})
			hdt.temp_room = room
			hdt.temp_door.invalid = (hdt.temp_room == nil)
			hdt.temp_room.Doors = append(hdt.temp_room.Doors, hdt.temp_door)
		}
		if hdt.temp_room == nil {
			hdt.temp_door.invalid = true
		} else {
			other_room, _ := hdt.house.Floors[0].findRoomForDoor(hdt.temp_room, hdt.temp_door)
			hdt.temp_door.invalid = (other_room == nil)
		}
	}

	floor := hdt.house.Floors[hdt.current_floor]
	if group.IsPressed(gin.AnyMouseLButton) {
		if hdt.temp_door != nil {
			other_room, other_door := floor.findRoomForDoor(hdt.temp_room, hdt.temp_door)
			if other_room != nil {
				other_room.Doors = append(other_room.Doors, other_door)
				hdt.temp_door.temporary = false
				hdt.temp_door = nil
				hdt.prev_door = nil
			}
		} else {
			hdt.temp_room, hdt.temp_door = hdt.viewer.FindClosestExistingDoor(bx, by)
			if hdt.temp_door != nil {
				hdt.prev_door = new(Door)
				*hdt.prev_door = *hdt.temp_door
				hdt.prev_room = hdt.temp_room
				hdt.temp_door.temporary = true
				room, door := hdt.house.Floors[0].FindMatchingDoor(hdt.temp_room, hdt.temp_door)
				if room != nil {
					algorithm.Choose(&room.Doors, func(d *Door) bool {
						return d != door
					})
				}
			}
		}
		return true
	}

	return false
}
func (hdt *houseDoorTab) Collapse() {
	hdt.onEscape()
}
func (hdt *houseDoorTab) Expand() {
}
func (hdt *houseDoorTab) Reload() {
	hdt.onEscape()
}

type houseRelicsTab struct {
	*gui.VerticalTable

	spawn_name *gui.TextEditLine
	make_spawn *gui.Button
	typed_name string

	house  *HouseDef
	viewer *HouseViewer

	// Which floor we are viewing and editing
	current_floor int

	temp_relic, prev_relic *SpawnPoint

	drag_anchor struct{ x, y float32 }
}

// TODO(tmckee#34): 'hdt' seems to be an artifact of copy-pasting
// 'houseDoorTab'? -_-
func (hdt *houseRelicsTab) newSpawn() {
	hdt.temp_relic = new(SpawnPoint)
	hdt.temp_relic.Name = hdt.spawn_name.GetText()
	hdt.temp_relic.X = 10000
	hdt.temp_relic.Dx = 2
	hdt.temp_relic.Dy = 2
	hdt.temp_relic.temporary = true
	hdt.temp_relic.invalid = true
	hdt.house.Floors[0].Spawns = append(hdt.house.Floors[0].Spawns, hdt.temp_relic)
}

func makeHouseRelicsTab(house *HouseDef, viewer *HouseViewer) *houseRelicsTab {
	var hdt houseRelicsTab
	hdt.VerticalTable = gui.MakeVerticalTable()
	hdt.house = house
	hdt.viewer = viewer

	hdt.VerticalTable.AddChild(gui.MakeTextLine("standard_18", "Spawns", 300, 1, 1, 1, 1))
	hdt.spawn_name = gui.MakeTextEditLine("standard_18", "", 300, 1, 1, 1, 1)
	hdt.VerticalTable.AddChild(hdt.spawn_name)

	hdt.make_spawn = gui.MakeButton("standard_18", "New Spawn Point", 300, 1, 1, 1, 1, func(gui.EventHandlingContext, int64) {
		hdt.newSpawn()
	})
	hdt.VerticalTable.AddChild(hdt.make_spawn)

	return &hdt
}

func (hdt *houseRelicsTab) onEscape() {
	if hdt.temp_relic != nil {
		if hdt.prev_relic != nil {
			*hdt.temp_relic = *hdt.prev_relic
			hdt.prev_relic = nil
		} else {
			algorithm.Choose(&hdt.house.Floors[0].Spawns, func(s *SpawnPoint) bool {
				return s != hdt.temp_relic
			})
		}
		hdt.temp_relic = nil
	}
}

func (hdt *houseRelicsTab) markTempSpawnValidity() {
	hdt.temp_relic.invalid = false
	floor := hdt.house.Floors[0]
	var room *Room
	x, y := hdt.temp_relic.FloorPos()
	for ix := BoardSpaceUnit(0); ix < hdt.temp_relic.Dx; ix++ {
		for iy := BoardSpaceUnit(0); iy < hdt.temp_relic.Dy; iy++ {
			room_at, furn_at, _ := floor.RoomFurnSpawnAtPos(x+ix, y+iy)
			if room == nil {
				room = room_at
			}
			if room_at == nil || room_at != room || furn_at != nil {
				hdt.temp_relic.invalid = true
				return
			}
		}
	}
}

func (hdt *houseRelicsTab) Think(ui *gui.Gui, t int64) {
	defer hdt.VerticalTable.Think(ui, t)
	mx, my := ui.GetLastMousePosition().XY()
	rbx, rby := hdt.viewer.WindowToBoard(mx, my)
	bx := BoardSpaceUnit(roundDown(rbx - hdt.drag_anchor.x + 0.5))
	by := BoardSpaceUnit(roundDown(rby - hdt.drag_anchor.y + 0.5))
	if hdt.temp_relic != nil {
		hdt.temp_relic.X = bx
		hdt.temp_relic.Y = by
		deltaX := gin.In().GetKeyById(gin.AnyRight).FramePressCount() - gin.In().GetKeyById(gin.AnyLeft).FramePressCount()
		hdt.temp_relic.Dx += BoardSpaceUnit(deltaX)
		if hdt.temp_relic.Dx < 1 {
			hdt.temp_relic.Dx = 1
		}
		if hdt.temp_relic.Dx > 10 {
			hdt.temp_relic.Dx = 10
		}
		deltaY := gin.In().GetKeyById(gin.AnyUp).FramePressCount() - gin.In().GetKeyById(gin.AnyDown).FramePressCount()
		hdt.temp_relic.Dy += BoardSpaceUnit(deltaY)
		if hdt.temp_relic.Dy < 1 {
			hdt.temp_relic.Dy = 1
		}
		if hdt.temp_relic.Dy > 10 {
			hdt.temp_relic.Dy = 10
		}
		hdt.markTempSpawnValidity()
	} else {
		intx, inty := roundDown(rbx), roundDown(rby)
		_, _, spawn_at := hdt.house.Floors[0].RoomFurnSpawnAtPos(BoardSpaceUnitPair(intx, inty))
		if spawn_at != nil {
			hdt.spawn_name.SetText(spawn_at.Name)
		} else if hdt.spawn_name.IsBeingEdited() {
			hdt.typed_name = hdt.spawn_name.GetText()
		} else {
			hdt.spawn_name.SetText(hdt.typed_name)
		}
	}

	// TODO(tmckee): do we need to distinguish between 'N' and 'n'? This was
	// originally 'n'.
	if hdt.temp_relic == nil && gin.In().GetKeyById(gin.AnyKeyN).FramePressCount() > 0 && ui.FocusWidget() == nil {
		hdt.newSpawn()
	}
}

// Rounds a float32 down, instead of towards zero
func roundDown(f float32) int {
	if f >= 0 {
		return int(f)
	}
	return int(f - 1)
}

func (hdt *houseRelicsTab) Respond(ui *gui.Gui, group gui.EventGroup) bool {
	if hdt.VerticalTable.Respond(ui, group) {
		return true
	}

	if group.IsPressed(gin.AnyEscape) {
		hdt.onEscape()
		return true
	}

	if group.IsPressed(gin.AnyBackspace) || group.IsPressed(gin.AnyKeyDelete) {
		algorithm.Choose(&hdt.house.Floors[0].Spawns, func(s *SpawnPoint) bool {
			return s != hdt.temp_relic
		})
		hdt.temp_relic = nil
		hdt.prev_relic = nil
		return true
	}

	floor := hdt.house.Floors[hdt.current_floor]
	if group.IsPressed(gin.AnyMouseLButton) {
		if mpos, ok := ui.UseMousePosition(group); ok {
			if hdt.temp_relic != nil {
				if !hdt.temp_relic.invalid {
					hdt.temp_relic.temporary = false
					hdt.temp_relic = nil
				}
			} else {
				for _, sp := range floor.Spawns {
					fbx, fby := hdt.viewer.WindowToBoard(mpos.X, mpos.Y)
					bx, by := BoardSpaceUnitPair(roundDown(fbx), roundDown(fby))
					x, y := sp.FloorPos()
					dx, dy := sp.Dims()
					if bx >= x && bx < x+dx && by >= y && by < y+dy {
						hdt.temp_relic = sp
						hdt.prev_relic = new(SpawnPoint)
						*hdt.prev_relic = *hdt.temp_relic
						hdt.temp_relic.temporary = true
						hdt.drag_anchor.x = fbx - float32(hdt.temp_relic.X)
						hdt.drag_anchor.y = fby - float32(hdt.temp_relic.Y)
						break
					}
				}
			}
		}
	}
	return false
}

func (hdt *houseRelicsTab) Collapse() {
	PopSpawnRegexp()
	hdt.onEscape()
}
func (hdt *houseRelicsTab) Expand() {
	PushSpawnRegexp(".*")
}
func (hdt *houseRelicsTab) Reload() {
	hdt.onEscape()
}

func (h *HouseDef) Save(path string) {
	base.SaveJson(path, h)
}

func LoadAllHousesInDir(dir string) {
	base.RemoveRegistry("houses")
	base.RegisterRegistry("houses", make(map[string]*HouseDef))
	base.RegisterAllObjectsInDir("houses", dir, ".house", "json")
}

func (h *HouseDef) setDoorsOpened(opened bool) {
	for _, floor := range h.Floors {
		for _, room := range floor.Rooms {
			for _, door := range room.Doors {
				door.Opened = opened
			}
		}
	}
}

type iamanidiotcontainer struct {
	Defname string
	*HouseDef
}

func MakeHouseFromName(name string) *HouseDef {
	var idiot iamanidiotcontainer
	idiot.Defname = name
	base.GetObject("houses", &idiot)
	idiot.HouseDef.setDoorsOpened(false)
	return idiot.HouseDef
}

func MakeHouseFromPath(path string) (*HouseDef, error) {
	var house HouseDef
	err := base.LoadAndProcessObject(path, "json", &house)
	if err != nil {
		return nil, err
	}
	house.Normalize()
	house.setDoorsOpened(false)
	return &house, nil
}
