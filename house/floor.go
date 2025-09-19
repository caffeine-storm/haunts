package house

import (
	"math"

	"github.com/MobRulesGames/haunts/house/perspective"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/util/algorithm"
)

type Floor struct {
	Rooms  []*Room `registry:"loadfrom-rooms"`
	Spawns []*SpawnPoint
}

func (f *Floor) getWallAlphas() []byte {
	ret := []byte{}

	for _, room := range f.Rooms {
		ret = append(ret, room.far_left.wall_alpha, room.far_right.wall_alpha)
	}

	return ret
}

func (f *Floor) canAddRoom(add *Room) bool {
	for _, room := range f.Rooms {
		if room.temporary {
			continue
		}
		if roomOverlap(room, add) {
			return false
		}
	}
	return true
}

func (f *Floor) FindMatchingDoor(room *Room, door *Door) (*Room, *Door) {
	for _, other_room := range f.Rooms {
		if other_room == room {
			continue
		}
		for _, other_door := range other_room.Doors {
			if door.Facing == FarLeft && other_door.Facing != NearRight {
				continue
			}
			if door.Facing == FarRight && other_door.Facing != NearLeft {
				continue
			}
			if door.Facing == NearLeft && other_door.Facing != FarRight {
				continue
			}
			if door.Facing == NearRight && other_door.Facing != FarLeft {
				continue
			}
			if door.Facing == FarLeft && other_room.Y != room.Y+room.Size.Dy {
				continue
			}
			if door.Facing == NearRight && room.Y != other_room.Y+other_room.Size.Dy {
				continue
			}
			if door.Facing == FarRight && other_room.X != room.X+room.Size.Dx {
				continue
			}
			if door.Facing == NearLeft && room.X != other_room.X+other_room.Size.Dx {
				continue
			}
			if door.Facing == FarLeft || door.Facing == NearRight {
				if door.Pos == other_door.Pos-(room.X-other_room.X) {
					return other_room, other_door
				}
			}
			if door.Facing == FarRight || door.Facing == NearLeft {
				if door.Pos == other_door.Pos-(room.Y-other_room.Y) {
					return other_room, other_door
				}
			}
		}
	}
	return nil, nil
}

func (f *Floor) findRoomForDoor(target *Room, door *Door) (*Room, *Door) {
	if !target.canAddDoor(door) {
		return nil, nil
	}

	if door.Facing == FarLeft {
		for _, room := range f.Rooms {
			if room.Y == target.Y+target.Size.Dy {
				temp := MakeDoor(door.Defname)
				temp.Pos = door.Pos - (room.X - target.X)
				temp.Facing = NearRight
				if room.canAddDoor(temp) {
					return room, temp
				}
			}
		}
	} else if door.Facing == FarRight {
		for _, room := range f.Rooms {
			if room.X == target.X+target.Size.Dx {
				temp := MakeDoor(door.Defname)
				temp.Pos = door.Pos - (room.Y - target.Y)
				temp.Facing = NearLeft
				if room.canAddDoor(temp) {
					return room, temp
				}
			}
		}
	}
	return nil, nil
}

func (f *Floor) canAddDoor(target *Room, door *Door) bool {
	r, _ := f.findRoomForDoor(target, door)
	return r != nil
}

func (f *Floor) removeInvalidDoors() {
	for _, room := range f.Rooms {
		algorithm.Choose(&room.Doors, func(a interface{}) bool {
			_, other_door := f.FindMatchingDoor(room, a.(*Door))
			return other_door != nil && !other_door.temporary
		})
	}
}

// TODO(tmckee#34): this should be named "GetAllTheThingsThatAreAtPos" or
// something.
func (f *Floor) RoomFurnSpawnAtPos(x, y BoardSpaceUnit) (room *Room, furn *Furniture, spawn *SpawnPoint) {
	for _, croom := range f.Rooms {
		rx, ry := croom.FloorPos()
		rdx, rdy := croom.Dims()
		if x < rx || y < ry || x >= rx+rdx || y >= ry+rdy {
			continue
		}
		room = croom
		for _, furniture := range room.Furniture {
			tx := x - rx
			ty := y - ry
			fx, fy := furniture.FloorPos()
			fdx, fdy := furniture.Dims()
			if tx < fx || ty < fy || tx >= fx+fdx || ty >= fy+fdy {
				continue
			}
			furn = furniture
			break
		}
		for _, sp := range f.Spawns {
			if sp.temporary {
				continue
			}
			if x >= sp.X && x < sp.X+sp.Dx && y >= sp.Y && y < sp.Y+sp.Dy {
				spawn = sp
				break
			}
		}
		return
	}
	return
}

