package feataff

import "alphaarc/pkg/actuate"

// --- Stateful control-protocol learning (half ② extended) ---
//
// Single-step CausalMapper learns "control X sets cell (r,c) to W" only when ONE
// action does it. Real interfaces are STATEFUL: arm-a-colour-then-place, move-a-
// cursor-then-press, pick-a-tool-then-apply -- the effect of a step depends on
// prior steps. PlanStatefulActuation searches short action SEQUENCES that achieve
// a specific target cell change, with the same change-pruning that keeps sequence
// search tractable: a prefix is only extended if its last step changed the board
// (an arm that lights a visible selection indicator counts; a dead click does not).
//
// This turns a derived GoalTarget (cell -> wanted colour) into the multi-step
// protocol that realises it -- the piece single-step planning couldn't supply.

// reaches reports whether the target cell already holds the target colour.
func reaches(g actuate.Grid, t actuate.CellChange) bool {
	return t.R >= 0 && t.R < len(g) && t.C >= 0 && t.C < len(g[t.R]) && g[t.R][t.C] == t.To
}

// PlanStatefulActuation searches (to maxDepth) for an action sequence that makes
// target's cell hold target.To, pruning any branch whose last step changed nothing.
// Offline it uses Reset to replay prefixes before trying each extension.
func PlanStatefulActuation(env Env, target actuate.CellChange, cands []actuate.Control, maxDepth int) ([]actuate.Control, bool) {
	var dfs func(prefix []actuate.Control) ([]actuate.Control, bool)
	dfs = func(prefix []actuate.Control) ([]actuate.Control, bool) {
		for _, c := range cands {
			cur := env.Reset()
			for _, p := range prefix {
				cur, _ = env.Step(p)
			}
			after, _ := env.Step(c)
			if gridsEqual(cur, after) {
				continue // PRUNE: a step that changes nothing is a dead end
			}
			seq := append(append([]actuate.Control{}, prefix...), c)
			if reaches(after, target) {
				return seq, true
			}
			if len(seq) < maxDepth {
				if r, ok := dfs(seq); ok {
					return r, true
				}
			}
		}
		return nil, false
	}
	return dfs(nil)
}
