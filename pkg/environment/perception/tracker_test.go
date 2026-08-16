package perception

import (
	"strings"
	"testing"
)

// grid5 places a color-c object of the given cells.
func gridWith(cells []Point, color int) [][]int {
	g := make([][]int, 24)
	for i := range g {
		g[i] = make([]int, 24)
	}
	for _, p := range cells {
		g[p.Y][p.X] = color
	}
	return g
}

// TestObjectTrackerKeepsIdentityThroughDeformation is the fix for the brittle
// shape hash: a body that DEFORMS in place (its exact shape changes) but stays
// the same color near the same spot must keep ONE stable id -- no "-new".
func TestObjectTrackerKeepsIdentityThroughDeformation(t *testing.T) {
	tr := NewObjectTracker()
	// frame 1: an L
	tok1 := tr.Track(gridWith([]Point{{5, 5}, {5, 6}, {6, 6}}, 3))
	// frame 2: same body, deformed (now a line) at roughly the same place
	tok2 := tr.Track(gridWith([]Point{{5, 5}, {6, 5}, {7, 5}}, 3))

	id1 := idOf(tok1)
	id2 := idOf(tok2)
	if id1 == "" || id2 == "" {
		t.Fatalf("FAIL: no identity token: %v / %v", tok1, tok2)
	}
	if id1 != id2 {
		t.Fatalf("FAIL: a deforming body lost its identity: %q -> %q", id1, id2)
	}

	// A genuinely new, far-away object of another color gets a NEW id.
	tok3 := tr.Track(gridWith([]Point{{5, 5}, {6, 5}, {7, 5}, {20, 20}}, 3))
	if !hasTwoDistinctIds(tok3) {
		t.Fatalf("FAIL: a distinct new body did not get its own id: %v", tok3)
	}
}

func idOf(tokens []string) string {
	for _, t := range tokens {
		if strings.Contains(t, "-color") {
			return strings.Split(t, "-")[0]
		}
	}
	return ""
}
func hasTwoDistinctIds(tokens []string) bool {
	seen := map[string]bool{}
	for _, t := range tokens {
		if strings.Contains(t, "-color") {
			seen[strings.Split(t, "-")[0]] = true
		}
	}
	return len(seen) >= 2
}
