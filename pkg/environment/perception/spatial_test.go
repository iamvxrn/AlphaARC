package perception

import (
	"strings"
	"testing"
)

// TestSalientTargetIsTheRareBlob: with three same-color bodies and one lone
// distinctive body, the target is the lone one -- the "key" guess.
func TestSalientTargetIsTheRareBlob(t *testing.T) {
	// 0 = background. color 3 appears three times (left), color 7 once (right).
	grid := [][]int{
		{3, 0, 0, 0, 0, 0, 7},
		{0, 0, 3, 0, 0, 0, 0},
		{0, 0, 0, 0, 3, 0, 0},
	}
	p, ok := SalientTargetCentroid(grid)
	if !ok {
		t.Fatal("expected a target")
	}
	// the lone color-7 blob is at column 6, row 0 -- far right, not among the 3s
	if p.X < 5 {
		t.Fatalf("target should be the rare right-side blob, got %+v", p)
	}
}

// TestApproachPreferenceRisesAsBodiesConverge: moving a body toward the salient
// target raises ApproachPreference -- the composer's gradient points the right
// way. This is the whole reason it can steer BestAction toward the key.
func TestApproachPreferenceRisesAsBodiesConverge(t *testing.T) {
	// target is the lone color-7 blob at the right edge; a color-3 mover starts
	// far left, then one cell closer.
	far := [][]int{
		{3, 0, 0, 0, 0, 0, 7},
	}
	near := [][]int{
		{0, 0, 0, 0, 0, 3, 7},
	}
	pFar := ApproachPreference(far)
	pNear := ApproachPreference(near)
	if !(pNear > pFar) {
		t.Fatalf("approach preference should rise as the mover nears the target: far=%.4f near=%.4f", pFar, pNear)
	}
	if pFar < 0 || pNear > spatialApproachWeight {
		t.Fatalf("approach preference out of [0,%.3f]: far=%.4f near=%.4f", spatialApproachWeight, pFar, pNear)
	}
}

// TestTargetDistanceNeedsTwoBodies: a single body (or none) has no "distance to
// something to bring together", so the composer stays silent rather than
// inventing a gradient from nothing.
func TestTargetDistanceNeedsTwoBodies(t *testing.T) {
	if _, ok := TargetDistance([][]int{{0, 0, 5, 0}}); ok {
		t.Fatal("one body should yield no target distance")
	}
	if p := ApproachPreference([][]int{{0, 0, 5, 0}}); p != 0 {
		t.Fatalf("one body should yield zero approach preference, got %.4f", p)
	}
}

// TestNumericTokensCarryMagnitude: the tokens encode object count and a
// quantized target-distance ring -- a coarse but real sense of number, distinct
// from the categorical identity tokens.
func TestNumericTokensCarryMagnitude(t *testing.T) {
	grid := [][]int{
		{3, 0, 0, 0, 0, 0, 7},
		{0, 0, 3, 0, 0, 0, 0},
	}
	toks := strings.Join(NumericTokens(grid), " ")
	if !strings.Contains(toks, "nobj") {
		t.Fatalf("expected an object-count token, got %q", toks)
	}
	if !strings.Contains(toks, "tdist") {
		t.Fatalf("expected a target-distance token, got %q", toks)
	}
}
