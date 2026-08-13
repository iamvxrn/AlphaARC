package bridge

import (
	"context"
	"testing"

	"protaxon/pkg/environment"
	"protaxon/pkg/environment/perception"
	"protaxon/pkg/graph"
	"protaxon/pkg/pipeline"
)

func TestActionSucceeded(t *testing.T) {
	cases := []struct {
		name     string
		previous string
		current  string
		want     bool
	}{
		{"no prior frame is a bootstrap success", "", "color3-cell0-0", true},
		{"identical observation means nothing changed", "color3-cell0-0", "color3-cell0-0", false},
		{"different observation means something changed", "color3-cell0-0", "color3-cell0-1", true},
		{"both empty (no blobs either frame) is unchanged", "empty", "empty", false},
	}
	for _, c := range cases {
		if got := actionSucceeded(c.previous, c.current); got != c.want {
			t.Fatalf("FAIL [%s]: expected %v, got %v", c.name, c.want, got)
		}
	}
}

func TestDescribeCategoryGraphStateReportsClusterActivationAndEdges(t *testing.T) {
	g := graph.NewGraph()
	g.AddNode(graph.NewNode(1, 0.5, 3))
	g.AddNode(graph.NewNode(2, 0.5, 3))
	g.AddLabel("color2-cell0-0", 1)
	g.AddLabel("color5-cell1-1", 2)
	g.Nodes[1].Activation = 0.8
	g.Nodes[2].Activation = 0.3
	g.AddEdge(1, 2, 0.42, false) // directed 1 -> 2 only

	got := DescribeCategoryGraphState(g, []string{"color2-cell0-0", "color5-cell1-1"})
	want := "color2-cell0-0: node=1 cluster=3 activation=0.8000 edges:->color5-cell1-1(w=0.4200) | color5-cell1-1: node=2 cluster=3 activation=0.3000"
	if got != want {
		t.Fatalf("FAIL:\nexpected %q\ngot      %q", want, got)
	}
}

func TestDescribeCategoryGraphStateReportsUnregisteredLabel(t *testing.T) {
	g := graph.NewGraph()
	got := DescribeCategoryGraphState(g, []string{"color9-cell9-9"})
	want := "color9-cell9-9: not yet in graph"
	if got != want {
		t.Fatalf("FAIL: expected %q, got %q", want, got)
	}
}

// TestDescribeCategoryGraphStateEdgeOrderFollowsInputLabelsNotMapOrder
// confirms a hub node's edges are listed in the SAME order the caller's
// labels slice was given, not Go's unordered map iteration over
// node.Edges -- otherwise this diagnostic would be non-deterministic
// output for the exact same graph state, defeating its own purpose.
func TestDescribeCategoryGraphStateEdgeOrderFollowsInputLabelsNotMapOrder(t *testing.T) {
	g := graph.NewGraph()
	g.AddNode(graph.NewNode(1, 0.5, 0))
	g.AddNode(graph.NewNode(2, 0.5, 0))
	g.AddNode(graph.NewNode(3, 0.5, 0))
	// AddLabel lowercases its keyword internally (pkg/graph/spreading.go),
	// and DescribeCategoryGraphState does a direct, non-lowercasing map
	// lookup -- so labels here must already be lowercase, exactly like
	// real perception.DescribeGridCells output always is ("color2-cell0-0"),
	// never uppercase single letters.
	g.AddLabel("a", 1)
	g.AddLabel("b", 2)
	g.AddLabel("c", 3)
	g.AddEdge(1, 2, 0.1, false)
	g.AddEdge(1, 3, 0.2, false)

	got := DescribeCategoryGraphState(g, []string{"a", "b", "c"})
	want := "a: node=1 cluster=0 activation=0.0000 edges:->b(w=0.1000),->c(w=0.2000) | b: node=2 cluster=0 activation=0.0000 | c: node=3 cluster=0 activation=0.0000"
	if got != want {
		t.Fatalf("FAIL:\nexpected %q\ngot      %q", want, got)
	}
}

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

// TestWinningBlobLabelFindsWinnerAmongAllNegativeActivations is a
// regression test for a real bug: bestActivation used to start at a
// hardcoded -1.0. A live run (2026-08-13) drove several nodes' Activation
// below -1.0 (to roughly -4.12, see MaxWeightMagnitude's doc comment in
// pkg/graph/eligibility.go), which meant EVERY candidate failed the
// `Activation > bestActivation` check and this silently returned "" even
// though a correct winner (the least-negative candidate) existed. With
// -Inf as the sentinel, the actual highest (least negative) candidate
// must always be found, however negative activations get.
func TestWinningBlobLabelFindsWinnerAmongAllNegativeActivations(t *testing.T) {
	g := graph.NewGraph()
	g.AddNode(graph.NewNode(1, 0.5, 0))
	g.AddNode(graph.NewNode(2, 0.5, 0))
	g.AddLabel("color2-cell0-0", 1)
	g.AddLabel("color5-cell1-1", 2)
	g.Nodes[1].Activation = -4.12
	g.Nodes[2].Activation = -2.5 // less negative -- must win

	got := winningBlobLabel(g, []int{1, 2})
	want := "color5-cell1-1"
	if got != want {
		t.Fatalf("FAIL: expected the least-negative candidate %q to win, got %q -- sentinel regression", want, got)
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

	// explorationRoll=1.0 guarantees no exploration override (Curiosity's
	// max is 1.0 and explorationRoll < curiosity is never true at exactly
	// 1.0), keeping this test's hand-traced result independent of the
	// curiosity/exploration mechanics under test elsewhere.
	action, obs, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 1, 2, 2, "", 0.1, 1.0)
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if want := perception.DescribeGridCells(grid, 1, 2, 2); obs != want {
		t.Fatalf("FAIL: expected returned observation %q to match DescribeGridCells, got %q", want, obs)
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

	action, _, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 2, 2, 2, "", 0.1, 1.0)
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
	action1, _, _, err := ChooseClickAction(ctx, engine, frame1, "investigate the scene", 1, 2, 2, "", 0.1, 1.0)
	if err != nil {
		t.Fatalf("FAIL: unexpected error on frame 1: %v", err)
	}
	if want := (environment.Action{ID: environment.Action6, X: 2, Y: 2}); action1 != want {
		t.Fatalf("FAIL: frame 1 expected %+v, got %+v", want, action1)
	}

	frame2 := gridWithCell(10, 10, 4, 3, 4)
	action2, _, _, err := ChooseClickAction(ctx, engine, frame2, "investigate the scene", 1, 2, 2, "", 0.1, 1.0)
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

	_, _, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 3, 2, 2, "", 0.1, 1.0)
	if err == nil {
		t.Fatalf("FAIL: expected an error when the grid has no blobs to click")
	}
}

