package offline

import (
	"protaxon/pkg/core"
	"protaxon/pkg/graph"
	"math/rand"
	"testing"
)

func TestLouvainNonTrivialClustering(t *testing.T) {
	sys := core.NewSystem()
	sys.Mode = core.Offline

	g := graph.NewGraph()

	for i := 0; i < 10; i++ {
		g.AddNode(graph.NewNode(i, 0.1, 0))
	}

	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if i != j {
				g.AddEdge(i, j, 0.9, false)
			}
		}
	}

	for i := 5; i < 10; i++ {
		for j := 5; j < 10; j++ {
			if i != j {
				g.AddEdge(i, j, 0.9, false)
			}
		}
	}

	g.AddEdge(2, 7, 0.02, true)

	t.Logf("Before Sleep: Total nodes=10, Total edges=%d", countEdges(g))

	stats := SubconsciousSleep(sys, g, 0.05)

	t.Logf("Sleep Execution: Pruned edges=%d, Final edges=%d, Clusters found=%d",
		stats.EdgesPruned, stats.FinalEdges, stats.ClustersFound)

	if stats.EdgesPruned != 1 {
		t.Fatalf("FAIL: Expected exactly 1 weak edge (<0.05) pruned, got %d", stats.EdgesPruned)
	}

	if stats.ClustersFound != 2 {
		t.Fatalf("FAIL: Expected exactly 2 non-trivial communities (cliques), got %d clusters!", stats.ClustersFound)
	}

	clique1ID := g.Nodes[0].ClusterID
	clique2ID := g.Nodes[5].ClusterID

	if clique1ID == clique2ID {
		t.Fatalf("FAIL: Louvain merged two distinct cliques into the same cluster ID %d!", clique1ID)
	}

	for i := 0; i < 5; i++ {
		if g.Nodes[i].ClusterID != clique1ID {
			t.Fatalf("FAIL: Node %d in Clique 1 has ClusterID %d, expected %d", i, g.Nodes[i].ClusterID, clique1ID)
		}
	}

	for i := 5; i < 10; i++ {
		if g.Nodes[i].ClusterID != clique2ID {
			t.Fatalf("FAIL: Node %d in Clique 2 has ClusterID %d, expected %d", i, g.Nodes[i].ClusterID, clique2ID)
		}
	}

	t.Logf("Louvain Non-Trivial Clustering PASS: Clique 1 (0..4) -> Cluster %d, Clique 2 (5..9) -> Cluster %d",
		clique1ID, clique2ID)
}

func TestLouvainNoisyGraphClustering(t *testing.T) {
	sys := core.NewSystem()
	sys.Mode = core.Offline
	rng := rand.New(rand.NewSource(42))

	g := graph.NewGraph()
	numNodes := 15

	for i := 0; i < numNodes; i++ {
		g.AddNode(graph.NewNode(i, 0.1, 0))
	}

	for cluster := 0; cluster < 3; cluster++ {
		start := cluster * 5
		for i := start; i < start+5; i++ {
			for j := start; j < start+5; j++ {
				if i != j && rng.Float64() < 0.8 {
					w := 0.4 + rng.Float64()*0.5
					g.AddEdge(i, j, w, false)
				}
			}
		}
	}

	for i := 0; i < numNodes; i++ {
		for j := 0; j < numNodes; j++ {
			if i/5 != j/5 && rng.Float64() < 0.15 {
				w := 0.08 + rng.Float64()*0.12
				g.AddEdge(i, j, w, true)
			}
		}
	}

	t.Logf("Noisy Graph Before Sleep: Total nodes=%d, Total edges=%d", numNodes, countEdges(g))

	stats := SubconsciousSleep(sys, g, 0.05)

	t.Logf("Noisy Graph Sleep Execution: Pruned edges=%d, Final edges=%d, Clusters found=%d",
		stats.EdgesPruned, stats.FinalEdges, stats.ClustersFound)

	correctPairs := 0
	totalPairs := 0

	for i := 0; i < numNodes; i++ {
		for j := i + 1; j < numNodes; j++ {
			sameGroundTruth := (i / 5) == (j / 5)
			sameAssignedCluster := (g.Nodes[i].ClusterID == g.Nodes[j].ClusterID)

			if sameGroundTruth == sameAssignedCluster {
				correctPairs++
			}
			totalPairs++
		}
	}

	purity := float64(correctPairs) / float64(totalPairs) * 100.0

	t.Logf("Noisy Louvain Cluster Purity: %d / %d pairs correct (%.2f%%)", correctPairs, totalPairs, purity)

	if stats.ClustersFound < 2 || stats.ClustersFound > 4 {
		t.Fatalf("FAIL: Louvain on noisy graph expected 2..4 clusters, got %d!", stats.ClustersFound)
	}

	if purity < 80.0 {
		t.Fatalf("FAIL: Cluster purity on noisy graph fell below 80%%! Got %.2f%%", purity)
	}
}

func TestLouvainBoundaryStressTest(t *testing.T) {
	sys := core.NewSystem()
	sys.Mode = core.Offline
	numNodes := 10

	noiseLevels := []float64{0.10, 0.20, 0.30, 0.40, 0.50, 0.60, 0.70, 0.80}

	t.Logf("=== LOUVAIN BOUNDARY STRESS TEST (Internal w=0.5..0.9 vs Inter-cluster Noise w) ===")

	for _, noiseW := range noiseLevels {
		rng := rand.New(rand.NewSource(777))
		g := graph.NewGraph()

		for i := 0; i < numNodes; i++ {
			g.AddNode(graph.NewNode(i, 0.1, 0))
		}

		// Clique 1 (0..4) & Clique 2 (5..9) internal edges w in [0.5, 0.9]
		for c := 0; c < 2; c++ {
			start := c * 5
			for i := start; i < start+5; i++ {
				for j := start; j < start+5; j++ {
					if i != j {
						w := 0.5 + rng.Float64()*0.4
						g.AddEdge(i, j, w, false)
					}
				}
			}
		}

		// Inter-cluster noise edges fixed at noiseW (well above pruning threshold 0.05)
		for i := 0; i < 5; i++ {
			for j := 5; j < 10; j++ {
				if rng.Float64() < 0.4 {
					g.AddEdge(i, j, noiseW, true)
					g.AddEdge(j, i, noiseW, true)
				}
			}
		}

		stats := SubconsciousSleep(sys, g, 0.05)

		correctPairs := 0
		totalPairs := 0

		for i := 0; i < numNodes; i++ {
			for j := i + 1; j < numNodes; j++ {
				sameGroundTruth := (i / 5) == (j / 5)
				sameAssignedCluster := (g.Nodes[i].ClusterID == g.Nodes[j].ClusterID)

				if sameGroundTruth == sameAssignedCluster {
					correctPairs++
				}
				totalPairs++
			}
		}

		purity := float64(correctPairs) / float64(totalPairs) * 100.0

		t.Logf("Noise Weight w=%.2f | Clusters Found=%2d | Cluster Purity=%5.1f%% (%d/%d pairs)",
			noiseW, stats.ClustersFound, purity, correctPairs, totalPairs)
	}
}

func countEdges(g *graph.Graph) int {
	total := 0
	for _, node := range g.Nodes {
		total += len(node.Edges)
	}
	return total
}
