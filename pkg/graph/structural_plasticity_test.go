package graph

import (
	"protaxon/pkg/core"
	"testing"
)

func TestFormCoActivationEdgesCreatesBidirectionalEdges(t *testing.T) {
	sys := core.NewSystem()
	g := NewGraph()
	g.AddNode(NewNode(1, 0.1, 0))
	g.AddNode(NewNode(2, 0.1, 0))
	g.AddNode(NewNode(3, 0.1, 0))

	formed := g.FormCoActivationEdges(sys, []int{1, 2, 3}, 0.15)

	// 3 nodes, every ordered pair (i != j) -> 3*2 = 6 directed edges.
	if formed != 6 {
		t.Fatalf("FAIL: expected 6 new directed edges among 3 mutually co-active nodes, got %d", formed)
	}
	for _, from := range []int{1, 2, 3} {
		for _, to := range []int{1, 2, 3} {
			if from == to {
				continue
			}
			edge, ok := g.Nodes[from].Edges[to]
			if !ok {
				t.Fatalf("FAIL: expected edge %d->%d to exist", from, to)
			}
			if edge.Weight != 0.15 {
				t.Fatalf("FAIL: expected new edge %d->%d weight 0.15, got %.4f", from, to, edge.Weight)
			}
		}
	}

	t.Logf("Structural Plasticity PASS: %d edges formed among 3 co-active nodes", formed)
}

func TestFormCoActivationEdgesNeverResetsExistingEdge(t *testing.T) {
	sys := core.NewSystem()
	g := NewGraph()
	g.AddNode(NewNode(1, 0.1, 0))
	g.AddNode(NewNode(2, 0.1, 0))
	g.AddEdge(1, 2, 0.83, false) // pretend Hebbian learning already strengthened this

	formed := g.FormCoActivationEdges(sys, []int{1, 2}, 0.15)

	if formed != 1 { // only 2->1 is missing; 1->2 already exists
		t.Fatalf("FAIL: expected exactly 1 new edge (2->1), got %d", formed)
	}
	if g.Nodes[1].Edges[2].Weight != 0.83 {
		t.Fatalf("FAIL: pre-existing edge 1->2 must not be reset by structural plasticity, got weight %.4f", g.Nodes[1].Edges[2].Weight)
	}

	t.Logf("Non-Destructive PASS: existing edge weight 0.83 preserved, only the missing direction (2->1) was formed")
}

func TestFormCoActivationEdgesSingleNodeFormsNothing(t *testing.T) {
	sys := core.NewSystem()
	g := NewGraph()
	g.AddNode(NewNode(1, 0.1, 0))

	formed := g.FormCoActivationEdges(sys, []int{1}, 0.15)
	if formed != 0 {
		t.Fatalf("FAIL: a single active node has nothing to co-fire with, expected 0 edges formed, got %d", formed)
	}
}
