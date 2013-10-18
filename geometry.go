package gfx

import (
	"errors"
	"github.com/go-gl/gl"
	"reflect"
	"runtime"
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
	panic("unreachable")
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

// vertexBytes gives the byte size of a specific piece of vertex data
func vertexBytes(v VertexFormat) int {
	const fsize = 4
	switch v {
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
	panic("unreachable")
}

// TODO: add tests for Stride and Count

// Stride gives the stride in bytes for a vertex buffer.
func (v VertexFormat) Stride() int {
	var i VertexFormat
	stride := 0
	for i = 1; i <= MaxVertexFormat; i <<= 1 {
		if v&i != 0 {
			stride += vertexBytes(i)
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

type IndexBuffer interface {
	bind()
	Slice(int, int) IndexBuffer
	Offset() int
	Count() int
}

type indexBuf struct {
	buf           gl.Buffer
	offset, count int
}

func (b *indexBuf) bind() {
	b.buf.Bind(gl.ELEMENT_ARRAY_BUFFER)
}

func (b *indexBuf) release() {
	trashbin.addBuffer(b.buf)
}

func (b *indexBuf) Slice(i, j int) IndexBuffer {
	if j < i || i < 0 || j < 0 || i >= b.count || j > b.count {
		panic("IndexBuffer Slice bounds out of range")
	}
	return &indexBuf{
		buf:    b.buf,
		offset: b.offset + i,
		count:  j - i,
	}
}

func (i *indexBuf) Offset() int {
	return i.offset
}

func (i *indexBuf) Count() int {
	return i.count
}

// VertexBuffer represents interleaved vertices for a VertexFormat set.
type VertexBuffer interface {
	bind()
	Count() int
	Format() VertexFormat
}

type vertexBuf struct {
	buf    gl.Buffer
	count  int
	format VertexFormat
}

func (b *vertexBuf) bind() {
	b.buf.Bind(gl.ARRAY_BUFFER)
}

func (b *vertexBuf) release() {
	trashbin.addBuffer(b.buf)
}

func (b *vertexBuf) Count() int {
	return b.count
}

func (b *vertexBuf) Format() VertexFormat {
	return b.format
}

// Geometry represents a piece of mesh that can be rendered in a single draw
// call.
type Geometry interface {
	Indices() IndexBuffer
	Vertices() VertexBuffer
	CopyFrom(GeometryData) error
}

type IndexBufferData interface {
	IndexCount() int
	CopyIndices(buf []uint16) error
}

type VertexBufferData interface {
	VertexCount() int
	VertexFormat() VertexFormat
	CopyVertices(buf []float32) error
}

type GeometryData interface {
	IndexBufferData
	VertexBufferData
}

type geometry struct {
	indices     *indexBuf
	vertices    *vertexBuf
	usage       Usage
	vertexArray gl.VertexArray
}

func (g *geometry) Indices() IndexBuffer {
	return g.indices
}

func (g *geometry) Vertices() VertexBuffer {
	return g.vertices
}

func NewGeometry(data GeometryData, usage Usage) (Geometry, error) {
	geom := initGeom(usage)
	geom.vertices.format = data.VertexFormat()
	err := geom.CopyFrom(data)
	if err != nil {
		return nil, err
	}
	return geom, nil
}

func initGeom(usage Usage) *geometry {
	var bufs [2]gl.Buffer
	gl.GenBuffers(bufs[:])
	geom := &geometry{
		indices: &indexBuf{
			buf: bufs[0],
		},
		vertices: &vertexBuf{
			buf: bufs[1],
		},
		usage:       usage,
		vertexArray: gl.GenVertexArray(),
	}
	runtime.SetFinalizer(geom.indices, (*indexBuf).release)
	runtime.SetFinalizer(geom.vertices, (*vertexBuf).release)
	// TODO: finalizer for geom; need to remove VAO
	return geom
}

var errBadVertexFormat = errors.New("gfx: bad vertex format")

func (g *geometry) CopyFrom(data GeometryData) error {
	vf := data.VertexFormat()
	if g.vertices.Format() != vf {
		return errBadVertexFormat
	}

	// if unmap returns false, the buffer we wrote to is no longer valid
	// and we need to try again. though, this is apparently uncommon in
	// modern drivers.
	g.indices.bind()
	idxlen := data.IndexCount()
	idxsize := 2 * idxlen
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, idxsize, nil, g.usage.gl())
	if idxsize > 0 {
		for stop := false; !stop; {
			ptr := gl.MapBuffer(gl.ELEMENT_ARRAY_BUFFER, gl.WRITE_ONLY)
			slicehdr := reflect.SliceHeader{
				Data: uintptr(ptr),
				Len:  idxlen,
				Cap:  idxlen,
			}
			slice := *(*[]uint16)(unsafe.Pointer(&slicehdr))
			err := data.CopyIndices(slice)
			stop = gl.UnmapBuffer(gl.ELEMENT_ARRAY_BUFFER)
			if err != nil {
				return err
			}
		}
	}
	g.indices.offset = 0
	g.indices.count = idxlen

	g.vertices.bind()
	vertlen := data.VertexCount()
	vertsize := vf.Stride() * vertlen
	gl.BufferData(gl.ARRAY_BUFFER, vertsize, nil, g.usage.gl())
	if vertsize > 0 {
		for stop := false; !stop; {
			ptr := gl.MapBuffer(gl.ARRAY_BUFFER, gl.WRITE_ONLY)
			slicehdr := reflect.SliceHeader{
				Data: uintptr(ptr),
				Len:  vertsize / 4,
				Cap:  vertsize / 4,
			}
			slice := *(*[]float32)(unsafe.Pointer(&slicehdr))
			err := data.CopyVertices(slice)
			stop = gl.UnmapBuffer(gl.ARRAY_BUFFER)
			if err != nil {
				return err
			}
		}
	}
	g.vertices.count = vertlen

	return nil
}

// StaticGeometry creates a static index/vertex buffer pair, uploads data to
// GPU and returns the representing Geometry. Panics if vertices length does
// not fit the format specified. There should be 3 float32's for every vertex
// channel. It is assumed the vertex data is interleaved.
func StaticGeometry(indices []uint16, vertices []float32, format VertexFormat) Geometry {
	// TODO: sanity check on vertices length based on VertexFormat?
	stride := format.Stride()
	if len(vertices)*4%stride != 0 {
		panic("gfx: vertex count does not fit vertex format")
	}

	var bufs [2]gl.Buffer
	gl.GenBuffers(bufs[:])
	bufs[0].Bind(gl.ELEMENT_ARRAY_BUFFER)
	size := len(indices) * int(unsafe.Sizeof(indices[0]))
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, size, &indices[0], gl.STATIC_DRAW)
	bufs[1].Bind(gl.ARRAY_BUFFER)
	size = len(vertices) * int(unsafe.Sizeof(vertices[0]))
	gl.BufferData(gl.ARRAY_BUFFER, size, &vertices[0], gl.STATIC_DRAW)

	geom := &geometry{
		indices: &indexBuf{
			buf:   bufs[0],
			count: len(indices),
		},
		vertices: &vertexBuf{
			buf:    bufs[1],
			count:  (len(vertices) * 4) / stride,
			format: format,
		},
	}
	runtime.SetFinalizer(geom.indices, (*indexBuf).release)
	runtime.SetFinalizer(geom.vertices, (*vertexBuf).release)
	return geom
}

type GeometryBuilder struct {
	vf       VertexFormat
	stride   int
	cur      int
	curvf    VertexFormat // data that's been set on the current vertex
	lastdata map[VertexFormat]int
	offsets  map[VertexFormat]int
	verts    []float32
	idxs     []uint16
	nextidx  uint16
}

func BuildGeometry(vf VertexFormat) *GeometryBuilder {
	return &GeometryBuilder{
		vf:       vf,
		stride:   vf.Stride() / 4,
		lastdata: make(map[VertexFormat]int, vf.Count()),
	}
}

// Clear resets buffers to zero length.
func (g *GeometryBuilder) Clear() {
	g.lastdata = make(map[VertexFormat]int, len(g.lastdata))
	g.cur = 0
	g.curvf = 0
	g.verts = g.verts[:0]
	g.idxs = g.idxs[:0]
	g.nextidx = 0
}

// TODO: add a test
func (g *GeometryBuilder) offset(v VertexFormat) int {
	if g.vf&v == 0 {
		panic(errBadVertexFormat)
	}
	if g.offsets == nil {
		offs := 0
		g.offsets = map[VertexFormat]int{}
		for i := VertexFormat(1); i <= MaxVertexFormat; i <<= 1 {
			if g.vf&i != 0 {
				g.offsets[i] = offs
				offs += vertexBytes(i) / 4
			}
		}
	}
	return g.offsets[v]
}

func (g *GeometryBuilder) next() {
	if len(g.verts) != 0 {
		g.cur += g.stride
	}
	g.curvf = 0
	zeros := make([]float32, g.stride)
	g.verts = append(g.verts, zeros...)
}

// fillVertex fills the rest of the vertex data using the last set data
// from a previous vertex
func (g *GeometryBuilder) fillVertex() {
	for i := VertexFormat(1); i <= MaxVertexFormat; i <<= 1 {
		if g.vf&i != 0 && g.curvf&i == 0 {
			offs, ok := g.lastdata[i]
			if ok {
				data := g.verts[offs : offs+vertexBytes(i)/4]
				g.set(i, data)
			}
		}
	}
}

func (g *GeometryBuilder) set(v VertexFormat, data []float32) {
	g.curvf |= v
	offs := g.cur + g.offset(v)
	g.lastdata[v] = offs
	copy(g.verts[offs:offs+len(data)], data)
}

// Position creates a new vertex and sets the vertex position.
// TODO: set unset data for previous vertex to the last data the user provided
func (g *GeometryBuilder) Position(x, y, z float32) *GeometryBuilder {
	g.fillVertex()
	g.next()
	g.set(VertexPosition, []float32{x, y, z})
	return g
}

// Color sets the vertex color.
func (gb *GeometryBuilder) Color(r, g, b float32) *GeometryBuilder {
	gb.set(VertexColor, []float32{r, g, b})
	return gb
}

// Normal sets the vertex normal.
func (g *GeometryBuilder) Normal(x, y, z float32) *GeometryBuilder {
	g.set(VertexNormal, []float32{x, y, z})
	return g
}

// Texcoord sets the vertex texture coordinate.
func (g *GeometryBuilder) Texcoord(u, v float32) *GeometryBuilder {
	g.set(VertexTexcoord, []float32{u, v})
	return g
}

// Indices appends new indices to the buffer that are relative to the maximum index in the buffer.
func (g *GeometryBuilder) Indices(idxs ...uint16) *GeometryBuilder {
	// TODO: could really use a test
	newnext := g.nextidx
	for i, idx := range idxs {
		idx += g.nextidx
		if idx >= newnext {
			newnext = idx + 1
		}
		idxs[i] = idx
	}
	g.nextidx = newnext
	g.idxs = append(g.idxs, idxs...)
	return g
}

func (g *GeometryBuilder) SetIndices(idxs ...uint16) {
	g.nextidx = 0
	g.idxs = make([]uint16, len(idxs))
	copy(g.idxs, idxs)
}

func (g *GeometryBuilder) VertexFormat() VertexFormat {
	return g.vf
}

// IndexCount returns the number of indices available.
func (g *GeometryBuilder) IndexCount() int {
	if g.idxs == nil {
		return g.VertexCount()
	}
	return len(g.idxs)
}

// CopyIndices copies the indices directly into buf. If IndexCount() does not
// match len(buf), an error is returned.
func (g *GeometryBuilder) CopyIndices(buf []uint16) error {
	if g.idxs != nil {
		// TODO: check len
		copy(buf, g.idxs)
		return nil
	}
	for i := range buf {
		buf[i] = uint16(i)
	}
	return nil
}

// VertexCount returns the number of vertices available.
func (g *GeometryBuilder) VertexCount() int {
	return len(g.verts) / g.stride
}

// CopyVertices copies the vertices directly into buf. If len(buf) does not equal VertexCount()*(VertexFormat().Stride()/4), an error is returned.
func (g *GeometryBuilder) CopyVertices(buf []float32) error {
	g.fillVertex()
	// TODO: check len
	copy(buf, g.verts)
	return nil
}

//func BuildGeometry(format VertexFormat) *GeometryBuilder {
//}
