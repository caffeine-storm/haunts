package texture_test

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
)

func TestBlockUntilLoaded(t *testing.T) {
	base.SetDatadir("../data")
	t.Run("should take a context with deadline", func(t *testing.T) {
		queue := rendertest.MakeDiscardingRenderQueue()
		texture.Init(queue)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()
		err := texture.BlockUntilLoaded(ctx, "not-going-to-load")
		if err == nil {
			t.Fatalf("there's no texture named 'not-going-to-load' so the timeout should have fired")
		}
	})

	t.Run("can load a texture", func(t *testing.T) {
		rendertest.WithGlForTest(50, 50, func(sys system.System, queue render.RenderQueueInterface) {
			texture.Init(queue)
			queue.Purge()

			texpath := path.Join(base.GetDataDir(), "textures", "cobweb.png")
			_, err := texture.LoadFromPath(texpath)
			if err != nil {
				panic(err)
			}

			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			defer cancel()
			err = texture.BlockUntilLoaded(ctx, texpath)
			if err != nil {
				t.Fatal(fmt.Errorf("cobweb.png should have loaded by now: %w", err))
			}
			delta := time.Now().Sub(start)
			t.Logf("timings: elapsed: %dms, budget: %dms, util: %.2f%%\n", delta.Milliseconds(), 250, float64(delta.Milliseconds())/float64(250))
		})
	})
}

func givenATimedOutContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestBlockUntilIdle(t *testing.T) {
	t.Run("should take a context with deadline", func(t *testing.T) {
		t.Run("is idle upon creation", func(t *testing.T) {
			queue := rendertest.MakeDiscardingRenderQueue()
			texture.Init(queue)
			ctx := givenATimedOutContext()

			// nothing's loading so it should be idle already
			err := texture.BlockUntilIdle(ctx)
			if err != nil {
				t.Fatalf("a fresh texture manager should be idle")
			}
		})
	})
}

func TestGetInFlightRequests(t *testing.T) {
	t.Run("should return a slice of strings", func(t *testing.T) {
		queue := rendertest.MakeDiscardingRenderQueue()
		texture.Init(queue)

		noPaths := texture.GetInFlightRequests()

		if len(noPaths) != 0 {
			t.Fatalf("no load requests were issued so none should be in flight")
		}
	})
}
