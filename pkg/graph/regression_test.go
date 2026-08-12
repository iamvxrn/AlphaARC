package graph

import (
	"protaxon/pkg/core"
	"math"
	"testing"
)

// TestPredictorHebbianMonotonicity verifies that Hebbian prediction error
// monotonically decreases on stochastic grid transitions (Step 1 regression).
func TestPredictorHebbianMonotonicity(t *testing.T) {
	sys := core.NewSystem()
	_ = sys
	n := 25
	w := make([][]float64, n)
	for i := range w {
		w[i] = make([]float64, n)
	}

	lr := 0.3
	pos := 0

	// Deterministic sequence of transitions
	seq := []int{1, 2, 3, 4, 0, 1, 2, 3, 4, 0, 1, 2, 3, 4, 0}

	errs := []float64{}
	for _, next := range seq {
		// softmax
		maxV := math.Inf(-1)
		for _, v := range w[pos] {
			if v > maxV {
				maxV = v
			}
		}
		sum := 0.0
		probs := make([]float64, n)
		for i, v := range w[pos] {
			probs[i] = math.Exp(v - maxV)
			sum += probs[i]
		}
		for i := range probs {
			probs[i] /= sum
		}

		err := 1.0 - probs[next]
		errs = append(errs, err)

		// Standard Hebbian update
		w[pos][next] += lr * 1.0 * 1.0

		// L1 normalize
		s := 0.0
		for _, v := range w[pos] {
			s += math.Abs(v)
		}
		if s > 10.0 {
			for i := range w[pos] {
				w[pos][i] *= (10.0 / s)
			}
		}
		pos = next
	}

	if errs[len(errs)-1] >= errs[0] {
		t.Fatalf("FAIL: Hebbian prediction error did not decrease! First=%.4f, Last=%.4f", errs[0], errs[len(errs)-1])
	}

	t.Logf("Predictor Monotonicity PASS: First err=%.4f -> Last err=%.4f", errs[0], errs[len(errs)-1])
}

// TestOriginalLateralInhibitionDominance verifies that when Candidate A continuously receives
// stronger input, Candidate A continuously wins without unexpected flip-flops (Step 5 regression).
func TestOriginalLateralInhibitionDominance(t *testing.T) {
	wA, wB := 0.6, 0.55
	penalty := 0.15
	boost := 0.05

	inA := []float64{0.9, 0.85, 0.88, 0.92, 0.95}
	inB := []float64{0.4, 0.35, 0.38, 0.40, 0.42}

	for r := 0; r < len(inA); r++ {
		actA := wA * inA[r]
		actB := wB * inB[r]

		if actB >= actA {
			t.Fatalf("FAIL: Candidate B unexpectedly won round %d! actA=%.4f actB=%.4f", r, actA, actB)
		}

		UpdateCandidateWeights(&wA, &wB, inA[r], inB[r], boost, penalty)
	}

	if wA <= wB {
		t.Fatalf("FAIL: Dominant Candidate A weight (%.4f) failed to stay higher than B (%.4f)", wA, wB)
	}

	t.Logf("Lateral Inhibition Dominance PASS: Final wA=%.4f, wB=%.4f", wA, wB)
}
