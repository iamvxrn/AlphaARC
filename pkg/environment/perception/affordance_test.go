package perception

import "testing"

// TestAffordanceLearnsRecolorAndGeneralizes: observing that clicking a color-3
// object recolored it to 5 lets the table (a) know color-3's effect, (b)
// generalize to a DIFFERENT color-3 object (1-shot over the class), and (c)
// predict a realistic next state used by the honest counterfactual.
func TestAffordanceLearnsRecolorAndGeneralizes(t *testing.T) {
	before := [][]int{
		{0, 3, 0, 0, 3, 0},
		{0, 3, 0, 0, 3, 0},
	}
	// clicking the left color-3 object (cells at col1) recolored it to 5.
	after := [][]int{
		{0, 5, 0, 0, 3, 0},
		{0, 5, 0, 0, 3, 0},
	}
	tbl := NewAffordanceTable()
	if tbl.Known(3) {
		t.Fatal("nothing learned yet, color 3 should be unknown")
	}
	tbl.ObserveClick(before, after, 1, 0) // clicked (x=1,y=0)

	op, ok := tbl.Operator(3)
	if !ok || op.Kind != OpRecolor || op.Color != 5 {
		t.Fatalf("expected a learned recolor->5 for color 3, got %+v ok=%v", op, ok)
	}

	// Generalize: predict clicking the OTHER color-3 object (right, col4) recolors
	// it to 5 as well -- a realistic next state, not a phantom.
	var rightObj Blob
	for _, b := range FindBlobs(after, BackgroundColor(after)) {
		if b.Color == 3 {
			rightObj = b
		}
	}
	pred, known := tbl.Predict(after, rightObj)
	if !known {
		t.Fatal("color 3 affordance should be known after one observation")
	}
	if pred[0][4] != 5 || pred[1][4] != 5 {
		t.Fatalf("predicted state should recolor the right color-3 object to 5, got rows %v/%v", pred[0], pred[1])
	}
}

// TestAffordanceUnknownStaysUnknown: an object class never clicked has no
// affordance, so LearnedPragmaticValue reports "unknown" (the caller treats it
// as epistemically worth probing, not as zero value).
func TestAffordanceUnknownStaysUnknown(t *testing.T) {
	grid := [][]int{{0, 7, 0}, {0, 7, 0}}
	tbl := NewAffordanceTable()
	var obj Blob
	for _, b := range FindBlobs(grid, BackgroundColor(grid)) {
		obj = b
	}
	if _, known := LearnedPragmaticValue(grid, obj, hypAllOneColor, tbl); known {
		t.Fatal("an unclicked class must report unknown, driving epistemic probing")
	}
}

// TestAffordanceLearnsDespawn: clicking an object that vanishes is learned as
// despawn, and predicted as clearing the object's cells to background.
func TestAffordanceLearnsDespawn(t *testing.T) {
	before := [][]int{{0, 4, 4, 0}}
	after := [][]int{{0, 0, 0, 0}}
	tbl := NewAffordanceTable()
	tbl.ObserveClick(before, after, 1, 0)
	op, ok := tbl.Operator(4)
	if !ok || op.Kind != OpDespawn {
		t.Fatalf("expected learned despawn for color 4, got %+v ok=%v", op, ok)
	}
}
