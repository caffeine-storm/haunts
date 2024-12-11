package texture_test

import (
	"context"
	"testing"
	"time"

	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/render/rendertest"
)

func TestBlockUntilLoaded(t *testing.T) {
	queue := rendertest.MakeDiscardingRenderQueue()
	texture.Init(queue)
	t.Run("should take a context with deadline", func(t *testing.T) {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Millisecond)
		texture.BlockUntilLoaded(ctx, "already-loaded")
	})
}
