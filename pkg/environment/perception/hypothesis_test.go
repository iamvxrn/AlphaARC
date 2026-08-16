package perception

import "testing"

// TestHypothesisScorersMeasureTheirGoal: each scorer peaks when its goal is met
// and is lower when violated -- the gradient the pragmatic loop climbs.
func TestHypothesisScorersMeasureTheirGoal(t *testing.T) {
	oneColor := [][]int{{0, 3, 3, 0}, {0, 3, 3, 0}}   // all foreground color 3
	twoColor := [][]int{{0, 3, 7, 0}, {0, 3, 7, 0}}   // two foreground colors
	if s1, s2 := hypAllOneColor(oneColor), hypAllOneColor(twoColor); !(s1 > s2) {
		t.Fatalf("all-one-color should score the uniform grid higher: %.3f vs %.3f", s1, s2)
	}
	if got := hypAllOneColor(oneColor); got != 1.0 {
		t.Fatalf("a fully-uniform foreground should score 1.0, got %.3f", got)
	}

	symH := [][]int{{5, 0, 0, 5}, {0, 7, 7, 0}}       // left-right mirror
	asymH := [][]int{{5, 0, 0, 0}, {0, 7, 0, 0}}      // not mirrored
	if s1, s2 := hypHorizontalSymmetry(symH), hypHorizontalSymmetry(asymH); !(s1 > s2) {
		t.Fatalf("horizontal-symmetry should score the mirrored grid higher: %.3f vs %.3f", s1, s2)
	}

	halvesEq := [][]int{{3, 0, 3, 0}, {0, 5, 0, 5}}   // left half == right half
	halvesNe := [][]int{{3, 0, 0, 0}, {0, 5, 0, 0}}
	if s1, s2 := hypHalvesMatch(halvesEq), hypHalvesMatch(halvesNe); !(s1 > s2) {
		t.Fatalf("halves-match should score matching halves higher: %.3f vs %.3f", s1, s2)
	}
}

// TestHypothesisTesterRotatesAndConfirms: the tester rotates off a stalled
// hypothesis, cycles through all candidates, and once confirmed stops rotating.
func TestHypothesisTesterRotatesAndConfirms(t *testing.T) {
	tr := NewHypothesisTester()
	first := tr.Current().Name
	flat := [][]int{{0, 0}, {0, 0}} // scores 0 on every hypothesis -> always stale

	// patience steps with no improvement should rotate exactly once.
	rotated := false
	for i := 0; i < hypRotatePatience; i++ {
		if tr.Observe(flat) {
			rotated = true
		}
	}
	if !rotated {
		t.Fatal("expected a rotation after the patience window on a stalled hypothesis")
	}
	if tr.Current().Name == first {
		t.Fatalf("hypothesis should have changed after rotation, still %q", first)
	}

	// Refute keeps moving; Confirm freezes.
	tr.Refute()
	locked := tr.Current().Name
	tr.Confirm()
	tr.Refute() // must be a no-op now
	for i := 0; i < hypRotatePatience+2; i++ {
		tr.Observe(flat) // must not rotate once confirmed
	}
	if tr.Current().Name != locked {
		t.Fatalf("a confirmed hypothesis must not rotate: was %q, now %q", locked, tr.Current().Name)
	}
	if !tr.Confirmed() {
		t.Fatal("Confirmed() should be true after Confirm()")
	}
}
