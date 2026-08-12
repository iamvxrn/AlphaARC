package graph

import (
	"protaxon/pkg/core"
	"testing"
)

func buildRouterTestGraph() *Graph {
	g := NewGraph()
	// Cluster 0: three competing candidates.
	g.AddNode(NewNode(1, 0.1, 0))
	g.AddNode(NewNode(2, 0.1, 0))
	g.AddNode(NewNode(3, 0.1, 0))
	// Cluster 1: a lone candidate, nothing to compete with.
	g.AddNode(NewNode(10, 0.1, 1))
	return g
}

// TestRouteCompetingClustersPicksOneWinnerPerCluster verifies the router
// narrows multiple same-cluster candidates down to a single, highest-
// activation winner, while a candidate alone in its own cluster survives
// unsuppressed since it has no competition.
func TestRouteCompetingClustersPicksOneWinnerPerCluster(t *testing.T) {
	sys := core.NewSystem()
	g := buildRouterTestGraph()

	g.Nodes[1].Activation = 0.4
	g.Nodes[2].Activation = 0.9 // strongest in cluster 0
	g.Nodes[3].Activation = 0.5
	g.Nodes[10].Activation = 0.2

	winners := g.RouteCompetingClusters(sys, []int{1, 2, 3, 10}, 0.1)

	if len(winners) != 2 {
		t.Fatalf("FAIL: expected 2 winners (1 per cluster), got %v", winners)
	}
	if winners[0] != 2 || winners[1] != 10 {
		t.Fatalf("FAIL: expected winners [2 10] (cluster-0 champion + lone cluster-1 candidate), got %v", winners)
	}

	if g.Nodes[2].Activation != 0.9 {
		t.Fatalf("FAIL: winner node 2's activation must be untouched, got %.4f", g.Nodes[2].Activation)
	}
	if g.Nodes[1].Activation >= 0.4 || g.Nodes[3].Activation >= 0.5 {
		t.Fatalf("FAIL: losing candidates 1 and 3 must be suppressed, got act1=%.4f act3=%.4f", g.Nodes[1].Activation, g.Nodes[3].Activation)
	}
	if g.Nodes[10].Activation != 0.2 {
		t.Fatalf("FAIL: lone candidate 10 must be untouched (no competitor in its cluster), got %.4f", g.Nodes[10].Activation)
	}

	t.Logf("Router PASS: cluster 0 winner=2 (act=%.2f, suppressed 1->%.4f 3->%.4f), cluster 1 lone survivor=10 (untouched act=%.2f)",
		g.Nodes[2].Activation, g.Nodes[1].Activation, g.Nodes[3].Activation, g.Nodes[10].Activation)
}

// TestRouteCompetingClustersEmptyInput verifies an empty candidate list
// produces an empty, non-panicking result.
func TestRouteCompetingClustersEmptyInput(t *testing.T) {
	sys := core.NewSystem()
	g := buildRouterTestGraph()

	winners := g.RouteCompetingClusters(sys, []int{}, 0.1)
	if len(winners) != 0 {
		t.Fatalf("FAIL: expected no winners for empty candidate list, got %v", winners)
	}
}
