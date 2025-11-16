package base_test

import (
	"testing"

	"github.com/caffeine-storm/gl"
	"github.com/runningwild/glop/render/rendertest/testbuilder"
)

func TestGlVersion(t *testing.T) {
	testbuilder.Run(func() {
		versionString := gl.GetString(gl.VERSION)
		t.Logf("versionString: %q\n", versionString)

		if versionString == "" {
			t.Error("gl.GetString(gl.VERSION) must not return the empty string once OpenGL is initialized")
		}
	})
}
