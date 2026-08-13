package bridge

import (
	"context"
	"testing"

	"protaxon/pkg/environment"
	"protaxon/pkg/graph"
	"protaxon/pkg/pipeline"
)

func TestLooksLikeBlobLabel(t *testing.T) {
	cases := []struct {
		word string
		want bool
	}{
		{"color2-cell0-0", true},
		{"color15-cell7-3", true},
		{"north", false},
		{"color2", false}, // no "-cell" part
		{"empty", false},
		{"aligned", false},
	}
	for _, c := range cases {
		if got := looksLikeBlobLabel(c.word); got != c.want {
			t.Fatalf("FAIL [%q]: expected %v, got %v", c.word, c.want, got)
		}
	}
}

func TestWinningBlobLabelIgnoresNonBlobLabelsEvenAtHigherActivation(t *testing.T) {
	g := graph.NewGraph()
	g.AddNode(graph.NewNode(1, 0.5, 0))
	g.AddNode(graph.NewNode(2, 0.5, 0))
	g.AddNode(graph.NewNode(3, 0.5, 0))
	g.AddLabel("north", 1)
	g.AddLabel("color2-cell0-0", 2)
	g.AddLabel("color5-cell1-1", 3)
	g.Nodes[1].Activation = 0.9 // highest overall, but not blob-shaped -- must be ignored
	g.Nodes[2].Activation = 0.5
	g.Nodes[3].Activation = 0.8 // highest AMONG blob-shaped labels

	got := winningBlobLabel(g, []int{1, 2, 3})
	want := "color5-cell1-1"
	if got != want {
		t.Fatalf("FAIL: expected %q, got %q", want, got)
	}
}

func TestWinningBlobLabelReturnsEmptyWhenNoneQualify(t *testing.T) {
	g := graph.NewGraph()
	g.AddNode(graph.NewNode(1, 0.5, 0))
	g.AddLabel("north", 1)
	g.Nodes[1].Activation = 1.0

	got := winningBlobLabel(g, []int{1})
	if got != "" {
		t.Fatalf("FAIL: expected empty string when no active node has a blob-shaped label, got %q", got)
	}
}

// gridWithCell builds a gridW x gridH grid (all background=0) with a single
// filled cell at (x,y) in the given color -- the minimal single-blob case.
func gridWithCell(gridW, gridH, x, y, color int) [][]int {
	grid := make([][]int, gridH)
	for row := range grid {
		grid[row] = make([]int, gridW)
	}
	grid[y][x] = color
	return grid
}

// TestChooseClickActionSingleBlobTrivialWin hand-traces the base case: one
// blob, no competition needed, so ChooseClickAction must click exactly its
// centroid.
func TestChooseClickActionSingleBlobTrivialWin(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	grid := gridWithCell(10, 10, 3, 3, 4)

	action, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 1, 2, 2)
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	want := environment.Action{ID: environment.Action6, X: 3, Y: 3}
	if action != want {
		t.Fatalf("FAIL: expected %+v, got %+v", want, action)
	}
}

// TestChooseClickActionPicksHighestRankedBlobOnFreshEngine hand-traces a
// two-blob competition on a completely fresh engine. Confirmed from
// source before writing this: LookupSeeds seeds every matched label at
// exactly 1.0 (pkg/graph/spreading.go), fresh concept nodes have no edges
// between them yet so SpreadingActivation can't change that, and
// EnsureConceptNodes gives every fresh concept node ClusterID 0, so both
// candidates land in the same router cluster and genuinely compete via
// ResolveLateralInhibition, which resolves an exact activation tie by
// keeping whichever candidate it iterates first -- members are
// sort.Ints'd, so the lower node ID (the one EnsureConceptNodes created
// first, i.e. whichever word appears first in the observation string)
// wins. DescribeGridCells/RankedLabeledBlobs put the larger blob first,
// so the larger blob's label is created first and wins on this exact
// mechanism -- not by coincidence.
//
// Grid: a 6-cell color-3 rectangle at rows 1-2, cols 1-3 (centroid (2,1))
// vs. a single color-7 cell at (5,5) (centroid (5,5)). The 6-cell blob
// must win.
func TestChooseClickActionPicksHighestRankedBlobOnFreshEngine(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()

	grid := make([][]int, 10)
	for y := range grid {
		grid[y] = make([]int, 10)
	}
	for _, x := range []int{1, 2, 3} {
		grid[1][x] = 3
		grid[2][x] = 3
	}
	grid[5][5] = 7

	action, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 2, 2, 2)
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	want := environment.Action{ID: environment.Action6, X: 2, Y: 1}
	if action != want {
		t.Fatalf("FAIL: expected the larger (6-cell) blob's centroid %+v, got %+v -- did the tie-break or ranking change?", want, action)
	}
}

// TestChooseClickActionBindsToCurrentFrameNotStaleCentroid proves the bind
// step re-derives coordinates from whatever grid is passed THIS call,
// rather than caching a blob object from a previous call. Both frames
// produce the exact same category label ("color4-cell0-0" at this coarse
// 2x2 lattice on a 10x10 grid -- (2,2) and (4,3) both floor to bucket
// (0,0)), so if the second call still returned (2,2), that would mean
// something is quietly caching the first frame's blob instead of
// re-reading the second frame's actual pixels.
func TestChooseClickActionBindsToCurrentFrameNotStaleCentroid(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()

	frame1 := gridWithCell(10, 10, 2, 2, 4)
	action1, _, err := ChooseClickAction(ctx, engine, frame1, "investigate the scene", 1, 2, 2)
	if err != nil {
		t.Fatalf("FAIL: unexpected error on frame 1: %v", err)
	}
	if want := (environment.Action{ID: environment.Action6, X: 2, Y: 2}); action1 != want {
		t.Fatalf("FAIL: frame 1 expected %+v, got %+v", want, action1)
	}

	frame2 := gridWithCell(10, 10, 4, 3, 4)
	action2, _, err := ChooseClickAction(ctx, engine, frame2, "investigate the scene", 1, 2, 2)
	if err != nil {
		t.Fatalf("FAIL: unexpected error on frame 2: %v", err)
	}
	if want := (environment.Action{ID: environment.Action6, X: 4, Y: 3}); action2 != want {
		t.Fatalf("FAIL: frame 2 expected %+v (this frame's actual centroid), got %+v -- looks like a stale/cached centroid from frame 1", want, action2)
	}
}

func TestChooseClickActionErrorsOnEmptyGrid(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	grid := [][]int{{0, 0}, {0, 0}}

	_, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 3, 2, 2)
	if err == nil {
		t.Fatalf("FAIL: expected an error when the grid has no blobs to click")
	}
}
