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
// board, each tried once from a fresh reset (clean attribution). This is the
// game-agnostic answer to "the goal cell is not the clickable trigger": discover
// the real actuators by causation, not by how salient they look. Order preserved,
// deduped by the caller's candidate list.
func ReachableControls(env Env, cands []actuate.Control) []actuate.Control {
	var out []actuate.Control
	for _, c := range cands {
		before := env.Reset()
		after, _ := env.Step(c)
		if !gridsEqual(before, after) {
			out = append(out, c)
		}
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
