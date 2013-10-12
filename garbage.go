package gfx

import (
	"github.com/go-gl/gl"
	"sync"
)

type garbage struct {
	// Contention is probably non-existent with the current GC, but who knows
	// in the future.
	sync.Mutex
	buffers []gl.Buffer
}

// TODO: this needs to be thread-local!! might have to do this shit in C, or at
// least a thread-local allocation.
var trashbin garbage

func (g *garbage) addBuffer(b gl.Buffer) {
	g.Lock()
	g.buffers = append(g.buffers, b)
	g.Unlock()
}

func (g *garbage) release() {
	g.Lock()
	defer g.Unlock()

	if len(g.buffers) > 0 {
		gl.DeleteBuffers(g.buffers)
		g.buffers = nil
	}
}

// releaseGarbage is called at certain checkpoints to release GPU resources
// after their references have been GCed. This is needed to make the GL calls
// on the correct thread.
func releaseGarbage() {
	trashbin.release()
}

/*
// Release cleans up all GPU resources that are no longer referenced. This
// works via garbage collection so any global references should be niled
// accordingly.
func Release() {
	runtime.GC()
	releaseGarbage()
}
*/
