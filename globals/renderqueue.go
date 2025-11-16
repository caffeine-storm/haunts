package globals

import (
	"fmt"

	"github.com/caffeine-storm/glop/render"
)

var (
	renderQueue      render.RenderQueueInterface
	renderQueueState render.RenderQueueState
)

func SetRenderQueue(queue render.RenderQueueInterface) {
	renderQueue = queue
}

func RenderQueue() render.RenderQueueInterface {
	if renderQueue == nil {
		panic(fmt.Errorf("Need to call SetRenderQueue before RenderQueue()"))
	}
	return renderQueue
}

func SetRenderQueueState(queueState render.RenderQueueState) {
	renderQueueState = queueState
}

func RenderQueueState() render.RenderQueueState {
	if renderQueueState == nil {
		panic(fmt.Errorf("Need to call SetRenderQueueState before RenderQueueState()"))
	}
	return renderQueueState
}
