package pipeline

import (
	"protaxon/pkg/graph"
	"context"
	"testing"
)

func TestPredictiveLoopPureNeuromorphic(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	t.Logf("=== RUNNING PURE NEUROMORPHIC (ZERO-LLM) PREDICTIVE CYCLE TEST ===")

	for step := 1; step <= 5; step++ {
		success := (step != 3)
		res, err := engine.RunPredictiveCycle(ctx, "Observation: sensory input stimulus", "Achieve target balance", success)
		if err != nil {
			t.Fatalf("Step %d failed: %v", step, err)
		}

		t.Logf("Step %d | Success=%t | PredErr=%.4f | DriveErr=%.4f | SleepTriggered=%t | PredTrust=%.2f ActTrust=%.2f",
			res.StepIndex, success, res.PredictionError, res.DriveError, res.SleepTriggered,
			engine.Predictor.TrustScore(), engine.Actor.TrustScore())

		if res.StepIndex == 5 && !res.SleepTriggered {
			t.Fatalf("Expected Subconscious Sleep to trigger on Step 5!")
		}
	}

	t.Logf("Pure Neuromorphic Predictive Loop Test PASS: 5 steps completed successfully!")
}

// TestActiveSubgraphFromSpreadingActivation verifies the wired-up
// Observation -> Node Lookup -> Spreading Activation -> Sub-graph Extraction
// pipeline (Stage 2): an observation whose text matches registered concept
// labels must seed and spread real graph activation into ActiveNodeIDs,
// while an observation with no label matches must yield an empty subgraph
// instead of falling back to stale leftover node state.
func TestActiveSubgraphFromSpreadingActivation(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	res, err := engine.RunPredictiveCycle(ctx, "Sensory Stimulus: Log anomaly pattern detected", "Isolate faulty node", true)
	if err != nil {
		t.Fatalf("Step failed: %v", err)
	}
	if len(res.ActiveNodeIDs) == 0 {
		t.Fatalf("FAIL: expected non-empty ActiveNodeIDs for observation matching 'stimulus'/'anomaly' labels, got %v", res.ActiveNodeIDs)
	}
	t.Logf("Matched observation -> ActiveNodeIDs=%v", res.ActiveNodeIDs)

	engine2 := NewEngine()
	res2, err := engine2.RunPredictiveCycle(ctx, "Unrelated text with no registered concept keywords", "Noop", true)
	if err != nil {
		t.Fatalf("Step failed: %v", err)
	}
	if len(res2.ActiveNodeIDs) != 0 {
		t.Fatalf("FAIL: expected empty ActiveNodeIDs for observation with no label matches, got %v", res2.ActiveNodeIDs)
	}
	t.Logf("Unmatched observation -> ActiveNodeIDs=%v (expected empty)", res2.ActiveNodeIDs)
}

