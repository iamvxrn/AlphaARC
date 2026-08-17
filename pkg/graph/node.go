package graph

import (
	"alphaarc/pkg/core"
	"math"
)

type Edge struct {
	TargetID       int
	Weight         float64
	IsCrossCluster bool

	// Eligibility is the accumulating temporal-credit-assignment trace for
	// this edge (Sutton & Barto TD(lambda) style): it grows when the edge's
	// source and target fire together, and fades each tick via
	// DecayEligibilityTraces, so a reward arriving a few ticks after the
	// firing can still reach it through HebbianUpdateWithEligibility.
	Eligibility float64
}

type Node struct {
	ID         int
	Activation float64
	Threshold  float64
	ClusterID  int
	Edges      map[int]*Edge // O(1) sparse adjacency list

	// IsAbstraction marks a node created by offline compression (Stage 3) as
	// standing in for a collapsed group of original nodes, listed in Members.
	IsAbstraction bool
	Members       []int
}

func NewNode(id int, threshold float64, clusterID int) *Node {
	return &Node{
		ID:         id,
		Activation: 0.0,
		Threshold:  threshold,
		ClusterID:  clusterID,
		Edges:      make(map[int]*Edge),
	}
}

type Graph struct {
	Nodes map[int]*Node
	// Labels maps lowercase keyword tokens to node IDs, used by LookupSeeds
	// to translate a raw text Observation into Spreading Activation seed nodes.
	Labels map[string][]int
}

func NewGraph() *Graph {
	return &Graph{
		Nodes:  make(map[int]*Node),
		Labels: make(map[string][]int),
	}
}

func (g *Graph) AddNode(node *Node) {
	g.Nodes[node.ID] = node
}

func (g *Graph) AddEdge(fromID, toID int, weight float64, crossCluster bool) {
	from, existsFrom := g.Nodes[fromID]
	_, existsTo := g.Nodes[toID]
	if !existsFrom || !existsTo {
		return
	}
	from.Edges[toID] = &Edge{
		TargetID:       toID,
		Weight:         weight,
		IsCrossCluster: crossCluster,
	}
}

// Propagate executes one step of forward activation propagation with threshold gating.
func (g *Graph) Propagate(sys *core.System, inputs map[int]float64) map[int]float64 {
	sys.OnlineGuard("Graph.Propagate")

	// Set initial activations
	nextActivations := make(map[int]float64)
	for id, node := range g.Nodes {
		inAct := inputs[id]
		if inAct < node.Threshold {
			node.Activation = 0.0
		} else {
			node.Activation = inAct
		}
		nextActivations[id] = node.Activation
	}

	// Propagate along edges
	for _, node := range g.Nodes {
		if node.Activation == 0 {
			continue
		}
		for targetID, edge := range node.Edges {
			thresh := node.Threshold
			decay := 1.0
			if edge.IsCrossCluster {
				thresh = math.Max(thresh, 0.5)
				decay = 0.5
			}
			if node.Activation >= thresh {
				nextActivations[targetID] += node.Activation * edge.Weight * decay
			}
		}
	}

	// Update node activations
	for id, act := range nextActivations {
		g.Nodes[id].Activation = act
	}

	return nextActivations
}
