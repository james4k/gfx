package gfx

import (
	"github.com/go-gl/gl"
	"reflect"
)

type Shader struct {
	prog         gl.Program
	vertexAttrs  VertexAttributes
	vertexFormat VertexFormat

	indexCount  int
	indexOffset int

	prevArrays []gl.AttribLocation
}

type ModelMaterial struct {
	*Shader "model.shader"

	viewMatrix           Mat4
	viewMatrixInverse    Mat4
	projectionMatrix     Mat4
	viewProjectionMatrix Mat4
	viewPosition         Vec3

	worldMatrix       Mat4
	worldNormalMatrix Mat3

	indices  IndexBuffer
	vertices VertexBuffer

	//Diffuse Sampler

	// Idea: for shader combos, put inputs in an embedded struct with a tag
	// defining the condition. or maybe just a tag is good enough..who knows
}

type ShaderSource interface {
	typ() gl.GLenum
	source() string
}

type VertexShader string
type FragmentShader string

func (v VertexShader) typ() gl.GLenum {
	return gl.VERTEX_SHADER
}

func (v VertexShader) source() string {
	return string(v)
}

func (f FragmentShader) typ() gl.GLenum {
	return gl.FRAGMENT_SHADER
}

func (f FragmentShader) source() string {
	return string(f)
}

func BuildShader(attrs VertexAttributes, srcs ...ShaderSource) *Shader {
	shader := &Shader{
		vertexAttrs:  attrs.clone(),
		vertexFormat: attrs.Format(),
	}
	shader.prog = gl.CreateProgram()
	ss := make([]gl.Shader, len(srcs))
	for i, src := range srcs {
		s := gl.CreateShader(src.typ())
		s.Source(src.source())
		s.Compile()
		// TODO: check for compile errors
		println(s.GetInfoLog())
		shader.prog.AttachShader(s)
		ss[i] = s
	}
	shader.prog.Link()
	println(shader.prog.GetInfoLog())
	// TODO: check for link errors
	// TODO: we can probably return an error...but do we provide compile/link
	// errors or throw those at a logger?
	// TODO: add finalizer for program

	// No longer need shader objects with a fully built program.
	for _, s := range ss {
		shader.prog.DetachShader(s)
		s.Delete()
	}
	return shader
}

func (s *Shader) load() {
	// TODO: compile/attach shaders and such

	// Projection/view uniforms
	//s.viewM = s.prog.GetUniformLocation("ViewM")
}

func (s *Shader) Use() {
	// checkpoint here for releasing unused GL resources
	releaseGarbage()

	s.prog.Use()
}

// SetUniforms takes struct fields with "uniform" tag and assigns their values
// to the shader's uniform variables.
func (s *Shader) SetUniforms(data interface{}) {
	// TODO: recurse down embedded structs to find their fields
	val := reflect.ValueOf(data)
	typ := val.Type()
	n := val.NumField()
	for i := 0; i < n; i++ {
		f := typ.Field(i)
		v := val.Field(i)
		// TODO: skip unexported fields.. they will panic!
		name := f.Tag.Get("uniform")
		if name == "" {
			continue
		}
		iface := v.Interface()
		switch iface.(type) {
		// TODO: float arrays up to [4]float32
		// TODO: samplers
		case float32:
			u := s.prog.GetUniformLocation(name)
			u.Uniform1f(iface.(float32))
		case Mat3:
			u := s.prog.GetUniformLocation(name)
			u.UniformMatrix3f(false, iface.(Mat3).Pointer())
		case Mat4:
			u := s.prog.GetUniformLocation(name)
			u.UniformMatrix4f(false, iface.(Mat4).Pointer())
		case [16]float32:
			u := s.prog.GetUniformLocation(name)
			val := iface.([16]float32)
			u.UniformMatrix4f(false, &val)
		default:
			sampler := iface.(Sampler)
			if sampler != nil {
				u := s.prog.GetUniformLocation(name)
				gl.ActiveTexture(gl.TEXTURE0)
				sampler.bind()
				u.Uniform1i(0)
			}
		}
	}
}

// SetGeometry sets the vertex attributes and binds the index buffer.
func (s *Shader) SetGeometry(geom Geometry) {
	vertices := geom.Vertices()
	if s.vertexFormat != vertices.Format() {
	}
	var attrib gl.AttribLocation
	vertices.bind()

	for _, a := range s.prevArrays {
		a.DisableArray()
	}
	s.prevArrays = s.prevArrays[:0]

	var i VertexFormat
	offset := 0
	stride := s.vertexFormat.Stride()
	for i = 1; i <= MaxVertexFormat; i <<= 1 {
		if s.vertexFormat&i == 0 {
			continue
		}
		name, ok := s.vertexAttrs[i]
		if !ok {
			// TODO: return error or something?
			break
		}
		attrib = s.prog.GetAttribLocation(name)
		if attrib >= 0 {
			attrib.EnableArray()
			elems := uint(vertexBytes(i)) / 4
			attrib.AttribPointer(elems, gl.FLOAT, false, stride, uintptr(offset))
			s.prevArrays = append(s.prevArrays, attrib)
		}
		offset += vertexBytes(i)
	}

	indices := geom.Indices()
	s.indexCount = indices.Count()
	s.indexOffset = indices.Offset()
	indices.bind()
}

// Draw makes a glDrawElements call using the previously set uniforms and
// geometry.
func (s *Shader) Draw() {
	gl.DrawElements(gl.TRIANGLES, s.indexCount, gl.UNSIGNED_SHORT, uintptr(s.indexOffset))
}