func (f *Floor) render(region gui.Region, focusx, focusy, angle, zoom float32, drawables []Drawable, los_tex *LosTexture, floor_drawers []RenderOnFloorer) {
	logging.Trace("Floor.render", "rooms", f.Rooms, "region", region)
	roomsToDraw := make([]*Room, len(f.Rooms))
	copy(roomsToDraw, f.Rooms)
	// Do not include temporary objects in the ordering, since they will likely
	// overlap with other objects and make it difficult to determine the proper
	// ordering. Just draw the temporary ones last.
	num_temp := 0
	for i := range roomsToDraw {
		if roomsToDraw[i].temporary {
			roomsToDraw[num_temp], roomsToDraw[i] = roomsToDraw[i], roomsToDraw[num_temp]
			num_temp++
		}
	}
	placed := OrderRectObjects(roomsToDraw[num_temp:])
	roomsToDraw = roomsToDraw[0:num_temp]
	for i := range placed {
		roomsToDraw = append(roomsToDraw, placed[i])
	}

	alpha_map := make(map[*Room]byte)
	los_map := make(map[*Room]byte)

	// First pass over the rooms - this will determine at what alpha the rooms
	// should be drawn. We will use this data later to determine the alpha for
	// the doors of adjacent rooms.
	for i := len(roomsToDraw) - 1; i >= 0; i-- {
		room := roomsToDraw[i]
		los_alpha := room.getMaxLosAlpha(los_tex)
		room.SetupGlStuff(&RoomRealGl{})
		tx := (focusx + 3) - float32(room.X+room.Size.Dx)
		if tx < 0 {
			tx = 0
		}
		ty := (focusy + 3) - float32(room.Y+room.Size.Dy)
		if ty < 0 {
			ty = 0
		}
		if tx < ty {
			tx = ty
		}
		// z := math.Log10(float64(zoom))
		z := float64(zoom) / 10
		v := math.Pow(z, float64(2*tx)/3)
		if v > 255 {
			v = 255
		}
		bv := 255 - byte(v)
		alpha_map[room] = byte((int(bv) * int(los_alpha)) >> 8)
		los_map[room] = los_alpha
	}

	logging.Debug("Floor.render: after first pass", "alpha_map", alpha_map, "los_map", los_map)

	// Second pass - this time we fill in the alpha that we should use for the
	// doors, using the values we've already calculated in the first pass.
	for _, r1 := range f.Rooms {
		r1.far_right.wall_alpha = 255
		r1.far_left.wall_alpha = 255
		for _, r2 := range f.Rooms {
			if r1 == r2 {
				continue
			}
			left, right := r2.getNearWallAlpha(los_tex)
			r1_rect := ImageRect(r1.X, r1.Y+r1.Size.Dy, r1.X+r1.Size.Dx, r1.Y+r1.Size.Dy+1)
			r2_rect := ImageRect(r2.X, r2.Y, r2.X+r2.Size.Dx, r2.Y+r2.Size.Dy)
			if r1_rect.Overlaps(r2_rect) {
				// If there is an open door between the two then we'll tone down the
				// alpha, otherwise we won't treat it any differently
				for _, d1 := range r1.Doors {
					for _, d2 := range r2.Doors {
						if d1 == d2 {
							r1.far_left.wall_alpha = byte((int(left) * 200) >> 8)
						}
					}
				}
			}
			r1_rect = ImageRect(r1.X+r1.Size.Dx, r1.Y, r1.X+r1.Size.Dx+1, r1.Y+r1.Size.Dy)
			if r1_rect.Overlaps(r2_rect) {
				for _, d1 := range r1.Doors {
					for _, d2 := range r2.Doors {
						if d1 == d2 {
							r1.far_right.wall_alpha = byte((int(right) * 200) >> 8)
						}
					}
				}
			}
		}
	}

	logging.Debug("Floor.render after second pass", "wall alphas", f.getWallAlphas(), "roomsToDraw", roomsToDraw)

	// Third pass - now that we know what alpha to use on the rooms, walls, and
	// doors we can actually render everything.  We still need to go back to
	// front though.
	for i := len(roomsToDraw) - 1; i >= 0; i-- {
		room := roomsToDraw[i]
		fx := focusx - float32(room.X)
		fy := focusy - float32(room.Y)
		matrices := perspective.MakeRoomMats(room.Size.GetDx(), room.Size.GetDy(), region, fx, fy, angle, zoom)
		v := alpha_map[room]
		if los_map[room] > 5 {
			room.Render(matrices, zoom, v, drawables, los_tex, floor_drawers)
		}
	}
}
