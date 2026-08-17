package graph

import (
	"alphaarc/pkg/core"
	"math"
)

// HebbianUpdate performs local weight updates on active edges, modulated by dopamine.
func (g *Graph) HebbianUpdate(sys *core.System, learningRate float64, dopamine float64, l1Cap float64) {
	sys.OnlineGuard("Graph.HebbianUpdate")

	for _, node := range g.Nodes {
		if node.Activation == 0 {
			continue
		}
		for targetID, edge := range node.Edges {
			targetNode, exists := g.Nodes[targetID]
			if !exists || targetNode.Activation == 0 {
				continue
			}
			// Hebbian plastic update: Delta W = lr * dopamine * pre * post
			delta := learningRate * dopamine * node.Activation * targetNode.Activation
			edge.Weight += delta
		}
		NormalizeNodeL1(node, l1Cap)
	}
}

// NormalizeNodeL1 caps the sum of absolute outgoing weights for a node,
// preserving MinWeightFloor for non-zero weights to prevent dead-agent lock.
func NormalizeNodeL1(node *Node, cap float64) {
	sum := 0.0
	for _, edge := range node.Edges {
		sum += math.Abs(edge.Weight)
	}
	if sum > cap && sum > 0 {
		scale := cap / sum
		for _, edge := range node.Edges {
			edge.Weight *= scale
			if edge.Weight > 0 && edge.Weight < MinWeightFloor {
				edge.Weight = MinWeightFloor
			}
		}
	}
}
