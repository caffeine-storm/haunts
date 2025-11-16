package house

import (
	"fmt"
	"runtime"

	"github.com/MobRulesGames/haunts/texture"
	"github.com/caffeine-storm/gl"
	"github.com/caffeine-storm/glop/render"
)

const (
	LosMinVisibility       = 32
	LosVisibilityThreshold = 200
	LosTextureSize         = 128
	LosTextureSizeSquared  = LosTextureSize * LosTextureSize
)

// A LosTexture is defined over a square portion of a grid, and if a pixel is
// non-black it indicates that there is visibility to that pixel from the
// center of the texture.  The texture is a square with a size that is a power
// of two, so the center is defined as the pixel to the lower-left of the
// actual center of the texture.
type LosTexture struct {
	// An off-render-thread copy of the texture data assumed to be in the
	// gl.Texture. Synchronizing changes to the data isn't free, however; see
	// Remap().
	pix []byte
	p2d [][]byte

	// TODO: this sucks
	tex   gl.Texture // used off render thread
	gltex gl.Texture // used on render thread

	// The texture needs to be created on the render thread, so we use this to
	// get the texture after it's been made.
	rec chan gl.Texture
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

	texels := makeTexelData(lt.pix)
	dim := len(lt.p2d)

	// TODO(tmckee): there must be a better way...
	renderQueue := texture.GetRenderQueue()
	renderQueue.Queue(func(render.RenderQueueState) {
		gl.Enable(gl.TEXTURE_2D)
		lt.gltex = gl.GenTexture()
		lt.gltex.Bind(gl.TEXTURE_2D)
		defer lt.gltex.Unbind(gl.TEXTURE_2D)
		gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)

		internalFormat := gl.ALPHA
		noMipMap := 0
		width := dim
		height := dim
		border := 0
		clientDataFormat := gl.GLenum(gl.ALPHA)
		clientDataType := gl.GLenum(gl.BYTE)
		gl.TexImage2D(gl.TEXTURE_2D, noMipMap, internalFormat, width, height, border, clientDataFormat, clientDataType, texels)

		lt.rec <- lt.gltex
		runtime.SetFinalizer(&lt, func(lt *LosTexture) {
			renderQueue.Queue(func(render.RenderQueueState) {
				lt.gltex.Delete()
			})
		})
	})

	return &lt
}

func (lt *LosTexture) String() string {
	return fmt.Sprintf("LosTexture{len(pix): %d, tex: %d", len(lt.pix), lt.tex)
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
		// TODO(tmckee): we should assert that this doesn't happen in the tests but
		// it might not be valid to assert against this case in production... lazy
		// initialization and whatnot.
		return
	}

	// Need to use a copy of the data on the render thread so that we don't have
	// concurrent read/writes.
	texelData := makeTexelData(lt.pix)
	dim := len(lt.p2d)

	renderQueue := texture.GetRenderQueue()
	renderQueue.Queue(func(render.RenderQueueState) {
		gl.Enable(gl.TEXTURE_2D)
		lt.gltex.Bind(gl.TEXTURE_2D)
		levelOfDetail := int(0)
		internalFormat := gl.ALPHA
		noBorder := int(0)
		gl.TexImage2D(gl.TEXTURE_2D, levelOfDetail, internalFormat, dim, dim, noBorder, gl.ALPHA, gl.UNSIGNED_BYTE, texelData)
	})
}

// Binds the texture, run on the render thread
func (lt *LosTexture) Bind() {
	render.MustBeOnRenderThread()
	if lt.gltex == 0 {
		panic(fmt.Errorf("must not happen... how can we be on the render thread but not have already set gltex?"))
	}
	lt.gltex.Bind(gl.TEXTURE_2D)
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

func makeTexelData(bs []byte) []byte {
	texelData := make([]byte, len(bs))
	copy(texelData, bs)
	return texelData
}
