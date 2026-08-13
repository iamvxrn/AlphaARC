package pipeline

import (
	"protaxon/pkg/goals"
	"protaxon/pkg/graph"
	"context"
	"fmt"
	"math"
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
// Observation -> Node Lookup-or-Create -> Spreading Activation -> Sub-graph
// Extraction -> Router pipeline (Stage 2): an observation whose text matches
// (or, since organic graph growth was added, creates) concept nodes must
// seed and spread real graph activation into a non-empty ActiveNodeIDs,
// while an observation with genuinely no meaningful content (empty string --
// no tokens survive tokenization at all, nothing to create or look up) must
// yield an empty subgraph instead of falling back to stale leftover node
// state. (An observation with unfamiliar-but-meaningful vocabulary no longer
// yields empty here now that EnsureConceptNodes creates nodes for novel
// words too -- see TestEnsureConceptNodesCreatesNewNodesForNovelVocabulary
// in pkg/graph for that behavior specifically.)
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
	res2, err := engine2.RunPredictiveCycle(ctx, "", "Noop", true)
	if err != nil {
		t.Fatalf("Step failed: %v", err)
	}
	if len(res2.ActiveNodeIDs) != 0 {
		t.Fatalf("FAIL: expected empty ActiveNodeIDs for an empty observation (no tokens at all), got %v", res2.ActiveNodeIDs)
	}
	t.Logf("Empty observation -> ActiveNodeIDs=%v (expected empty)", res2.ActiveNodeIDs)
}

// TestStage3AbstractionCompressionOnSleep verifies the Stage 3 wiring end to
// end inside the real engine: the seed demo graph (nodes 1-2-3, chain
// cohesion 0.65) must stay untouched by the 0.75 compression threshold on
// its own, while a densely interconnected cluster injected into the same
// graph (cohesion 0.9, fully disconnected from 1-2-3) must collapse into one
// abstraction node the moment Subconscious Sleep triggers at Step 5.
//
// The observation text is deliberately built entirely from stopwords/short
// tokens ("it is of the"), so EnsureConceptNodes creates nothing, seeds
// nothing, SpreadingActivation short-circuits before ever calling Propagate,
// and no node's Activation ever leaves its 0.0 default. With every node
// inactive, UpdateEligibilityTraces never bumps any edge's trace, so
// HebbianUpdateWithEligibility's `Eligibility == 0` guard skips every edge
// on every cycle. That keeps 1->2 and 2->3 pinned at their exact seed
// weights (0.7, 0.6 -> cohesion 0.65) instead of leaving it to chance
// whether 5 cycles of plasticity could drift the chain's cohesion across the
// 0.75 threshold -- and keeps the node count exactly what it was before the
// loop, since a stopword-only observation can't grow the graph either.
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
		res, err = engine.RunPredictiveCycle(ctx, "it is of the", "Achieve target balance", true)
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

// TestPrimaryClusterPicksHighestActivationWinner verifies the Stage 4 MoE
// gating logic directly: among a set of active node IDs spanning different
// clusters, primaryCluster must return the ClusterID of whichever node has
// the highest current Activation -- not the first, not the lowest ID.
func TestPrimaryClusterPicksHighestActivationWinner(t *testing.T) {
	g := graph.NewGraph()
	g.AddNode(graph.NewNode(1, 0.1, 0))
	g.AddNode(graph.NewNode(2, 0.1, 5))
	g.AddNode(graph.NewNode(3, 0.1, 9))
	g.Nodes[1].Activation = 0.3
	g.Nodes[2].Activation = 0.9
	g.Nodes[3].Activation = 0.5

	got := primaryCluster(g, []int{1, 2, 3})
	if got != 5 {
		t.Fatalf("FAIL: expected cluster 5 (node 2's cluster, highest activation 0.9), got %d", got)
	}

	if empty := primaryCluster(g, []int{}); empty != 0 {
		t.Fatalf("FAIL: expected fallback cluster 0 for no active nodes, got %d", empty)
	}
}

