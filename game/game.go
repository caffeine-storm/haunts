package game

import (
	"math/rand"
	"sort"

	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
)

func (g *Game) SpawnEntity(spawn *Entity, x, y house.BoardSpaceUnit) bool {
	for i := range g.Ents {
		cx, cy := g.Ents[i].FloorPos()
		if cx == x && cy == y {
			logging.Warn("Can't spawn entity", "pos", []any{x, y}, "blockedby", g.Ents[i].Name)
			return false
		}
	}
	spawn.X = float64(x)
	spawn.Y = float64(y)
	spawn.Info.RoomsExplored[spawn.CurrentRoom()] = true
	g.Ents = append(g.Ents, spawn)
	return true
}

// Returns true iff the action was set
// This function will return false if there is no selected entity, if the
// action cannot be selected (because it is invalid or the entity has
// insufficient Ap), or if there is an action currently executing.
func (g *Game) SetCurrentAction(action Action) bool {
	if g.Action_state != noAction && g.Action_state != preppingAction {
		return false
	}
	// the action should be one that belongs to the current entity, if not then
	// we need to bail out immediately
	if g.selected_ent == nil {
		logging.Warn("Tried to SetCurrentAction() without a selected entity.")
		return action == nil
	}
	if action != nil {
		valid := false
		for _, a := range g.selected_ent.Actions {
			if a == action {
				valid = true
				break
			}
		}
		if !valid {
			logging.Warn("Tried to SetCurrentAction() with an action that did not belong to the selected entity.")
			return action == nil
		}
	}
	if g.current_action != nil {
		g.current_action.Cancel()
	}
	if action == nil {
		g.Action_state = noAction
	} else {
		g.Action_state = preppingAction
	}
	g.viewer.RemoveFloorDrawable(g.current_action)
	g.current_action = action
	if g.current_action != nil {
		g.viewer.AddFloorDrawable(g.current_action)
	}
	return true
}

type orderEntsBigToSmall []*Entity

func (o orderEntsBigToSmall) Len() int {
	return len(o)
}
func (o orderEntsBigToSmall) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}
func (o orderEntsBigToSmall) Less(i, j int) bool {
	return o[i].Dx*o[i].Dy > o[j].Dx*o[j].Dy
}

type orderSpawnsSmallToBig []*house.SpawnPoint

func (o orderSpawnsSmallToBig) Len() int {
	return len(o)
}
func (o orderSpawnsSmallToBig) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}
func (o orderSpawnsSmallToBig) Less(i, j int) bool {
	return o[i].Dx*o[i].Dy < o[j].Dx*o[j].Dy
}

type entSpawnPair struct {
	ent   *Entity
	spawn *house.SpawnPoint
}

// Distributes the ents among the spawn points.  Since this is done randomly
// it might not work, so there is a very small chance that not all spawns will
// have an ent given to them, even if it is possible to distrbiute them
// properly.  Regardless, at least some will be spawned.
func spawnEnts(g *Game, ents []*Entity, spawns []*house.SpawnPoint) {
	sort.Sort(orderSpawnsSmallToBig(spawns))
	sanity := 100
	var places []entSpawnPair
	for sanity > 0 {
		sanity--
		places = places[0:0]
		sort.Sort(orderEntsBigToSmall(ents))
		//slightly shuffle the ents
		for i := range ents {
			j := i + rand.Intn(5) - 2
			if j >= 0 && j < len(ents) {
				ents[i], ents[j] = ents[j], ents[i]
			}
		}
		// Go through each ent and try to place it in an unused spawn point
		used_spawns := make(map[*house.SpawnPoint]bool)
		for _, ent := range ents {
			for _, spawn := range spawns {
				if used_spawns[spawn] {
					continue
				}
				if int(spawn.Dx) < ent.Dx || int(spawn.Dy) < ent.Dy {
					continue
				}
				used_spawns[spawn] = true
				places = append(places, entSpawnPair{ent, spawn})
				break
			}
		}
		if len(places) == len(spawns) {
			break
		}
	}
	if sanity > 0 {
		logging.Debug("Placed all objects", "remaning sanity", sanity)
	} else {
		logging.Warn("Out of sanity while placing objects", "placed", len(places), "requested", len(spawns))
	}
	for _, place := range places {
		place.ent.X = float64(int(place.spawn.X) + rand.Intn(int(place.spawn.Dx)-place.ent.Dx+1))
		place.ent.Y = float64(int(place.spawn.Y) + rand.Intn(int(place.spawn.Dy)-place.ent.Dy+1))
		g.viewer.AddDrawable(place.ent)
		g.Ents = append(g.Ents, place.ent)
		logging.Debug("placing", "object", place.ent.Name, "pos", []any{place.ent.X, place.ent.Y})
	}
}
