package pipeline

import (
	"context"
	"testing"
)

// TestForwardModelLearnsToPredictAlternatingStream is the behavioral proof
// the structural identity test (TestRunPredictiveCycleTrainsForwardModelOn-
// RealizedTransition) can't give: that the forward model doesn't just train
// on the realized transition mechanically, but actually CONVERGES -- its
// cross-cycle forecast error falls as it learns a predictable stream.
//
// The stream alternates two disjoint observations A,B,A,B,... The only way
// forecast error can drop is if the model learns the transitions "after A
// comes B" and "after B comes A" -- an autoencoder of the current frame
// (what the predictor wrongly was before the forward-model fix) could never
// reduce cross-cycle error here, since the next frame is always the OTHER
// observation, never a copy of the current one.
//
// Thresholds are deliberately loose (10x, not the ~390x actually observed
// on the deterministic run this was written against) so ordinary drift in
// the MLP/graph internals can't make it flaky while still failing hard if
// learning regresses to no-better-than-chance.
func TestForwardModelLearnsToPredictAlternatingStream(t *testing.T) {
	ctx := context.Background()
	e := NewEngine()
	obsA := "color1-cell0-0 color2-cell1-1 color3-cell2-2"
	obsB := "color7-cell5-5 color8-cell6-6 color9-cell7-7"

	const cycles = 40
	errs := make([]float64, cycles)
	for i := 0; i < cycles; i++ {
		obs := obsA
		if i%2 == 1 {
			obs = obsB
		}
		res, err := e.RunPredictiveCycle(ctx, obs, "goal", true)
		if err != nil {
			t.Fatalf("FAIL: cycle %d error: %v", i+1, err)
		}
		errs[i] = res.ForecastError
	}

	// Early window = cycles 2..11 (skip cycle 1, whose error is 0 by
	// definition -- nothing was forecast yet). Late window = last 10 cycles.
	early := mean(errs[1:11])
	late := mean(errs[cycles-10:])

	if early < 0.02 {
		t.Fatalf("FAIL: expected a real early forecast error to learn down from, got early mean %.5f (stream may be trivially predictable)", early)
	}
	if late >= early*0.1 {
		t.Fatalf("FAIL: forecast error did not converge -- late mean %.5f is not <10x below early mean %.5f; the forward model isn't learning the alternation", late, early)
	}
	t.Logf("forward-model learning confirmed: early(cycles 2-11) mean=%.5f -> late(last 10) mean=%.5f (%.0fx reduction)", early, late, early/late)
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}
