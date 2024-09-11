package status_test

import (
	"github.com/MobRulesGames/haunts/game/status"
	"github.com/orfjackal/gospec/src/gospec"
	"testing"
)

func TestAllSpecs(t *testing.T) {
	status.RegisterAllConditions()
	r := gospec.NewRunner()
	r.AddSpec(ConditionsSpec)
	gospec.MainGoTest(r, t)
}
