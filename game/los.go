package game

import (
	"fmt"
	"regexp"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/mrgnet"
	"github.com/go-gl-legacy/gl"
)

type Purpose int

const (
	PurposeNone Purpose = iota
	PurposeRelic
	PurposeMystery
	PurposeCleanse
)

type LosMode int

const (
	LosModeNone LosMode = iota
	LosModeBlind
	LosModeAll
	LosModeEntities
	LosModeRooms
)

type turnState int

const (
	// Waiting for the script to finish Init()
	turnStateInit turnState = iota

	// Waiting for the script to finish RoundStart()
	turnStateStart

	// Waiting for or running an Ai action
	turnStateAiAction

	// Waiting for the script to finish OnAction()
	turnStateScriptOnAction

	// Humans and Ai are done, now the script can run some actions if it wants
	turnStateMainPhaseOver

	// Waiting for the script to finish OnEnd()
	turnStateEnd
)

type sideLosData struct {
	mode LosMode
	tex  *house.LosTexture
}

type Waypoint struct {
	Name   string
	Side   Side
	X, Y   float64
	Radius float64
	Active bool
	drawn  bool
	// Color, maybe?
}

func (wp *Waypoint) Dims() (house.BoardSpaceUnit, house.BoardSpaceUnit) {
	return house.BoardSpaceUnitPair(2*wp.Radius, 2*wp.Radius)
}

func (wp *Waypoint) FloorPos() (house.BoardSpaceUnit, house.BoardSpaceUnit) {
	return house.BoardSpaceUnitPair(wp.X, wp.Y)
}

func (wp *Waypoint) RenderOnFloor() {
	if !wp.Active {
		return
	}
	wp.drawn = true
	gl.Color4ub(200, 0, 0, 128)
	base.EnableShader("waypoint")
	base.SetUniformF("waypoint", "radius", float32(wp.Radius))

	t := float32(time.Now().UnixNano()%1e15) / 1.0e9
	base.SetUniformF("waypoint", "time", t)
	gl.Begin(gl.QUADS)
	gl.TexCoord2i(0, 1)
	gl.Vertex2i(int(wp.X-wp.Radius), int(wp.Y-wp.Radius))
	gl.TexCoord2i(0, 0)
	gl.Vertex2i(int(wp.X-wp.Radius), int(wp.Y+wp.Radius))
	gl.TexCoord2i(1, 0)
	gl.Vertex2i(int(wp.X+wp.Radius), int(wp.Y+wp.Radius))
	gl.TexCoord2i(1, 1)
	gl.Vertex2i(int(wp.X+wp.Radius), int(wp.Y-wp.Radius))
	gl.End()

	base.EnableShader("")
}

type gameDataTransient struct {
	los struct {
		denizens, intruders sideLosData

		// When merging the los from different entities we'll do it here, and we
		// keep it around to avoid reallocating it every time we need it.
		full_merger []bool
		merger      [][]bool
	}

	// Used to sync up with the script, the value passed is usually nil, but
	// whenever an action happens it will get passed along this channel too.
	comm struct {
		script_to_game chan interface{}
		game_to_script chan interface{}
	}

	script *gameScript

	// Indicates if we're waiting for a script to run or something
	Turn_state   turnState
	Action_state actionState

	net struct {
		key  mrgnet.GameKey
		game *mrgnet.Game
		side Side
	}
}

func (gdt *gameDataTransient) alloc() {
	if gdt.los.denizens.tex != nil {
		return
	}
	// TODO(#8): farm these MakeLosTexture calls out to a render thread and
	// update house.MakeLosTexture to expect to be called from a render thread.
	gdt.los.denizens.tex = house.MakeLosTexture()
	gdt.los.intruders.tex = house.MakeLosTexture()
	gdt.los.full_merger = make([]bool, house.LosTextureSizeSquared)
	gdt.los.merger = make([][]bool, house.LosTextureSize)
	for i := range gdt.los.merger {
		gdt.los.merger[i] = gdt.los.full_merger[i*house.LosTextureSize : (i+1)*house.LosTextureSize]
	}

	gdt.comm.script_to_game = make(chan interface{}, 1)
	gdt.comm.game_to_script = make(chan interface{}, 1)

	logging.Info("gdt.alloc leaving gdt.script as nil")
}

