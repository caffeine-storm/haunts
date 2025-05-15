package gametest

import (
	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/gui"
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
