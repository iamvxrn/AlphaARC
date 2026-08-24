package feataff

import (
	"strings"

	"alphaarc/pkg/actuate"
)

// sigOf is a cheap content signature: an action leaving it unchanged is inert.
func sigOf(g actuate.Grid) string {
	var b strings.Builder
	for _, row := range g {
		for _, v := range row {
			b.WriteByte(byte('0' + v%64))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

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

// PlanStatefulActuationLive achieves target in a SINGLE continuous episode -- one
// Reset, then greedy forward steps, NO reset-per-branch (so it fits a cumulative
// live budget where the offline reset search can't run). Forward pruning by grid
// signature: an inert step keeps the signature and is never retried from that state;
// a state-changing step reaches a new signature where actions are available again,
// so an arm->place chain forms. Returns steps spent and whether the target was
// reached within budget.
func PlanStatefulActuationLive(env Env, target actuate.CellChange, cands []actuate.Control, budget int) (steps int, ok bool) {
	cur := env.Reset()
	if reaches(cur, target) {
		return 0, true
	}
	tried := map[string]map[actuate.Control]bool{}
	for steps < budget {
		s := sigOf(cur)
		if tried[s] == nil {
			tried[s] = map[actuate.Control]bool{}
		}
		var chosen actuate.Control
		found := false
		for _, c := range cands {
			if !tried[s][c] {
				chosen, found = c, true
				break
			}
		}
		if !found {
			return steps, false // dead end: everything from this state tried
		}
		tried[s][chosen] = true
		after, _ := env.Step(chosen)
		steps++
		if reaches(after, target) {
			return steps, true
		}
		cur = after // inert -> same sig -> chosen stays tried; changer -> new sig
	}
	return steps, false
}
