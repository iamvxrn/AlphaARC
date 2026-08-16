package perception

import (
	"fmt"
	"sort"
)

// ObjectTracker gives objects STABLE identity across frames by CONTINUITY, not
// by exact pixel shape. The shape-hash identity (ObjectSignature) proved too
// brittle live: a body that deforms or flickers by one pixel got a brand-new
// hash every frame, so the graph treated each frame as new objects being born
// (log: coarse positions unchanged, yet shape ids churned; 81 clusters / ~2000
// edges by step 25). A real object is "the same body" if, frame to frame, it's
// the same color, near where it was, and roughly the same size -- tolerant to
// deformation. That is what a tracker matches on.
//
// Track ingests a frame and returns, per object, a stable identity token
// ("obj<id>-color<c>") plus, if it moved, a motion token ("obj<id>-up" etc.).
// Persistent ids survive deformation and motion; the graph and forward model
// finally get one node per body instead of one per pixel-shape.
type ObjectTracker struct {
	objs    []trackedObj
	nextID  int
	maxDist int // max centroid jump (in cells) still counted as the same body
}

type trackedObj struct {
	id, color  int
	cx, cy     int
	size       int
}

func NewObjectTracker() *ObjectTracker { return &ObjectTracker{nextID: 1, maxDist: 8} }

func (t *ObjectTracker) Track(grid [][]int) []string {
	blobs := FindBlobs(grid, BackgroundColor(grid))
	used := make([]bool, len(t.objs))
	next := make([]trackedObj, 0, len(blobs))
	tokens := make([]string, 0, len(blobs)*2)

	for _, b := range blobs {
		best, bestD := -1, t.maxDist*t.maxDist+1
		for j, o := range t.objs {
			if used[j] || o.color != b.Color {
				continue
			}
			dx, dy := o.cx-b.Centroid.X, o.cy-b.Centroid.Y
			if d := dx*dx + dy*dy; d < bestD {
				bestD, best = d, j
			}
		}
		var id int
		if best >= 0 {
			o := t.objs[best]
			id, used[best] = o.id, true
			// motion token if the body's centroid shifted
			if dx, dy := b.Centroid.X-o.cx, b.Centroid.Y-o.cy; dx != 0 || dy != 0 {
				tokens = append(tokens, fmt.Sprintf("obj%d-%s", id, moveDirection(dx, dy)))
			}
		} else {
			id = t.nextID
			t.nextID++
		}
		tokens = append(tokens, fmt.Sprintf("obj%d-color%d", id, b.Color))
		next = append(next, trackedObj{id: id, color: b.Color, cx: b.Centroid.X, cy: b.Centroid.Y, size: len(b.Cells)})
	}
	t.objs = next
	return tokens
}

// LabeledObjects returns the click candidates in the SAME vocabulary the graph
// and observation speak: one entry per object tracked in the most recent Track
// call, labeled "obj<id>-color<c>" (byte-identical to the identity token Track
// emits into the observation) and carrying that object's centroid. Ordered
// largest object first, matching RankedLabeledBlobs' convention.
//
// This is the Fix-3 reconnection. Previously the click candidates were cell-
// labeled ("color<n>-cell<c>-<r>") while the graph held only obj-id nodes, so
// winningBlobLabel matched nothing, every candidate read "not yet in graph",
// and the graph was out of click selection entirely. With candidate labels that
// match the graph's node labels, the winning category can be bound back to a
// concrete object centroid and the graph rejoins the decision. Call after Track.
func (t *ObjectTracker) LabeledObjects() []LabeledBlob {
	objs := make([]trackedObj, len(t.objs))
	copy(objs, t.objs)
	sort.SliceStable(objs, func(i, j int) bool { return objs[i].size > objs[j].size })
	out := make([]LabeledBlob, 0, len(objs))
	for _, o := range objs {
		out = append(out, LabeledBlob{
			Blob:  Blob{Color: o.color, Centroid: Point{X: o.cx, Y: o.cy}},
			Label: fmt.Sprintf("obj%d-color%d", o.id, o.color),
		})
	}
	return out
}

// TopologySignature is a frame's OBJECT-LEVEL fingerprint: the sorted multiset
// of tracked object identities present, ignoring their absolute positions. Two
// frames with the same signature differ only by object motion/deformation, not
// by which bodies exist -- so a pure "conveyor belt" tick (bodies sliding, none
// created/destroyed) is recognizably NOT progress. Call after Track.
func (t *ObjectTracker) TopologySignature() string {
	ids := make([]int, len(t.objs))
	for i, o := range t.objs {
		ids[i] = o.id
	}
	// simple order-independent hash of the id set
	sum := 0
	for _, id := range ids {
		sum += id*id + id
	}
	return fmt.Sprintf("topo-%d-%d", len(ids), sum)
}
