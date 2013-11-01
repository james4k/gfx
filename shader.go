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
		// TODO: user should pass in an io.Writer for us to write the log to
		// TODO: does glGetError give us anything more definitive?
		println(s.GetInfoLog())
		shader.prog.AttachShader(s)
		ss[i] = s
	}
	shader.prog.Link()
	// TODO: user should pass in an io.Writer for us to write the log to
	// TODO: does glGetError give us anything more definitive?
	println(shader.prog.GetInfoLog())
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

func (s *Shader) VertexFormat() VertexFormat {
	return s.vertexFormat
}

// Use puts the shader as the active program to bind data to and execute.
func (s *Shader) Use() {
	// checkpoint here for releasing unused GL resources
	releaseGarbage()

	s.prog.Use()
}

// SetUniforms takes struct fields with "uniform" tag and assigns their values
// to the shader's uniform variables.
// TODO: really need to return an error; a lot of room for user error here,
// with uniform names and shit that need to be right.
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
			sampler := iface.(Sampler2D)
			if sampler != nil {
				u := s.prog.GetUniformLocation(name)
				// TODO: need to select texture unit
				gl.ActiveTexture(gl.TEXTURE0)
				sampler.bind()
				u.Uniform1i(0)
			}
		}
	}
}

type GeometryLayout struct {
	vao    gl.VertexArray
	geom   Geometry
	shader *Shader
}

func (g *GeometryLayout) Geometry() Geometry {
	return g.geom
}

// LayoutGeometry builds a vertex array object holding vertex attribute locations and
// buffer pointers for the given geometry.
func (s *Shader) LayoutGeometry(geom Geometry) *GeometryLayout {
	vertices := geom.Vertices()
	if s.vertexFormat != vertices.Format() {
		// TODO: really need to return an error; mainly for vertex format
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
			attrib.AttribPointer(attribElems(i), attribType(i), attribNormalized(i), stride, uintptr(offset))
			attrib.EnableArray()
		}
		offset += attribBytes(i)
	}

	geom.Indices().bind()
	gl.VertexArray(0).Bind()

	layout := &GeometryLayout{
		vao:    vao,
		geom:   geom,
		shader: s,
	}
	// TODO: finalizer to delete VAO
	return layout
}

// SetGeometryLayout binds the underlying vertex array object that holds the buffer pointers.
func (s *Shader) SetGeometryLayout(layout *GeometryLayout) error {
	// TODO: come up with a more sophistcated layout compatibility check. abstract this away from the shader somehow.
	if layout.shader != s {
		return errors.New("gfx: geometry layout not compatible with this shader")
	}
	indices := layout.Geometry().Indices()
	s.indexCount = indices.Count()
	s.indexOffset = indices.Offset()
	layout.vao.Bind()
	return nil
}

// Draw makes a glDrawElements call using the previously set uniforms and
// geometry.
// TODO: Need to think about glDrawElementsInstanced, and per-instance vertex
// attributes (glVertexAttribDivisor). Will probably need a separate
// DrawInstanced() method, or maybe add an instances parameter, or a
// SetInstances method? Hmmz
func (s *Shader) Draw() {
	gl.DrawElements(gl.TRIANGLES, s.indexCount, gl.UNSIGNED_SHORT, uintptr(s.indexOffset))
	gl.VertexArray(0).Bind()
}
