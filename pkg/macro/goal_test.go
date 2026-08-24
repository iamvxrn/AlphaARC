package macro

import "testing"

// A 6x6 grid tiled with a 2x2 motif [[1,2],[3,4]], but ONE cell is corrupted.
// GoalTargets should derive exactly that cell -> the colour completing the tiling.
func TestGoalTargets_CompletesTiling(t *testing.T) {
	bg := 0
	motif := [][]int{{1, 2}, {3, 4}}
	g := make([][]int, 6)
	for r := range g {
		g[r] = make([]int, 6)
		for c := range g[r] {
			g[r][c] = motif[r%2][c%2]
		}
	}
	g[3][4] = 9 // corrupt one cell (class (1,0) -> should be 3)

	ts := GoalTargets(g, bg)
	if len(ts) != 1 {
		t.Fatalf("expected exactly 1 goal target, got %d: %v", len(ts), ts)
	}
	if ts[0].R != 3 || ts[0].C != 4 || ts[0].Want != 3 {
		t.Fatalf("expected target (3,4)->3, got %+v", ts[0])
	}
}

// A clean tiling has no goal (nothing to fix); noise has no reference.
func TestGoalTargets_NoneWhenPerfectOrNoise(t *testing.T) {
	bg := 0
	perfect := [][]int{{1, 2, 1, 2}, {3, 4, 3, 4}, {1, 2, 1, 2}, {3, 4, 3, 4}}
	if ts := GoalTargets(perfect, bg); ts != nil {
		t.Fatalf("perfect tiling should yield no targets, got %v", ts)
	}
}

// Multiple corruptions of the same tiling are all recovered.
func TestGoalTargets_MultipleHoles(t *testing.T) {
	bg := 0
	motif := [][]int{{5, 6}, {7, 8}}
	g := make([][]int, 8)
	for r := range g {
		g[r] = make([]int, 8)
		for c := range g[r] {
			g[r][c] = motif[r%2][c%2]
		}
	}
	g[0][0], g[5][3] = 0, 0 // two holes
	ts := GoalTargets(g, bg)
	if len(ts) != 2 {
		t.Fatalf("expected 2 targets, got %d: %v", len(ts), ts)
	}
	want := map[[2]int]int{{0, 0}: 5, {5, 3}: 8}
	for _, gt := range ts {
		if want[[2]int{gt.R, gt.C}] != gt.Want {
			t.Fatalf("bad target %+v (want map %v)", gt, want)
		}
	}
}
