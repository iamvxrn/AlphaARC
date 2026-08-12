package offline

import (
	"protaxon/pkg/core"
	"protaxon/pkg/graph"
	"testing"
)

// buildCliqueWithExternalLinks builds a 5-node fully-connected clique
// (weight 0.8, cohesive) plus one external node 100 linked in both
// directions, all with ClusterID assigned manually so the test is isolated
// from Louvain's own convergence behavior (covered separately in
// sleep_test.go).
func buildCliqueWithExternalLinks() *graph.Graph {
	g := graph.NewGraph()
	for i := 0; i < 5; i++ {
		g.AddNode(graph.NewNode(i, 0.1, 0))
	}
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if i != j {
				g.AddEdge(i, j, 0.8, false)
			}
		}
	}
	g.AddNode(graph.NewNode(100, 0.1, 1))

	// Outbound (member -> external): should average to 0.5 on the abstraction node.
	g.AddEdge(0, 100, 0.4, true)
	g.AddEdge(2, 100, 0.6, true)
	// Inbound (external -> member): should average to 0.2 on the abstraction node.
	g.AddEdge(100, 1, 0.2, true)

	for i := 0; i < 5; i++ {
		g.Nodes[i].ClusterID = 0
	}
	g.Nodes[100].ClusterID = 1

	return g
}

func TestCompressGraphAbstractionsCollapsesCohesiveCluster(t *testing.T) {
	sys := core.NewSystem()
	sys.Mode = core.Offline
	g := buildCliqueWithExternalLinks()

	stats := CompressGraphAbstractions(sys, g, 0.7)

	if stats.ClustersCompressed != 1 {
		t.Fatalf("FAIL: expected exactly 1 cluster compressed, got %d", stats.ClustersCompressed)
	}
	if stats.NodesAbsorbed != 5 {
		t.Fatalf("FAIL: expected 5 nodes absorbed, got %d", stats.NodesAbsorbed)
	}
	if len(g.Nodes) != 2 { // 1 abstraction node + external node 100
		t.Fatalf("FAIL: expected 2 remaining nodes (1 abstraction + external), got %d: %v", len(g.Nodes), nodeIDs(g))
	}
	for i := 0; i < 5; i++ {
		if _, exists := g.Nodes[i]; exists {
			t.Fatalf("FAIL: absorbed member node %d should no longer exist in graph", i)
		}
	}

	var abs *graph.Node
	for id, node := range g.Nodes {
		if id != 100 {
			abs = node
		}
	}
	if abs == nil || !abs.IsAbstraction {
		t.Fatalf("FAIL: expected a single IsAbstraction node besides external node 100")
	}
	if len(abs.Members) != 5 {
		t.Fatalf("FAIL: expected abstraction node to list 5 original members, got %v", abs.Members)
	}
	for i := 0; i < 5; i++ {
		found := false
		for _, m := range abs.Members {
			if m == i {
				found = true
			}
		}
		if !found {
			t.Fatalf("FAIL: abstraction Members missing original node %d: %v", i, abs.Members)
		}
	}

	t.Logf("Compression PASS: 5-node clique (cohesion=0.80) -> abstraction node %d, Members=%v", abs.ID, abs.Members)
}

func TestCompressGraphAbstractionsPreservesExternalConnectivity(t *testing.T) {
	sys := core.NewSystem()
	sys.Mode = core.Offline
	g := buildCliqueWithExternalLinks()

	CompressGraphAbstractions(sys, g, 0.7)

	var absID int = -1
	for id := range g.Nodes {
		if id != 100 {
			absID = id
		}
	}
	if absID == -1 {
		t.Fatalf("FAIL: no abstraction node found")
	}

	outEdge, ok := g.Nodes[absID].Edges[100]
	if !ok {
		t.Fatalf("FAIL: expected redirected outbound edge abstraction->100")
	}
	if diff := outEdge.Weight - 0.5; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("FAIL: expected averaged outbound weight 0.5 (from 0.4,0.6), got %.6f", outEdge.Weight)
	}

	inEdge, ok := g.Nodes[100].Edges[absID]
	if !ok {
		t.Fatalf("FAIL: expected redirected inbound edge 100->abstraction")
	}
	if diff := inEdge.Weight - 0.2; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("FAIL: expected averaged inbound weight 0.2, got %.6f", inEdge.Weight)
	}

	t.Logf("External Connectivity PASS: abstraction->100 = %.4f (expected 0.5), 100->abstraction = %.4f (expected 0.2)",
		outEdge.Weight, inEdge.Weight)
}

func TestCompressGraphAbstractionsRemapsLabels(t *testing.T) {
	sys := core.NewSystem()
	sys.Mode = core.Offline
	g := buildCliqueWithExternalLinks()
	g.AddLabel("concept", 2)
	g.AddLabel("concept", 3) // two members sharing one keyword must dedupe to one entry

	CompressGraphAbstractions(sys, g, 0.7)

	ids, ok := g.Labels["concept"]
	if !ok || len(ids) != 1 {
		t.Fatalf("FAIL: expected label 'concept' remapped to exactly 1 deduped abstraction ID, got %v", ids)
	}
	if _, exists := g.Nodes[ids[0]]; !exists {
		t.Fatalf("FAIL: remapped label points at a node ID that doesn't exist: %d", ids[0])
	}
	if !g.Nodes[ids[0]].IsAbstraction {
		t.Fatalf("FAIL: remapped label should point at the abstraction node")
	}

	t.Logf("Label Remap PASS: 'concept' (originally nodes 2 and 3) -> abstraction node %d", ids[0])
}

func TestCompressGraphAbstractionsLeavesSparseClusterUntouched(t *testing.T) {
	sys := core.NewSystem()
	sys.Mode = core.Offline
	g := graph.NewGraph()

	g.AddNode(graph.NewNode(0, 0.1, 0))
	g.AddNode(graph.NewNode(1, 0.1, 0))
	g.AddEdge(0, 1, 0.2, false)
	g.AddEdge(1, 0, 0.2, false)

	stats := CompressGraphAbstractions(sys, g, 0.7)

	if stats.ClustersCompressed != 0 {
		t.Fatalf("FAIL: expected 0 clusters compressed for cohesion 0.2 < threshold 0.7, got %d", stats.ClustersCompressed)
	}
	if len(g.Nodes) != 2 {
		t.Fatalf("FAIL: expected both original nodes to survive untouched, got %d nodes", len(g.Nodes))
	}
	if g.Nodes[0].IsAbstraction || g.Nodes[1].IsAbstraction {
		t.Fatalf("FAIL: nodes below cohesion threshold must not become abstraction nodes")
	}

	t.Logf("Below-Threshold PASS: cohesion 0.20 < threshold 0.70, cluster left untouched")
}

func TestCompressGraphAbstractionsSkipsSingletonClusters(t *testing.T) {
	sys := core.NewSystem()
	sys.Mode = core.Offline
	g := graph.NewGraph()
	g.AddNode(graph.NewNode(0, 0.1, 0))
	g.AddNode(graph.NewNode(1, 0.1, 1))

	stats := CompressGraphAbstractions(sys, g, 0.0)

	if stats.ClustersCompressed != 0 {
		t.Fatalf("FAIL: singleton clusters must never be compressed, got %d compressed", stats.ClustersCompressed)
	}
	if len(g.Nodes) != 2 {
		t.Fatalf("FAIL: expected both singleton nodes to survive, got %d", len(g.Nodes))
	}
}

func nodeIDs(g *graph.Graph) []int {
	ids := make([]int, 0, len(g.Nodes))
	for id := range g.Nodes {
		ids = append(ids, id)
	}
	return ids
}
