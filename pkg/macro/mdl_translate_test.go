package macro

import "testing"

// Translate basics -----------------------------------------------------------

// Blank grid saves nothing under Repeat either.
func TestTranslate_BlankSavesNothing(t *testing.T) {
	blank := [][]int{{0, 0, 0, 0}, {0, 0, 0, 0}}
	if p := TranslatePreference(blank, 0); p != 0 {
		t.Fatalf("blank must save 0, got %d", p)
	}
}

// A solid single-colour flood is trivially periodic at every period, yet the
// non-trivial-tile guard credits it ZERO — the dark room's periodic door is shut.
func TestTranslate_UniformFloodEarnsNothing(t *testing.T) {
	flood := [][]int{{1, 1, 1, 1}, {1, 1, 1, 1}}
	if p := TranslatePreference(flood, 0); p != 0 {
		t.Fatalf("uniform flood must earn 0 translate savings, got %d", p)
	}
}

// A genuine repeating pattern is compressed.
func TestTranslate_PeriodicSaves(t *testing.T) {
	g := [][]int{{1, 2, 1, 2}} // period 2, non-trivial tile {1,2}
	if p := TranslatePreference(g, 0); p != 2 {
		t.Fatalf("period-2 pattern should save 2, got %d", p)
	}
}

// Gradient both ways: breaking a periodic cell drops the saving; restoring lifts it.
func TestTranslate_GradientBothWays(t *testing.T) {
	g := [][]int{{1, 2, 1, 2}}
	base := TranslatePreference(g, 0)
	broken := cloneGrid(g)
	broken[0][2] = 5
	if got := TranslatePreference(broken, 0); got != base-1 {
		t.Fatalf("breaking one periodic cell should drop by 1: base=%d got=%d", base, got)
	}
	restored := cloneGrid(broken)
	restored[0][2] = 1
	if got := TranslatePreference(restored, 0); got != base {
		t.Fatalf("restoring should recover: base=%d got=%d", base, got)
	}
}

// THE emergence proof --------------------------------------------------------
//
// No primitive is named as the goal. BestPrimitive scores every primitive on the
// single currency of bits saved and returns the winner. On a symmetric-but-not-
// periodic grid Reflect must win; on a periodic-but-not-symmetric grid Translate
// must win. Which regularity "matters" is decided per grid by the data alone.

func TestEmergence_SelectsReflectOnSymmetry(t *testing.T) {
	symmetric := [][]int{
		{1, 2, 2, 1},
		{3, 4, 4, 3},
	}
	prim, save := BestPrimitive(symmetric, 0)
	if prim.Name != "Reflect" {
		t.Fatalf("symmetric grid should select Reflect, got %s (save=%d)", prim.Name, save)
	}
	// sanity: reflection genuinely out-saves translation here
	if SymmetryPreference(symmetric, 0) <= TranslatePreference(symmetric, 0) {
		t.Fatalf("expected reflect savings > translate on symmetric grid (R=%d T=%d)",
			SymmetryPreference(symmetric, 0), TranslatePreference(symmetric, 0))
	}
}

func TestEmergence_SelectsTranslateOnPeriodicity(t *testing.T) {
	periodic := [][]int{
		{1, 2, 1, 2},
		{3, 4, 3, 4},
	}
	prim, save := BestPrimitive(periodic, 0)
	if prim.Name != "Translate" {
		t.Fatalf("periodic grid should select Translate, got %s (save=%d)", prim.Name, save)
	}
	if TranslatePreference(periodic, 0) <= SymmetryPreference(periodic, 0) {
		t.Fatalf("expected translate savings > reflect on periodic grid (R=%d T=%d)",
			SymmetryPreference(periodic, 0), TranslatePreference(periodic, 0))
	}
}
