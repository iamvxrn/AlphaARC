package pipeline

import (
	"context"
	"testing"

	"alphaarc/pkg/environment/perception"
)

// blobRole is one synthetic recurring visual object: a small filled square
// that jitters by a few pixels around a fixed base position from frame to
// frame, the way a real game element might drift slightly without actually
// being a different thing.
type blobRole struct {
	color        int
	baseX, baseY int
	jitterSeed   int
}

// makeBlobFrame renders roles onto a 64x64 grid (matching environment.GridSize)
// as 3x3 filled squares. Jitter is a small deterministic function of
// frameIdx, not math/rand, so results are 100% reproducible and every
// possible (x,y) an experimenter needs to hand-check is derivable from the
// formula alone: jx = (frameIdx*7+jitterSeed)%5 - 2, jy = (frameIdx*11+jitterSeed)%5 - 2,
// each in [-2, 2].
func makeBlobFrame(frameIdx int, roles []blobRole) [][]int {
	const size = 64
	grid := make([][]int, size)
	for y := range grid {
		grid[y] = make([]int, size)
	}
	for _, r := range roles {
		jx := (frameIdx*7+r.jitterSeed)%5 - 2
		jy := (frameIdx*11+r.jitterSeed)%5 - 2
		cx, cy := r.baseX+jx, r.baseY+jy
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				y, x := cy+dy, cx+dx
				if y >= 0 && y < size && x >= 0 && x < size {
					grid[y][x] = r.color
				}
			}
		}
	}
	return grid
}

// TestPerceptionCategoryGranularityAndCohesion is an EXPERIMENT, not a
// hand-traced correctness test: it runs the exact same 30-frame synthetic
// sequence (3 recurring blob "roles" that jitter ±2px around fixed base
// positions -- a stand-in for a real game element drifting slightly
// without being a different thing) through two fresh engines, one
// perceiving via DescribeGridCells at a COARSE 4x4 lattice (16px buckets),
// one at a FINE 16x16 lattice (4px buckets), and logs comparative
// diagnostics for a human to read, not a pass/fail cohesion threshold --
// nobody has verified in advance what the "right" answer is.
//
// Hand-computed going in (bucket = floor(coord / bucketSize)): at the
// coarse 16px-bucket resolution, all three roles' ±2px jitter stays inside
// one bucket every frame (base coords 10 and 50 are both far from any
// 16px boundary), so every role collapses to exactly ONE stable graph
// token. At the fine 4px-bucket resolution, both roles' base coordinates
// (10, 50) sit close enough to a 4px boundary that jitter crosses it: since
// jx and jy are both functions of the same frameIdx (not independent),
// only 5 distinct (jx,jy) pairs ever occur per role, which lands each role
// in exactly 3 distinct fine buckets over the 30-frame run (independently
// re-derived, not 4 -- the corner combination where both x and y cross
// their boundary in the same frame never actually occurs here). Either
// way, 3 buckets for what is, by construction, the exact same recurring
// object is the concrete mechanism the referent-instability argument
// predicts -- this test measures what it actually does to the graph,
// instead of assuming.
func TestPerceptionCategoryGranularityAndCohesion(t *testing.T) {
	roles := []blobRole{
		{color: 2, baseX: 10, baseY: 10, jitterSeed: 0},
		{color: 5, baseX: 50, baseY: 50, jitterSeed: 1},
		{color: 7, baseX: 10, baseY: 50, jitterSeed: 2},
	}
	const frameCount = 30

	run := func(cols, rows int) (finalNodes, sleepCount, abstractionsCreated, distinctObs int, peakCohesion float64) {
		ctx := context.Background()
		engine := NewEngine()
		initialNodes := len(engine.Graph.Nodes)
		seen := map[string]bool{}

		for i := 0; i < frameCount; i++ {
			grid := makeBlobFrame(i, roles)
			obs := perception.DescribeGridCells(grid, len(roles), cols, rows)
			seen[obs] = true
			res, err := engine.RunPredictiveCycle(ctx, obs, "investigate the scene", true)
			if err != nil {
				t.Fatalf("cycle %d (cols=%d rows=%d) failed: %v", i, cols, rows, err)
			}
			if res.SleepTriggered {
				sleepCount++
			}
			abstractionsCreated += res.AbstractionsCreated
			if res.MaxCohesionObserved > peakCohesion {
				peakCohesion = res.MaxCohesionObserved
			}
		}

		finalNodes = len(engine.Graph.Nodes) - initialNodes
		distinctObs = len(seen)
		return
	}

	coarseNodes, coarseSleep, coarseAbstractions, coarseDistinctObs, coarsePeakCohesion := run(4, 4)
	fineNodes, fineSleep, fineAbstractions, fineDistinctObs, finePeakCohesion := run(16, 16)

	t.Logf("COARSE (4x4, 16px buckets): %d distinct observation strings, +%d nodes over %d frames, %d sleep cycles, %d abstractions created, peak cohesion=%.4f",
		coarseDistinctObs, coarseNodes, frameCount, coarseSleep, coarseAbstractions, coarsePeakCohesion)
	t.Logf("FINE   (16x16, 4px buckets): %d distinct observation strings, +%d nodes over %d frames, %d sleep cycles, %d abstractions created, peak cohesion=%.4f",
		fineDistinctObs, fineNodes, frameCount, fineSleep, fineAbstractions, finePeakCohesion)

	// This checks raw vocabulary diversity BEFORE any pipeline processing --
	// the one thing actually guaranteed by CellToken's bucket math alone,
	// independent of what Stage 3 compression later does to node counts.
	// Node counts (logged above) are NOT asserted on: Step 9's
	// CompressGraphAbstractions can shrink coarse's count more than fine's
	// (stronger, more consistent co-activation compresses more readily), so
	// "fine >= coarse in final node count" is not actually guaranteed by
	// construction and would be a fragile thing to hard-assert on.
	if fineDistinctObs < coarseDistinctObs {
		t.Fatalf("FAIL (sanity check, not the experiment's real question): expected the fine lattice to produce at least as many distinct observation strings as the coarse one for the same jittering roles, got fine=%d coarse=%d -- CellToken's bucket math may be wrong, re-derive it before trusting the cohesion numbers above", fineDistinctObs, coarseDistinctObs)
	}

	t.Logf("READ THIS, DON'T JUST TRUST THE PASS: this test cannot fail on the actual research question (does finer granularity hurt cohesion) -- it only sanity-checks raw vocabulary diversity, not final node counts (which compression can reshuffle in either direction). The peak cohesion numbers above are the real result; compare them by hand.")
}
