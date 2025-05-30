package house

import (
	"runtime"

	"github.com/MobRulesGames/haunts/texture"
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/render"
)

const LosMinVisibility = 32
const LosVisibilityThreshold = 200
const LosTextureSize = 128
const LosTextureSizeSquared = LosTextureSize * LosTextureSize

// A LosTexture is defined over a square portion of a grid, and if a pixel is
// non-black it indicates that there is visibility to that pixel from the
// center of the texture.  The texture is a square with a size that is a power
// of two, so the center is defined as the pixel to the lower-left of the
// actual center of the texture.
type LosTexture struct {
	pix []byte
	p2d [][]byte
	tex gl.Texture

	// The texture needs to be created on the render thread, so we use this to
	// get the texture after it's been made.
	rec chan gl.Texture
}

func losTextureFinalize(lt *LosTexture, renderQueue render.RenderQueueInterface) {
	renderQueue.Queue(func(render.RenderQueueState) {
		gl.Enable(gl.TEXTURE_2D)
		lt.tex.Delete()
	})
}

// Creates a LosTexture with the specified size, which must be a power of two.
func MakeLosTexture() *LosTexture {
	var lt LosTexture
	lt.pix = make([]byte, LosTextureSizeSquared)
	lt.p2d = make([][]byte, LosTextureSize)
	lt.rec = make(chan gl.Texture, 1)
	for i := 0; i < LosTextureSize; i++ {
		lt.p2d[i] = lt.pix[i*LosTextureSize : (i+1)*LosTextureSize]
	}

	// TODO(tmckee): there must be a better way...
	renderQueue := texture.GetRenderQueue()
	renderQueue.Queue(func(render.RenderQueueState) {
		gl.Enable(gl.TEXTURE_2D)
		tex := gl.GenTexture()
		tex.Bind(gl.TEXTURE_2D)
		defer tex.Unbind(gl.TEXTURE_2D)
		gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.ALPHA, len(lt.p2d), len(lt.p2d), 0, gl.ALPHA, gl.BYTE, lt.pix)
		lt.rec <- tex
		runtime.SetFinalizer(&lt, func(lt *LosTexture) {
			losTextureFinalize(lt, renderQueue)
		})
	})

	return &lt
}

// If the texture has been created this returns true, otherwise it checks for
// the finished texture and returns true if it is available, false otherwise.
func (lt *LosTexture) ready() bool {
	if lt.tex != 0 {
		return true
	}
	select {
	case lt.tex = <-lt.rec:
		return true
	default:
	}
	return false
}

// Updates OpenGl with any changes that have been made to the texture.
// OpenGl calls in this method are run on the render thread
func (lt *LosTexture) Remap() {
	if !lt.ready() {
		return
	}
	renderQueue := texture.GetRenderQueue()
	renderQueue.Queue(func(render.RenderQueueState) {
		gl.Enable(gl.TEXTURE_2D)
		lt.tex.Bind(gl.TEXTURE_2D)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, len(lt.p2d), len(lt.p2d), gl.ALPHA, gl.UNSIGNED_BYTE, lt.pix)
	})
}

// TODO(tmckee): ummm... doesn't this _have_ to be run on the render thread?
// Binds the texture, not run on the render thread
func (lt *LosTexture) Bind() {
	lt.ready()
	lt.tex.Bind(gl.TEXTURE_2D)
}

// Clears the texture so that all pixels are set to the specified value
func (lt *LosTexture) Clear(v byte) {
	for i := range lt.pix {
		lt.pix[i] = v
	}
}

// Returns the length of a side of this texture
func (lt *LosTexture) Size() int {
	return len(lt.p2d)
}

// Returns a convenient 2d slice over the texture
func (lt *LosTexture) Pix() [][]byte {
	return lt.p2d
}
