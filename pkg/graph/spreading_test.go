package graph

import (
	"alphaarc/pkg/core"
	"testing"
)

func buildChainGraph() *Graph {
	g := NewGraph()
	g.AddNode(NewNode(1, 0.8, 0))
	g.AddNode(NewNode(2, 0.5, 0))
	g.AddNode(NewNode(3, 0.2, 0))
	g.AddEdge(1, 2, 0.9, false)
	g.AddEdge(2, 3, 0.8, false)
	return g
}

// TestLookupSeedsTokenizesObservation verifies the "Observation -> Node Lookup"
// step: registered keywords are matched case-insensitively against tokens in
// a raw observation string, punctuation is stripped, and unmatched tokens are
// ignored.
func TestLookupSeedsTokenizesObservation(t *testing.T) {
	g := buildChainGraph()
	g.AddLabel("Anomaly", 1)
	g.AddLabel("latency", 3)

	seeds := g.LookupSeeds("Sensory Stimulus: Log ANOMALY pattern detected, latency spike.")

	if seeds[1] != 1.0 {
		t.Fatalf("FAIL: expected node 1 seeded at 1.0 via 'Anomaly' label, got %v", seeds[1])
	}
	if seeds[3] != 1.0 {
		t.Fatalf("FAIL: expected node 3 seeded at 1.0 via 'latency' label, got %v", seeds[3])
	}
	if _, ok := seeds[2]; ok {
		t.Fatalf("FAIL: node 2 has no registered label and must not be seeded")
	}

	empty := g.LookupSeeds("nothing relevant here")
	if len(empty) != 0 {
		t.Fatalf("FAIL: expected no seeds for observation with no label matches, got %v", empty)
	}
}

// TestSpreadingActivationReachesMultiHopNodes verifies that SpreadingActivation
// genuinely propagates across multiple hops (1->2->3) over several calls to
// Propagate, reaching nodes a single-hop Propagate call could never activate.
func TestSpreadingActivationReachesMultiHopNodes(t *testing.T) {
	sys := core.NewSystem()

	g1 := buildChainGraph()
	oneHop := g1.SpreadingActivation(sys, map[int]float64{1: 1.0}, 1, 0.8)
	if oneHop[3] != 0 {
		t.Fatalf("FAIL: single-hop spread should not yet reach node 3 (2 edges away), got %.4f", oneHop[3])
	}

	g3 := buildChainGraph()
	threeHop := g3.SpreadingActivation(sys, map[int]float64{1: 1.0}, 3, 0.8)
	if threeHop[3] <= 0 {
		t.Fatalf("FAIL: 3-hop spread should reach node 3 with positive activation, got %.4f", threeHop[3])
	}
	if threeHop[1] != 1.0 {
		t.Fatalf("FAIL: seed node 1 should stay re-injected at full strength each hop, got %.4f", threeHop[1])
	}

	t.Logf("Spreading Activation PASS: 1-hop node3=%.4f, 3-hop node3=%.4f", oneHop[3], threeHop[3])
}

// TestSpreadingActivationEmptySeeds verifies that an observation with no
// matching labels (i.e. no seeds) produces an empty active subgraph rather
// than panicking or falling back to stale leftover node activations.
func TestSpreadingActivationEmptySeeds(t *testing.T) {
	sys := core.NewSystem()
	g := buildChainGraph()

	activations := g.SpreadingActivation(sys, map[int]float64{}, 3, 0.8)
	if len(activations) != 0 {
		t.Fatalf("FAIL: expected empty activations for empty seed set, got %v", activations)
	}
}

// TestExtractActiveSubgraphSortedAndFiltered verifies threshold filtering and
// deterministic sorted-ID output for downstream Agent Context construction.
func TestExtractActiveSubgraphSortedAndFiltered(t *testing.T) {
	activations := map[int]float64{5: 0.05, 3: 2.0, 1: 1.5, 9: 0.2}

	ids := ExtractActiveSubgraph(activations, 0.1)
	expected := []int{1, 3, 9}
	if len(ids) != len(expected) {
		t.Fatalf("FAIL: expected %v, got %v", expected, ids)
	}
	for i := range expected {
		if ids[i] != expected[i] {
			t.Fatalf("FAIL: expected sorted %v, got %v", expected, ids)
		}
	}
}
