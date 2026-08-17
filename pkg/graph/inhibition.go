package graph

import (
	"alphaarc/pkg/core"
	"math"
)

const (
	// MinWeightFloor prevents the "dead-agent problem" by ensuring suppressed weights
	// never drop to hard 0.0, allowing candidates to reactivate if inputs shift.
	MinWeightFloor = 0.001
)

// ResolveLateralInhibition performs Winner-Takes-All competition among candidate node IDs.
// The candidate with the highest activation survives, while weaker candidates are suppressed/inhibited.
func (g *Graph) ResolveLateralInhibition(sys *core.System, candidateIDs []int, penalty float64) *Node {
	sys.OnlineGuard("Graph.ResolveLateralInhibition")

	if len(candidateIDs) == 0 {
		return nil
	}

	var winner *Node
	maxActivation := -1.0

	for _, id := range candidateIDs {
		node, exists := g.Nodes[id]
		if !exists {
			continue
		}
		if node.Activation > maxActivation {
			maxActivation = node.Activation
			winner = node
		}
	}

	if winner == nil {
		return nil
	}

	// Suppress/inhibit non-winners with MinWeightFloor protection on activation
	for _, id := range candidateIDs {
		if id == winner.ID {
			continue
		}
		node, exists := g.Nodes[id]
		if exists {
			node.Activation *= penalty
			if node.Activation < MinWeightFloor {
				node.Activation = MinWeightFloor
			}
		}
	}

	return winner
}

// UpdateCandidateWeights performs sensory-modulated competitive weight updates between candidates.
// Sensory input (inA, inB) directly drives plasticity, preventing zombie-agent locks when inputs reverse.
func UpdateCandidateWeights(wA, wB *float64, inA, inB, boost, penalty float64) {
	actA := (*wA) * inA
	actB := (*wB) * inB

	if actA >= actB {
		*wA += boost * inA
		*wB -= penalty * (1.0 - inB)
	} else {
		*wB += boost * inB
		*wA -= penalty * (1.0 - inA)
	}

	// Enforce floor floor clamp before applying sensory leak
	*wA = math.Max(MinWeightFloor, *wA)
	*wB = math.Max(MinWeightFloor, *wB)

	// Sensory plasticity leak: any positive input stimulus (in > 0)
	// allows a suppressed candidate at floor to escape and recover.
	if inA > 0 {
		*wA += 0.05 * inA
	}
	if inB > 0 {
		*wB += 0.05 * inB
	}
}