// TestPrimaryClusterFindsWinnerAmongAllNegativeActivations is a regression
// test for the same bug class already found and fixed once in
// winningBlobLabel (pkg/environment/bridge/bridge_click_test.go's
// TestWinningBlobLabelFindsWinnerAmongAllNegativeActivations): a hardcoded
// -1.0 sentinel silently breaks once real activations go more negative than
// that (observed live, roughly -4.12, before edge weights got a floor --
// Node.Activation itself is still unbounded even now). With math.Inf(-1),
// the actual highest (least negative) candidate's cluster must always win.
func TestPrimaryClusterFindsWinnerAmongAllNegativeActivations(t *testing.T) {
	g := graph.NewGraph()
	g.AddNode(graph.NewNode(1, 0.1, 3))
	g.AddNode(graph.NewNode(2, 0.1, 7))
	g.Nodes[1].Activation = -4.12
	g.Nodes[2].Activation = -2.5 // less negative -- must win

	got := primaryCluster(g, []int{1, 2})
	if got != 7 {
		t.Fatalf("FAIL: expected cluster 7 (node 2's cluster, least-negative activation -2.5), got %d -- sentinel regression", got)
	}
}

// TestSpecialistForCreatesDistinctDeterministicAgents verifies the Stage 4
// MoE pool itself: specialistFor must lazily create exactly one specialist
// per distinct ClusterID, cache and reuse it on repeat calls (not create
// duplicates), and give each cluster's specialist a genuinely different,
// deterministic MLP -- not a clone of the cluster-0 generalist.
func TestSpecialistForCreatesDistinctDeterministicAgents(t *testing.T) {
	engine := NewEngine()

	if len(engine.Predictors) != 1 {
		t.Fatalf("FAIL: expected pool of exactly 1 (cluster 0) right after NewEngine, got %d", len(engine.Predictors))
	}

	p5a := engine.specialistFor(5)
	p5b := engine.specialistFor(5)
	if p5a != p5b {
		t.Fatalf("FAIL: repeat calls for the same cluster must return the SAME cached specialist instance")
	}
	if len(engine.Predictors) != 2 {
		t.Fatalf("FAIL: expected pool size 2 (cluster 0 + cluster 5) after one new cluster, got %d", len(engine.Predictors))
	}
	if p5a.ID() != "pred-cluster-5" {
		t.Fatalf("FAIL: expected specialist ID 'pred-cluster-5', got %q", p5a.ID())
	}

	p7 := engine.specialistFor(7)
	if p7 == p5a || p7 == engine.Predictor {
		t.Fatalf("FAIL: cluster 7's specialist must be a distinct instance from cluster 5's and cluster 0's")
	}
	if len(engine.Predictors) != 3 {
		t.Fatalf("FAIL: expected pool size 3 after a second new cluster, got %d", len(engine.Predictors))
	}

	// Deterministic, genuinely different MLPs: same input, different seeds
	// (1005 vs 1007) must not produce identical forward-pass output.
	probe := make([]float64, engine.PredictorDim)
	for i := range probe {
		probe[i] = 1.0
	}
	_, out5 := p5a.MLP.Forward(probe)
	_, out7 := p7.MLP.Forward(probe)
	identical := true
	for i := range out5 {
		if out5[i] != out7[i] {
			identical = false
			break
		}
	}
	if identical {
		t.Fatalf("FAIL: cluster 5 and cluster 7 specialists produced identical output -- seeds aren't actually differentiating them")
	}

	t.Logf("MoE Pool PASS: 3 distinct specialists (cluster 0/5/7), cached correctly, deterministically different MLPs")
}

