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
	action, obs, _, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 1, 2, 2, "", false, 0.1, 1.0, NewOutcomeMemory(), "")
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if want := perception.DescribeGridStructural(grid, 1, 2, 2); obs != want {
		t.Fatalf("FAIL: expected returned observation %q to match DescribeGridStructural, got %q", want, obs)
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

	action, _, _, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 2, 2, 2, "", false, 0.1, 1.0, NewOutcomeMemory(), "")
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
	action1, _, _, _, err := ChooseClickAction(ctx, engine, frame1, "investigate the scene", 1, 2, 2, "", false, 0.1, 1.0, NewOutcomeMemory(), "")
	if err != nil {
		t.Fatalf("FAIL: unexpected error on frame 1: %v", err)
	}
	if want := (environment.Action{ID: environment.Action6, X: 2, Y: 2}); action1 != want {
		t.Fatalf("FAIL: frame 1 expected %+v, got %+v", want, action1)
	}

	frame2 := gridWithCell(10, 10, 4, 3, 4)
	action2, _, _, _, err := ChooseClickAction(ctx, engine, frame2, "investigate the scene", 1, 2, 2, "", false, 0.1, 1.0, NewOutcomeMemory(), "")
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

	_, _, _, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 3, 2, 2, "", false, 0.1, 1.0, NewOutcomeMemory(), "")
	if err == nil {
		t.Fatalf("FAIL: expected an error when the grid has no blobs to click")
	}
}

// TestChooseClickActionNilMemoryDoesNotPanic confirms a nil *OutcomeMemory
// (a caller that forgot to construct one) gets silently substituted with a
// fresh instance rather than panicking on memory.Record/SuccessRate.
func TestChooseClickActionNilMemoryDoesNotPanic(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	grid := gridWithCell(10, 10, 3, 3, 4)

	action, _, _, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 1, 2, 2, "", false, 0.1, 1.0, nil, "")
	if err != nil {
		t.Fatalf("FAIL: unexpected error with nil memory: %v", err)
	}
	want := environment.Action{ID: environment.Action6, X: 3, Y: 3}
	if action != want {
		t.Fatalf("FAIL: expected %+v even with nil memory, got %+v", want, action)
	}
}

// TestChooseClickActionLevelsCompletedIncreasedForcesSuccessDespiteUnchangedGrid
// is a regression test for the real gap confirmed 2026-08-13: a 300-action
// live run had the grid-changed proxy alone driving every success/failure
// judgment, and OutcomeMemory correctly, mechanically exploited a click that
// satisfied that proxy without ever completing a level. Call 1 is a
// bootstrap success (Curiosity falls 0.5 -> 0.4, see
// TestChooseClickActionCuriosityFallsOnSuccessAndRisesOnFailure for the
// baseline this contrasts with). Call 2 sees the EXACT SAME grid -- the
// proxy alone (actionSucceeded) would say false and Curiosity would rise
// back to 0.5 -- but levelsCompletedIncreased=true is passed, so
// actualSuccess must be forced true anyway: Curiosity must keep FALLING (to
// 0.3), not rise, and the previously clicked label must be recorded as a
// success, not a failure.
func TestChooseClickActionLevelsCompletedIncreasedForcesSuccessDespiteUnchangedGrid(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	grid := gridWithCell(10, 10, 3, 3, 4)
	const step = 0.1

	memory := NewOutcomeMemory()
	start := engine.Homeostasis.Curiosity
	_, obs1, label1, _, err := ChooseClickAction(ctx, engine, grid, "goal", 1, 2, 2, "", false, step, 1.0, memory, "")
	if err != nil {
		t.Fatalf("FAIL: unexpected error on call 1: %v", err)
	}
	if got, want := engine.Homeostasis.Curiosity, start-step; !approxEqual(got, want) {
		t.Fatalf("FAIL: expected curiosity to fall to %.4f after a bootstrap success, got %.4f", want, got)
	}

	_, _, _, _, err = ChooseClickAction(ctx, engine, grid, "goal", 1, 2, 2, obs1, true, step, 1.0, memory, label1)
	if err != nil {
		t.Fatalf("FAIL: unexpected error on call 2: %v", err)
	}
	if got, want := engine.Homeostasis.Curiosity, start-2*step; !approxEqual(got, want) {
		t.Fatalf("FAIL: expected curiosity to keep falling to %.4f (levelsCompletedIncreased must override the unchanged-grid proxy), got %.4f", want, got)
	}
	// levelsCompletedIncreased forces actualSuccess AND (branch C) reinforces
	// the recent sequence, so the label is recorded, proven, and rate 1.0
	// (the forced success plus the level-completion credit are all successes).
	rate, attempts := memory.SuccessRate(label1)
	if attempts < 1 {
		t.Fatalf("FAIL: expected %q to be recorded after call 2, got %d attempts", label1, attempts)
	}
	if rate != 1.0 {
		t.Fatalf("FAIL: expected rate 1.0 (levelsCompletedIncreased forces success despite an identical grid), got %.4f", rate)
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

	memory := NewOutcomeMemory()
	start := engine.Homeostasis.Curiosity
	_, obs1, _, _, err := ChooseClickAction(ctx, engine, grid, "goal", 1, 2, 2, "", false, step, 1.0, memory, "")
	if err != nil {
		t.Fatalf("FAIL: unexpected error on call 1: %v", err)
	}
	if got, want := engine.Homeostasis.Curiosity, start-step; !approxEqual(got, want) {
		t.Fatalf("FAIL: expected curiosity to fall to %.4f after a bootstrap success, got %.4f", want, got)
	}

	_, _, _, _, err = ChooseClickAction(ctx, engine, grid, "goal", 1, 2, 2, obs1, false, step, 1.0, memory, "")
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

	action, _, _, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 3, 2, 2, "", false, 0.1, 0.1, NewOutcomeMemory(), "")
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

	action, _, _, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 3, 2, 2, "", false, 0.1, 0.4, NewOutcomeMemory(), "")
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	want := environment.Action{ID: environment.Action6, X: 2, Y: 1}
	if action != want {
		t.Fatalf("FAIL: expected the default (6-cell) WTA winner %+v when roll meets curiosity exactly, got %+v", want, action)
	}
}

