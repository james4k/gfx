package gfx

import (
	"errors"
	"github.com/go-gl/gl"
	"reflect"
	"unsafe"
)

type Usage uint16

const (
	StaticDraw Usage = iota
	DynamicDraw
	StreamDraw
	StaticCopy
	DynamicCopy
	StreamCopy
)

func (u Usage) gl() gl.GLenum {
	switch u {
	case StaticDraw:
		return gl.STATIC_DRAW
	case DynamicDraw:
		return gl.DYNAMIC_DRAW
	case StreamDraw:
		return gl.STREAM_DRAW
	case StaticCopy:
		return gl.STATIC_COPY
	case DynamicCopy:
		return gl.DYNAMIC_COPY
	case StreamCopy:
		return gl.STREAM_COPY
	default:
		return gl.STATIC_DRAW
	}
}

type VertexFormat uint32

// TODO: maybe we don't need all of these texcoords...we need a better way for
// the user to extend these.
const (
	VertexPosition VertexFormat = 1 << iota
	VertexColor
	VertexColor1
	VertexNormal
	VertexTangent
	VertexBitangent
	VertexTexcoord
	VertexTexcoord1
	VertexTexcoord2
	VertexTexcoord3
	VertexTexcoord4
	VertexTexcoord5
	VertexTexcoord6
	VertexTexcoord7
	VertexUserData
	VertexUserData1
	VertexUserData2
	VertexUserData3
	MaxVertexFormat = VertexUserData3
)

// TODO: we need to convert the following 4 funcs into a table or something

// AttribBytes gives the byte size of a specific piece of vertex data
func (v VertexFormat) AttribBytes() int {
	const fsize = 4
	switch v {
	case VertexColor,
		VertexColor1:
		// RGBA, 8-bit channels
		return 4
	case VertexTexcoord,
		VertexTexcoord1,
		VertexTexcoord2,
		VertexTexcoord3,
		VertexTexcoord4,
		VertexTexcoord5,
		VertexTexcoord6,
		VertexTexcoord7:
		return 2 * fsize
	case VertexUserData,
		VertexUserData1,
		VertexUserData2,
		VertexUserData3:
		return 4 * fsize
	default:
		return 3 * fsize
	}
}

// attribType gives the GL type of a specific piece of vertex data
func (v VertexFormat) attribType() gl.GLenum {
	switch v {
	case VertexColor,
		VertexColor1:
		return gl.UNSIGNED_BYTE
	default:
		return gl.FLOAT
	}
}

// attribNormalized specifies integral value to be normalized to [0.0-1.0] for unsigned, yata yata.
func (v VertexFormat) attribNormalized() bool {
	switch v {
	case VertexColor,
		VertexColor1:
		return true
	default:
		return false
	}
}

// attribElems gives the number of elements for a specific piece of vertex data
func (v VertexFormat) attribElems() uint {
	switch v {
	case VertexColor,
		VertexColor1:
		return 4
	case VertexTexcoord,
		VertexTexcoord1,
		VertexTexcoord2,
		VertexTexcoord3,
		VertexTexcoord4,
		VertexTexcoord5,
		VertexTexcoord6,
		VertexTexcoord7:
		return 2
	case VertexUserData,
		VertexUserData1,
		VertexUserData2,
		VertexUserData3:
		return 4
	default:
		return 3
	}
}

// TODO: add tests for Stride and Count

// Stride gives the stride in bytes for a vertex buffer.
func (v VertexFormat) Stride() int {
	var i VertexFormat
	stride := 0
	for i = 1; i <= MaxVertexFormat; i <<= 1 {
		if v&i != 0 {
			stride += i.AttribBytes()
		}
	}
	return stride
}

func (v VertexFormat) Count() int {
	var i VertexFormat
	count := 0
	for i = 1; i <= MaxVertexFormat; i <<= 1 {
		if v&i != 0 {
			count++
		}
	}
	return count
}

