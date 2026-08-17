package graph

import "alphaarc/pkg/core"

// FormCoActivationEdges implements structural plasticity ("cells that fire
// together, wire together"): for every ordered pair of nodes in activeIDs
// that don't yet have an edge between them, it creates one at initialWeight.
// HebbianUpdate and HebbianUpdateWithEligibility only ever strengthen or
// weaken edges that already exist -- this is what lets the graph's topology
// itself grow from which concepts actually co-occur in real experience,
// instead of staying frozen at whatever edges it was constructed with.
//
// Existing edges are left untouched (never reset back to initialWeight), so
// this is safe to call every cycle without erasing prior Hebbian learning.
// Returns the number of new edges formed.
func (g *Graph) FormCoActivationEdges(sys *core.System, activeIDs []int, initialWeight float64) int {
	sys.OnlineGuard("Graph.FormCoActivationEdges")

	formed := 0
	for _, from := range activeIDs {
		fromNode, exists := g.Nodes[from]
		if !exists {
			continue
		}
		for _, to := range activeIDs {
			if from == to {
				continue
			}
			if _, hasEdge := fromNode.Edges[to]; hasEdge {
				continue
			}
			if _, toExists := g.Nodes[to]; !toExists {
				continue
			}
			g.AddEdge(from, to, initialWeight, false)
			formed++
		}
	}
	return formed
}
