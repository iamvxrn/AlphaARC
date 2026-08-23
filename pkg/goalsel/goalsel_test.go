package goalsel

import "testing"

// probeExp returns a fixed set of interventions (the experiments the causal
// mapper would run). Features are ENTANGLED -- no intervention isolates one --
// but the coupling RATIO varies across interventions, which is enough to
// attribute causality. Hidden reward rule: fk drove it, fdist is a correlate.
type probeExp struct{ ivs []Intervention }

func (e probeExp) Probe(confounded []string) []Intervention { return e.ivs }

// Capstone: a distractor rose in lock-step with the true feature into a reward
// (confounded), and NO experiment can move either alone -- yet varied coupling
// ratios let the selector attribute causation to fk.
func TestGoalSelector_ResolvesEntangledCorrelationVsCausation(t *testing.T) {
	s := New([]string{"fk", "fdist", "fnoise"}, 5, 3.0)

	// PASSIVE: fk and fdist rise together 0->5 (synchronous) into a reward.
	for step := 1; step <= 5; step++ {
		s.Observe(map[string]float64{"fk": float64(step), "fdist": float64(step), "fnoise": 0}, step == 5)
	}
	tied, ok := s.Confounded()
	if !ok || !contains(tied, "fk") || !contains(tied, "fdist") {
		t.Fatalf("expected fk/fdist confounded from synchronous movement, got %v ok=%v", tied, ok)
	}

	// ACTIVE: entangled interventions with DIFFERENT ratios (never isolated):
	//   both move + reward; fdist-heavy + no reward; fk-heavy + reward.
	exp := probeExp{ivs: []Intervention{
		{Deltas: map[string]float64{"fk": 5, "fdist": 5, "fnoise": 0}, Reward: true},
		{Deltas: map[string]float64{"fk": 1, "fdist": 5, "fnoise": 0}, Reward: false},
		{Deltas: map[string]float64{"fk": 5, "fdist": 1, "fnoise": 0}, Reward: true},
	}}
	if !s.Disambiguate(exp) {
		t.Fatal("Disambiguate should have run on the confound set")
	}

	feat, dir, ok := s.Goal()
	if !ok || feat != "fk" {
		t.Fatalf("goal should resolve to fk (Δfk bigger when rewarded), got %q hyps=%v", feat, s.Hypotheses())
	}
	if dir != 1 {
		t.Fatalf("direction should be +1, got %d", dir)
	}
	var ck, cd float64
	for _, h := range s.Hypotheses() {
		if h.Feature == "fk" {
			ck = h.Credit
		}
		if h.Feature == "fdist" {
			cd = h.Credit
		}
	}
	if ck <= cd {
		t.Fatalf("fk credit (%.2f) must exceed spurious fdist (%.2f)", ck, cd)
	}
}

// When two features are PERFECTLY entangled (constant ratio across every
// intervention -- no experiment ever changes their proportion) they are
// operationally indistinguishable, so the selector MERGES them instead of
// looping forever.
func TestGoalSelector_MergesInseparableFeatures(t *testing.T) {
	s := New([]string{"fa", "fb"}, 4, 2.0)
	for step := 1; step <= 4; step++ {
		s.Observe(map[string]float64{"fa": float64(step), "fb": float64(2 * step)}, step == 4)
	}
	if _, ok := s.Confounded(); !ok {
		t.Fatal("fa/fb should be confounded")
	}
	// Every intervention keeps the ratio fa:fb = 1:2 -> inseparable.
	exp := probeExp{ivs: []Intervention{
		{Deltas: map[string]float64{"fa": 2, "fb": 4}, Reward: true},
		{Deltas: map[string]float64{"fa": 3, "fb": 6}, Reward: true},
	}}
	if !s.Disambiguate(exp) {
		t.Fatal("Disambiguate should act")
	}
	if len(s.Merged()) == 0 {
		t.Fatal("inseparable features must be merged, not left dangling")
	}
	feat, _, ok := s.Goal()
	if !ok || feat != "fa+fb" {
		t.Fatalf("goal should be the merged feature 'fa+fb', got %q merged=%v", feat, s.Merged())
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
			t.Fatalf("inert feature must never be credited, got %.1f", h.Credit)
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