// TestMoERoutingAcrossClusters verifies the Stage 4 wiring end to end inside
// RunPredictiveCycle: an observation whose winning concept lives in a
// non-default cluster must be handled by that cluster's own specialist (pool
// grows, SpecialistCluster reported correctly), while an observation that
// resolves to nothing active falls back to the cluster-0 generalist.
func TestMoERoutingAcrossClusters(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	engine.Graph.AddNode(graph.NewNode(10, 0.1, 1))
	engine.Graph.AddLabel("quantum", 10)
	engine.Graph.AddNode(graph.NewNode(20, 0.1, 2))
	engine.Graph.AddLabel("photon", 20)

	// Cycles 1 and 2 deliberately share ZERO vocabulary (not even filler
	// words): EnsureConceptNodes creates fresh ClusterID-0 nodes for any
	// novel token, and FormCoActivationEdges wires them to whatever else is
	// active that cycle. A token reused across cycles (e.g. a generic filler
	// word appearing in both observations) would keep a node alive across
	// cycles and let SpreadingActivation's edge-feedback loop pull its
	// activation above the "clean" cluster's, silently reassigning
	// primaryCluster's winner. Distinct vocabulary per cycle avoids that by
	// construction: nothing from cycle 1 is still active/seeded in cycle 2.
	res1, err := engine.RunPredictiveCycle(ctx, "quantum flux zeta", "n/a", true)
	if err != nil {
		t.Fatalf("cycle 1 failed: %v", err)
	}
	if res1.SpecialistCluster != 1 {
		t.Fatalf("FAIL: expected cycle 1 routed to cluster 1 (node 10's cluster), got %d", res1.SpecialistCluster)
	}
	if res1.PredictorPoolSize != 2 {
		t.Fatalf("FAIL: expected pool size 2 after cluster 1's specialist was created, got %d", res1.PredictorPoolSize)
	}

	res2, err := engine.RunPredictiveCycle(ctx, "photon wave gamma", "n/a", true)
	if err != nil {
		t.Fatalf("cycle 2 failed: %v", err)
	}
	if res2.SpecialistCluster != 2 {
		t.Fatalf("FAIL: expected cycle 2 routed to cluster 2 (node 20's cluster), got %d", res2.SpecialistCluster)
	}
	if res2.PredictorPoolSize != 3 {
		t.Fatalf("FAIL: expected pool size 3 after cluster 2's specialist was created, got %d", res2.PredictorPoolSize)
	}

	res3, err := engine.RunPredictiveCycle(ctx, "it is of the", "n/a", true)
	if err != nil {
		t.Fatalf("cycle 3 failed: %v", err)
	}
	if res3.SpecialistCluster != 0 {
		t.Fatalf("FAIL: expected cycle 3 (nothing active) to fall back to cluster 0, got %d", res3.SpecialistCluster)
	}
	if res3.PredictorPoolSize != 3 {
		t.Fatalf("FAIL: expected pool size to stay at 3 (no new cluster needed), got %d", res3.PredictorPoolSize)
	}

	t.Logf("MoE Routing PASS: cluster 1 -> pool=2, cluster 2 -> pool=3, fallback to cluster 0 -> pool stays 3")
}