// VertexAttributes maps shader attributes by name to specific vertex data,
// and as a whole a complete VertexFormat for geometry.
type VertexAttributes map[VertexFormat]string

var DefaultVertexAttributes = VertexAttributes{
	VertexPosition: "Position",
}

// Format returns a VertexFormat bitmask determined by the mapped attributes.
func (v VertexAttributes) Format() VertexFormat {
	var mask VertexFormat
	for k, _ := range v {
		mask |= k
	}
	return mask
}

func (v VertexAttributes) clone() VertexAttributes {
	v2 := make(VertexAttributes, len(v))
	for k, v := range v {
		v2[k] = v
	}
	return v2
}

var ErrBadVertexFormat = errors.New("gfx: bad vertex format")
var errMapBufferFailed = errors.New("gfx: mapbuffer failed")

// VertexBuffer represents interleaved vertices for a VertexFormat set.
type VertexBuffer struct {
	buf    gl.Buffer
	count  int
	format VertexFormat
}

func (b *VertexBuffer) bind() {
	b.buf.Bind(gl.ARRAY_BUFFER)
}

func (b *VertexBuffer) Release() {
	b.buf.Delete()
}

func (b *VertexBuffer) Count() int {
	return b.count
}

func (b *VertexBuffer) Format() VertexFormat {
	return b.format
}

func (b *VertexBuffer) SetVertices(src []byte, usage Usage) error {
	gl.VertexArray(0).Bind()
	b.bind()
	// set size of buffer and invalidate it
	gl.BufferData(gl.ARRAY_BUFFER, len(src), nil, usage.gl())
	if len(src) > 0 {
		// if unmap returns false, the buffer we wrote to is no longer valid and we
		// need to try again. though, this is apparently uncommon in modern
		// drivers. this means it is not feasible to compute/copy vertices directly
		// into the mapped buffer. however, it would be nice to provide a
		// failure-capable API to do this.
		const maxretries = 5
		retries := 0
		for ; retries < maxretries; retries++ {
			ptr := gl.MapBuffer(gl.ARRAY_BUFFER, gl.WRITE_ONLY)
			slicehdr := reflect.SliceHeader{
				Data: uintptr(ptr),
				Len:  len(src),
				Cap:  len(src),
			}
			dest := *(*[]byte)(unsafe.Pointer(&slicehdr))
			copy(dest, src)
			if gl.UnmapBuffer(gl.ARRAY_BUFFER) {
				break
			}
		}
		if retries == maxretries {
			return errMapBufferFailed
		}
	}
	b.count = len(src) / b.format.Stride()
	return nil
}

type IndexBuffer struct {
	buf gl.Buffer
	//offset int
	count int
}

func (b *IndexBuffer) bind() {
	b.buf.Bind(gl.ELEMENT_ARRAY_BUFFER)
}

func (b *IndexBuffer) Release() {
	b.buf.Delete()
}

/*
func (b *IndexBuffer) Slice(i, j int) IndexBuffer {
	if j < i || i < 0 || j < 0 || i >= b.count || j > b.count {
		panic("IndexBuffer.Slice bounds out of range")
	}
	return IndexBuffer{
		buf:    b.buf,
		offset: b.offset + i,
		count:  j - i,
	}
}

func (b *IndexBuffer) Offset() int {
	return b.offset
}
*/

func (b *IndexBuffer) Count() int {
	return b.count
}

func (b *IndexBuffer) SetIndices(src []uint16, usage Usage) error {
	gl.VertexArray(0).Bind()
	b.bind()
	idxsize := 2 * len(src)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, idxsize, nil, usage.gl())
	if idxsize > 0 {
		const maxretries = 5
		retries := 0
		for ; retries < maxretries; retries++ {
			ptr := gl.MapBuffer(gl.ELEMENT_ARRAY_BUFFER, gl.WRITE_ONLY)
			slicehdr := reflect.SliceHeader{
				Data: uintptr(ptr),
				Len:  len(src),
				Cap:  len(src),
			}
			dest := *(*[]uint16)(unsafe.Pointer(&slicehdr))
			copy(dest, src)
			if gl.UnmapBuffer(gl.ELEMENT_ARRAY_BUFFER) {
				break
			}
		}
		if retries == maxretries {
			return errMapBufferFailed
		}
	}
	//b.offset = 0
	b.count = len(src)
	return nil
}