func approxEqual(a, b float64) bool {
	diff := a - b
	return diff < 1e-9 && diff > -1e-9
}

// TestChooseClickActionCuriosityFallsOnSuccessAndRisesOnFailure hand-traces
// Curiosity's dynamics across two calls on a fresh engine (starts 0.5, see
// pkg/homeostasis.NewState). Call 1 is a bootstrap success (no prior
// observation) -> Curiosity falls by curiosityStep. Call 2 sees the exact
// same grid, so the observation is identical -> actionSucceeded is false
// -> Curiosity rises by curiosityStep, landing back at the start value.
func TestChooseClickActionCuriosityFallsOnSuccessAndRisesOnFailure(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	grid := gridWithCell(10, 10, 3, 3, 4)
	const step = 0.1

	start := engine.Homeostasis.Curiosity
	_, obs1, _, err := ChooseClickAction(ctx, engine, grid, "goal", 1, 2, 2, "", step, 1.0)
	if err != nil {
		t.Fatalf("FAIL: unexpected error on call 1: %v", err)
	}
	if got, want := engine.Homeostasis.Curiosity, start-step; !approxEqual(got, want) {
		t.Fatalf("FAIL: expected curiosity to fall to %.4f after a bootstrap success, got %.4f", want, got)
	}

	_, _, _, err = ChooseClickAction(ctx, engine, grid, "goal", 1, 2, 2, obs1, step, 1.0)
	if err != nil {
		t.Fatalf("FAIL: unexpected error on call 2: %v", err)
	}
	if got, want := engine.Homeostasis.Curiosity, start; !approxEqual(got, want) {
		t.Fatalf("FAIL: expected curiosity to rise back to %.4f after an identical (failed) frame, got %.4f", want, got)
	}
}

// threeRankedBlobsGrid builds a 10x10 grid with three blobs of distinct
// sizes -- a 6-cell color-3 rectangle (centroid (2,1), ranked first), a
// 2-cell color-5 pair (centroid (5,5), ranked second), and a 1-cell
// color-7 dot (centroid (8,8), ranked third) -- reusing the same ranking
// mechanics TestChooseClickActionPicksHighestRankedBlobOnFreshEngine
// already confirmed: on a fresh engine, the largest blob's label is
// created first and wins WTA's exact-activation tie by lowest node ID.
func threeRankedBlobsGrid() [][]int {
	grid := make([][]int, 10)
	for y := range grid {
		grid[y] = make([]int, 10)
	}
	for _, x := range []int{1, 2, 3} {
		grid[1][x] = 3
		grid[2][x] = 3
	}
	grid[5][5] = 5
	grid[5][6] = 5
	grid[8][8] = 7
	return grid
}

// TestChooseClickActionExplorationPicksNonDefaultBlob: on a fresh engine's
// first call, a bootstrap success drops Curiosity from 0.5 to 0.4.
// explorationRoll=0.1 < 0.4, so exploration must trigger. It excludes the
// default WTA winner (the 6-cell blob at index 0) and picks deterministically
// among the 2 remaining blobs via normalizedRoll = 0.1/0.4 = 0.25,
// idx = int(0.25*2) = 0 -> the first remaining ranked blob, the 2-cell
// blob at (5,5) -- NOT the 6-cell default.
func TestChooseClickActionExplorationPicksNonDefaultBlob(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	grid := threeRankedBlobsGrid()

	action, _, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 3, 2, 2, "", 0.1, 0.1)
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	want := environment.Action{ID: environment.Action6, X: 5, Y: 5}
	if action != want {
		t.Fatalf("FAIL: expected exploration to pick the 2-cell blob's centroid %+v (not the default 6-cell winner (2,1)), got %+v", want, action)
	}
}

// TestChooseClickActionNoExplorationWhenRollMeetsCuriosity: same setup as
// above, but explorationRoll=0.4 exactly meets (does not fall below)
// post-update Curiosity 0.4, so exploration must NOT trigger -- the
// default WTA winner (the 6-cell blob, centroid (2,1)) must be returned.
func TestChooseClickActionNoExplorationWhenRollMeetsCuriosity(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	grid := threeRankedBlobsGrid()

	action, _, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 3, 2, 2, "", 0.1, 0.4)
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	want := environment.Action{ID: environment.Action6, X: 2, Y: 1}
	if action != want {
		t.Fatalf("FAIL: expected the default (6-cell) WTA winner %+v when roll meets curiosity exactly, got %+v", want, action)
	}
}
