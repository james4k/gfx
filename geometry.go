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

// TODO: we need to convert the following 4 funcs into a table or something

// attribBytes gives the byte size of a specific piece of vertex data
func attribBytes(v VertexFormat) int {
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
	panic("unreachable")
}

// attribType gives the GL type of a specific piece of vertex data
func attribType(v VertexFormat) gl.GLenum {
	switch v {
	case VertexColor,
		VertexColor1:
		return gl.UNSIGNED_BYTE
	default:
		return gl.FLOAT
	}
	panic("unreachable")
}

// attribNormalized specifies integral value to be normalized to [0.0-1.0] for unsigned, yata yata.
func attribNormalized(v VertexFormat) bool {
	switch v {
	case VertexColor,
		VertexColor1:
		return true
	default:
		return false
	}
	panic("unreachable")
}

// attribElems gives the number of elements for a specific piece of vertex data
func attribElems(v VertexFormat) uint {
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
	panic("unreachable")
}

// TODO: add tests for Stride and Count

// Stride gives the stride in bytes for a vertex buffer.
func (v VertexFormat) Stride() int {
	var i VertexFormat
	stride := 0
	for i = 1; i <= MaxVertexFormat; i <<= 1 {
		if v&i != 0 {
			stride += attribBytes(i)
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

var errBadVertexFormat = errors.New("gfx: bad vertex format")
var errMapBufferFailed = errors.New("gfx: mapbuffer failed")

// VertexBuffer represents interleaved vertices for a VertexFormat set.
type VertexBuffer struct {
	buf    gl.Buffer
	count  int
	format VertexFormat
}

func (b VertexBuffer) bind() {
	b.buf.Bind(gl.ARRAY_BUFFER)
}

func (b VertexBuffer) Release() {
	b.buf.Delete()
}

func (b VertexBuffer) Count() int {
	return b.count
}

func (b VertexBuffer) Format() VertexFormat {
	return b.format
}

func (b *VertexBuffer) SetVertices(src []byte, usage Usage, vf VertexFormat) error {
	if b.Format() != vf {
		return errBadVertexFormat
	}
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
	b.count = len(src) / vf.Stride()
	return nil
}

type IndexBuffer struct {
	buf gl.Buffer
	//offset int
	count int
}

func (b IndexBuffer) bind() {
	b.buf.Bind(gl.ELEMENT_ARRAY_BUFFER)
}

func (b IndexBuffer) Release() {
	b.buf.Delete()
}

/*
func (b IndexBuffer) Slice(i, j int) IndexBuffer {
	if j < i || i < 0 || j < 0 || i >= b.count || j > b.count {
		panic("IndexBuffer.Slice bounds out of range")
	}
	return IndexBuffer{
		buf:    b.buf,
		offset: b.offset + i,
		count:  j - i,
	}
}

func (b IndexBuffer) Offset() int {
	return b.offset
}
*/

func (b IndexBuffer) Count() int {
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
func (b *IndexBuffer) SetIndices32(p []uint32) error {
	panic("NO.")
}

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

func (g Geometry) Release() {
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

type GeometryBuffer struct {
	vf       VertexFormat
	stride   int
	cur      int
	curvf    VertexFormat // data that's been set on the current vertex
	lastdata map[VertexFormat]int
	offsets  map[VertexFormat]int
	verts    []byte
	idxs     []uint16
	nextidx  uint16
}

func NewGeometryBuffer(vf VertexFormat) *GeometryBuffer {
	return &GeometryBuffer{
		vf:       vf,
		stride:   vf.Stride(),
		lastdata: make(map[VertexFormat]int, vf.Count()),
	}
}

// Clear resets buffers to zero length.
func (g *GeometryBuffer) Clear() {
	g.lastdata = make(map[VertexFormat]int, len(g.lastdata))
	g.cur = 0
	g.curvf = 0
	g.verts = g.verts[:0]
	g.idxs = g.idxs[:0]
	g.nextidx = 0
}

// TODO: add a test
func (g *GeometryBuffer) offset(v VertexFormat) int {
	if g.vf&v == 0 {
		panic(errBadVertexFormat)
	}
	if g.offsets == nil {
		offs := 0
		g.offsets = map[VertexFormat]int{}
		for i := VertexFormat(1); i <= MaxVertexFormat; i <<= 1 {
			if g.vf&i != 0 {
				g.offsets[i] = offs
				offs += attribBytes(i)
			}
		}
	}
	return g.offsets[v]
}

func (g *GeometryBuffer) next() {
	if len(g.verts) != 0 {
		g.cur += g.stride
	}
	g.curvf = 0
	zeros := make([]uint8, g.stride)
	g.verts = append(g.verts, zeros...)
}

// fillVertex fills the rest of the vertex data using the last set data
// from a previous vertex
func (g *GeometryBuffer) fillVertex() {
	/*
		for i := VertexFormat(1); i <= MaxVertexFormat; i <<= 1 {
			if g.vf&i != 0 && g.curvf&i == 0 {
				offs, ok := g.lastdata[i]
				if ok {
					data := g.verts[offs : offs+attribBytes(i)/4]
					g.set(i, data)
				}
			}
		}
	*/
	for i, offs := range g.lastdata {
		if g.vf&i != 0 && g.curvf&i == 0 {
			data := g.verts[offs : offs+attribBytes(i)]
			g.set(i, data)
		}
	}
}

func (g *GeometryBuffer) set(v VertexFormat, data []uint8) {
	g.curvf |= v
	offs := g.cur + g.offset(v)
	g.lastdata[v] = offs
	copy(g.verts[offs:offs+len(data)], data)
}

func (g *GeometryBuffer) setf(v VertexFormat, data []float32) {
	sz := len(data) * 4
	slicehdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&data[0])),
		Len:  sz,
		Cap:  sz,
	}
	slice := *(*[]uint8)(unsafe.Pointer(&slicehdr))
	g.set(v, slice)
}

// Position creates a new vertex and sets the vertex position.
func (g *GeometryBuffer) Position(x, y, z float32) *GeometryBuffer {
	g.fillVertex()
	g.next()
	g.setf(VertexPosition, []float32{x, y, z})
	return g
}

// Color sets the vertex color.
func (gb *GeometryBuffer) Color(r, g, b, a uint8) *GeometryBuffer {
	gb.set(VertexColor, []uint8{r, g, b, a})
	return gb
}

func (gb *GeometryBuffer) Colorf(r, g, b, a float32) *GeometryBuffer {
	gb.set(VertexColor, []uint8{
		uint8(r * 255.0), uint8(g * 255.0), uint8(b * 255.0), uint8(a * 255.0),
	})
	return gb
}

// Normal sets the vertex normal.
func (g *GeometryBuffer) Normal(x, y, z float32) *GeometryBuffer {
	g.setf(VertexNormal, []float32{x, y, z})
	return g
}

// Texcoord sets the vertex texture coordinate.
func (g *GeometryBuffer) Texcoord(u, v float32) *GeometryBuffer {
	g.setf(VertexTexcoord, []float32{u, v})
	return g
}

// Indices appends new indices to the buffer that are relative to the maximum index in the buffer.
func (g *GeometryBuffer) Indices(idxs ...uint16) *GeometryBuffer {
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

func (g *GeometryBuffer) SetIndices(idxs ...uint16) {
	g.nextidx = 0
	g.idxs = make([]uint16, len(idxs))
	copy(g.idxs, idxs)
}

func (g *GeometryBuffer) VertexFormat() VertexFormat {
	return g.vf
}

// IndexCount returns the number of indices available.
func (g *GeometryBuffer) IndexCount() int {
	if g.idxs == nil {
		return g.VertexCount()
	}
	return len(g.idxs)
}

// CopyIndices copies the indices to dest. If IndexCount() does not
// match len(buf), an error is returned.
func (g *GeometryBuffer) CopyIndices(dest *IndexBuffer, usage Usage) error {
	// TODO: sanity check on len, as described in doc
	return dest.SetIndices(g.idxs, usage)
}

// VertexCount returns the number of vertices available.
func (g *GeometryBuffer) VertexCount() int {
	return len(g.verts) / g.stride
}

// CopyVertices copies the vertices to dest. If len(buf) does not equal
// VertexCount()*VertexFormat.Stride(), an error is returned.
func (g *GeometryBuffer) CopyVertices(dest *VertexBuffer, usage Usage) error {
	// TODO: sanity check on len, as described in doc
	g.fillVertex()
	return dest.SetVertices(g.verts, usage, g.VertexFormat())
}
