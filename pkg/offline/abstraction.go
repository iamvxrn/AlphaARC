package offline

import (
	"protaxon/pkg/core"
	"protaxon/pkg/graph"
	"sort"
)

type CompressionStats struct {
	ClustersEvaluated   int
	ClustersCompressed  int
	NodesAbsorbed       int
	AbstractionsCreated int
}

// CompressGraphAbstractions implements Stage 3 abstraction compression.
//
// For each Louvain community (Node.ClusterID, assigned by
// RunLouvainCommunityDetection / SubconsciousSleep), it computes the mean
// intra-cluster edge weight ("cohesion") as a graph-topology proxy for
// information-bottleneck-style redundancy: a cluster whose members are
// strongly, densely interconnected carries little information beyond "these
// fire together," so it is collapsed into a single higher-level abstraction
// node once cohesion clears cohesionThreshold. External connectivity is
// preserved (redirected edges to/from absorbed members are weight-averaged
// onto the new node, never dropped), and any Graph.Labels entries pointing at
// an absorbed member are remapped to the abstraction node so Stage 2's
// Observation -> Node Lookup keeps working after a sleep cycle compresses
// the graph.
//
// This is a deliberately simplified proxy for the Information Bottleneck
// principle (Tishby & Zaslavsky), not an implementation of its variational
// objective — it is a topological stand-in scoped to what a sparse graph of
// this kind can support.
func CompressGraphAbstractions(sys *core.System, g *graph.Graph, cohesionThreshold float64) CompressionStats {
	sys.OfflineGuard("CompressGraphAbstractions")

	stats := CompressionStats{}

	clusters := make(map[int][]int)
	nodeIDs := make([]int, 0, len(g.Nodes))
	for id, node := range g.Nodes {
		clusters[node.ClusterID] = append(clusters[node.ClusterID], id)
		nodeIDs = append(nodeIDs, id)
	}
	sort.Ints(nodeIDs)

	clusterKeys := make([]int, 0, len(clusters))
	for cID := range clusters {
		clusterKeys = append(clusterKeys, cID)
	}
	sort.Ints(clusterKeys)

	nextID := 0
	for _, id := range nodeIDs {
		if id >= nextID {
			nextID = id + 1
		}
	}

	for _, cID := range clusterKeys {
		members := append([]int(nil), clusters[cID]...)
		sort.Ints(members)
		stats.ClustersEvaluated++

		if len(members) < 2 {
			continue
		}

		cohesion, edgeCount := intraClusterCohesion(g, members)
		if edgeCount == 0 || cohesion < cohesionThreshold {
			continue
		}

		abstractID := nextID
		nextID++

		absorbCluster(g, members, abstractID, cID)

		stats.ClustersCompressed++
		stats.NodesAbsorbed += len(members)
		stats.AbstractionsCreated++
	}

	// Recompute IsCrossCluster on every remaining edge now that some
	// endpoints may have been redirected to a new abstraction node.
	for _, node := range g.Nodes {
		for targetID, edge := range node.Edges {
			if targetNode, exists := g.Nodes[targetID]; exists {
				edge.IsCrossCluster = node.ClusterID != targetNode.ClusterID
			}
		}
	}

	return stats
}

// intraClusterCohesion returns the mean weight of directed edges whose
// source and target both belong to members, and how many such edges exist.
func intraClusterCohesion(g *graph.Graph, members []int) (float64, int) {
	memberSet := make(map[int]bool, len(members))
	for _, id := range members {
		memberSet[id] = true
	}

	sum := 0.0
	count := 0
	for _, id := range members {
		node, exists := g.Nodes[id]
		if !exists {
			continue
		}
		for targetID, edge := range node.Edges {
			if memberSet[targetID] {
				sum += edge.Weight
				count++
			}
		}
	}
	if count == 0 {
		return 0, 0
	}
	return sum / float64(count), count
}

// absorbCluster collapses members into one new abstraction node (abstractID,
// in Louvain community clusterID), preserving external connectivity by
// weight-averaging redirected edges and remapping any Graph.Labels entries
// pointing at an absorbed member.
func absorbCluster(g *graph.Graph, members []int, abstractID, clusterID int) {
	memberSet := make(map[int]bool, len(members))
	thresholdSum := 0.0
	flatMembers := make([]int, 0, len(members))

	for _, id := range members {
		memberSet[id] = true
		node := g.Nodes[id]
		thresholdSum += node.Threshold
		if node.IsAbstraction {
			flatMembers = append(flatMembers, node.Members...)
		} else {
			flatMembers = append(flatMembers, id)
		}
	}
	sort.Ints(flatMembers)

	abstractNode := graph.NewNode(abstractID, thresholdSum/float64(len(members)), clusterID)
	abstractNode.IsAbstraction = true
	abstractNode.Members = flatMembers
	g.AddNode(abstractNode)

	// Redirect outbound edges (member -> external), weight-averaged per target.
	outWeights := make(map[int][]float64)
	for _, id := range members {
		node := g.Nodes[id]
		for targetID, edge := range node.Edges {
			if memberSet[targetID] {
				continue // intra-cluster edge, absorbed into the merge
			}
			outWeights[targetID] = append(outWeights[targetID], edge.Weight)
		}
	}
	for targetID, weights := range outWeights {
		g.AddEdge(abstractID, targetID, average(weights), false)
	}

	// Redirect inbound edges (external -> member), weight-averaged per source.
	inWeights := make(map[int][]float64)
	for id, node := range g.Nodes {
		if memberSet[id] || id == abstractID {
			continue
		}
		for targetID, edge := range node.Edges {
			if memberSet[targetID] {
				inWeights[id] = append(inWeights[id], edge.Weight)
				delete(node.Edges, targetID)
			}
		}
	}
	for sourceID, weights := range inWeights {
		g.AddEdge(sourceID, abstractID, average(weights), false)
	}

	// Remap Observation -> Node Lookup labels pointing at absorbed members.
	for keyword, ids := range g.Labels {
		changed := false
		seen := make(map[int]bool, len(ids))
		remapped := make([]int, 0, len(ids))
		for _, id := range ids {
			if memberSet[id] {
				id = abstractID
				changed = true
			}
			if !seen[id] {
				seen[id] = true
				remapped = append(remapped, id)
			}
		}
		if changed {
			g.Labels[keyword] = remapped
		}
	}

	for _, id := range members {
		delete(g.Nodes, id)
	}
}

func average(vals []float64) float64 {
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}
