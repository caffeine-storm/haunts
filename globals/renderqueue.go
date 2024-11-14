package globals

import "github.com/runningwild/glop/render"

var renderQueueState render.RenderQueueState

func SetRenderQueueState(queueState render.RenderQueueState) {
	renderQueueState = queueState
}

func RenderQueueState() render.RenderQueueState {
	return renderQueueState
}
