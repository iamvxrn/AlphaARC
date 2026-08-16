package perception

import (
	"strings"
	"testing"
)

// TestObjectMotionsDetectsDirection: an object that slid up must produce an
// "...-up" motion token keyed on its (stable) identity.
func TestObjectMotionsDetectsDirection(t *testing.T) {
	g := func(r int) [][]int {
		m := make([][]int, 10)
		for i := range m {
			m[i] = make([]int, 10)
		}
		m[r][4], m[r][5] = 7, 7 // a color-7 pair at row r
		return m
	}
	prev := g(6)
	cur := g(3) // same object moved up 3 rows

	sig := ObjectSignature(Blob{Color: 7, Cells: []Point{{4, 3}, {5, 3}}})
	motions := ObjectMotions(prev, cur)
	joined := strings.Join(motions, " ")
	if !strings.Contains(joined, sig+"-up") {
		t.Fatalf("FAIL: expected %q, got motions %v", sig+"-up", motions)
	}
	// No motion when nothing moves.
	if m := ObjectMotions(prev, prev); len(m) != 0 {
		t.Fatalf("FAIL: expected no motion for identical frames, got %v", m)
	}
}
