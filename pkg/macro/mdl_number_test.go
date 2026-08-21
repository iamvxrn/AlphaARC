package macro

import "testing"

// Numerosity basics ----------------------------------------------------------

func TestCount_BlankAndLoneObjectSaveNothing(t *testing.T) {
	blank := [][]int{{0, 0, 0}, {0, 0, 0}}
	if p := NumerosityPreference(blank, 0); p != 0 {
		t.Fatalf("blank must save 0, got %d", p)
	}
	lone := [][]int{{1, 1, 1, 0, 0}, {0, 0, 0, 0, 0}}
	if p := NumerosityPreference(lone, 0); p != 0 {
		t.Fatalf("a single object saves 0 (nothing to share), got %d", p)
	}
}

func TestCount_RepeatedObjectsSave(t *testing.T) {
	// Two identical horizontal trominoes (3 cells each) => (2-1)*3 = 3.
	g := [][]int{
		{1, 1, 1, 0, 0},
		{0, 0, 0, 0, 0},
		{0, 0, 1, 1, 1},
	}
	if p := NumerosityPreference(g, 0); p != 3 {
		t.Fatalf("two identical trominoes should save 3, got %d", p)
	}
}

// The degenerate cheat is shut: scattered single-cell dots earn nothing.
func TestCount_ScatteredDotsEarnNothing(t *testing.T) {
	dots := [][]int{
		{1, 0, 0, 0, 0},
		{0, 0, 1, 0, 0},
		{0, 0, 0, 0, 1},
	}
	if p := NumerosityPreference(dots, 0); p != 0 {
		t.Fatalf("scattered single-cell dots must earn 0 (not counting), got %d", p)
	}
}

// Gradient: making one of two identical objects different destroys the shared
// description and drops the saving.
func TestCount_GradientBreaks(t *testing.T) {
	g := [][]int{
		{1, 1, 1, 0, 0},
		{0, 0, 0, 0, 0},
		{0, 0, 1, 1, 1},
	}
	base := NumerosityPreference(g, 0)
	broken := cloneGrid(g)
	broken[2][4] = 2 // splits the second tromino, no identical pair remains
	if got := NumerosityPreference(broken, 0); got >= base {
		t.Fatalf("breaking object identity must drop the saving: base=%d got=%d", base, got)
	}
}

// THE cross-domain emergence proof -------------------------------------------
//
// A grid of repeated identical objects that is NEITHER globally symmetric NOR
// periodic. Count (number domain) must out-save Reflect and Translate (geometry
// domain), and BestPrimitive must select it — proving the drive is not "a
// geometry scorer" but a general bits-saved currency spanning Core-Knowledge
// domains.
func TestEmergence_SelectsCountOnRepeatedObjects(t *testing.T) {
	g := [][]int{
		{1, 1, 1, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1, 1, 1, 0},
		{0, 0, 0, 0, 0, 0, 0, 0},
		{0, 1, 1, 1, 0, 0, 0, 0},
	}
	r := SymmetryPreference(g, 0)
	tr := TranslatePreference(g, 0)
	n := NumerosityPreference(g, 0)
	t.Logf("savings: Reflect=%d Translate=%d Count=%d", r, tr, n)

	if n <= r || n <= tr {
		t.Fatalf("Count must out-save geometry here: R=%d T=%d Count=%d", r, tr, n)
	}
	prim, save := BestPrimitive(g, 0)
	if prim.Name != "Count" {
		t.Fatalf("repeated-objects grid should select Count, got %s (save=%d)", prim.Name, save)
	}
}
