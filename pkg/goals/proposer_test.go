package goals

import (
	"protaxon/pkg/memory"
	"testing"
)

func TestProposeIrreversibleTarget(t *testing.T) {
	h := memory.NewHindsightMemory(3, 10, 100)

	startState := []float64{1, 0, 0}
	stateA := []float64{0.9, 0.1, 0}
	stateB := []float64{0, 1, 0}
	stateC := []float64{0, 0, 1} // Deepest

	// Episode 1: start <-> A (reversible noise)
	h.BeginEpisode("ep1")
	h.RecordTransition(startState, "goA", stateA)
	h.RecordTransition(stateA, "goStart", startState)
	h.EndEpisode(false)

	// Episode 2: start -> B -> C (irreversible)
	h.BeginEpisode("ep2")
	h.RecordTransition(startState, "goB", stateB)
	h.RecordTransition(stateB, "goC", stateC)
	h.EndEpisode(false)

	target := ProposeIrreversibleTarget(h, startState, 0.99)
	if target == nil {
		t.Fatal("FAIL: expected an irreversible target, got nil")
	}

	sim := memory.CosineSimilarity(target, stateC)
	if sim < 0.99 {
		t.Fatalf("FAIL: expected target to be stateC, got similarity %f", sim)
	}

	t.Logf("Found irreversible target: %v", target)
}

func TestNoIrreversibleTarget(t *testing.T) {
	h := memory.NewHindsightMemory(2, 10, 100)
	startState := []float64{1, 0}
	stateA := []float64{0, 1}

	h.BeginEpisode("ep1")
	h.RecordTransition(startState, "goA", stateA)
	h.RecordTransition(stateA, "goStart", startState)
	h.EndEpisode(false)

	target := ProposeIrreversibleTarget(h, startState, 0.99)
	if target != nil {
		t.Fatalf("FAIL: expected nil target when everything is reversible, got %v", target)
	}
}
