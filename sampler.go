package gfx

import (
	"github.com/go-gl/gl"
	"image"
)

type Sampler2D struct {
	tex gl.Texture
}

// Image takes an image and returns a 2D Sampler. Currently only takes
// *image.NRGBA, *image.RGBA, *image.Alpha, and *image.Gray. No processing is
// done on the image data, such as premultiplying alpha or linearization.
func Image(img image.Image) (*Sampler2D, error) {
	switch img.(type) {
	case *image.NRGBA:
		nrgba := img.(*image.NRGBA)
		size := nrgba.Rect.Size()
		return imageRGBA(nrgba.Pix, size.X, size.Y)
	case *image.RGBA:
		rgba := img.(*image.RGBA)
		size := rgba.Rect.Size()
		return imageRGBA(rgba.Pix, size.X, size.Y)
	case *image.Alpha:
		alpha := img.(*image.Alpha)
		size := alpha.Rect.Size()
		return imageAlpha(alpha.Pix, size.X, size.Y)
	case *image.Gray:
		gray := img.(*image.Gray)
		size := gray.Rect.Size()
		return imageAlpha(gray.Pix, size.X, size.Y)
	default:
		return nil, image.ErrFormat
	}
}

func (s *Sampler2D) Delete() {
	s.tex.Delete()
}

func (s *Sampler2D) bind() {
	s.tex.Bind(gl.TEXTURE_2D)
}

func imageRGBA(pix []byte, width, height int) (*Sampler2D, error) {
	s := &Sampler2D{
		tex: gl.GenTexture(),
	}
	s.bind()
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, pix)
	return s, nil
}

func imageAlpha(pix []byte, width, height int) (*Sampler2D, error) {
	s := &Sampler2D{
		tex: gl.GenTexture(),
	}
	s.bind()
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.R8, width, height, 0, gl.RED, gl.UNSIGNED_BYTE, pix)
	return s, nil
}