// TestHierarchicalGoalStackWiring verifies the Stage 4 piece 2 wiring end to
// end inside RunPredictiveCycle: the first cycle must auto-bootstrap a root
// External goal from the flat `goal` string (backward compat for every
// caller that never touches e.Goals directly); an explicitly pushed
// sub-goal must become the new top and receive its own accumulating scope
// (the ClusterIDs actually active while it was being pursued) separately
// from the root; popping it must hand control back to the root goal.
func TestHierarchicalGoalStackWiring(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	// Cycle 1: stack starts empty -> auto-bootstraps a root External goal.
	res1, err := engine.RunPredictiveCycle(ctx, "Sensory Stimulus: system check", "Fix system", true)
	if err != nil {
		t.Fatalf("cycle 1 failed: %v", err)
	}
	if res1.GoalStackDepth != 1 {
		t.Fatalf("FAIL: expected auto-bootstrapped depth 1 after cycle 1, got %d", res1.GoalStackDepth)
	}
	if res1.CurrentGoal != "Fix system" {
		t.Fatalf("FAIL: expected root goal 'Fix system', got %q", res1.CurrentGoal)
	}
	root := engine.Goals.Top()
	if len(root.ScopeClusters) == 0 {
		t.Fatalf("FAIL: expected the root goal to have accumulated at least one active ClusterID from cycle 1")
	}

	// Explicitly push a sub-goal -- real hierarchy, not auto-bootstrapped.
	sub := engine.Goals.Push("Isolate faulty node", goals.TypeInternal, 0.8, engine.StepCounter)
	if engine.Goals.Depth() != 2 {
		t.Fatalf("FAIL: expected depth 2 right after pushing a sub-goal, got %d", engine.Goals.Depth())
	}

	// Cycle 2: goal string repeated on purpose -- with a non-empty stack,
	// Step 0's auto-bootstrap must NOT fire again (no accidental second
	// root goal), and this cycle's activity must land on the SUB-goal.
	res2, err := engine.RunPredictiveCycle(ctx, "Sensory Stimulus: anomaly detected", "Fix system", true)
	if err != nil {
		t.Fatalf("cycle 2 failed: %v", err)
	}
	if res2.GoalStackDepth != 2 {
		t.Fatalf("FAIL: expected depth to stay at 2 (no duplicate auto-bootstrap), got %d", res2.GoalStackDepth)
	}
	if res2.CurrentGoal != "Isolate faulty node" {
		t.Fatalf("FAIL: expected current goal to be the sub-goal 'Isolate faulty node', got %q", res2.CurrentGoal)
	}
	if len(sub.ScopeClusters) == 0 {
		t.Fatalf("FAIL: expected the sub-goal to have accumulated at least one active ClusterID from cycle 2")
	}

	ancestry := engine.Goals.Ancestry()
	if len(ancestry) != 2 || ancestry[0].Description != "Fix system" || ancestry[1].Description != "Isolate faulty node" {
		t.Fatalf("FAIL: expected ancestry [Fix system, Isolate faulty node], got %v", describeGoals(ancestry))
	}

	// Pop the sub-goal -- control returns to the root.
	popped := engine.Goals.Pop()
	if popped.Description != "Isolate faulty node" || !popped.Satisfied {
		t.Fatalf("FAIL: expected to pop the satisfied sub-goal, got %q satisfied=%v", popped.Description, popped.Satisfied)
	}

	res3, err := engine.RunPredictiveCycle(ctx, "Sensory Stimulus: recovery check", "Fix system", true)
	if err != nil {
		t.Fatalf("cycle 3 failed: %v", err)
	}
	if res3.GoalStackDepth != 1 {
		t.Fatalf("FAIL: expected depth back to 1 after popping the sub-goal, got %d", res3.GoalStackDepth)
	}
	if res3.CurrentGoal != "Fix system" {
		t.Fatalf("FAIL: expected control back on the root goal 'Fix system', got %q", res3.CurrentGoal)
	}

	t.Logf("Hierarchical Goal Stack PASS: root(depth1) -> push sub-goal(depth2, scope=%v) -> pop -> back to root(depth1)",
		sub.ScopeClusters)
}

func describeGoals(gs []*goals.Goal) []string {
	out := make([]string, len(gs))
	for i, g := range gs {
		out[i] = g.Description
	}
	return out
}

