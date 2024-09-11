package base_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
)

type TestEntity struct {
	Defname string
	*TestPayload
	Discriminator int
}

// Instances of the payload should be identical amongst different Embedders
type TestPayload struct {
	Name string
}

func TestRegistry(t *testing.T) {
	t.Run("GetObject-CanAssignPayload", func(t *testing.T) {
		base.SetDatadir("go-test-data")

		aPayload := &TestPayload{
			Name: "a payload",
		}

		regMap := map[string]*TestPayload{
			"testkey": aPayload,
		}
		base.RegisterRegistry("test-reg", regMap)

		lookup := TestEntity{
			Defname:       "testkey",
			TestPayload:   nil,
			Discriminator: 42,
		}
		base.GetObject("test-reg", &lookup)

		if lookup.TestPayload != aPayload {
			t.Error("expected 'base.GetObject' to update the TestPayload field from nil to", aPayload, "but got", lookup.TestPayload)
		}
	})
}
