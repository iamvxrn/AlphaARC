package graph

import "protaxon/pkg/core"

// UpdateEligibilityTraces accumulates trace strength on every edge whose
// source and target are both active this tick: Eligibility += pre*post.
// Call once per ONLINE tick, after DecayEligibilityTraces and before
// HebbianUpdateWithEligibility.
func (g *Graph) UpdateEligibilityTraces(sys *core.System) {
	sys.OnlineGuard("Graph.UpdateEligibilityTraces")

	for _, node := range g.Nodes {
		if node.Activation == 0 {
			continue
		}
		for targetID, edge := range node.Edges {
			targetNode, exists := g.Nodes[targetID]
			if !exists || targetNode.Activation == 0 {
				continue
			}
			edge.Eligibility += node.Activation * targetNode.Activation
		}
	}
}

// DecayEligibilityTraces multiplies every edge's trace by decay (expected in
// (0,1)), fading credit for firings from earlier ticks rather than erasing
// it instantly. Call once per ONLINE tick, before UpdateEligibilityTraces.
func (g *Graph) DecayEligibilityTraces(sys *core.System, decay float64) {
	sys.OnlineGuard("Graph.DecayEligibilityTraces")

	for _, node := range g.Nodes {
		for _, edge := range node.Edges {
			edge.Eligibility *= decay
		}
	}
}

// HebbianUpdateWithEligibility is HebbianUpdate's temporal-credit-assignment
// counterpart: plasticity is driven by each edge's accumulated, decayed
// Eligibility trace instead of this tick's instantaneous pre*post product,
// so an edge that fired a few ticks before a reward arrives still gets
// credited, proportional to how much its trace has decayed since. It does
// not gate on the source node's current Activation the way HebbianUpdate
// does -- a zero-activation node this tick can still carry a nonzero trace
// from a recent firing. Existing HebbianUpdate is left untouched for callers
// that don't want trace-based credit assignment.
func (g *Graph) HebbianUpdateWithEligibility(sys *core.System, learningRate, dopamine, l1Cap float64) {
	sys.OnlineGuard("Graph.HebbianUpdateWithEligibility")

	for _, node := range g.Nodes {
		for _, edge := range node.Edges {
			if edge.Eligibility == 0 {
				continue
			}
			edge.Weight += learningRate * dopamine * edge.Eligibility
		}
		NormalizeNodeL1(node, l1Cap)
	}
}
