package main

import (
	"j4k.co/gfx"
)

var vs gfx.VertexShader = `
uniform mat4 WorldViewProjectionM;

attribute vec3 Position;
attribute vec3 Color;
attribute vec2 UV;
attribute vec3 Normal;

varying vec2 uv;
varying vec3 color;

void main() {
	vec3 lightdir = vec3(0.0, 0.7, -0.7);
	vec4 worldNormal = WorldViewProjectionM * vec4(Normal, 1.0);
	float costheta = clamp(dot(worldNormal.xyz, lightdir), 0.0, 1.0);
	vec3 light = vec3(0.4) + vec3(1.0, 1.0, 1.0) * costheta;
	color = Color * light;
	uv = UV;
	gl_Position = WorldViewProjectionM * vec4(Position, 1.0);
}`

var fs gfx.FragmentShader = `
uniform sampler2D Diffuse;

varying vec2 uv;
varying vec3 color;

void main() {
	gl_FragColor = vec4(texture2D(Diffuse, uv).xyz * color, 1.0);
}`
