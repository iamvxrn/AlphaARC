package offline

import (
	"protaxon/pkg/graph"
	"sort"
)

// RunLouvainCommunityDetection partitions graph nodes into dense communities.
func RunLouvainCommunityDetection(g *graph.Graph) map[int]int {
	communities := make(map[int]int)

	// Sort node IDs deterministically to avoid Go map iteration randomness
	nodeIDs := make([]int, 0, len(g.Nodes))
	for id := range g.Nodes {
		communities[id] = id
		nodeIDs = append(nodeIDs, id)
	}
	sort.Ints(nodeIDs)

	improved := true
	maxPasses := 10

	for pass := 0; pass < maxPasses && improved; pass++ {
		improved = false
		for _, id := range nodeIDs {
			node := g.Nodes[id]
			currentComm := communities[id]

			// Compute total edge weight from node to each neighboring community
			commWeights := make(map[int]float64)
			for targetID, edge := range node.Edges {
				targetComm := communities[targetID]
				commWeights[targetComm] += edge.Weight
			}

			bestComm := currentComm
			bestWeight := commWeights[currentComm]

			// Sort neighbor community IDs for deterministic tie-breaking
			commIDs := make([]int, 0, len(commWeights))
			for cID := range commWeights {
				commIDs = append(commIDs, cID)
			}
			sort.Ints(commIDs)

			for _, cID := range commIDs {
				w := commWeights[cID]
				if w > bestWeight || (w == bestWeight && cID < bestComm) {
					bestWeight = w
					bestComm = cID
				}
			}

			if bestComm != currentComm {
				communities[id] = bestComm
				improved = true
			}
		}
	}

	// Renumber community IDs to contiguous 0..N-1 IDs deterministically
	uniqueComms := make(map[int]int)
	nextID := 0

	// Sort community keys for stable output mapping
	allComms := make([]int, 0)
	for _, cID := range communities {
		found := false
		for _, u := range allComms {
			if u == cID {
				found = true
				break
			}
		}
		if !found {
			allComms = append(allComms, cID)
		}
	}
	sort.Ints(allComms)
	for _, cID := range allComms {
		uniqueComms[cID] = nextID
		nextID++
	}

	finalMap := make(map[int]int)
	for _, id := range nodeIDs {
		finalMap[id] = uniqueComms[communities[id]]
	}

	return finalMap
}
