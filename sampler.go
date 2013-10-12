package gfx

import (
	"github.com/go-gl/gl"
	"image"
)

// TODO: we need to supprt image.Image ideally as a source, as well as
// streaming sources for minimum copying..but that also requires
// PIXEL_UNPACK_BUFFER
type Sampler interface {
	bind()
}

// Image takes an image and returns a 2D Sampler. Currently only takes
// *image.NRGBA, *image.RGBA. No processing is done on the image data, such
// as premultiplying alpha.
func Image(img image.Image) (Sampler, error) {
	switch img.(type) {
	case *image.NRGBA:
		nrgba := img.(*image.NRGBA)
		size := nrgba.Rect.Size()
		return imageRGBA(nrgba.Pix, size.X, size.Y)
	case *image.RGBA:
		rgba := img.(*image.RGBA)
		size := rgba.Rect.Size()
		return imageRGBA(rgba.Pix, size.X, size.Y)
	default:
		return nil, image.ErrFormat
	}
	panic("unreachable")
}

type texture2d struct {
	tex gl.Texture
}

func (t *texture2d) bind() {
	t.tex.Bind(gl.TEXTURE_2D)
}

func imageRGBA(pix []byte, width, height int) (Sampler, error) {
	// TODO: finalizer
	t := &texture2d{
		tex: gl.GenTexture(),
	}
	t.bind()
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, pix)
	return t, nil
}
