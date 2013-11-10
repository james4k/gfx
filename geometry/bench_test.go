package geometry_test

import (
	"j4k.co/gfx"
	"j4k.co/gfx/geometry"
	"testing"
)

const builderQuads = 40 * 40

func BenchmarkBuilderTinyVerts(b *testing.B) {
	bdr := geometry.NewBuilder(gfx.VertexPosition)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for q := 0; q < builderQuads; q++ {
			bdr.Position(0, 0, 0)
			bdr.Position(1, 0, 0)
			bdr.Position(1, 1, 0)
			bdr.Position(0, 1, 0)
			bdr.Indices(0, 1, 2, 2, 0, 3)
		}
	}
}

func BenchmarkBuilderFatVerts(b *testing.B) {
	bdr := geometry.NewBuilder(gfx.VertexPosition | gfx.VertexColor |
		gfx.VertexTexcoord)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for q := 0; q < builderQuads; q++ {
			bdr.Position(0, 0, 0).Color(128, 0, 255, 255).Texcoord(0, 0)
			bdr.Position(1, 0, 0)
			bdr.Position(1, 1, 0)
			bdr.Position(0, 1, 0)
			bdr.Indices(0, 1, 2, 2, 0, 3)
		}
	}
}
