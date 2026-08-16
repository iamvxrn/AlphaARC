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

// TestHypothesisTesterRotatesAndWarmStarts: the tester rotates off a stalled
// hypothesis, and a WarmStart brings the winning hypothesis to the head while
// KEEPING rotation open (each level may differ) and counting the win.
func TestHypothesisTesterRotatesAndWarmStarts(t *testing.T) {
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

	// WarmStart: the current (won) hypothesis goes to the head, wins increments,
	// and it gets a fresh patience window -- but rotation is NOT permanently locked.
	won := tr.Current().Name
	tr.WarmStart()
	if tr.Current().Name != won {
		t.Fatalf("warm-start should keep the won hypothesis %q at the head, got %q", won, tr.Current().Name)
	}
	if tr.Wins() != 1 {
		t.Fatalf("Wins() should be 1 after one WarmStart, got %d", tr.Wins())
	}
	// still rotates if the warm-started hypothesis then stalls on the new level.
	rotatedAgain := false
	for i := 0; i < hypRotatePatience; i++ {
		if tr.Observe(flat) {
			rotatedAgain = true
		}
	}
	if !rotatedAgain {
		t.Fatal("warm-started hypothesis must still rotate when it stalls (rotation stays open)")
	}
}

// TestPragmaticValueRewardsGoalAdvancingClicks: the counterfactual lookahead
// gives a POSITIVE value to acting on an object whose recolor/removal advances
// the current hypothesis, and ~0 when the goal is already met (nothing helps).
func TestPragmaticValueRewardsGoalAdvancingClicks(t *testing.T) {
	// background 0; foreground is mostly color 3 with one stray color 7. Fixing
	// the stray advances all-one-color.
	grid := [][]int{
		{0, 3, 3, 0, 7, 0},
		{0, 3, 3, 0, 0, 0},
	}
	var stray Blob
	for _, b := range FindBlobs(grid, BackgroundColor(grid)) {
		if b.Color == 7 {
			stray = b
		}
	}
	if v := PragmaticValue(grid, stray, hypAllOneColor); !(v > 0) {
		t.Fatalf("acting on the stray off-color object should have positive pragmatic value, got %.4f", v)
	}

	// An already-uniform foreground: no mutation can improve all-one-color, so
	// every object's pragmatic value is ~0.
	uniform := [][]int{
		{0, 3, 3, 0},
		{0, 3, 3, 0},
	}
	for _, b := range FindBlobs(uniform, BackgroundColor(uniform)) {
		if v := PragmaticValue(uniform, b, hypAllOneColor); v != 0 {
			t.Fatalf("a uniform grid should give 0 pragmatic value, got %.4f", v)
		}
	}
}

// TestTopologyInvariants: connectivity, enclosure, and gravity each peak when
// their goal is met and score lower when violated -- new Core-Knowledge gradients.
func TestTopologyInvariants(t *testing.T) {
	// connectivity: one joined bar vs two separate pieces (bg 0 kept majority).
	joined := [][]int{{0, 0, 3, 3, 3, 0, 0}}
	split := [][]int{{0, 3, 0, 0, 0, 3, 0}}
	if a, b := hypConnectivity(joined), hypConnectivity(split); !(a > b) {
		t.Fatalf("connectivity should score the joined body higher: %.3f vs %.3f", a, b)
	}
	if got := hypConnectivity(joined); got != 1.0 {
		t.Fatalf("a single connected body should score 1.0, got %.3f", got)
	}

	// enclosure: a ring around an interior bg cell vs an open shape (embedded in
	// a bg field so 0 stays the background).
	ring := [][]int{
		{0, 0, 0, 0, 0},
		{0, 3, 3, 3, 0},
		{0, 3, 0, 3, 0},
		{0, 3, 3, 3, 0},
		{0, 0, 0, 0, 0},
	}
	open := [][]int{
		{0, 0, 0, 0, 0},
		{0, 3, 3, 3, 0},
		{0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0},
	}
	if a, b := hypEnclosure(ring), hypEnclosure(open); !(a > b) {
		t.Fatalf("enclosure should score the ring higher: %.3f vs %.3f", a, b)
	}

	// gravity: settled at the bottom vs floating at the top.
	settled := [][]int{
		{0, 0, 0},
		{0, 0, 0},
		{3, 3, 3},
	}
	floating := [][]int{
		{3, 3, 3},
		{0, 0, 0},
		{0, 0, 0},
	}
	if a, b := hypGravity(settled), hypGravity(floating); !(a > b) {
		t.Fatalf("gravity should score the settled body higher: %.3f vs %.3f", a, b)
	}
}

// TestCompressionRewardsRegularity: compression-as-goal scores a regular
// (uniform/repetitive) grid far above a scrambled one -- a general, non-
// enumerative regularity gradient the climb machinery can pursue.
func TestCompressionRewardsRegularity(t *testing.T) {
	// uniform region: maximally compressible.
	uniform := make([][]int, 10)
	for r := range uniform {
		uniform[r] = make([]int, 10)
		for c := range uniform[r] {
			uniform[r][c] = 5
		}
	}
	// scrambled: a deterministic high-entropy fill (many colors, no runs).
	scrambled := make([][]int, 10)
	for r := range scrambled {
		scrambled[r] = make([]int, 10)
		for c := range scrambled[r] {
			scrambled[r][c] = (r*7 + c*3 + r*c) % 10
		}
	}
	u, s := hypCompression(uniform), hypCompression(scrambled)
	if !(u > s) {
		t.Fatalf("compression should score the regular grid higher: uniform=%.3f scrambled=%.3f", u, s)
	}
	if !(u > 0.7) {
		t.Fatalf("a uniform grid should be highly compressible (>0.7), got %.3f", u)
	}

	// a half-then-mirror (repetitive) grid beats the scramble too.
	striped := make([][]int, 10)
	for r := range striped {
		striped[r] = make([]int, 10)
		for c := range striped[r] {
			striped[r][c] = c % 2 * 3 // vertical stripes -> compresses well column-major
		}
	}
	if hypCompression(striped) <= s {
		t.Fatalf("a striped (repetitive) grid should beat the scramble: striped=%.3f scrambled=%.3f", hypCompression(striped), s)
	}
}
