package offline

import (
	"protaxon/pkg/core"
	"protaxon/pkg/graph"
	"fmt"
)

type SleepStats struct {
	EdgesPruned    int
	ClustersFound  int
	InitialEdges   int
	FinalEdges     int
}

// SubconsciousSleep performs OFFLINE consolidation: edge pruning & Louvain re-clustering.
func SubconsciousSleep(sys *core.System, g *graph.Graph, pruneThreshold float64) SleepStats {
	sys.OfflineGuard("SubconsciousSleep")

	stats := SleepStats{}

	// Step 1: Count initial edges & Prune weak edges
	for _, node := range g.Nodes {
		stats.InitialEdges += len(node.Edges)
		for targetID, edge := range node.Edges {
			if edge.Weight < pruneThreshold {
				delete(node.Edges, targetID)
				stats.EdgesPruned++
			}
		}
		stats.FinalEdges += len(node.Edges)
	}

	// Step 2: Run Louvain Community Detection
	clusterMap := RunLouvainCommunityDetection(g)
	uniqueClusters := make(map[int]bool)
	for _, cID := range clusterMap {
		uniqueClusters[cID] = true
	}
	stats.ClustersFound = len(uniqueClusters)

	// Step 3: Assign ClusterIDs & update IsCrossCluster flags
	for nodeID, clusterID := range clusterMap {
		if node, exists := g.Nodes[nodeID]; exists {
			node.ClusterID = clusterID
		}
	}

	for _, node := range g.Nodes {
		for targetID, edge := range node.Edges {
			targetNode, exists := g.Nodes[targetID]
			if exists {
				edge.IsCrossCluster = (node.ClusterID != targetNode.ClusterID)
			}
		}
	}

	fmt.Printf("[OFFLINE / SLEEP] Pruned %d weak edges (<%.3f). Total edges: %d -> %d. Clusters: %d\n",
		stats.EdgesPruned, pruneThreshold, stats.InitialEdges, stats.FinalEdges, stats.ClustersFound)

	return stats
}
