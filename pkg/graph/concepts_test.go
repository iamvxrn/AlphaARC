package graph

import "testing"

func conceptTestNodeIDs(g *Graph) []int {
	ids := make([]int, 0, len(g.Nodes))
	for id := range g.Nodes {
		ids = append(ids, id)
	}
	return ids
}

func TestEnsureConceptNodesCreatesNewNodesForNovelVocabulary(t *testing.T) {
	g := NewGraph()

	seeds := g.EnsureConceptNodes("Unexpected latency spike detected upstream", 0.5)

	// "latency", "spike", "detected", "upstream" all qualify (len>=3, not
	// stopwords); "unexpected" too. That's 5 new concept nodes.
	if len(g.Nodes) != 5 {
		t.Fatalf("FAIL: expected 5 newly created concept nodes, got %d: %v", len(g.Nodes), conceptTestNodeIDs(g))
	}
	if len(seeds) != 5 {
		t.Fatalf("FAIL: expected 5 seed activations for the newly created concepts, got %v", seeds)
	}
	for _, ids := range g.Labels {
		if len(ids) != 1 {
			t.Fatalf("FAIL: expected each new label to map to exactly 1 node, got %v", ids)
		}
	}

	t.Logf("Concept Creation PASS: %d nodes created from novel vocabulary, labels=%v", len(g.Nodes), g.Labels)
}

func TestEnsureConceptNodesReusesExistingLabelsWithoutDuplicating(t *testing.T) {
	g := NewGraph()
	g.AddNode(NewNode(0, 0.5, 0))
	g.AddLabel("latency", 0)

	seeds := g.EnsureConceptNodes("Latency spike detected", 0.5)

	// "latency" already exists -> reused, not duplicated. "spike" and
	// "detected" are new -> 2 new nodes. Total 3 nodes (1 original + 2 new).
	if len(g.Nodes) != 3 {
		t.Fatalf("FAIL: expected 3 total nodes (1 reused + 2 new), got %d: %v", len(g.Nodes), conceptTestNodeIDs(g))
	}
	if seeds[0] != 1.0 {
		t.Fatalf("FAIL: expected existing node 0 to be seeded via reused 'latency' label, got %v", seeds)
	}
	if len(g.Labels["latency"]) != 1 {
		t.Fatalf("FAIL: 'latency' label must still map to exactly 1 node (no duplicate registration), got %v", g.Labels["latency"])
	}

	t.Logf("Reuse PASS: 'latency' reused node 0, 2 genuinely new concepts created, total nodes=%d", len(g.Nodes))
}

func TestEnsureConceptNodesFiltersStopwordsAndShortTokens(t *testing.T) {
	g := NewGraph()

	seeds := g.EnsureConceptNodes("it is of the to a an", 0.5)

	if len(g.Nodes) != 0 {
		t.Fatalf("FAIL: expected 0 nodes created from an all-stopword observation, got %d: %v", len(g.Nodes), conceptTestNodeIDs(g))
	}
	if len(seeds) != 0 {
		t.Fatalf("FAIL: expected 0 seeds from an all-stopword observation, got %v", seeds)
	}

	t.Logf("Stopword Filter PASS: all-stopword observation produced 0 nodes")
}
