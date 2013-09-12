package main

import (
	"j4k.co/gfx"
)

var vs gfx.VertexShader = `
uniform mat4 WorldViewProjectionM;

attribute vec3 Position;
attribute vec3 Color;
attribute vec3 Normal;

varying vec3 color;

void main() {
	color = Color;
	//color = vec3(0, 1, 0);
	gl_Position = WorldViewProjectionM * vec4(Position, 1.0);
	//gl_Position.xyz = vec3(0);
}`

var fs gfx.FragmentShader = `
varying vec3 color;

void main() {
	gl_FragColor = vec4(color, 1.0);
}`
