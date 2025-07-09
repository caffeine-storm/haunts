package registry

import (
	"path/filepath"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/status"
	"github.com/MobRulesGames/haunts/house"
)

func LoadAllRegistries() {
	datadir := base.GetDataDir()
	house.LoadAllFurnitureInDir(filepath.Join(datadir, "furniture"))
	house.LoadAllDecalsInDir(filepath.Join(datadir, "textures"))
	house.LoadAllRoomsInDir(filepath.Join(datadir, "rooms"))
	house.LoadAllDoorsInDir(filepath.Join(datadir, "doors"))
	house.LoadAllHousesInDir(filepath.Join(datadir, "houses"))
	game.LoadAllGearInDir(filepath.Join(datadir, "gear"))
	game.RegisterActions()
	status.RegisterAllConditions()
}
