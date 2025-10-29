package game

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
