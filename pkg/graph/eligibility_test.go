package graph

import (
	"protaxon/pkg/core"
	"testing"
)

func buildTwoNodeEdge() *Graph {
	g := NewGraph()
	g.AddNode(NewNode(1, 0.1, 0))
	g.AddNode(NewNode(2, 0.1, 0))
	g.AddEdge(1, 2, 0.5, false)
	return g
}

// TestEligibilityTraceAccumulatesAndPersists verifies traces build up when
// an edge's endpoints co-fire, and survive (rather than reset) on a later
// tick where neither endpoint fires -- the persistence that makes temporal
// credit assignment possible at all.
func TestEligibilityTraceAccumulatesAndPersists(t *testing.T) {
	sys := core.NewSystem()
	g := buildTwoNodeEdge()

	g.Nodes[1].Activation = 0.8
	g.Nodes[2].Activation = 0.6
	g.UpdateEligibilityTraces(sys)

	got := g.Nodes[1].Edges[2].Eligibility
	want := 0.8 * 0.6
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("FAIL: expected trace %.4f after co-firing, got %.4f", want, got)
	}

	g.DecayEligibilityTraces(sys, 0.5)
	afterDecay := g.Nodes[1].Edges[2].Eligibility
	if diff := afterDecay - want*0.5; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("FAIL: expected trace halved to %.4f, got %.4f", want*0.5, afterDecay)
	}

	// Neither endpoint fires this tick -- trace must persist, not reset.
	g.Nodes[1].Activation = 0
	g.Nodes[2].Activation = 0
	g.UpdateEligibilityTraces(sys)

	stillThere := g.Nodes[1].Edges[2].Eligibility
	if diff := stillThere - afterDecay; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("FAIL: trace must persist across a non-firing tick, got %.4f want %.4f", stillThere, afterDecay)
	}

	t.Logf("Eligibility Trace PASS: fired=%.4f -> decayed=%.4f -> persisted=%.4f", got, afterDecay, stillThere)
}

// TestHebbianUpdateWithEligibilityCreditsPastFiring is the key differentiator:
// an edge fires on tick 1, then goes quiet (Activation=0) on tick 2 when the
// reward actually lands. Plain HebbianUpdate must skip it entirely (its
// pre*post gate is zero); HebbianUpdateWithEligibility must still credit it
// via the residual trace. This is what "temporal credit assignment" means
// concretely, not just in the abstract.
func TestHebbianUpdateWithEligibilityCreditsPastFiring(t *testing.T) {
	sys := core.NewSystem()

	plain := buildTwoNodeEdge()
	traced := buildTwoNodeEdge()

	// Tick 1: both fire.
	for _, g := range []*Graph{plain, traced} {
		g.Nodes[1].Activation = 1.0
		g.Nodes[2].Activation = 1.0
	}
	traced.DecayEligibilityTraces(sys, 0.7) // no-op on a fresh trace, but exercises real call order
	traced.UpdateEligibilityTraces(sys)

	// Tick 2: reward arrives, but neither node is active anymore.
	for _, g := range []*Graph{plain, traced} {
		g.Nodes[1].Activation = 0
		g.Nodes[2].Activation = 0
	}
	traced.DecayEligibilityTraces(sys, 0.7)
	traced.UpdateEligibilityTraces(sys) // no new firing; trace stays at 1.0*0.7

	reward := 1.0
	learningRate := 0.1
	plainBefore := plain.Nodes[1].Edges[2].Weight
	tracedBefore := traced.Nodes[1].Edges[2].Weight

	plain.HebbianUpdate(sys, learningRate, reward, 10.0)
	traced.HebbianUpdateWithEligibility(sys, learningRate, reward, 10.0)

	plainAfter := plain.Nodes[1].Edges[2].Weight
	tracedAfter := traced.Nodes[1].Edges[2].Weight

	if plainAfter != plainBefore {
		t.Fatalf("FAIL: plain HebbianUpdate should be a no-op when Activation=0 this tick, weight moved %.6f -> %.6f", plainBefore, plainAfter)
	}

	wantDelta := learningRate * reward * (1.0 * 0.7)
	gotDelta := tracedAfter - tracedBefore
	if diff := gotDelta - wantDelta; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("FAIL: expected eligibility-driven delta %.6f, got %.6f", wantDelta, gotDelta)
	}

	t.Logf("Temporal Credit Assignment PASS: plain Hebbian frozen at %.4f (correct no-op); eligibility-driven moved %.4f -> %.4f via residual trace",
		plainAfter, tracedBefore, tracedAfter)
}

// TestEligibilityTraceAccumulatesAcrossRepeatedFiring verifies the trace
// genuinely accumulates (Sutton & Barto "accumulating" trace) rather than
// just replacing its previous value: firing twice in a short window must
// leave a larger trace than firing once.
func TestEligibilityTraceAccumulatesAcrossRepeatedFiring(t *testing.T) {
	sys := core.NewSystem()
	g := buildTwoNodeEdge()

	g.Nodes[1].Activation = 1.0
	g.Nodes[2].Activation = 1.0
	g.UpdateEligibilityTraces(sys) // trace = 1.0
	g.DecayEligibilityTraces(sys, 0.7)
	g.UpdateEligibilityTraces(sys) // trace = 0.7 + 1.0 = 1.7

	got := g.Nodes[1].Edges[2].Eligibility
	want := 0.7 + 1.0
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("FAIL: expected accumulating trace %.4f after 2 firings, got %.4f", want, got)
	}
	if got <= 1.0 {
		t.Fatalf("FAIL: repeated firing must accumulate above a single firing's trace (1.0), got %.4f", got)
	}

	t.Logf("Accumulating Trace PASS: two firings within decay window -> trace=%.4f (> single-firing 1.0)", got)
}