// TestChooseClickActionProvenOutcomeOverridesDefaultAndExploration: on
// threeRankedBlobsGrid, the 1-cell color-7 blob at (8,8) is ranked LAST and
// would never win WTA or a low-probability exploration roll on its own.
// But with 3 recorded successes already in OutcomeMemory for its label
// ("color7-cell1-1" at this grid's cols=2,rows=2 resolution -- centroid
// (8,8) floors to col=8*2/10=1, row=8*2/10=1), it must be preferred over
// BOTH the WTA default (the 6-cell blob) and exploration (explorationRoll
// is set high enough, 1.0, that exploration wouldn't trigger anyway) --
// real accumulated evidence outranks structural graph state.
func TestChooseClickActionProvenOutcomeOverridesDefaultAndExploration(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	grid := threeRankedBlobsGrid()

	memory := NewOutcomeMemory()
	for i := 0; i < 3; i++ {
		memory.Record("color7-cell1-1", true)
	}

	action, _, clickedLabel, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 3, 2, 2, "", false, 0.1, 1.0, memory, "")
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	want := environment.Action{ID: environment.Action6, X: 8, Y: 8}
	if action != want {
		t.Fatalf("FAIL: expected the proven-track-record blob's centroid %+v, got %+v -- override not applied", want, action)
	}
	if clickedLabel != "color7-cell1-1" {
		t.Fatalf("FAIL: expected returned clickedLabel %q, got %q", "color7-cell1-1", clickedLabel)
	}
}

