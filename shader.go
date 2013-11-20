package gfx

import (
	"errors"
	"github.com/go-gl/gl"
	"reflect"
)

type Shader struct {
	prog         gl.Program
	vertexAttrs  VertexAttributes
	vertexFormat VertexFormat

	indexCount  int
	indexOffset int
	indexType   gl.GLenum
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
		println(s.GetInfoLog())
		shader.prog.AttachShader(s)
		ss[i] = s
	}
	shader.prog.Link()
	println(shader.prog.GetInfoLog())

	// No longer need shader objects with a fully built program.
	for _, s := range ss {
		shader.prog.DetachShader(s)
		s.Delete()
	}
	return shader
}

func (s *Shader) Delete() {
	s.prog.Delete()
}

func (s *Shader) VertexFormat() VertexFormat {
	return s.vertexFormat
}

// Use puts the shader as the active program to bind data to and execute.
func (s *Shader) Use() {
	s.prog.Use()
}

// SetUniforms takes struct fields with "uniform" tag and assigns their values
// to the shader's uniform variables.
func (s *Shader) SetUniforms(data interface{}) {
	val := reflect.ValueOf(data)
	typ := val.Type()
	n := val.NumField()
	for i := 0; i < n; i++ {
		v := val.Field(i)
		if !v.CanInterface() {
			continue
		}
		f := typ.Field(i)
		name := f.Tag.Get("uniform")
		if name == "" {
			continue
		}
		iface := v.Interface()
		switch iface.(type) {
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
		case *Sampler2D:
			u := s.prog.GetUniformLocation(name)
			sampler := iface.(*Sampler2D)
			gl.ActiveTexture(gl.TEXTURE0)
			sampler.bind()
			u.Uniform1i(0)
		}
	}
}

type GeometryLayout struct {
	vao    gl.VertexArray
	idxbuf *IndexBuffer
	shader *Shader
}

// LayoutGeometry builds a vertex array object holding vertex attribute locations and
// buffer pointers for the given geometry.
func LayoutGeometry(s *Shader, geom *Geometry) *GeometryLayout {
	vertices := geom.VertexBuffer
	if s.vertexFormat != vertices.Format() {
		panic("moo")
	}
	vao := gl.GenVertexArray()
	vao.Bind()

	var attrib gl.AttribLocation
	vertices.bind()

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
			attrib.AttribPointer(i.attribElems(), i.attribType(), i.attribNormalized(), stride, uintptr(offset))
			attrib.EnableArray()
		}
		offset += i.AttribBytes()
	}

	geom.IndexBuffer.bind()

	layout := &GeometryLayout{
		vao:    vao,
		idxbuf: &geom.IndexBuffer,
		shader: s,
	}
	return layout
}

func (g *GeometryLayout) Delete() {
	g.vao.Delete()
}

// SetGeometry binds the underlying vertex array object that holds the buffer pointers.
func (s *Shader) SetGeometry(layout *GeometryLayout) error {
	if layout.shader != s {
		return errors.New("gfx: geometry layout not compatible with this shader")
	}
	s.indexCount = layout.idxbuf.Count()
	s.indexType = layout.idxbuf.elemtype
	//s.indexOffset = indices.Offset()
	layout.vao.Bind()
	return nil
}

// Draw makes a glDrawElements call using the previously set uniforms and
// geometry.
func (s *Shader) Draw() {
	gl.DrawElements(gl.TRIANGLES, s.indexCount, s.indexType, uintptr(s.indexOffset))
}
