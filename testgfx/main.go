// Copyright 2012 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"github.com/Jragonmiris/mathgl"
	"github.com/go-gl/gl"
	glfw "github.com/go-gl/glfw3"
	"image"
	"image/png"
	"io"
	"j4k.co/gfx"
	"os"
)

const (
	Title  = "Spinning Gopher"
	Width  = 640
	Height = 480
)

var (
	texture    gl.Texture
	rotx, roty float32
	ambient    []float32 = []float32{0.5, 0.5, 0.5, 1}
	diffuse    []float32 = []float32{1, 1, 1, 1}
	lightpos   []float32 = []float32{-5, 5, 10, 0}
)

func errorCallback(err glfw.ErrorCode, desc string) {
	fmt.Printf("%v: %v\n", err, desc)
}

func main() {
	glfw.SetErrorCallback(errorCallback)

	if !glfw.Init() {
		panic("Can't init glfw!")
	}
	defer glfw.Terminate()

	window, err := glfw.CreateWindow(Width, Height, Title, nil, nil)
	if err != nil {
		panic(err)
	}

	window.MakeContextCurrent()

	glfw.SwapInterval(1)

	gl.Init()

	if err := initScene(); err != nil {
		fmt.Fprintf(os.Stderr, "init: %s\n", err)
		return
	}
	defer destroyScene()

	for !window.ShouldClose() {
		drawScene()
		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func createTexture(r io.Reader) (gfx.Sampler, error) {
	img, err := png.Decode(r)
	if err != nil {
		return nil, err
	}

	rgbaImg, ok := img.(*image.NRGBA)
	if !ok {
		return nil, errors.New("texture must be an NRGBA image")
	}
	return gfx.Image(rgbaImg)

	/*
		textureId := gl.GenTexture()
		textureId.Bind(gl.TEXTURE_2D)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)

		// flip image: first pixel is lower left corner
		imgWidth, imgHeight := img.Bounds().Dx(), img.Bounds().Dy()
		data := make([]byte, imgWidth*imgHeight*4)
		lineLen := imgWidth * 4
		dest := len(data) - lineLen
		for src := 0; src < len(rgbaImg.Pix); src += rgbaImg.Stride {
			copy(data[dest:dest+lineLen], rgbaImg.Pix[src:src+rgbaImg.Stride])
			dest -= lineLen
		}
		gl.TexImage2D(gl.TEXTURE_2D, 0, 4, imgWidth, imgHeight, 0, gl.RGBA, gl.UNSIGNED_BYTE, data)

		return textureId, nil
	*/
}

var cube struct {
	shader *gfx.Shader

	projM          mathgl.Mat4f
	viewM          mathgl.Mat4f
	worldM         mathgl.Mat4f
	WorldViewProjM [16]float32 `uniform:"WorldViewProjectionM"`
	Diffuse        gfx.Sampler `uniform:"Diffuse"`
	geom           gfx.Geometry
}

func initScene() (err error) {
	gl.Enable(gl.TEXTURE_2D)
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.LIGHTING)

	gl.ClearColor(0.5, 0.5, 0.5, 0.0)
	gl.ClearDepth(1)
	gl.DepthFunc(gl.LEQUAL)

	gl.Lightfv(gl.LIGHT0, gl.AMBIENT, ambient)
	gl.Lightfv(gl.LIGHT0, gl.DIFFUSE, diffuse)
	gl.Lightfv(gl.LIGHT0, gl.POSITION, lightpos)
	gl.Enable(gl.LIGHT0)

	gl.Viewport(0, 0, Width, Height)
	/*gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	gl.Frustum(-1, 1, -1, 1, 1.0, 10.0)
	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
	*/

	cube.projM = mathgl.Ortho(-3, 3, -3, 3, -10.0, 10.0)
	cube.viewM = mathgl.Ident4f()
	cube.worldM = mathgl.Translate3D(0, 0, 2)
	cube.WorldViewProjM = [16]float32(cube.projM.Mul4(cube.viewM).Mul4(cube.worldM))

	goph, err := os.Open("gopher.png")
	if err != nil {
		panic(err)
	}
	defer goph.Close()

	// TODO: create cube geometry. could build out the vertex buffer manually, but this is a good place to do a mesh builder type thing.
	//cube.geom = ... mesh builder?

	gfx.DefaultVertexAttributes[gfx.VertexPosition] = "Position"
	gfx.DefaultVertexAttributes[gfx.VertexColor] = "Color"
	gfx.DefaultVertexAttributes[gfx.VertexTexcoord] = "UV"
	gfx.DefaultVertexAttributes[gfx.VertexNormal] = "Normal"
	vfmt := gfx.DefaultVertexAttributes.Format()
	builder := gfx.BuildGeometry(vfmt)
	builder.Position(-1, -1, 1).Texcoord(0, 0).Color(1, 1, 1).Normal(0, 0, 1)
	builder.Position(1, -1, 1).Texcoord(1, 0)
	builder.Position(1, 1, 1).Texcoord(1, 1)
	builder.Position(-1, 1, 1).Texcoord(0, 1)
	builder.Indices(0, 1, 2, 2, 0, 3)
	builder.Position(-1, -1, -1).Texcoord(0, 0).Normal(0, 0, -1)
	builder.Position(-1, 1, -1).Texcoord(1, 0)
	builder.Position(1, 1, -1).Texcoord(1, 1)
	builder.Position(1, -1, -1).Texcoord(0, 1)
	builder.Indices(0, 1, 2, 2, 0, 3)
	builder.Position(-1, 1, -1).Texcoord(0, 0).Normal(0, 1, 0)
	builder.Position(-1, 1, 1).Texcoord(1, 0)
	builder.Position(1, 1, 1).Texcoord(1, 1)
	builder.Position(1, 1, -1).Texcoord(0, 1)
	builder.Indices(0, 1, 2, 2, 0, 3)
	builder.Position(-1, -1, -1).Texcoord(0, 0).Normal(0, -1, 0)
	builder.Position(1, -1, -1).Texcoord(1, 0)
	builder.Position(1, -1, 1).Texcoord(1, 1)
	builder.Position(-1, -1, 1).Texcoord(0, 1)
	builder.Indices(0, 1, 2, 2, 0, 3)
	builder.Position(1, -1, -1).Texcoord(0, 0).Normal(1, 0, 0)
	builder.Position(1, 1, -1).Texcoord(1, 0)
	builder.Position(1, 1, 1).Texcoord(1, 1)
	builder.Position(1, -1, 1).Texcoord(0, 1)
	builder.Indices(0, 1, 2, 2, 0, 3)
	builder.Position(-1, -1, -1).Texcoord(0, 0).Normal(-1, 0, 0)
	builder.Position(-1, -1, 1).Texcoord(1, 0)
	builder.Position(-1, 1, 1).Texcoord(1, 1)
	builder.Position(-1, 1, -1).Texcoord(0, 1)
	builder.Indices(0, 1, 2, 2, 0, 3)
	cube.geom, err = gfx.NewGeometry(builder, gfx.StaticDraw)
	if err != nil {
		panic(err)
	}

	shader := gfx.BuildShader(gfx.DefaultVertexAttributes, vs, fs)
	cube.shader = shader
	cube.Diffuse, err = createTexture(goph)
	if err != nil {
		panic(err)
	}
	return
}

func destroyScene() {
	texture.Delete()
}

func drawScene() {
	rotx += 0.5
	roty += 0.5

	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	rotmat := mathgl.HomogRotate3DX(rotx).Mul4(mathgl.HomogRotate3DY(roty))
	cube.worldM = mathgl.Translate3D(0, 0, 2).Mul4(rotmat)
	cube.WorldViewProjM = [16]float32(cube.projM.Mul4(cube.viewM).Mul4(cube.worldM))

	cube.shader.Use()
	cube.shader.SetUniforms(cube)
	cube.shader.SetGeometry(cube.geom)
	cube.shader.Draw()
}
