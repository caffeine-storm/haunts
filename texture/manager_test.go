package texture_test

import (
	"context"
	"path"
	"testing"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/runningwild/glop/glog"
	"github.com/runningwild/glop/render/rendertest"
)

func TestBlockUntilLoaded(t *testing.T) {
	// TODO(tmckee): BLECH! we're doing this to call base.SetupLogger() which
	// should not be coupled to this test.
	base.SetDatadir("../data")
	queue := rendertest.MakeDiscardingRenderQueue()
	texture.Init(queue)
	t.Run("should take a context with deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()
		err := texture.BlockUntilLoaded(ctx, "not-going-to-load")
		if err == nil {
			t.Fatalf("there's no texture named 'not-going-to-load' so the timeout should have fired")
		}
	})

	t.Run("can load a texture", func(t *testing.T) {
		t.Skip()
		base.SetLogLevel(glog.LevelTrace)
		base.Log().Trace("a test trace message")
		texpath := path.Join(base.GetDataDir(), "textures", "cobweb.png")
		_, err := texture.LoadFromPath(texpath)
		if err != nil {
			panic(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
		defer cancel()
		err = texture.BlockUntilLoaded(ctx, texpath)
		if err != nil {
			t.Fatalf("cobweb.png should have loaded by now")
		}
	})
}
