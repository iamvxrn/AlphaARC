package feataff

import "alphaarc/pkg/actuate"

// --- Sequence actuation: temporal depth with change-pruning ---
//
// Single-control pursuit is exhausted (the live scan: no single click cracks any
// game but vc33). The hard games are STATEFUL: arm-then-apply, step-onto-then-
// press, multi-step assembly -- the reward needs a SEQUENCE of actions, not one.
//
// The trap is combinatorial explosion: on a ~30-150 action live budget, blind
// 2-step search dies before it finds reward. The pruning rule that makes it
// tractable (temporal reachability): a prefix may only be EXTENDED if its final
// step caused a MEASURABLE change (a board diff -- which, for pure-grid readout
// features, subsumes any feature-readout delta). A step that changes nothing (a
// wall, an empty click, an unarmed apply) is a dead end and its whole subtree is
// pruned instantly. So the branching factor collapses to "actions that actually
// do something", exactly the set worth chaining.
//
// This offline searcher uses Reset to try alternative branches (free offline) so
// the macro-action math can be debugged without budget noise; the pruning is the
// same logic the live single-episode driver will use to stay within budget.

// SeqResult is the outcome of a sequence search.
type SeqResult struct {
	Steps  []actuate.Control // the winning sequence (nil if none)
	Won    bool              // reward reached
	Probes int               // real env.Step calls made (the live-budget cost proxy)
}

// DiscoverSequence searches action sequences up to maxDepth for one that reaches
// the sparse reward, pruning any branch whose last step caused no change. cands is
// the candidate action set (reachable/ salient controls). Offline: Reset is used
// to replay a prefix before trying each extension.
func DiscoverSequence(env Env, feats []Feature, cands []actuate.Control, maxDepth int) SeqResult {
	probes := 0

	// replay resets and re-applies prefix, returning the grid after the prefix.
	replay := func(prefix []actuate.Control) actuate.Grid {
		cur := env.Reset()
		for _, p := range prefix {
			cur, _ = env.Step(p)
			probes++
		}
		return cur
	}

	var dfs func(prefix []actuate.Control) (SeqResult, bool)
	dfs = func(prefix []actuate.Control) (SeqResult, bool) {
		for _, c := range cands {
			before := replay(prefix)
			after, reward := env.Step(c)
			probes++
			if gridsEqual(before, after) {
				continue // PRUNE: zero-change step is a dead end, do not extend it
			}
			seq := append(append([]actuate.Control{}, prefix...), c)
			if reward {
				return SeqResult{Steps: seq, Won: true, Probes: probes}, true
			}
			if len(seq) < maxDepth {
				if res, ok := dfs(seq); ok {
					return res, true
				}
			}
		}
		return SeqResult{}, false
	}

	if res, ok := dfs(nil); ok {
		return res
	}
	return SeqResult{Won: false, Probes: probes}
}
