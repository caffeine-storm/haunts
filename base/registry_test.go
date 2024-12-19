package base_test

import (
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/stretchr/testify/assert"
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

type InvalidEntity struct {
	Defname string
	*invalidPayloadField
	Discriminator int
}

// Using an 'unexported' type for the embedded payload will not work because
// golang reflection refuses to assign through unexported fields.
type invalidPayloadField struct {
	Name string
}

func TestRegistry(t *testing.T) {
	base.SetDatadir("testdata")

	t.Run("GetObject-CanAssignPayload", func(t *testing.T) {
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

	t.Run("GetObject-CannotAssignToUnexportedPayloadField", func(t *testing.T) {
		aPayload := &invalidPayloadField{
			Name: "an invalid payload",
		}

		regMap := map[string]*invalidPayloadField{
			"testkey": aPayload,
		}
		base.RegisterRegistry("test-reg", regMap)

		lookup := InvalidEntity{
			Defname:             "testkey",
			invalidPayloadField: nil,
			Discriminator:       42,
		}

		assert.Panics(t, func() {
			base.GetObject("test-reg", &lookup)
		}, "expected the registry to be unable to assign through an unexported field")
	})
}
