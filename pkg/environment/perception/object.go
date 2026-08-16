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
