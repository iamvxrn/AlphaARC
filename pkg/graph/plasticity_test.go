package graph

import (
	"math"
	"testing"
)

func TestL1NormalizationWithFloor(t *testing.T) {
	node := NewNode(1, 0.1, 0)
	node.Edges[2] = &Edge{TargetID: 2, Weight: 15.0} // Exceeds cap=10.0
	node.Edges[3] = &Edge{TargetID: 3, Weight: MinWeightFloor}

	sumBefore := 0.0
	for _, edge := range node.Edges {
		sumBefore += math.Abs(edge.Weight)
	}

	cap := 10.0
	NormalizeNodeL1(node, cap)

	sumAfter := 0.0
	for _, edge := range node.Edges {
		sumAfter += math.Abs(edge.Weight)
	}

	w2 := node.Edges[2].Weight
	w3 := node.Edges[3].Weight

	t.Logf("L1 Normalization Test (cap=%.1f):", cap)
	t.Logf("  sum|w| before: %.4f", sumBefore)
	t.Logf("  sum|w| after : %.4f", sumAfter)
	t.Logf("  dominant edge w2: %.4f", w2)
	t.Logf("  floor edge w3   : %.4f (MinWeightFloor=%.4f)", w3, MinWeightFloor)

	if w3 < MinWeightFloor {
		t.Fatalf("FAIL: Normalization shrank floor edge below MinWeightFloor! w3=%.6f", w3)
	}
}

func TestL1NormalizationMultipleFloors(t *testing.T) {
	node := NewNode(1, 0.1, 0)
	cap := 10.0

	// Test N = 10, 50, 100 floor edges
	counts := []int{10, 50, 100}

	for _, N := range counts {
		node.Edges = make(map[int]*Edge)
		node.Edges[0] = &Edge{TargetID: 0, Weight: 15.0} // Dominant edge

		for i := 1; i <= N; i++ {
			node.Edges[i] = &Edge{TargetID: i, Weight: MinWeightFloor}
		}

		sumBefore := 0.0
		for _, edge := range node.Edges {
			sumBefore += math.Abs(edge.Weight)
		}

		NormalizeNodeL1(node, cap)

		sumAfter := 0.0
		for _, edge := range node.Edges {
			sumAfter += math.Abs(edge.Weight)
		}

		expectedMaxSum := cap + float64(N)*MinWeightFloor

		t.Logf("Multiple Floors (N=%d floor edges): sum|w| before=%.4f -> after=%.4f (bound=%.4f)",
			N, sumBefore, sumAfter, expectedMaxSum)

		if sumAfter > expectedMaxSum+1e-6 {
			t.Fatalf("FAIL: sum|w| (%.4f) exceeded theoretical bound (%.4f) for N=%d floor edges!",
				sumAfter, expectedMaxSum, N)
		}
	}
}
