package texture_test

import (
	"testing"
	"unsafe"

	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/render/rendertest"
	"github.com/runningwild/glop/system"
)

type point struct {
	x, y, z, s, t float32
}

func TestTextureDrawElements(t *testing.T) {
	// Try to draw a textured quad across some triangles.
	extent := float32(64.0)

	geometry := []point{
		// Top left
		{x: 0, y: extent, z: 0, s: 0, t: 1},
		// Top right
		{x: extent, y: extent, z: 0, s: 1, t: 1},
		// Bottom right
		{x: extent, y: 0, z: 0, s: 1, t: 0},
		// Bottom left
		{x: 0, y: 0, z: 0, s: 0, t: 0},
	}
	indices := []uint16{
		0, 1, 2,
		0, 2, 3,
	}
	rendertest.DeprecatedWithGlForTest(64, 64, func(sys system.System, queue render.RenderQueueInterface) {
		queue.Queue(func(render.RenderQueueState) {
			gl.Enable(gl.TEXTURE_2D)
			defer gl.Disable(gl.TEXTURE_2D)

			gl.Enable(gl.BLEND)
			gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
			defer gl.Disable(gl.BLEND)

			gl.Enable(gl.STENCIL_TEST)
			defer gl.Disable(gl.STENCIL_TEST)
			gl.ClearStencil(0)
			gl.Clear(gl.STENCIL_BUFFER_BIT)
			gl.StencilFunc(gl.ALWAYS, 1, 1)
			gl.StencilOp(gl.KEEP, gl.KEEP, gl.KEEP)

			// upload vertices
			vertBuf := gl.GenBuffer()
			vertBuf.Bind(gl.ARRAY_BUFFER)
			geometryByteCount := len(geometry) * int(unsafe.Sizeof(geometry[0]))
			gl.BufferData(gl.ARRAY_BUFFER, geometryByteCount, geometry, gl.STATIC_DRAW)

			// upload indices
			idxBuf := gl.GenBuffer()
			idxBuf.Bind(gl.ELEMENT_ARRAY_BUFFER)
			indicesByteCount := len(indices) * int(unsafe.Sizeof(indices[0]))
			gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, indicesByteCount, indices, gl.STATIC_DRAW)
			// teach opengl where the vertices are
			gl.VertexPointer(3, gl.FLOAT, int(unsafe.Sizeof(point{})), nil)

			// teach opengl where the texture co-ords are
			gl.TexCoordPointer(2, gl.FLOAT, int(unsafe.Sizeof(point{})), unsafe.Offsetof(geometry[0].s))

			// setup a texture
			tex := rendertest.GivenATexture("checker/0.png")
			gl.ActiveTexture(gl.TEXTURE0)
			tex.Bind(gl.TEXTURE_2D)

			gl.EnableClientState(gl.VERTEX_ARRAY)
			defer gl.DisableClientState(gl.VERTEX_ARRAY)
			gl.EnableClientState(gl.TEXTURE_COORD_ARRAY)
			defer gl.DisableClientState(gl.TEXTURE_COORD_ARRAY)

			// draw 2 elements
			gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_SHORT, nil)
		})
		queue.Purge()

		stringResult := rendertest.ShouldLookLikeFile(queue, "checker")
		if stringResult != "" {
			t.Fatalf("image mismatch: %s", stringResult)
		}
	})
}
