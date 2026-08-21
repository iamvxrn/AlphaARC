package macro

import "testing"

// --- Sterile mock game -------------------------------------------------------
//
// A deterministic ARC-like environment: the only action is to click a background
// cell, which paints it the foreground colour. The environment's hidden goal is
// "become compressible"; the agent is NEVER told this. Its ONLY drive is
// DrivePreference (bits saved by the best primitive). We prove the PHYSICS of
// that drive in isolation — free of A*, perception noise, or the real API — so
// that if the live run later stalls we already know the drive itself is sound.
//
// The crux is the sparse gradient: from a blank grid a single click saves 0 bits
// (a lone cell has no mirror partner), so a 1-step (greedy) agent sees a flat
// landscape and cannot move. Only a 2-step planner can see "click here, then its
// mirror -> +1" and act. That planning is the Active Inference half of the loop.

// lookaheadValue is the best DrivePreference reachable within `depth` click-moves
// (depth 0 = the grid as-is). The agent may also stop early, so the value is
// monotone in depth.
func lookaheadValue(g [][]int, bg, fg, depth int) int {
	v := DrivePreference(g, bg)
	if depth <= 0 {
		return v
	}
	for r := range g {
		for c := range g[r] {
			if g[r][c] != bg {
				continue
			}
			ng := cloneGrid(g)
			ng[r][c] = fg
			if vv := lookaheadValue(ng, bg, fg, depth-1); vv > v {
				v = vv
			}
		}
	}
	return v
}

// bestMove returns the click that maximizes the value reachable within `horizon`
// moves. ok=false means no click improves on standing still (a local optimum).
func bestMove(g [][]int, bg, fg, horizon int) (int, int, bool) {
	best := DrivePreference(g, bg)
	br, bc := -1, -1
	for r := range g {
		for c := range g[r] {
			if g[r][c] != bg {
				continue
			}
			ng := cloneGrid(g)
			ng[r][c] = fg
			if v := lookaheadValue(ng, bg, fg, horizon-1); v > best {
				best, br, bc = v, r, c
			}
		}
	}
	return br, bc, br != -1
}

// runAgent drives the mock loop with a planner of the given horizon until it is
// stuck or the budget runs out, returning the final grid and the savings trace.
func runAgent(start [][]int, bg, fg, horizon, budget int) ([][]int, []int) {
	g := cloneGrid(start)
	trace := []int{DrivePreference(g, bg)}
	for i := 0; i < budget; i++ {
		r, c, ok := bestMove(g, bg, fg, horizon)
		if !ok {
			break
		}
		g[r][c] = fg
		trace = append(trace, DrivePreference(g, bg))
	}
	return g, trace
}

// Proof 1: a greedy (1-step) agent is STUCK on the sparse gradient — it makes no
// progress from blank, because no single click saves a bit.
func TestActiveInference_GreedyStuckOnSparseGradient(t *testing.T) {
	blank := [][]int{
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}
	final, trace := runAgent(blank, 0, 1, /*horizon*/ 1, /*budget*/ 20)
	t.Logf("greedy trace: %v", trace)
	if fg := countForeground(final, 0); fg != 0 {
		t.Fatalf("greedy agent should be stuck at blank, but placed %d cells", fg)
	}
	if DrivePreference(final, 0) != 0 {
		t.Fatalf("greedy agent should reach 0 savings, got %d", DrivePreference(final, 0))
	}
}

// Proof 2: a 2-step planner CROSSES the sparse gradient — driven only by
// compression savings, with no symmetry scorer and no target, it builds
// compressible (symmetric) structure from the same blank start.
func TestActiveInference_LookaheadCrossesSparseGradient(t *testing.T) {
	blank := [][]int{
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}
	final, trace := runAgent(blank, 0, 1, /*horizon*/ 2, /*budget*/ 20)
	t.Logf("lookahead trace: %v", trace)
	t.Logf("final grid: %v", final)

	if DrivePreference(final, 0) <= 0 {
		t.Fatalf("lookahead agent failed to earn any compression savings")
	}
	if countForeground(final, 0) == 0 {
		t.Fatalf("lookahead agent placed nothing")
	}
	if !isHSymmetric(final) {
		t.Fatalf("drive should build symmetric (compressible) structure:\n%v", final)
	}
	// The trace must be strictly better than the greedy one: it left 0.
	if trace[len(trace)-1] <= 0 {
		t.Fatalf("final savings must exceed the greedy dead-end of 0")
	}
}
