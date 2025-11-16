package texture

import (
	"github.com/caffeine-storm/gl"
	"github.com/caffeine-storm/glu"
)

var error_texture gl.Texture

func makeErrorTexture() {
	gl.Enable(gl.TEXTURE_2D)
	error_texture = gl.GenTexture()
	error_texture.Bind(gl.TEXTURE_2D)
	gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	singleTexel := []byte{255, 0, 255, 255}
	glu.Build2DMipmaps(gl.TEXTURE_2D, gl.RGBA, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, singleTexel)
}
