package goalsel

import "testing"

// groundTruthExp is the synthetic environment's causal oracle. The HIDDEN reward
// rule is: reward iff feature "fk" reaches the threshold. "fdist" is a
// distractor that (during normal play) moved in lock-step with fk but has NO
// causal role. Intervene isolates a single feature from a fresh reset.
type groundTruthExp struct {
	threshold float64
}

func (e groundTruthExp) Intervene(move string, keep []string) (reward bool, ok bool) {
	// From a fresh reset (fk=0, fdist=0), an isolating control pushes only `move`
	// to the threshold. Reward fires iff fk ended up at/over threshold.
	switch move {
	case "fk":
		return true, true // moving fk alone crosses the hidden threshold -> reward
	case "fdist":
		return false, true // moving fdist alone leaves fk at 0 -> no reward
	default:
		return false, true
	}
}

// Capstone: a distractor moved synchronously with the true goal feature right
// before the reward; the selector must NOT permanently glue them, but resolve
// causation by intervention.
func TestGoalSelector_ResolvesCorrelationVsCausation(t *testing.T) {
	s := New([]string{"fk", "fdist", "fnoise"}, 5, 3.0)

	// PASSIVE PHASE: two episodes of correlated play. fk and fdist rise together
	// 0->5 over 5 steps (synchronous!), fnoise wanders without net movement;
	// reward fires on the step fk hits 5. So fk and fdist get credited together.
	for ep := 0; ep < 2; ep++ {
		for step := 1; step <= 5; step++ {
			v := float64(step)
			noise := 0.0
			if step%2 == 0 {
				noise = 1.0 // jitters but net delta over the window ~0
			}
			s.Observe(map[string]float64{"fk": v, "fdist": v, "fnoise": noise}, step == 5)
		}
		// reset trajectories between episodes by feeding a fresh baseline
		s.Observe(map[string]float64{"fk": 0, "fdist": 0, "fnoise": 0}, false)
	}

	// After correlated rewards, fk and fdist are CONFOUNDED (tied), goal ambiguous.
	tied, ok := s.Confounded()
	if !ok {
		t.Fatalf("expected a confound set from synchronous movement, got none (hyps=%v)", s.Hypotheses())
	}
	if !contains(tied, "fk") || !contains(tied, "fdist") {
		t.Fatalf("confound set should tie fk and fdist, got %v", tied)
	}

	// ACTIVE PHASE: break the tie by intervention.
	if !s.Disambiguate(groundTruthExp{threshold: 5}) {
		t.Fatal("Disambiguate should have run interventions on the confound set")
	}

	// The selector must now name fk (causal), not fdist (spurious correlate).
	feat, dir, ok := s.Goal()
	if !ok || feat != "fk" {
		t.Fatalf("goal should resolve to fk after intervention, got %q (ok=%v) hyps=%v", feat, ok, s.Hypotheses())
	}
	if dir != 1 {
		t.Fatalf("goal direction should be +1 (increase fk), got %d", dir)
	}
	// fdist must have been demoted below fk.
	var ck, cd float64
	for _, h := range s.Hypotheses() {
		if h.Feature == "fk" {
			ck = h.Credit
		}
		if h.Feature == "fdist" {
			cd = h.Credit
		}
	}
	if !(ck > cd) {
		t.Fatalf("fk credit (%.1f) must exceed the spurious fdist (%.1f) after intervention", ck, cd)
	}
}

// A pure distractor that never moves toward reward is never credited.
func TestGoalSelector_IgnoresInertFeature(t *testing.T) {
	s := New([]string{"fk", "fstatic"}, 3, 2.0)
	for step := 1; step <= 3; step++ {
		s.Observe(map[string]float64{"fk": float64(step), "fstatic": 0}, step == 3)
	}
	feat, _, ok := s.Goal()
	if !ok || feat != "fk" {
		t.Fatalf("only fk moved, goal should be fk, got %q ok=%v", feat, ok)
	}
	for _, h := range s.Hypotheses() {
		if h.Feature == "fstatic" && h.Credit != 0 {
			t.Fatalf("an inert feature must never be credited, got %.1f", h.Credit)
		}
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
