package feataff

import "alphaarc/pkg/actuate"

// --- Control REACHABILITY: causal control discovery ---
//
// The perceptual candidate set (ResidualControls: residual + object centroids)
// assumes the actuator LOOKS salient -- the anomaly cell or a visible blob. On
// several real games that assumption fails: the compression-relevant cell is NOT
// the clickable trigger (vc33's grow button is off the pattern; ft09's mismatched
// block is inert to a direct click; a hidden button changes the board from a bg
// cell). When the salient controls move no feature, the honest fix is to stop
// trusting saliency and trust OBSERVED CAUSATION: sweep coarsely and keep only
// the clicks that actually change the board -- those are the real actuators, and
// only they can possibly move a goal feature.

// SweepControls returns a coarse grid of click candidates at the given spacing,
// always including the last row and column so edge/corner triggers aren't missed.
// This is the un-prioritised candidate pool for causal discovery, NOT a play
// policy -- it is filtered by ReachableControls before anything acts on it.
func SweepControls(g actuate.Grid, step int) []actuate.Control {
	if step < 1 {
		step = 1
	}
	h := len(g)
	if h == 0 {
		return nil
	}
	w := len(g[0])
	seen := map[[2]int]bool{}
	var cs []actuate.Control
	add := func(x, y int) {
		if x < 0 || y < 0 || y >= h || x >= w {
			return
		}
		k := [2]int{x, y}
		if seen[k] {
			return
		}
		seen[k] = true
		cs = append(cs, actuate.Control{Kind: "click", X: x, Y: y})
	}
	for r := 0; r < h; r += step {
		for c := 0; c < w; c += step {
			add(c, r)
		}
		add(w-1, r) // last column of this row
	}
	for c := 0; c < w; c += step { // last row
		add(c, h-1)
	}
	add(w-1, h-1) // bottom-right corner
	return cs
}

// ReachableControls returns the subset of cands whose click actually changes the
// board -- the real actuators, discovered by causation rather than by how salient
// they look (the game-agnostic answer to "the goal cell is not the clickable
// trigger"). It sweeps SEQUENTIALLY from a SINGLE reset (1 Reset + N Steps, like
// real play) rather than resetting per candidate: on a live game with a bounded
// budget the RESET is the expensive/limited op, and a per-candidate reset sweep
// exhausts the session (observed killing ft09 mid-run). Accumulation is fine for
// DISCOVERY -- a click that changes the board is reachable regardless of prior
// state; clean per-feature attribution is done afterwards by a small Explore over
// just the discovered actuators. Order preserved.
func ReachableControls(env Env, cands []actuate.Control) []actuate.Control {
	prev := env.Reset()
	var out []actuate.Control
	for _, c := range cands {
		after, _ := env.Step(c)
		if !gridsEqual(prev, after) {
			out = append(out, c)
		}
		prev = after
	}
	return out
}

func gridsEqual(a, b actuate.Grid) bool {
	if len(a) != len(b) {
		return false
	}
	for r := range a {
		if len(a[r]) != len(b[r]) {
			return false
		}
		for c := range a[r] {
			if a[r][c] != b[r][c] {
				return false
			}
		}
	}
	return true
}
