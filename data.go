package gfx

type Vec3 interface {
	Pointer() *[3]float32
}

type Mat3 interface {
	Pointer() *[9]float32
}

type Mat4 interface {
	Pointer() *[16]float32
}