// TestConflictResolutionAcrossClusters verifies the Stage 4 piece 3 wiring
// (CONCEPT.md Section 16): when two clusters are simultaneously active in
// one cycle, both specialists get to predict, the higher-trust one wins,
// and -- critically -- the losing candidate is NOT discarded, it's kept in
// Engine.RecentConflicts alongside the winner.
func TestConflictResolutionAcrossClusters(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	engine.Graph.AddNode(graph.NewNode(10, 0.1, 1))
	engine.Graph.AddLabel("quantum", 10)
	engine.Graph.AddNode(graph.NewNode(20, 0.1, 2))
	engine.Graph.AddLabel("photon", 20)

	// Give cluster 2's specialist a trust head start so the winner is
	// determined by a value this test controls directly, not by having to
	// hand-simulate MLP confidence output.
	engine.specialistFor(2).SetTrustScore(1.5)

	res, err := engine.RunPredictiveCycle(ctx, "quantum photon", "n/a", true)
	if err != nil {
		t.Fatalf("cycle failed: %v", err)
	}

	if !res.ConflictDetected {
		t.Fatalf("FAIL: expected a conflict to be detected with 2 simultaneously active clusters")
	}
	if res.ConflictCandidates != 2 {
		t.Fatalf("FAIL: expected exactly 2 conflict candidates, got %d", res.ConflictCandidates)
	}
	if res.ConflictWinner != "pred-cluster-2" {
		t.Fatalf("FAIL: expected 'pred-cluster-2' (trust 1.5) to win over 'pred-cluster-1' (trust 1.0), got %q", res.ConflictWinner)
	}

	if len(engine.RecentConflicts) != 1 {
		t.Fatalf("FAIL: expected exactly 1 recorded conflict, got %d", len(engine.RecentConflicts))
	}
	sources := make(map[string]bool)
	for _, c := range engine.RecentConflicts[0].Candidates {
		sources[c.Source] = true
	}
	if !sources["pred-cluster-1"] || !sources["pred-cluster-2"] {
		t.Fatalf("FAIL: expected the record to preserve BOTH candidates (winner and loser), got %v", sources)
	}

	t.Logf("Conflict Resolution PASS: 2 candidates, winner=%s, loser preserved in RecentConflicts (CONCEPT.md Section 16: never erase a losing position)", res.ConflictWinner)
}

// TestNoConflictWhenOnlyOneClusterActive verifies conflict resolution stays
// silent (no false positives) on the overwhelmingly common case: a single
// cluster active, exactly what every other pre-existing test already relies
// on implicitly.
func TestNoConflictWhenOnlyOneClusterActive(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	res, err := engine.RunPredictiveCycle(ctx, "Sensory Stimulus: system check", "n/a", true)
	if err != nil {
		t.Fatalf("cycle failed: %v", err)
	}
	if res.ConflictDetected {
		t.Fatalf("FAIL: expected no conflict when only cluster 0 is active, got ConflictDetected=true (winner=%q)", res.ConflictWinner)
	}
	if len(engine.RecentConflicts) != 0 {
		t.Fatalf("FAIL: expected zero recorded conflicts, got %d", len(engine.RecentConflicts))
	}
}

// TestAshbyShockRecoveryAtScale verifies Stage 4 piece 3's other half: the
// Ashby homeostasis shock-recovery mechanism, already verified in Stage 1
// (pkg/homeostasis/hormones_test.go) on a single-agent engine, must still
// work identically once the specialist pool has grown to non-trivial size.
// Homeostasis is a single shared Engine field, independent of Predictors --
// so this is a real empirical check that scaling the agent population
// doesn't silently break it, not just an assumption.
func TestAshbyShockRecoveryAtScale(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	// Grow the specialist pool first: 8 distinct clusters, one concept each.
	for i := 0; i < 8; i++ {
		clusterID := i + 1
		nodeID := 100 + i
		label := fmt.Sprintf("concept%d", i)
		engine.Graph.AddNode(graph.NewNode(nodeID, 0.1, clusterID))
		engine.Graph.AddLabel(label, nodeID)
		if _, err := engine.RunPredictiveCycle(ctx, label, "grow pool", true); err != nil {
			t.Fatalf("pool-growth cycle %d failed: %v", i, err)
		}
	}
	if len(engine.Predictors) < 9 { // cluster 0 (default) + 8 new
		t.Fatalf("FAIL: expected specialist pool grown to at least 9 before the shock test, got %d", len(engine.Predictors))
	}
	poolSizeAtShockTest := len(engine.Predictors)

	// Baseline: settle with a few successes (mirrors Stage 1's own
	// hormones_test.go setup, just replayed inside the real engine).
	for i := 0; i < 3; i++ {
		if _, err := engine.RunPredictiveCycle(ctx, "it is of the", "steady", true); err != nil {
			t.Fatalf("baseline cycle failed: %v", err)
		}
	}
	baselineCortisol := engine.Homeostasis.Cortisol

	// Shock: a burst of failures.
	for i := 0; i < 3; i++ {
		if _, err := engine.RunPredictiveCycle(ctx, "it is of the", "shock", false); err != nil {
			t.Fatalf("shock cycle failed: %v", err)
		}
	}
	shockedCortisol := engine.Homeostasis.Cortisol
	if shockedCortisol <= baselineCortisol {
		t.Fatalf("FAIL: expected Cortisol to rise after a burst of failures (baseline=%.4f, after shock=%.4f)", baselineCortisol, shockedCortisol)
	}

	// Recovery: sustained success.
	for i := 0; i < 15; i++ {
		if _, err := engine.RunPredictiveCycle(ctx, "it is of the", "recover", true); err != nil {
			t.Fatalf("recovery cycle failed: %v", err)
		}
	}
	recoveredCortisol := engine.Homeostasis.Cortisol
	if recoveredCortisol >= shockedCortisol {
		t.Fatalf("FAIL: expected Cortisol to recover (drop) after sustained success (post-shock=%.4f, after recovery=%.4f)", shockedCortisol, recoveredCortisol)
	}

	t.Logf("Ashby Shock Recovery at Scale PASS: pool size=%d, Cortisol baseline=%.4f -> shocked=%.4f -> recovered=%.4f",
		poolSizeAtShockTest, baselineCortisol, shockedCortisol, recoveredCortisol)
}