// TestChooseClickActionPredictableOutcomeSuppressesProvenOverride is the
// epistemic-escape (variant B): identical setup to the proven-override test
// above (3 proven successes for the last-ranked blob), but the engine is
// armed so THIS cycle's forecast error is low (the forward model already
// predicts the scene -- PendingPrediction is set within 0.1 of the actual
// observation embedding on one dimension, MSE ~0.0005 < predictableForecast-
// Error). A predictable outcome makes the proven action epistemically
// exhausted, so the override must YIELD and the WTA default (the 6-cell blob
// at (2,1)) is clicked instead -- the fix for the 976/1000 single-point
// lock-in a live run showed. Contrast the override-test above, where the
// cold engine's forecast error is 0 (not predictable) so the override stands.
func TestChooseClickActionPredictableOutcomeSuppressesProvenOverride(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	grid := threeRankedBlobsGrid()

	memory := NewOutcomeMemory()
	for i := 0; i < 3; i++ {
		memory.Record("color7-cell1-1", true)
	}

	// Arm the forward model to predict this frame well: PendingPrediction ~=
	// the observation's embedding, so RunPredictiveCycle reports a tiny error
	// this cycle. Past warmup (ForecastSamples) with a HIGH running norm, so
	// the tiny error registers as SETTLED both absolutely and relative to the
	// norm -- exercising the relativized predictability signal.
	engine.ForecastSamples = 10
	engine.ForecastErrorEMA = 0.5
	obs := perception.DescribeGridStructural(grid, 3, 2, 2) // must match what ChooseClickAction now feeds the cycle
	pv := pipeline.ObservationVector(obs)
	pend := append([]float64(nil), pv...)
	pend[0] += 0.1 // MSE = 0.1^2 / 20 = 0.0005, a settled (well-predicted) spot
	engine.PendingPrediction = pend

	action, _, clickedLabel, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 3, 2, 2, "", false, 0.1, 1.0, memory, "")
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if clickedLabel == "color7-cell1-1" {
		t.Fatalf("FAIL: proven action was still exploited despite a predictable (epistemically exhausted) outcome -- override not suppressed")
	}
	want := environment.Action{ID: environment.Action6, X: 2, Y: 1}
	if action != want {
		t.Fatalf("FAIL: expected the WTA default (6-cell blob at (2,1)) once the proven override yields, got %+v (clickedLabel=%q)", action, clickedLabel)
	}
}

// TestChooseClickActionInsufficientAttemptsDoNotOverride confirms the
// minProvenAttempts gate: only 2 recorded successes (below the threshold
// of 3) for the last-ranked blob must NOT override the WTA default, unlike
// the 3-attempt case above.
func TestChooseClickActionInsufficientAttemptsDoNotOverride(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	grid := threeRankedBlobsGrid()

	memory := NewOutcomeMemory()
	memory.Record("color7-cell1-1", true)
	memory.Record("color7-cell1-1", true)

	action, _, _, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 3, 2, 2, "", false, 0.1, 1.0, memory, "")
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	want := environment.Action{ID: environment.Action6, X: 2, Y: 1}
	if action != want {
		t.Fatalf("FAIL: expected the default (6-cell) WTA winner %+v since only 2 attempts is below minProvenAttempts, got %+v", want, action)
	}
}

// TestChooseClickActionRecordsPreviousClickedLabelOutcome is an
// integration test for the actual recording wiring, not just
// OutcomeMemory's own unit tests: call 1 clicks the default (6-cell) blob
// and returns its label; call 2, on the SAME grid (so actualSuccess is
// false), passes that label back in as previousClickedLabel and must
// record exactly one failed attempt against it.
func TestChooseClickActionRecordsPreviousClickedLabelOutcome(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	grid := threeRankedBlobsGrid()
	memory := NewOutcomeMemory()

	_, obs1, label1, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 3, 2, 2, "", false, 0.1, 1.0, memory, "")
	if err != nil {
		t.Fatalf("FAIL: unexpected error on call 1: %v", err)
	}
	if label1 != "color3-cell0-0" {
		t.Fatalf("FAIL: expected call 1 to click %q (the default WTA winner), got %q", "color3-cell0-0", label1)
	}
	if _, attempts := memory.SuccessRate(label1); attempts != 0 {
		t.Fatalf("FAIL: expected 0 recorded attempts before call 2 (nothing to attribute the first click's outcome to yet), got %d", attempts)
	}

	if _, _, _, _, err := ChooseClickAction(ctx, engine, grid, "investigate the scene", 3, 2, 2, obs1, false, 0.1, 1.0, memory, label1); err != nil {
		t.Fatalf("FAIL: unexpected error on call 2: %v", err)
	}
	rate, attempts := memory.SuccessRate(label1)
	if attempts != 1 {
		t.Fatalf("FAIL: expected exactly 1 recorded attempt for %q after call 2, got %d", label1, attempts)
	}
	if rate != 0.0 {
		t.Fatalf("FAIL: expected rate 0.0 (identical grid -> actualSuccess=false), got %.4f", rate)
	}
}
