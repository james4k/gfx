package scenes

type Scene struct {
	Node
}

func New() *Scene {
	s := &Scene{}
	initObject(s)
	return s
}

type Camera struct {
	Node
}

type Mesh struct {
	Node
	geometry Geometry
	material Material
}
