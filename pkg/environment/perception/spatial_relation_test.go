package perception

import (
	"strings"
	"testing"
)

// TestSpatialRelationTokens: the relational perception layer detects touching,
// containment (inside/outside), and alignment -- the Core-Knowledge relations
// the observation previously lacked.
func TestSpatialRelationTokens(t *testing.T) {
	// A ring of 3 with a lone 7 enclosed inside it, and a separate 5 aligned in
	// the same column as part of the ring.
	grid := [][]int{
		{0, 0, 0, 0, 0, 0},
		{0, 3, 3, 3, 0, 0},
		{0, 3, 7, 3, 0, 0},
		{0, 3, 3, 3, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}
	toks := strings.Join(SpatialRelationTokens(grid), " ")
	// the 7 is walled off by the 3-ring -> inside.
	if !strings.Contains(toks, "inside") {
		t.Fatalf("expected an inside/containment token for the enclosed 7, got %q", toks)
	}

	// touching: two adjacent different-color objects.
	touching := [][]int{{0, 3, 7, 0}}
	if !strings.Contains(strings.Join(SpatialRelationTokens(touching), " "), "touch") {
		t.Fatalf("adjacent different objects should produce a touch token")
	}
	// not touching: a gap between them.
	apart := [][]int{{3, 0, 0, 7}}
	if strings.Contains(strings.Join(SpatialRelationTokens(apart), " "), "touch") {
		t.Fatalf("separated objects should NOT touch")
	}

	// alignment: two objects sharing a column.
	aligned := [][]int{
		{0, 3, 0},
		{0, 0, 0},
		{0, 7, 0},
	}
	if !strings.Contains(strings.Join(SpatialRelationTokens(aligned), " "), "aligned") {
		t.Fatalf("objects sharing a column should produce an aligned token")
	}
}
