package macro

import "testing"

// P1: blank grid discovers nothing (dark-room-proof, like every primitive).
func TestDiscover_BlankFindsNothing(t *testing.T) {
	blank := [][]int{{0, 0, 0, 0}, {0, 0, 0, 0}}
	if name, s := DiscoverTransform(blank, 0); s != 0 || name != "" {
		t.Fatalf("blank must discover nothing, got %q=%d", name, s)
	}
}

// P2: parity with the hand-authored Reflect -- on a horizontally symmetric grid
// the meta-operator discovers reflectH and its savings equal SymmetryPreference.
func TestDiscover_ReproducesReflect(t *testing.T) {
	bg := 0
	g := [][]int{
		{1, 2, 2, 1},
		{3, 4, 4, 3},
	}
	name, s := DiscoverTransform(g, bg)
	if name != "reflectH" {
		t.Fatalf("expected to discover reflectH, got %q", name)
	}
	if s != SymmetryPreference(g, bg) {
		t.Fatalf("discovered savings %d must match SymmetryPreference %d", s, SymmetryPreference(g, bg))
	}
}

// P3 (the real point): a grid with 180-degree rotational symmetry but NO
// reflection or period -- a regularity the hand-authored set (Reflect/Translate/
// Count) MISSES. The meta-operator must DISCOVER rot180 from the data and beat
// every hand-authored primitive. This is operator growth in miniature: the
// specific regularity was found, not written in code.
func TestDiscover_FindsRotationTheHandSetMisses(t *testing.T) {
	bg := 0
	// Point-symmetric about the centre (g[r][c] == g[H-1-r][W-1-c]) but not
	// mirror-symmetric on either axis and not periodic.
	g := [][]int{
		{1, 2, 0, 0},
		{0, 0, 3, 4},
		{4, 3, 0, 0},
		{0, 0, 2, 1},
	}
	name, s := DiscoverTransform(g, bg)
	if name != "rot180" {
		t.Fatalf("expected to discover rot180, got %q (savings %d)", name, s)
	}
	if s <= 0 {
		t.Fatalf("rot180 must explain cells, got %d", s)
	}
	// It must strictly beat the entire hand-authored grammar on this grid.
	refl := SymmetryPreference(g, bg)
	tr := TranslatePreference(g, bg)
	cnt := NumerosityPreference(g, bg)
	corr := CorrespondencePreference(g, bg)
	t.Logf("discovered %s=%d | Reflect=%d Translate=%d Count=%d Correspondence=%d", name, s, refl, tr, cnt, corr)
	if s <= refl || s <= tr || s <= cnt || s <= corr {
		t.Fatalf("discovered rot180=%d must beat the hand-authored set (R=%d T=%d Cnt=%d Corr=%d)", s, refl, tr, cnt, corr)
	}
}

// P4: the degenerate cheat is shut -- identity is not in the family, so a grid
// with no real symmetry discovers nothing rather than "explaining" every cell.
func TestDiscover_NoFakeSymmetry(t *testing.T) {
	bg := 0
	g := [][]int{
		{1, 2, 3, 4},
		{5, 6, 7, 8},
	}
	if name, s := DiscoverTransform(g, bg); s != 0 {
		t.Fatalf("an asymmetric grid must discover nothing, got %q=%d", name, s)
	}
}

// P5: color-permutation symmetry -- a grid that is horizontally mirror-symmetric
// only AFTER a colour swap (1<->5, 2<->6). Exact Reflect scores 0; the
// color-perm family must find it.
func TestDiscover_ColorPermSymmetry(t *testing.T) {
	bg := 0
	// left half {1,2}, right half is its mirror recoloured to {5,6}.
	g := [][]int{
		{1, 2, 6, 5},
		{2, 1, 5, 6},
	}
	if s := SymmetryPreference(g, bg); s != 0 {
		t.Fatalf("exact reflect should be 0 on a colour-swapped mirror, got %d", s)
	}
	if s := ColorPermSymmetry(g, bg); s <= 0 {
		t.Fatalf("color-perm symmetry should be >0 on a colour-swapped mirror, got %d", s)
	}
	// blank stays 0 (dark-room-proof)
	if s := ColorPermSymmetry([][]int{{0, 0, 0, 0}}, bg); s != 0 {
		t.Fatalf("blank must be 0, got %d", s)
	}
}

// P6: periodicity up to a colour permutation. A row that repeats in SHAPE with a
// per-tile recolour (and a repeated colour that breaks mirror symmetry) -> exact
// Translate and the involution color-perm both score 0, but the periodic family
// finds the shift+relabel.
func TestDiscover_PeriodicColorPerm(t *testing.T) {
	// tiles [x, 2] repeated with a per-tile recolour of x (1->3->5, 4->6->8); the
	// interleaved repeated colour 2 is the background here (as perception would
	// pick it) and breaks mirror symmetry. Exact Translate and Reflect score 0;
	// the periodic family is the strongest explanation.
	bg := 2
	g := [][]int{
		{1, 2, 3, 2, 5, 2},
		{4, 2, 6, 2, 8, 2},
	}
	if s := TranslatePreference(g, bg); s != 0 {
		t.Fatalf("exact translate should be 0 on a per-tile recolour, got %d", s)
	}
	if s := SymmetryPreference(g, bg); s != 0 {
		t.Fatalf("exact reflect should be 0, got %d", s)
	}
	if PeriodicColorPerm(g, bg) <= ColorPermSymmetry(g, bg) {
		t.Fatalf("periodic-color-perm (%d) should dominate the involution color-perm (%d) on a periodic-recolour grid",
			PeriodicColorPerm(g, bg), ColorPermSymmetry(g, bg))
	}
	if s := PeriodicColorPerm(g, bg); s <= 0 {
		t.Fatalf("periodic-color-perm should be >0, got %d", s)
	}
	if s := PeriodicColorPerm([][]int{{0, 0, 0, 0, 0, 0}}, 0); s != 0 {
		t.Fatalf("blank must be 0, got %d", s)
	}
	// injectivity guard: a many-to-one colour collapse must not be credited
	collapse := [][]int{{1, 2, 9, 9, 9, 9}} // period-2 maps 1->9 and 2->9 (collapse)
	if s := colorPermPeriodSavings(collapse, 0, 0, 2); s != 0 {
		t.Fatalf("non-injective (collapsing) colour map must score 0, got %d", s)
	}
}