// TODO: 32-bit indices...maybe need another type altogether
//func (b *IndexBuffer) SetIndices32(p []uint32) error {
//panic("NO.")
//}

type VertexData interface {
	VertexCount() int
	VertexFormat() VertexFormat
	CopyVertices(dest *VertexBuffer, usage Usage) error
}

type IndexData interface {
	IndexCount() int
	CopyIndices(dest *IndexBuffer, usage Usage) error
}

// Geometry represents a piece of mesh that can be rendered in a single
// draw call. It may or may not contain an index buffer, but always has
// a vertex buffer.
type Geometry struct {
	usage Usage
	VertexBuffer
	IndexBuffer
}

// NewGeometry copies vertices from src as well as indices if IndexData
// is implemented, into newly allocated buffer objects.
func NewGeometry(src VertexData, usage Usage) (*Geometry, error) {
	srcidx, ok := src.(IndexData)
	geom := allocGeom(usage, ok)
	geom.VertexBuffer.format = src.VertexFormat()
	err := src.CopyVertices(&geom.VertexBuffer, usage)
	if err != nil {
		return nil, err
	}
	if ok {
		err := srcidx.CopyIndices(&geom.IndexBuffer, usage)
		if err != nil {
			return nil, err
		}
	}
	return geom, nil
}

func allocGeom(usage Usage, hasIndex bool) *Geometry {
	geom := &Geometry{
		usage: usage,
	}
	if hasIndex {
		var bufs [2]gl.Buffer
		gl.GenBuffers(bufs[:])
		geom.VertexBuffer.buf = bufs[0]
		geom.IndexBuffer.buf = bufs[1]
	} else {
		geom.VertexBuffer.buf = gl.GenBuffer()
	}
	return geom
}

func (g *Geometry) Release() {
	g.VertexBuffer.Release()
	g.IndexBuffer.Release()
}

// CopyFrom copies vertices from src as well as indices if IndexData
// is implemented.
func (g *Geometry) CopyFrom(src VertexData) error {
	err := src.CopyVertices(&g.VertexBuffer, g.usage)
	if err != nil {
		return err
	}
	if srcidx, ok := src.(IndexData); ok {
		err := srcidx.CopyIndices(&g.IndexBuffer, g.usage)
		if err != nil {
			return err
		}
	}
	return nil
}

// StaticGeometry creates a static index/vertex buffer pair, uploads data to
// GPU and returns the representing Geometry. Panics if vertices length does
// not fit the format specified. There should be 3 float32's for every vertex
// channel. It is assumed the vertex data is interleaved.
/*
func StaticGeometry(indices []uint16, vertices []float32, format VertexFormat) Geometry {
	// TODO: sanity check on vertices length based on VertexFormat?
	stride := format.Stride()
	if len(vertices)*4%stride != 0 {
		panic("gfx: vertex count does not fit vertex format")
	}

	geom := initGeom(StaticDraw)

	var bufs [2]gl.Buffer
	gl.GenBuffers(bufs[:])
	bufs[0].Bind(gl.ELEMENT_ARRAY_BUFFER)
	size := len(indices) * int(unsafe.Sizeof(indices[0]))
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, size, &indices[0], gl.STATIC_DRAW)
	bufs[1].Bind(gl.ARRAY_BUFFER)
	size = len(vertices) * int(unsafe.Sizeof(vertices[0]))
	gl.BufferData(gl.ARRAY_BUFFER, size, &vertices[0], gl.STATIC_DRAW)

	return geom
}
*/
