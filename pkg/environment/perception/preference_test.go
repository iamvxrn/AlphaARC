package perception

import "testing"

// gridColors places each color at an isolated cell -> one 1-cell blob each.
func gridColors(colors []int) [][]int {
	g := make([][]int, len(colors))
	for i := range g {
		g[i] = make([]int, len(colors)*2)
	}
	for i, c := range colors {
		g[i][i*2] = c
	}
	return g
}

// TestStructureScorePrefersConsolidation: a grid whose blobs share one color
// (ordered) must score higher than one where every blob is a different color
// (scattered) -- the source-of-meaning signal points at structure.
func TestStructureScorePrefersConsolidation(t *testing.T) {
	ordered := StructureScore(gridColors([]int{5, 5, 5, 5}))     // one group of 4 -> 1.0
	scattered := StructureScore(gridColors([]int{1, 2, 3, 4}))   // four groups of 1 -> 0.25
	if !(ordered > scattered) {
		t.Fatalf("FAIL: ordered grid did not score above scattered: %.3f vs %.3f", ordered, scattered)
	}
	if ordered != 1.0 {
		t.Fatalf("FAIL: fully-consolidated grid should score 1.0, got %.3f", ordered)
	}
	if StructureScore([][]int{{0, 0}, {0, 0}}) != 0 {
		t.Fatalf("FAIL: empty grid should score 0")
	}
}
