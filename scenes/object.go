package scenes

type NodeID uint32

type Object interface {
	node() *Node
	ID() NodeID
	Parent() Object
	Child(int) Object
	NumChildren() int
	SetParent(Object)
	AddChild(Object)
	Remove()
}

func initObject(o Object) {
	// TODO: verify this is more than a *Node
	n := o.(*Node)
	n.outer = o
}

type Node struct {
	id       NodeID
	outer    Object
	parent   Object
	children []Object
}

func (o *Node) node() *Node {
	return o
}

func (o *Node) ID() NodeID {
	return o.id
}

func (o *Node) Parent() Object {
	return o.parent
}

func (o *Node) Child(i int) Object {
	return o.children[i]
}

func (o *Node) NumChildren() int {
	return len(o.children)
}

func (o *Node) SetParent(p Object) {
	if p == nil {
		o.Remove()
	} else {
		p.AddChild(o.outer)
	}
}

func (o *Node) AddChild(c Object) {
	c.Remove()
	c.node().parent = o.outer
	o.children = append(o.children, c)
}

func (o *Node) Remove() {
	if o.parent == nil {
		return
	}
	p := o.parent.node()
	for i, c := range o.children {
		if o.id == c.ID() {
			p.children = append(p.children[:i], p.children[i+1:]...)
			return
		}
	}
}

func (o *Node) CreateCamera() *Camera {
	cam := &Camera{}
	o.AddChild(cam)
	return cam
}

func (o *Node) CreateMesh(geom Geometry, mat Material) *Mesh {
	mesh := &Mesh{
		geometry: geom,
		material: mat,
	}
	o.AddChild(mesh)
	return mesh
}