// TestOrganicGraphGrowthAndCompression closes the Stage 3 -> Stage 4 gate
// noted in ARCHITECTURE.md: Stage 3 (pkg/offline/abstraction_test.go) was
// previously only verified against hand-injected synthetic clusters. This
// runs the real engine through the same 10 observations used in
// cmd/protaxon-stage1's benchmark demo, twice over (20 cycles, 4
// Subconscious Sleep triggers), letting EnsureConceptNodes and
// FormCoActivationEdges (both new) grow the graph's structure from real,
// repeated experience instead of a single injected clique.
//
// What's hard-asserted is only what's guaranteed by construction, independent
// of the MLP's reward dynamics (already flagged in review as not reliably
// hand-traceable over many steps: reward = 1-predErr can go negative, so
// whether Hebbian plasticity strengthens or weakens any given edge isn't
// something this test can predict in advance): the graph must grow well
// beyond its 3-node seed, since the vocabulary below is far richer than the
// original 8 registered labels, and at least 3 Subconscious Sleep cycles
// must fire (a pure StepCounter%5==0 structural fact). Whether that organic
// structure ever clears Stage 3's 0.75 cohesion threshold on its own is
// reported via t.Logf with the real numbers, not asserted -- that's the
// actual open empirical question this test exists to answer, per Rule #5.
func TestOrganicGraphGrowthAndCompression(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	observations := []struct {
		Observation string
		Goal        string
		Success     bool
	}{
		{"Sensory Stimulus S_1: Log anomaly pattern detected", "Isolate faulty node", true},
		{"Sensory Stimulus S_2: Dynamic graph load shift", "Balance edge weights", true},
		{"Sensory Stimulus S_3: Network delay perturbation", "Mitigate latency spike", false},
		{"Sensory Stimulus S_4: Rerouting via secondary path", "Restore latency parity", true},
		{"Sensory Stimulus S_5: System stabilization tick", "Consolidate graph memory", true},
		{"Sensory Stimulus S_6: Query join request", "Minimize graph energy", true},
		{"Sensory Stimulus S_7: Unindexed edge traversal", "Add index edge", false},
		{"Sensory Stimulus S_8: Re-traversing with index hint", "Achieve fast recall", true},
		{"Sensory Stimulus S_9: Summarize optimization state", "Report stability", true},
		{"Sensory Stimulus S_10: Final consolidation tick", "Maintain homeostatic balance", true},
	}

	initialNodes := len(engine.Graph.Nodes)

	var last *CycleResult
	var err error
	sleepCount := 0
	totalAbstractionsCreated := 0
	peakCohesion := 0.0
	mlpRewardSum := 0.0
	mlpRewardMin := math.Inf(1)
	mlpRewardMax := math.Inf(-1)
	negativeMLPRewardCount := 0
	structuralPositiveCount := 0
	cycleCount := 0
	for pass := 0; pass < 2; pass++ {
		for _, obs := range observations {
			last, err = engine.RunPredictiveCycle(ctx, obs.Observation, obs.Goal, obs.Success)
			if err != nil {
				t.Fatalf("cycle failed (pass %d, %q): %v", pass, obs.Observation, err)
			}
			if last.SleepTriggered {
				sleepCount++
			}
			totalAbstractionsCreated += last.AbstractionsCreated
			if last.MaxCohesionObserved > peakCohesion {
				peakCohesion = last.MaxCohesionObserved
			}

			// mlpReward is diagnostic only now -- it no longer drives Hebbian
			// plasticity (see predictive_loop.go Step 6: structuralReward,
			// derived from obs.Success, does that job since 2026-08-12).
			// Kept here to separately track the Predictor MLP's own health.
			mlpReward := 1.0 - last.PredictionError
			mlpRewardSum += mlpReward
			cycleCount++
			if mlpReward < mlpRewardMin {
				mlpRewardMin = mlpReward
			}
			if mlpReward > mlpRewardMax {
				mlpRewardMax = mlpReward
			}
			if mlpReward < 0 {
				negativeMLPRewardCount++
			}
			if obs.Success {
				structuralPositiveCount++
			}
		}
	}

	finalNodes := len(engine.Graph.Nodes)
	totalEdges := 0
	for _, node := range engine.Graph.Nodes {
		totalEdges += len(node.Edges)
	}

	if finalNodes <= initialNodes {
		t.Fatalf("FAIL: expected the graph to grow beyond its %d-node seed from 20 cycles of varied real vocabulary, still at %d", initialNodes, finalNodes)
	}
	if sleepCount < 3 {
		t.Fatalf("FAIL: expected at least 3 Subconscious Sleep cycles across 20 steps (every 5th), got %d", sleepCount)
	}

	t.Logf("Organic Growth PASS: %d -> %d nodes, %d total directed edges, %d sleep cycles, %d total AbstractionsCreated, peak cohesion observed=%.4f (threshold 0.75) across the run",
		initialNodes, finalNodes, totalEdges, sleepCount, totalAbstractionsCreated, peakCohesion)

	t.Logf("MoE Pool Diagnostic: final PredictorPoolSize=%d (starts at 1; grows only if Louvain-diversified clusters actually win the router's competition at some point -- not hard-asserted here, see TestMoERoutingAcrossClusters for a deterministic guarantee)",
		last.PredictorPoolSize)

	t.Logf("MLP Reward Diagnostic (1-PredictionError, %d cycles, NO LONGER drives Hebbian): mean=%.4f min=%.4f max=%.4f negative_count=%d/%d (%.0f%%)",
		cycleCount, mlpRewardSum/float64(cycleCount), mlpRewardMin, mlpRewardMax, negativeMLPRewardCount, cycleCount,
		100*float64(negativeMLPRewardCount)/float64(cycleCount))
	t.Logf("Structural Reward Diagnostic (actualSuccess-derived +1/-1, %d cycles, DOES drive Hebbian): positive=%d/%d (%.0f%%), mean=%.4f",
		cycleCount, structuralPositiveCount, cycleCount, 100*float64(structuralPositiveCount)/float64(cycleCount),
		(float64(structuralPositiveCount)-float64(cycleCount-structuralPositiveCount))/float64(cycleCount))

	if totalAbstractionsCreated > 0 {
		t.Logf("Stage 3 -> Stage 4 gate: compression fired on ORGANIC structure %d time(s) during this run -- not just the synthetic clusters in abstraction_test.go", totalAbstractionsCreated)
	} else {
		t.Logf("Stage 3 -> Stage 4 gate: NOT yet cleared -- 0 organic compressions this run (peak cohesion %.4f vs threshold 0.75). Longer runs or denser co-occurrence may be needed before Stage 4 (see ARCHITECTURE.md Stage 4 gating note).", peakCohesion)
	}
}
