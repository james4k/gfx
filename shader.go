package gfx

import (
	"errors"
	"fmt"
	"github.com/go-gl/gl"
	"reflect"
	"unsafe"
)

type Shader struct {
	prog         gl.Program
	vertexAttrs  VertexAttributes
	vertexFormat VertexFormat
	texlocs      []gl.UniformLocation
	indexCount   int
	indexOffset  int
	indexType    gl.GLenum
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

// texunit finds a previously assigned texture unit for loc, or
// selects the next one.
func (s *Shader) texunit(loc gl.UniformLocation) int {
	for i, v := range s.texlocs {
		if loc == v {
			return i
		}
	}
	v := len(s.texlocs)
	s.texlocs = append(s.texlocs, loc)
	return v
}

// Use puts the shader as the active program to bind data to and execute.
func (s *Shader) Use() {
	s.prog.Use()
	s.texlocs = s.texlocs[:]
}

// AssignUniforms takes struct fields with "uniform" tag and assigns their values
// to the shader's uniform variables. data must be a pointer to a struct.
func (s *Shader) AssignUniforms(data interface{}) error {
	val := reflect.ValueOf(data)
	ptr := val.Pointer()
	val = val.Elem()
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
		err := s.assign(unsafe.Pointer(ptr+f.Offset), v, f.Type, name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Shader) assign(ptr unsafe.Pointer, val reflect.Value, typ reflect.Type, name string) error {
	u := s.prog.GetUniformLocation(name)
	if typ.Kind() == reflect.Ptr {
		if s.assignPrimitive(unsafe.Pointer(val.Pointer()), typ.Elem(), u) {
			return nil
		}
	} else if s.assignPrimitive(ptr, typ, u) {
		return nil
	}
	iface := val.Interface()
	switch iface.(type) {
	// special types
	case *Sampler2D:
		sampler := iface.(*Sampler2D)
		texunit := s.texunit(u)
		gl.ActiveTexture(gl.TEXTURE0 + gl.GLenum(texunit))
		sampler.bind()
		u.Uniform1i(texunit)
	default:
		return fmt.Errorf("gfx: invalid uniform type %v", typ)
	}
	return nil
}

func (s *Shader) assignPrimitive(ptr unsafe.Pointer, typ reflect.Type, u gl.UniformLocation) bool {
	switch typ.Kind() {
	// basic primitives
	case reflect.Int:
		u.Uniform1i(*(*int)(ptr))
	case reflect.Int32:
		u.Uniform1i(int(*(*int32)(ptr)))
	case reflect.Float32:
		u.Uniform1f(*(*float32)(ptr))
	// arrays represent vectors or matrices
	case reflect.Array:
		size := typ.Len()
		elemtyp := typ.Elem()
		switch elemtyp.Kind() {
		case reflect.Int32:
			switch size {
			case 2:
				slice := (*(*[2]int32)(ptr))[:]
				u.Uniform2iv(1, slice)
			case 3:
				slice := (*(*[3]int32)(ptr))[:]
				u.Uniform3iv(1, slice)
			case 4:
				slice := (*(*[4]int32)(ptr))[:]
				u.Uniform4iv(1, slice)
			default:
				return false
			}
		case reflect.Float32:
			switch size {
			case 2:
				slice := (*(*[2]float32)(ptr))[:]
				u.Uniform2fv(1, slice)
			case 3:
				slice := (*(*[3]float32)(ptr))[:]
				u.Uniform3fv(1, slice)
			case 4:
				slice := (*(*[4]float32)(ptr))[:]
				u.Uniform4fv(1, slice)
			case 9:
				matptr := (*[9]float32)(ptr)
				u.UniformMatrix3f(false, matptr)
			case 16:
				matptr := (*[16]float32)(ptr)
				u.UniformMatrix4f(false, matptr)
			default:
				return false
			}
		default:
			return false
		}
	default:
		return false
	}
	return true
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
