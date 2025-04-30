package housetest

import "github.com/MobRulesGames/haunts/house"

func GivenAHouseDef() *house.HouseDef {
	return house.MakeHouseFromName("tutorial")
}

func MakeStubbedFloor() *house.Floor {
	return &house.Floor{
		Rooms:  []*house.Room{},
		Spawns: []*house.SpawnPoint{},
	}
}

func MakeStubbedHouseDef() *house.HouseDef {
	return &house.HouseDef{
		Name: "stubbed",
		Floors: []*house.Floor{
			MakeStubbedFloor(),
		},
	}
}
