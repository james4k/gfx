// Copyright 2012 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is an example taken from github.com/examples/glfw3 that's been modified
// to test out some abstraction ideas.

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
	"runtime"
)

const (
	Title  = "Spinning Gopher"
	Width  = 640
	Height = 480
)

var (
	rotx, roty float32
)

func errorCallback(err glfw.ErrorCode, desc string) {
	fmt.Printf("%v: %v\n", err, desc)
}

func main() {
	runtime.LockOSThread()

	glfw.SetErrorCallback(errorCallback)

	if !glfw.Init() {
		panic("Can't init glfw!")
	}
	defer glfw.Terminate()

	// must be done in main thread or we get a nasty stderr message from glfw,
	// although it does seem to 'work'
	window, err := glfw.CreateWindow(Width, Height, Title, nil, nil)
	if err != nil {
		panic(err)
	}

	// separate thread for drawing so that we don't block on the event thread.
	// most obvious benefit is that we continue to render during window
	// resizes.
	go func() {
		runtime.LockOSThread()

		window.MakeContextCurrent()
		glfw.SwapInterval(1)
		gl.Init()
		if err := initScene(); err != nil {
			fmt.Fprintf(os.Stderr, "init: %s\n", err)
			return
		}

		for !window.ShouldClose() {
			drawScene()
			window.SwapBuffers()
		}
		os.Exit(0)
	}()

	for {
		glfw.WaitEvents()
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
	gl.Enable(gl.DEPTH_TEST)

	gl.Viewport(0, 0, Width, Height)
	gl.ClearColor(0.5, 0.5, 0.5, 0.0)
	gl.ClearDepth(1)
	gl.DepthFunc(gl.LEQUAL)

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
