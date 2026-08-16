package perception

import (
	"fmt"
	"hash/fnv"
	"sort"
)

// ObjectSignature is the first Core-Knowledge primitive: OBJECTNESS. It gives a
// blob a position-INVARIANT identity -- its color plus its SHAPE (the set of
// its cells translated so the top-left is the origin, sorted). Two blobs of the
// same color and shape get the same signature no matter WHERE they are, so an
// object that moved one pixel is recognized as the SAME object, not a brand-new
// one.
//
// This is the fix for the fragmentation seen in a live run: act-5 slid objects
// up a row (color3-cell15-13 -> cell14-13 -> cell13-13), and because identity
// was baked into position (the cellX-Y label), the graph treated each shift as
// the death of one object and the birth of another (fresh nodes every step),
// forgetting the history of interacting with that body. Shape-based identity
// separates WHAT a thing is (stable) from WHERE it is (changes) -- the ground
// every later geometric/causal prior stands on.
//
// Limitation (honest): two DISTINCT objects with identical color+shape share a
// signature; telling them apart needs cross-frame tracking (proximity), the
// next primitive. Rotations/reflections give different signatures too -- a
// separate geometry prior, later.
func ObjectSignature(b Blob) string {
	if len(b.Cells) == 0 {
		return fmt.Sprintf("obj-color%d-empty", b.Color)
	}
	minX, minY := b.Cells[0].X, b.Cells[0].Y
	for _, c := range b.Cells {
		if c.X < minX {
			minX = c.X
		}
		if c.Y < minY {
			minY = c.Y
		}
	}
	offsets := make([]Point, len(b.Cells))
	for i, c := range b.Cells {
		offsets[i] = Point{X: c.X - minX, Y: c.Y - minY}
	}
	sort.Slice(offsets, func(i, j int) bool {
		if offsets[i].Y != offsets[j].Y {
			return offsets[i].Y < offsets[j].Y
		}
		return offsets[i].X < offsets[j].X
	})
	h := fnv.New32a()
	for _, o := range offsets {
		fmt.Fprintf(h, "%d,%d;", o.X, o.Y)
	}
	return fmt.Sprintf("obj-color%d-shape%08x", b.Color, h.Sum32())
}

// ObjectTokens returns the position-invariant identity signature of every
// object (blob) in the grid -- the stable "these bodies are present" tokens.
// Wired into the observation, they give the graph a PERSISTENT node per object
// that survives the object moving, unlike the position-baked color-cell labels
// that spawn a fresh node every pixel-shift.
func ObjectTokens(grid [][]int) []string {
	blobs := FindBlobs(grid, BackgroundColor(grid))
	out := make([]string, len(blobs))
	for i, b := range blobs {
		out[i] = ObjectSignature(b)
	}
	return out
}

// ObjectMotions is Core-Knowledge brick 2: MOTION. It matches each object in
// cur to the same-identity object in prev (nearest one if several share a
// shape) and emits a token for how it moved -- e.g. "obj-color3-shapeXXXX-up".
// This turns a pixel-by-pixel position change into an explicit, stable motion
// concept the forward model can learn a causal rule over ("act-5 -> that body
// moves up"), instead of re-deriving it from churning position labels every
// frame. Objects with no match in prev are "-new" (appeared); unmoved objects
// emit nothing.
func ObjectMotions(prev, cur [][]int) []string {
	prevBySig := make(map[string][]Point)
	for _, b := range FindBlobs(prev, BackgroundColor(prev)) {
		s := ObjectSignature(b)
		prevBySig[s] = append(prevBySig[s], b.Centroid)
	}
	var out []string
	for _, b := range FindBlobs(cur, BackgroundColor(cur)) {
		sig := ObjectSignature(b)
		cands := prevBySig[sig]
		if len(cands) == 0 {
			out = append(out, sig+"-new")
			continue
		}
		best, bestD := cands[0], dist2(b.Centroid, cands[0])
		for _, p := range cands[1:] {
			if d := dist2(b.Centroid, p); d < bestD {
				best, bestD = p, d
			}
		}
		dx, dy := b.Centroid.X-best.X, b.Centroid.Y-best.Y
		if dx == 0 && dy == 0 {
			continue
		}
		out = append(out, sig+"-"+moveDirection(dx, dy))
	}
	return out
}

func dist2(a, b Point) int { return (a.X-b.X)*(a.X-b.X) + (a.Y-b.Y)*(a.Y-b.Y) }

// direction names the dominant axis of a move (rows are y; up = smaller y).
func moveDirection(dx, dy int) string {
	if abs(dy) >= abs(dx) {
		if dy < 0 {
			return "up"
		}
		return "down"
	}
	if dx < 0 {
		return "left"
	}
	return "right"
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