// TestStage3AbstractionCompressionOnSleep verifies the Stage 3 wiring end to
// end inside the real engine: the seed demo graph (nodes 1-2-3, chain
// cohesion 0.65) must stay untouched by the 0.75 compression threshold on
// its own, while a densely interconnected cluster injected into the same
// graph (cohesion 0.9, fully disconnected from 1-2-3) must collapse into one
// abstraction node the moment Subconscious Sleep triggers at Step 5.
//
// The observation text deliberately matches none of the engine's registered
// labels, so LookupSeeds seeds nothing, SpreadingActivation short-circuits
// before ever calling Propagate, and no node's Activation ever leaves its
// 0.0 default. With every node inactive, UpdateEligibilityTraces never bumps
// any edge's trace, so HebbianUpdateWithEligibility's `Eligibility == 0`
// guard skips every edge on every cycle. That keeps 1->2 and 2->3 pinned at
// their exact seed weights (0.7, 0.6 -> cohesion 0.65) instead of leaving it
// to chance whether 5 cycles of plasticity could drift the chain's cohesion
// across the 0.75 threshold.
func TestStage3AbstractionCompressionOnSleep(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	for i := 10; i < 15; i++ {
		engine.Graph.AddNode(graph.NewNode(i, 0.1, 0))
	}
	for i := 10; i < 15; i++ {
		for j := 10; j < 15; j++ {
			if i != j {
				engine.Graph.AddEdge(i, j, 0.9, false)
			}
		}
	}

	if len(engine.Graph.Nodes) != 8 {
		t.Fatalf("FAIL: expected 8 seed nodes (3 original + 5 injected), got %d", len(engine.Graph.Nodes))
	}

	var res *CycleResult
	var err error
	for step := 1; step <= 5; step++ {
		res, err = engine.RunPredictiveCycle(ctx, "Idle background maintenance tick", "Achieve target balance", true)
		if err != nil {
			t.Fatalf("Step %d failed: %v", step, err)
		}
	}

	if !res.SleepTriggered {
		t.Fatalf("FAIL: expected Subconscious Sleep to trigger on Step 5")
	}
	if res.AbstractionsCreated != 1 {
		t.Fatalf("FAIL: expected exactly 1 abstraction node created from the injected cluster, got %d", res.AbstractionsCreated)
	}
	if res.NodesAbsorbed != 5 {
		t.Fatalf("FAIL: expected 5 injected nodes absorbed, got %d", res.NodesAbsorbed)
	}

	for i := 1; i <= 3; i++ {
		if _, exists := engine.Graph.Nodes[i]; !exists {
			t.Fatalf("FAIL: original seed node %d (chain cohesion 0.65 < threshold 0.75) must survive uncompressed", i)
		}
	}
	for i := 10; i < 15; i++ {
		if _, exists := engine.Graph.Nodes[i]; exists {
			t.Fatalf("FAIL: injected cluster node %d (cohesion 0.9 >= threshold 0.75) should have been absorbed", i)
		}
	}
	if len(engine.Graph.Nodes) != 4 {
		t.Fatalf("FAIL: expected 4 remaining nodes (3 original + 1 abstraction), got %d", len(engine.Graph.Nodes))
	}

	t.Logf("Stage 3 PASS: injected cohesive cluster (0.90) compressed to 1 abstraction node; seed chain (0.65) left untouched. Final node count=%d",
		len(engine.Graph.Nodes))
}

// TestDynamicCompetitionRouterNarrowsActiveSubgraph verifies the Stage 2
// router wiring end to end: the engine's 3 seed nodes all share ClusterID 0
// by construction, so an observation that seeds more than one of them (here
// "stimulus" -> node 1 and "anomaly" -> node 2, which 3-hop spreading also
// carries to node 3) must be narrowed down to exactly 1 surviving winner --
// proving RouteCompetingClusters actually ran, not just that some nodes
// happened to clear the activation threshold.
func TestDynamicCompetitionRouterNarrowsActiveSubgraph(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	res, err := engine.RunPredictiveCycle(ctx, "Sensory Stimulus: Log anomaly pattern detected", "Isolate faulty node", true)
	if err != nil {
		t.Fatalf("Step failed: %v", err)
	}

	if len(res.ActiveNodeIDs) != 1 {
		t.Fatalf("FAIL: expected exactly 1 winner after same-cluster competition, got %v", res.ActiveNodeIDs)
	}

	t.Logf("Dynamic Competition Router PASS: multiple same-cluster candidates narrowed to single winner %v", res.ActiveNodeIDs)
}

// TestEligibilityTracesAccumulateDuringRealCycle verifies the Stage 2
// eligibility-trace wiring end to end: after a cycle where nodes actually
// activate, at least one edge among the co-active nodes must carry a nonzero
// Eligibility trace afterward -- proving DecayEligibilityTraces /
// UpdateEligibilityTraces / HebbianUpdateWithEligibility are genuinely being
// invoked from RunPredictiveCycle, not dead code sitting unused in pkg/graph.
func TestEligibilityTracesAccumulateDuringRealCycle(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	_, err := engine.RunPredictiveCycle(ctx, "Sensory Stimulus: Log anomaly pattern detected", "Isolate faulty node", true)
	if err != nil {
		t.Fatalf("Step failed: %v", err)
	}

	found := false
	for _, node := range engine.Graph.Nodes {
		for _, edge := range node.Edges {
			if edge.Eligibility != 0 {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("FAIL: expected at least one edge with a nonzero Eligibility trace after a cycle with active nodes")
	}

	t.Logf("Eligibility Trace Wiring PASS: at least one edge accumulated a nonzero trace during the real predictive cycle")
}
