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

// Workspace segmentation: a small tiled workspace with a hole, embedded in a big
// background field. Whole-grid GoalTargets is fooled (bg is the majority ->
// guarded to nil); SegmentedGoalTargets crops to the content box and finds the hole.
func TestGoalTargets_SegmentationFindsEmbeddedWorkspace(t *testing.T) {
	bg := 7
	g := make([][]int, 10)
	for r := range g {
		g[r] = make([]int, 10)
		for c := range g[r] {
			g[r][c] = bg
		}
	}
	// 4x4 tiled workspace (motif [[1,2],[3,4]]) at rows 3-6, cols 3-6
	motif := [][]int{{1, 2}, {3, 4}}
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			g[3+r][3+c] = motif[r%2][c%2]
		}
	}
	g[4][6] = 9 // corrupt one workspace cell: (4,6) class (1,1) -> want 4

	// whole-grid deriver refuses (background dominates -> no real reference)
	if ts := GoalTargets(g, bg); ts != nil {
		t.Fatalf("whole-grid deriver should refuse the bg-dominated grid, got %v", ts)
	}
	// segmentation crops to the workspace and finds exactly the hole, in orig coords
	ts := SegmentedGoalTargets(g, bg)
	if len(ts) != 1 || ts[0].R != 4 || ts[0].C != 6 || ts[0].Want != 4 {
		t.Fatalf("segmentation should find (4,6)->4, got %v", ts)
	}
}

// No erasing: a target that would set a cell TO background is never emitted, and a
// tiling whose reference is mostly background is refused (the vc33 "1516 targets"
// degenerate).
func TestGoalTargets_RefusesBackgroundReference(t *testing.T) {
	bg := 3
	// mostly-bg grid with a small stripe of content: the majority tile is bg.
	g := make([][]int, 8)
	for r := range g {
		g[r] = make([]int, 8)
		for c := range g[r] {
			g[r][c] = bg
		}
	}
	for r := 0; r < 8; r++ {
		g[r][4] = 5 // a single column of content
	}
	if ts := SegmentedGoalTargets(g, bg); ts != nil {
		t.Fatalf("a bg-majority reference must be refused (no erase-to-bg goal), got %v", ts)
	}
}
