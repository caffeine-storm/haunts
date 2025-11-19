package gametest

import (
	"path/filepath"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/mrgnet"
	"github.com/MobRulesGames/haunts/texture"
)

func GivenAScript() string {
	// TODO(tmckee): we should prefer an in-memory []byte instead of reading the
	// file in game.startGameScript
	return filepath.Join("versus", "basic.lua")
}

func GivenAHouseName() string {
	return "tutorial"
}

func GivenAScenario() game.Scenario {
	return game.Scenario{
		Script:    GivenAScript(),
		HouseName: GivenAHouseName(),
	}
}

func GivenAPlayer() *game.Player {
	return &game.Player{}
}

func GivenAGamePanelForScenario(scenario game.Scenario) *game.GamePanel {
	player := GivenAPlayer()
	noSpecialData := map[string]string{}
	noGameKey := mrgnet.GameKey("")
	ret := game.MakeGamePanel(scenario, player, noSpecialData, noGameKey)

	queue := texture.GetRenderQueue()
	queue.Purge()
	ret.SetLosModeAll()
	queue.Purge()

	return ret
}

func GivenAGamePanel() *game.GamePanel {
	scenario := GivenAScenario()
	return GivenAGamePanelForScenario(scenario)
}

func GivenAGame() *game.Game {
	panel := GivenAGamePanel()

	return panel.GetGame()
}

func GivenAnEntity() *game.Entity {
	g := GivenAGame()
	return game.MakeEntity("Bosch's Ghost", g)
}

func textureObjectForPath(path string) texture.Object {
	return texture.Object{
		Path: base.Path(filepath.Join(base.GetDataDir(), path)),
	}
}

func GivenSomeOptions() []game.Option {
	tex1 := textureObjectForPath("ui/cute1.png")
	tex2 := textureObjectForPath("ui/cute2.png")
	tex3 := textureObjectForPath("ui/cute3.png")
	tex4 := textureObjectForPath("ui/cute4.png")
	basicOptions := []*game.OptionBasic{
		{
			Id:        "some-id--though-really-its-a-script-name--1",
			HouseName: "Lvl_01_Haunted_House", // a '.house' file from data/houses but without the .house extension
			Small:     tex1,
			Large:     tex3,
			Text:      "#1",
			Size:      12,
		},
		{
			Id:        "some-id--though-really-its-a-script-name--2",
			HouseName: "Lvl_01_Haunted_House", // a '.house' file from data/houses but without the .house extension
			Small:     tex2,
			Large:     tex4,
			Text:      "#2",
			Size:      12,
		},
		{
			Id:        "some-id--though-really-its-a-script-name--3",
			HouseName: "Lvl_01_Haunted_House", // a '.house' file from data/houses but without the .house extension
			Small:     tex3,
			Large:     tex1,
			Text:      "#3",
			Size:      12,
		},
		{
			Id:        "some-id--though-really-its-a-script-name--4",
			HouseName: "Lvl_01_Haunted_House", // a '.house' file from data/houses but without the .house extension
			Small:     tex4,
			Large:     tex2,
			Text:      "#4",
			Size:      12,
		},
	}

	// Make sure each option has a non-zero alpha by calling Think on each
	ret := make([]game.Option, len(basicOptions))
	for i, opt := range basicOptions {
		hovered := false
		selected := false
		selectable := true
		opt.Think(hovered, selected, selectable, 5)

		ret[i] = opt
	}

	return ret
}
