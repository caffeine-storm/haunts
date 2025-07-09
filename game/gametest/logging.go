package gametest

import (
	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/smartystreets/goconvey/convey"
)

type drawWithLoggingTrace struct {
	Drawer
}

func (it *drawWithLoggingTrace) Draw(reg gui.Region, ctx gui.DrawingContext) {
	logging.TraceBracket(func() {
		it.Drawer.Draw(reg, ctx)
	})
}

// Wrap a Drawer with a logging.TraceBracket so as to get logging traces when
// drawing. Useful while fixing up a not-drawing Drawer.
func DrawWithTrace(d Drawer) *drawWithLoggingTrace {
	return &drawWithLoggingTrace{
		Drawer: d,
	}
}

// Like RunDrawingTest but will enable logging traces when the
// object-under-test is Draw()ing.
func RunTracedDrawingTest(c convey.C, builder func() Drawer, ref rendertest.TestDataReference) {
	RunDrawingTest(c, func() Drawer {
		return DrawWithTrace(builder())
	}, ref)
}
