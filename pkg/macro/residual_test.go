package macro

import "testing"

func hasCell(cells []Cell, r, c int) bool {
	for _, cl := range cells {
		if cl.R == r && cl.C == c {
			return true
		}
	}
	return false
}

// A periodic texture with a single anomaly: the residual must be exactly that
// anomaly, and NOT the regular cells around it.
func TestResidual_TranslateIsolatesAnomaly(t *testing.T) {
	grid := [][]int{
		{1, 2, 1, 2, 1, 2},
		{1, 2, 1, 2, 1, 2},
		{1, 2, 9, 2, 1, 2}, // 9 at (2,2) breaks the period
	}
	if bp, _ := BestPrimitive(grid, 0); bp.Name != "Translate" {
		t.Fatalf("expected Translate to win, got %s", bp.Name)
	}
	res := ResidualCells(grid, 0)
	if len(res) != 1 || !hasCell(res, 2, 2) {
		t.Fatalf("residual should be exactly the anomaly (2,2), got %v", res)
	}
	x, y, ok := ResidualTarget(grid, 0)
	if !ok || x != 2 || y != 2 {
		t.Fatalf("target should point at the anomaly (x=2,y=2), got (%d,%d,%v)", x, y, ok)
	}
}

// A perfectly periodic grid has NO residual — nothing to attend to.
func TestResidual_PurePeriodicHasNone(t *testing.T) {
	grid := [][]int{
		{1, 2, 1, 2},
		{1, 2, 1, 2},
	}
	if res := ResidualCells(grid, 0); len(res) != 0 {
		t.Fatalf("pure periodic grid should have no residual, got %v", res)
	}
}

// A mostly-symmetric field with one broken cell: the residual must contain the
// anomaly.
func TestResidual_ReflectIsolatesAnomaly(t *testing.T) {
	grid := [][]int{
		{1, 2, 2, 1},
		{3, 4, 4, 3},
		{5, 6, 9, 5}, // (2,2) should mirror (2,1)=6 but is 9
	}
	if bp, _ := BestPrimitive(grid, 0); bp.Name != "Reflect" {
		t.Fatalf("expected Reflect to win, got %s", bp.Name)
	}
	res := ResidualCells(grid, 0)
	if !hasCell(res, 2, 2) {
		t.Fatalf("residual should include the broken cell (2,2), got %v", res)
	}
}

// Repeated identical objects plus one odd object: the residual is the odd object.
func TestResidual_CountIsolatesOddObject(t *testing.T) {
	grid := [][]int{
		{1, 1, 1, 0, 0, 0},
		{0, 0, 0, 0, 0, 0},
		{0, 0, 0, 1, 1, 1},
		{0, 0, 0, 0, 0, 0},
		{2, 2, 0, 0, 0, 0}, // the odd object
	}
	if bp, _ := BestPrimitive(grid, 0); bp.Name != "Count" {
		t.Fatalf("expected Count to win, got %s", bp.Name)
	}
	res := ResidualCells(grid, 0)
	if !hasCell(res, 4, 0) || !hasCell(res, 4, 1) {
		t.Fatalf("residual should be the odd object cells (4,0),(4,1), got %v", res)
	}
	// and must NOT flag the repeated trominoes
	if hasCell(res, 0, 0) || hasCell(res, 2, 3) {
		t.Fatalf("residual must not include the repeated objects, got %v", res)
	}
}
