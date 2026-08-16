package perception

import "testing"

// TestMacroValueBeatsSingleClick: transforming a WHOLE class (macro) advances
// the hypothesis far more than a single click of it -- the leverage that
// unsticks a flat per-object climb. Here recoloring all stray color-3 objects to
// the majority 5 makes the grid nearly all-one-color.
func TestMacroValueBeatsSingleClick(t *testing.T) {
	// mostly color 5, with three stray color-3 objects (bg 0).
	grid := [][]int{
		{0, 5, 0, 3, 0, 5, 0, 3, 0, 5, 0, 3, 0},
	}
	tbl := NewAffordanceTable()
	// teach: clicking a color-3 object recolors that cell to 5.
	before := [][]int{{0, 3, 0}}
	after := [][]int{{0, 5, 0}}
	tbl.ObserveClick(before, after, 1, 0)
	if !tbl.Known(3) {
		t.Fatal("should have learned color-3's recolor->5 affordance")
	}

	macro, ok := tbl.MacroValue(grid, 3, hypAllOneColor)
	if !ok || !(macro > 0) {
		t.Fatalf("the class-3 macro should positively advance all-one-color, got %.4f ok=%v", macro, ok)
	}

	// a single-object counterfactual (one 3 -> 5) advances it much less.
	var oneThree Blob
	for _, b := range FindBlobs(grid, BackgroundColor(grid)) {
		if b.Color == 3 {
			oneThree = b
			break
		}
	}
	single, _ := tbl.LookaheadValue(grid, oneThree, nil, hypAllOneColor, 1)
	singleDelta := single - scopedScore(grid, hypAllOneColor)
	if !(macro > singleDelta) {
		t.Fatalf("macro (%.4f) should beat a single click's delta (%.4f)", macro, singleDelta)
	}
}
