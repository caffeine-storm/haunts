package globals

import "github.com/runningwild/glop/render"

var renderQueueState render.RenderQueueState

func SetRenderQueueState(queueState render.RenderQueueState) {
	renderQueueState = queueState
}

func RenderQueueState() render.RenderQueueState {
	if renderQueueState == nil {
		panic("Need to call SetRenderQueueState before RenderQueueState()")
	}
	return renderQueueState
}
