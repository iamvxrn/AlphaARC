package macro

import "testing"

// bgGrid makes an h x w grid filled with bg.
func bgGrid(h, w, bg int) [][]int {
	g := make([][]int, h)
	for r := range g {
		g[r] = make([]int, w)
		for c := range g[r] {
			g[r][c] = bg
		}
	}
	return g
}

// place stamps box's cells into grid at (top,left).
func place(grid [][]int, top, left int, box [][]int) {
	for r := range box {
		for c := range box[r] {
			grid[top+r][left+c] = box[r][c]
		}
	}
}

// symBox: a 5x5 color-2 frame with a repeated 4/3 interior (each colour occurs
// several times, so a single missing cell yields residual 1, not a color-swap
// remap). Interior cells are diagonally isolated -> not their own components.
var symBox = [][]int{
	{2, 2, 2, 2, 2},
	{2, 4, 3, 4, 2},
	{2, 3, 4, 3, 2},
	{2, 4, 3, 4, 2},
	{2, 2, 2, 2, 2},
}

// P1: filling one correct cell of a partial pattern raises the saving by exactly
// 1 -- the dopamine click on DriveScore. And blank/single give 0.
func TestCorrespondence_FillRaisesSaving(t *testing.T) {
	bg := 9
	full := bgGrid(7, 14, bg)
	place(full, 1, 1, symBox)
	place(full, 1, 8, symBox)
	sFull := CorrespondenceSavings(full, bg)

	partial := bgGrid(7, 14, bg)
	place(partial, 1, 1, symBox)
	place(partial, 1, 8, symBox)
	partial[3][10] = bg // knock out one interior cell of the second box (center 4)
	sPartial := CorrespondenceSavings(partial, bg)

	if sFull != 24 {
		t.Fatalf("full match of a 5x5 pair should save area-0-pointer=24, got %d", sFull)
	}
	if sPartial != 23 {
		t.Fatalf("one missing cell should save 23 (residual 1), got %d", sPartial)
	}
	if sFull-sPartial != 1 {
		t.Fatalf("filling one cell must raise saving by exactly 1: full=%d partial=%d", sFull, sPartial)
	}
}

func TestCorrespondence_BlankAndSingleSaveNothing(t *testing.T) {
	bg := 9
	if s := CorrespondenceSavings(bgGrid(7, 7, bg), bg); s != 0 {
		t.Fatalf("blank must save 0, got %d", s)
	}
	one := bgGrid(7, 7, bg)
	place(one, 1, 1, symBox)
	if s := CorrespondenceSavings(one, bg); s != 0 {
		t.Fatalf("a single box (no pair) must save 0, got %d", s)
	}
}

// The degenerate cheat is shut: two identical SOLID blocks (no interior pattern)
// earn nothing -- Correspondence rewards matching PATTERNS, not floods.
func TestCorrespondence_SolidBlocksEarnNothing(t *testing.T) {
	bg := 9
	solid := [][]int{{2, 2, 2, 2, 2}, {2, 2, 2, 2, 2}, {2, 2, 2, 2, 2}, {2, 2, 2, 2, 2}, {2, 2, 2, 2, 2}}
	g := bgGrid(7, 14, bg)
	place(g, 1, 1, solid)
	place(g, 1, 8, solid)
	if s := CorrespondenceSavings(g, bg); s != 0 {
		t.Fatalf("two solid blocks must earn 0 (no pattern), got %d", s)
	}
}

// ColorSwap: a box that is the template recoloured (4<->3) still matches, via the
// O(area) majority map -- residual collapses to 0.
func TestCorrespondence_ColorSwapMatches(t *testing.T) {
	bg := 9
	swapped := [][]int{
		{2, 2, 2, 2, 2},
		{2, 3, 4, 3, 2},
		{2, 4, 3, 4, 2},
		{2, 3, 4, 3, 2},
		{2, 2, 2, 2, 2},
	}
	g := bgGrid(7, 14, bg)
	place(g, 1, 1, symBox)
	place(g, 1, 8, swapped)
	if s := CorrespondenceSavings(g, bg); s != 24 {
		t.Fatalf("a recoloured copy should match via ColorSwap (save 24), got %d", s)
	}
}

// Reflect: a box that is the horizontal mirror of the template matches via Reflect.
func TestCorrespondence_ReflectMatches(t *testing.T) {
	bg := 9
	tpl := [][]int{
		{2, 2, 2, 2, 2},
		{2, 4, 9, 9, 2},
		{2, 9, 9, 9, 2},
		{2, 9, 9, 3, 2},
		{2, 2, 2, 2, 2},
	}
	mir := [][]int{
		{2, 2, 2, 2, 2},
		{2, 9, 9, 4, 2},
		{2, 9, 9, 9, 2},
		{2, 3, 9, 9, 2},
		{2, 2, 2, 2, 2},
	}
	g := bgGrid(7, 14, bg)
	place(g, 1, 1, tpl)
	place(g, 1, 8, mir)
	if s := CorrespondenceSavings(g, bg); s != 24 {
		t.Fatalf("a mirror copy should match via Reflect (save 24), got %d", s)
	}
}

// Cross-primitive emergence: two matching template boxes placed with NO global
// symmetry/periodicity -- Correspondence must out-save the self-regularity
// primitives and BestPrimitive must select it.
func TestEmergence_SelectsCorrespondence(t *testing.T) {
	bg := 9
	g := bgGrid(16, 16, bg)
	place(g, 1, 1, symBox)   // top-left
	place(g, 9, 6, symBox)   // offset down-right: not a mirror or period of the first
	corr := CorrespondencePreference(g, bg)
	refl := SymmetryPreference(g, bg)
	tr := TranslatePreference(g, bg)
	cnt := NumerosityPreference(g, bg)
	t.Logf("savings: Correspondence=%d Reflect=%d Translate=%d Count=%d", corr, refl, tr, cnt)
	if corr <= refl || corr <= tr || corr <= cnt {
		t.Fatalf("Correspondence must dominate here: C=%d R=%d T=%d Cnt=%d", corr, refl, tr, cnt)
	}
	if p, _ := BestPrimitive(g, bg); p.Name != "Correspondence" {
		t.Fatalf("should select Correspondence, got %s", p.Name)
	}
}