type gameDataPrivate struct {
	// Hacky - but gives us a way to prevent selecting ents and whatnot while
	// any kind of modal dialog box is up.
	modal bool
}
type spawnLos struct {
	Pattern string
	r       *regexp.Regexp
}
type gameDataGobbable struct {
	// TODO: No idea if this thing can be loaded from the registry - should
	// probably figure that out at some point
	House *house.HouseDef
	Ents  []*Entity

	// True for online games
	Net bool

	// Set of all Entities that are still resident.  This is so we can safely
	// clean things up since they will all have ais running in the background
	// preventing them from getting GCed.
	all_ents_in_game   map[*Entity]bool
	all_ents_in_memory map[*Entity]bool

	// Regexps.  Any spawn points with names matching this pattern will grant
	// los to the appropriate side.
	Los_spawns struct {
		Denizens, Intruders spawnLos
	}

	// Next unique EntityId to be assigned
	Entity_id EntityId

	// Current player
	Side Side

	// Current turn number - incremented on each OnRound() so every two
	// indicates that a complete round has happened.
	Turn int

	// PRNG, need it here so that we serialize it along with everything
	// else so that replays work properly.
	Rand gobbablePrng

	// Waypoints, used for signaling things to the player on the map
	Waypoints []Waypoint

	// Transient data - none of the following are exported

	player_inactive bool

	viewer *house.HouseViewer

	// If the user is dragging around a new Entity to place, this is it
	new_ent *Entity

	selected_ent *Entity
	hovered_ent  *Entity

	// Stores the current acting entity - if it is an Ai controlled entity
	ai_ent *Entity

	// TODO(tmckee): this _is_ exported contrary to a comment above. Need to find
	// out if it shouldn't be exported or what.
	Ai struct {
		Path struct {
			Minions, Denizens, Intruders string
		}
		minions, denizens, intruders Ai
	}

	// If an Ai is executing currently it is referenced here
	active_ai Ai

	current_exec   ActionExec
	current_action Action
}

type actionState int

const (
	noAction actionState = iota

	// The Ai is running and determining the next action to run
	waitingAction

	// The player has selected an action and is determining whether or not to
	// use it, and how.
	preppingAction

	// Check the scripts to see if the action should be modified or cancelled.
	verifyingAction

	// An action is currently running, everything should pause while this runs.
	doingAction
)

func (as actionState) String() string {
	switch as {
	case noAction:
		return "noAction"
	case waitingAction:
		return "waitingAction"
	case preppingAction:
		return "preppingAction"
	case verifyingAction:
		return "verifyingAction"
	case doingAction:
		return "doingAction"
	}

	panic(fmt.Errorf("bad actionState number: %d", int(as)))
}

// x and y are given in room coordinates
func furnitureAt(room *house.Room, x, y house.BoardSpaceUnit) *house.Furniture {
	for _, f := range room.Furniture {
		fx, fy := f.FloorPos()
		fdx, fdy := f.Dims()
		if x >= fx && x < fx+fdx && y >= fy && y < fy+fdy {
			return f
		}
	}
	return nil
}

// x and y are given in floor coordinates
func roomAt(floor *house.Floor, x, y house.BoardSpaceUnit) *house.Room {
	for _, room := range floor.Rooms {
		rx, ry := room.FloorPos()
		rdx, rdy := room.Dims()
		if x >= rx && x < rx+rdx && y >= ry && y < ry+rdy {
			return room
		}
	}
	return nil
}

func connected(r, r2 *house.Room, x, y, x2, y2 house.BoardSpaceUnit) bool {
	if r == r2 {
		return true
	}
	x -= house.BoardSpaceUnit(r.X)
	y -= house.BoardSpaceUnit(r.Y)
	x2 -= house.BoardSpaceUnit(r2.X)
	y2 -= house.BoardSpaceUnit(r2.Y)
	var facing house.WallFacing
	if x == 0 && x2 != 0 {
		facing = house.NearLeft
	} else if y == 0 && y2 != 0 {
		facing = house.NearRight
	} else if x != 0 && x2 == 0 {
		facing = house.FarRight
	} else if y != 0 && y2 == 0 {
		facing = house.FarLeft
	} else {
		// This shouldn't happen, but in case it does we certainly shouldn't treat
		// it as an open door
		return false
	}
	for _, door := range r.Doors {
		if door.Facing != facing {
			continue
		}
		var pos house.BoardSpaceUnit
		switch facing {
		case house.NearLeft:
			fallthrough
		case house.FarRight:
			pos = y

		case house.NearRight:
			fallthrough
		case house.FarLeft:
			pos = x
		}
		if pos >= door.Pos && pos < door.Pos+door.Width {
			return door.IsOpened()
		}
	}
	return false
}

type exclusionGraph struct {
	side Side
	los  bool
	ex   map[*Entity]bool
	g    *Game
}

func (eg *exclusionGraph) Adjacent(v int) ([]int, []float64) {
	return eg.g.adjacent(v, eg.los, eg.side, eg.ex)
}
func (eg *exclusionGraph) NumVertex() int {
	return eg.g.numVertex()
}

type roomGraph struct {
	g *Game
}

func (rg *roomGraph) NumVertex() int {
	return len(rg.g.House.Floors[0].Rooms)
}

func (rg *roomGraph) Adjacent(n int) ([]int, []float64) {
	room := rg.g.House.Floors[0].Rooms[n]
	var adj []int
	var cost []float64
	for _, door := range room.Doors {
		other_room, _ := rg.g.House.Floors[0].FindMatchingDoor(room, door)
		if other_room != nil {
			for i := range rg.g.House.Floors[0].Rooms {
				if other_room == rg.g.House.Floors[0].Rooms[i] {
					adj = append(adj, i)
					cost = append(cost, 1)
					break
				}
			}
		}
	}
	return adj, cost
}
