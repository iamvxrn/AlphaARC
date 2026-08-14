package perception

import (
	"strings"
	"testing"
)

// gridFromColors builds a small grid placing each (color) at a distinct,
// isolated cell so every entry becomes its own 1-cell blob.
func gridFromColors(colors []int) [][]int {
	// One blob per color entry, each on its own row, well separated.
	grid := make([][]int, len(colors))
	for i := range grid {
		grid[i] = make([]int, len(colors)*2)
	}
	for i, c := range colors {
		grid[i][i*2] = c
	}
	return grid
}

// TestRelationalTokensAreSurfaceInvariant is the property that makes Stage-6
// transfer even possible: two grids with the SAME color-multiplicity structure
// but DIFFERENT actual colors must emit identical relational tokens.
func TestRelationalTokensAreSurfaceInvariant(t *testing.T) {
	// Structure: one color appears 3x, another 2x.
	a := gridFromColors([]int{5, 5, 5, 8, 8})
	b := gridFromColors([]int{1, 1, 1, 2, 2}) // same structure, different colors

	ta := strings.Join(RelationalTokens(a), " ")
	tb := strings.Join(RelationalTokens(b), " ")
	if ta != tb {
		t.Fatalf("FAIL: surface-different but structurally-identical grids gave different relational tokens: %q vs %q", ta, tb)
	}
	if ta != "cmult3 cmult2" {
		t.Fatalf("FAIL: expected the color-multiplicity signature %q, got %q", "cmult3 cmult2", ta)
	}
}

// TestRelationalTokensDistinguishDifferentStructure confirms the tokens aren't
// trivially constant: a genuinely different structure yields different tokens.
func TestRelationalTokensDistinguishDifferentStructure(t *testing.T) {
	threeAndTwo := strings.Join(RelationalTokens(gridFromColors([]int{5, 5, 5, 8, 8})), " ")
	twoAndTwo := strings.Join(RelationalTokens(gridFromColors([]int{5, 5, 8, 8})), " ")
	if threeAndTwo == twoAndTwo {
		t.Fatalf("FAIL: structurally different grids collapsed to the same tokens (%q) -- signature not discriminative", threeAndTwo)
	}
}
