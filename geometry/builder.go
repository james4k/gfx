package geometry

import (
	"j4k.co/gfx"
	"reflect"
	"unsafe"
)

type Builder struct {
	VertexBuilder
	IndexBuilder
}

func NewBuilder(vf gfx.VertexFormat) *Builder {
	return &Builder{
		VertexBuilder: VertexBuilder{
			vf:       vf,
			stride:   vf.Stride(),
			lastdata: make(map[gfx.VertexFormat]int, vf.Count()),
		},
	}
}

func (b *Builder) Clear() {
	b.VertexBuilder.Clear()
	b.IndexBuilder.Clear()
}

type VertexBuilder struct {
	vf       gfx.VertexFormat
	stride   int
	cur      int
	curvf    gfx.VertexFormat // data that's been set on the current vertex
	lastdata map[gfx.VertexFormat]int
	offsets  map[gfx.VertexFormat]int
	verts    []byte
}

func NewVertexBuilder(vf gfx.VertexFormat) *VertexBuilder {
	return &VertexBuilder{
		vf:       vf,
		stride:   vf.Stride(),
		lastdata: make(map[gfx.VertexFormat]int, vf.Count()),
	}
}

// Clear resets buffers to zero length.
func (b *VertexBuilder) Clear() {
	b.lastdata = make(map[gfx.VertexFormat]int, len(b.lastdata))
	b.cur = 0
	b.curvf = 0
	b.verts = b.verts[:0]
}

// TODO: add a test
func (b *VertexBuilder) offset(v gfx.VertexFormat) int {
	if b.vf&v == 0 {
		panic(gfx.ErrBadVertexFormat)
	}
	if b.offsets == nil {
		offs := 0
		b.offsets = map[gfx.VertexFormat]int{}
		for i := gfx.VertexFormat(1); i <= gfx.MaxVertexFormat; i <<= 1 {
			if b.vf&i != 0 {
				b.offsets[i] = offs
				offs += i.AttribBytes()
			}
		}
	}
	return b.offsets[v]
}

func (b *VertexBuilder) next() {
	if len(b.verts) != 0 {
		b.cur += b.stride
	}
	b.curvf = 0
	zeros := make([]uint8, b.stride)
	b.verts = append(b.verts, zeros...)
}

// fillVertex fills the rest of the vertex data using the last set data
// from a previous vertex
func (b *VertexBuilder) fillVertex() {
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
	for i, offs := range b.lastdata {
		if b.vf&i != 0 && b.curvf&i == 0 {
			data := b.verts[offs : offs+i.AttribBytes()]
			b.set(i, data)
		}
	}
}

func (b *VertexBuilder) set(v gfx.VertexFormat, data []uint8) {
	b.curvf |= v
	offs := b.cur + b.offset(v)
	b.lastdata[v] = offs
	copy(b.verts[offs:offs+len(data)], data)
}

func (b *VertexBuilder) setf(v gfx.VertexFormat, data []float32) {
	sz := len(data) * 4
	slicehdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&data[0])),
		Len:  sz,
		Cap:  sz,
	}
	slice := *(*[]uint8)(unsafe.Pointer(&slicehdr))
	b.set(v, slice)
}

// Position creates a new vertex and sets the vertex position.
func (b *VertexBuilder) Position(x, y, z float32) *VertexBuilder {
	b.fillVertex()
	b.next()
	b.setf(gfx.VertexPosition, []float32{x, y, z})
	return b
}

// Color sets the vertex color.
func (b *VertexBuilder) Color(red, green, blue, alpha uint8) *VertexBuilder {
	b.set(gfx.VertexColor, []uint8{red, green, blue, alpha})
	return b
}

func (b *VertexBuilder) Colorf(red, green, blue, alpha float32) *VertexBuilder {
	b.set(gfx.VertexColor, []uint8{
		uint8(red * 255.0), uint8(green * 255.0), uint8(blue * 255.0), uint8(alpha * 255.0),
	})
	return b
}

// Normal sets the vertex normal.
func (b *VertexBuilder) Normal(x, y, z float32) *VertexBuilder {
	b.setf(gfx.VertexNormal, []float32{x, y, z})
	return b
}

// Texcoord sets the vertex texture coordinate.
func (b *VertexBuilder) Texcoord(u, v float32) *VertexBuilder {
	b.setf(gfx.VertexTexcoord, []float32{u, v})
	return b
}

// VertexCount returns the number of vertices available.
func (b *VertexBuilder) VertexCount() int {
	return len(b.verts) / b.stride
}

// CopyVertices copies the vertices to dest. If len(buf) does not equal
// VertexCount()*VertexFormat.Stride(), an error is returned.
func (b *VertexBuilder) CopyVertices(dest *gfx.VertexBuffer, usage gfx.Usage) error {
	// TODO: sanity check on len, as described in doc
	b.fillVertex()
	if b.VertexFormat() != dest.Format() {
		return gfx.ErrBadVertexFormat
	}
	return dest.SetVertices(b.verts, usage)
}

func (b *VertexBuilder) VertexFormat() gfx.VertexFormat {
	return b.vf
}

type IndexBuilder struct {
	idxs    []uint16
	nextidx uint16
}

// Indices appends new indices to the buffer that are relative to the maximum index in the buffer.
func (b *IndexBuilder) Indices(idxs ...uint16) *IndexBuilder {
	// TODO: could really use a test
	newnext := b.nextidx
	for i, idx := range idxs {
		idx += b.nextidx
		if idx >= newnext {
			newnext = idx + 1
		}
		idxs[i] = idx
	}
	b.nextidx = newnext
	b.idxs = append(b.idxs, idxs...)
	return b
}

// SetIndices copies idxs into a new buffer.
func (b *IndexBuilder) SetIndices(idxs ...uint16) {
	b.nextidx = 0
	b.idxs = make([]uint16, len(idxs))
	copy(b.idxs, idxs)
}

// IndexCount returns the number of indices available.
func (b *IndexBuilder) IndexCount() int {
	return len(b.idxs)
}

// CopyIndices copies the indices to dest. If IndexCount() does not
// match len(buf), an error is returned.
func (b *IndexBuilder) CopyIndices(dest *gfx.IndexBuffer, usage gfx.Usage) error {
	// TODO: sanity check on len, as described in doc
	return dest.SetIndices(b.idxs, usage)
}

// Clear resets buffers to zero length.
func (b *IndexBuilder) Clear() {
	b.idxs = b.idxs[:0]
	b.nextidx = 0
}
