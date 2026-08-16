package perception

import "testing"

// TestRelationalAffordanceGeneralizes: a NON-LOCAL effect (clicking a green
// object flips a cell elsewhere) is learned relative to the trigger and
// generalizes to another green object at a different position -- the relational
// dynamics a 1-step sat is otherwise blind to.
func TestRelationalAffordanceGeneralizes(t *testing.T) {
	before := [][]int{
		{0, 3, 0, 0, 0, 0}, // green #1 at (1,0)
		{0, 0, 0, 0, 7, 0}, // remote target color7 at (4,1)
		{0, 0, 0, 0, 0, 0},
		{0, 3, 0, 0, 0, 0}, // green #2 at (1,3)
		{0, 0, 0, 0, 0, 0},
	}
	after := [][]int{ // clicked green #1; the REMOTE cell (4,1) turned 7->5
		{0, 3, 0, 0, 0, 0},
		{0, 0, 0, 0, 5, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 3, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
	}
	tbl := NewAffordanceTable()
	tbl.ObserveClick(before, after, 1, 0)
	if !tbl.Known(3) {
		t.Fatal("clicking color 3 had a visible effect; it should be learned")
	}

	// Generalize to green #2 (centroid (1,3)): the relative offset (+3,+1) lands
	// at (4,4), which should be recolored to 5.
	var green2 Blob
	for _, b := range FindBlobs(before, BackgroundColor(before)) {
		if b.Color == 3 && b.Centroid.Y == 3 {
			green2 = b
		}
	}
	pred, ok := tbl.Predict(before, green2)
	if !ok {
		t.Fatal("color 3 affordance should be known")
	}
	if pred[4][4] != 5 {
		t.Fatalf("relational effect should recolor (4,4) to 5 for green #2, got %d (row %v)", pred[4][4], pred[4])
	}
}

// TestLookaheadStepsBeyondOneStep: the 2-step rollout can reach a higher
// satisfaction than the first move alone yields -- the mechanism that steps over
// Δsat<=0 valleys a greedy 1-step agent is trapped in.
func TestLookaheadStepsBeyondOneStep(t *testing.T) {
	// score = share of color-5 cells.
	score := func(g [][]int) float64 {
		n, tot := 0, 0
		for _, row := range g {
			for _, c := range row {
				tot++
				if c == 5 {
					n++
				}
			}
		}
		if tot == 0 {
			return 0
		}
		return float64(n) / float64(tot)
	}
	grid := [][]int{{0, 3, 0, 4, 0}} // objA color3 at (1,0), objB color4 at (3,0)
	tbl := NewAffordanceTable()
	// clicking color3 recolors its own cell to 4 (no color5 yet -> flat sat)
	tbl.ObserveClick([][]int{{0, 3, 0, 4, 0}}, [][]int{{0, 4, 0, 4, 0}}, 1, 0)
	// clicking color4 recolors its own cell to 5
	tbl.ObserveClick([][]int{{0, 3, 0, 4, 0}}, [][]int{{0, 3, 0, 5, 0}}, 3, 0)

	blobs := FindBlobs(grid, BackgroundColor(grid))
	var objA Blob
	for _, b := range blobs {
		if b.Color == 3 {
			objA = b
		}
	}
	v1, ok1 := tbl.LookaheadValue(grid, objA, blobs, score, 1)
	v2, ok2 := tbl.LookaheadValue(grid, objA, blobs, score, 2)
	if !ok1 || !ok2 {
		t.Fatal("objA affordance is known, lookahead should succeed")
	}
	if !(v2 > v1) {
		t.Fatalf("2-step rollout should reach a higher sat than 1-step alone: v1=%.3f v2=%.3f", v1, v2)
	}
}

// TestLookaheadUnknownIsUnknown: an object whose class was never clicked cannot
// be rolled out -- reported unknown so the caller probes it (motor babbling).
func TestLookaheadUnknownIsUnknown(t *testing.T) {
	grid := [][]int{{0, 7, 0}}
	tbl := NewAffordanceTable()
	var obj Blob
	for _, b := range FindBlobs(grid, BackgroundColor(grid)) {
		obj = b
	}
	if _, ok := tbl.LookaheadValue(grid, obj, []Blob{obj}, hypAllOneColor, 2); ok {
		t.Fatal("an unclicked class must be unknown")
	}
}
