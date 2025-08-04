package gametest

import "github.com/runningwild/glop/gui"

type wrapper struct {
	fn func(gui.Region, gui.DrawingContext)
}

func (w *wrapper) Draw(r gui.Region, ctx gui.DrawingContext) {
	w.fn(r, ctx)
}

func DrawerAdapter(fn func(region gui.Region, _ gui.DrawingContext)) func() Drawer {
	return func() Drawer {
		return &wrapper{
			fn: fn,
		}
	}
}
