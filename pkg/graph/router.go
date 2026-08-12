package graph

import (
	"protaxon/pkg/core"
	"sort"
)

// RouteCompetingClusters is the dynamic competition router: candidate node
// IDs are grouped by their current ClusterID, and within any group with more
// than one candidate, Winner-Takes-All lateral inhibition
// (ResolveLateralInhibition) picks a single winner and suppresses the rest
// (their Activation is scaled down toward MinWeightFloor, same mechanism
// Stage 1 already verified). A candidate alone in its cluster passes through
// unsuppressed -- there is nothing for it to compete with.
//
// Which nodes end up competing, and who wins, is decided fresh from the
// graph's live Activation levels on every call -- there is no static
// priority list -- hence "dynamic." Returns the sorted, deduplicated list of
// surviving winner IDs.
func (g *Graph) RouteCompetingClusters(sys *core.System, candidateIDs []int, penalty float64) []int {
	sys.OnlineGuard("Graph.RouteCompetingClusters")

	groups := make(map[int][]int)
	for _, id := range candidateIDs {
		node, exists := g.Nodes[id]
		if !exists {
			continue
		}
		groups[node.ClusterID] = append(groups[node.ClusterID], id)
	}

	clusterIDs := make([]int, 0, len(groups))
	for cID := range groups {
		clusterIDs = append(clusterIDs, cID)
	}
	sort.Ints(clusterIDs)

	winners := make([]int, 0, len(candidateIDs))
	for _, cID := range clusterIDs {
		members := groups[cID]
		if len(members) == 1 {
			winners = append(winners, members[0])
			continue
		}
		sort.Ints(members)
		winner := g.ResolveLateralInhibition(sys, members, penalty)
		if winner != nil {
			winners = append(winners, winner.ID)
		}
	}

	sort.Ints(winners)
	return winners
}
