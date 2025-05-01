package game_test

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/house/housetest"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/cache"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/sprite"
	"github.com/stretchr/testify/require"
)

func givenASpriteManager() *sprite.Manager {
	rqi := rendertest.MakeStubbedRenderQueue()
	bb := cache.MakeRamByteBank()

	return sprite.MakeManager(rqi, func(s string) cache.ByteBank { return bb })
}

func givenAGame() *game.Game {
	houseDef := housetest.MakeStubbedHouseDef()
	spriteManager := givenASpriteManager()
	return game.MakeGame(houseDef, spriteManager)
}

func TestGobbableGameState(t *testing.T) {
	t.Run("ought to be able to gob-encode game state", func(t *testing.T) {
		require := require.New(t)

		base.SetDatadir("../data")
		rq := rendertest.MakeStubbedRenderQueue()
		texture.Init(rq)
		registry.LoadAllRegistries()
		gm := givenAGame()

		buf := &bytes.Buffer{}
		err := gob.NewEncoder(buf).Encode(gm)

		require.NoError(err)

		var newSt game.Game
		err = gob.NewDecoder(buf).Decode(&newSt)

		require.NoError(err)
	})
}
