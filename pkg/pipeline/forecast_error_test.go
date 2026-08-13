package pipeline

import (
	"context"
	"testing"
)

func approxEqual(a, b float64) bool {
	diff := a - b
	return diff < 1e-9 && diff > -1e-9
}

func TestVectorMSEComputesMeanSquaredError(t *testing.T) {
	a := []float64{1, 2, 3, 4}
	b := []float64{2, 2, 3, 0}
	// diffs: -1, 0, 0, 4 -> squared: 1, 0, 0, 16 -> mean = 17/4 = 4.25
	if got := vectorMSE(a, b); !approxEqual(got, 4.25) {
		t.Fatalf("FAIL: expected 4.25, got %.6f", got)
	}
}

func TestVectorMSEZeroForIdenticalVectors(t *testing.T) {
	a := []float64{1, -1, 0, 2.5}
	if got := vectorMSE(a, a); got != 0.0 {
		t.Fatalf("FAIL: expected 0.0 for identical vectors, got %.6f", got)
	}
}

func TestVectorMSEPanicsOnLengthMismatch(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("FAIL: expected a panic on mismatched vector lengths")
		}
	}()
	vectorMSE([]float64{1, 2}, []float64{1, 2, 3})
}

// TestRunPredictiveCycleForecastErrorIsZeroOnFirstCycle: a cold Engine has
// no PendingPrediction yet (nothing was forecast on a call that never
// happened), so the very first cycle must report ForecastError 0.0 rather
// than comparing against a nil/zero-value prediction as if it meant
// something. The cycle must still cache a prediction for next time.
func TestRunPredictiveCycleForecastErrorIsZeroOnFirstCycle(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	res, err := engine.RunPredictiveCycle(ctx, "color3-cell0-0 north", "goal", true)
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if res.ForecastError != 0.0 {
		t.Fatalf("FAIL: expected ForecastError 0.0 on a cold engine's first cycle, got %.6f", res.ForecastError)
	}
	if engine.PendingPrediction == nil {
		t.Fatalf("FAIL: expected PendingPrediction to be cached after the first cycle")
	}
	if len(engine.PendingPrediction) != ObservationVectorDim {
		t.Fatalf("FAIL: expected cached prediction of length %d, got %d", ObservationVectorDim, len(engine.PendingPrediction))
	}
}

// TestRunPredictiveCycleForecastErrorMatchesCachedPredictionVsActualObservationVector
// is the core wiring test: cycle 2's ForecastError must be exactly the MSE
// between cycle 1's forecast (res1.Prediction.ValueVector, now cached as
// PendingPrediction) and cycle 2's REAL content-based ObservationVector --
// a genuine cross-cycle expectation-vs-reality comparison, computed
// independently here via vectorMSE rather than assumed from the
// implementation's own internals.
func TestRunPredictiveCycleForecastErrorMatchesCachedPredictionVsActualObservationVector(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	res1, err := engine.RunPredictiveCycle(ctx, "color3-cell0-0 north", "goal", true)
	if err != nil {
		t.Fatalf("FAIL: unexpected error on cycle 1: %v", err)
	}

	obs2 := "color5-cell1-1 south aligned"
	res2, err := engine.RunPredictiveCycle(ctx, obs2, "goal", true)
	if err != nil {
		t.Fatalf("FAIL: unexpected error on cycle 2: %v", err)
	}

	want := vectorMSE(res1.Prediction.ValueVector, ObservationVector(obs2))
	if !approxEqual(res2.ForecastError, want) {
		t.Fatalf("FAIL: expected ForecastError %.6f (MSE between cycle 1's cached forecast and cycle 2's actual observation vector), got %.6f", want, res2.ForecastError)
	}
}

// TestRunPredictiveCycleSurpriseRaisesDopaminePlasticity is a differential
// test of the CORRECTED Dopamine direction: two engines run an identical
// bootstrap cycle 1 (same deterministic seeds, same observation, same
// actualSuccess -- NewEngine gives every fresh engine the same seeded MLP
// weights, so both evolve identically). Before cycle 2, one engine's
// PendingPrediction is overridden to exactly match cycle 2's real
// ObservationVector (a perfect forecast, ForecastError=0, surprise 0) and
// the other's to something wildly far away (a large ForecastError, surprise
// ~1) -- isolating the forecastError -> Dopamine wiring from whatever the
// real Predictor MLP happened to output (both engines otherwise get an
// identical drive-error-delta from UpdateHormones, since Energy/Curiosity/
// Stress evolve identically).
//
// Dopamine is the global plasticity multiplier: a prediction error is the
// signal worth LEARNING from, so surprise must turn plasticity UP. The
// surprised engine must therefore end with Dopamine strictly HIGHER than
// the perfectly-accurate one. (An earlier version asserted the opposite --
// rewarding accuracy/stasis -- which was the bug this direction fixes.)
func TestRunPredictiveCycleSurpriseRaisesDopaminePlasticity(t *testing.T) {
	ctx := context.Background()
	engineAccurate := NewEngine()
	engineSurprised := NewEngine()

	obs1 := "color3-cell0-0 north"
	if _, err := engineAccurate.RunPredictiveCycle(ctx, obs1, "goal", true); err != nil {
		t.Fatalf("FAIL: unexpected error (accurate engine, cycle 1): %v", err)
	}
	if _, err := engineSurprised.RunPredictiveCycle(ctx, obs1, "goal", true); err != nil {
		t.Fatalf("FAIL: unexpected error (surprised engine, cycle 1): %v", err)
	}

	obs2 := "color5-cell1-1 south"
	engineAccurate.PendingPrediction = ObservationVector(obs2)
	far := make([]float64, ObservationVectorDim)
	for i := range far {
		far[i] = 100.0
	}
	engineSurprised.PendingPrediction = far

	if _, err := engineAccurate.RunPredictiveCycle(ctx, obs2, "goal", true); err != nil {
		t.Fatalf("FAIL: unexpected error (accurate engine, cycle 2): %v", err)
	}
	if _, err := engineSurprised.RunPredictiveCycle(ctx, obs2, "goal", true); err != nil {
		t.Fatalf("FAIL: unexpected error (surprised engine, cycle 2): %v", err)
	}

	if engineSurprised.Homeostasis.Dopamine <= engineAccurate.Homeostasis.Dopamine {
		t.Fatalf("FAIL: expected surprise (high forecast error) to raise Dopamine/plasticity above an accurate forecast, got surprised=%.4f accurate=%.4f",
			engineSurprised.Homeostasis.Dopamine, engineAccurate.Homeostasis.Dopamine)
	}
}

// TestRunPredictiveCycleTrainsForwardModelOnRealizedTransition is the test
// that distinguishes a genuine forward model from the autoencoder it
// replaced. On cycle 2 (with the Predictors' Hopfield still empty, since the
// forward-model fix means cycle 1 does no training at all -- no previous
// specialist to train yet), the cached forecast PendingPrediction is the
// pure MLP forward pass of cycle 1's stateVector. This cycle then trains
// that same specialist on (cycle1 stateVector -> cycle2's realized
// stateVector); MLP.Train returns the loss of that forward pass against the
// realized target, computed with the exact weights that produced the
// forecast (unchanged since -- Process never trains). So the reported
// PredictionError (the training loss on the realized transition) must equal
// ForecastError (the independently-cached forecast vs. reality) to floating
// point. The old autoencoder trained on (stateVector -> stateVector), whose
// loss has no such relationship to the cross-cycle ForecastError -- this
// identity is the signature of training on the realized NEXT observation.
func TestRunPredictiveCycleTrainsForwardModelOnRealizedTransition(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	if _, err := engine.RunPredictiveCycle(ctx, "color3-cell0-0 north", "goal", true); err != nil {
		t.Fatalf("FAIL: unexpected error on cycle 1: %v", err)
	}
	// Cycle 1 does no predictor training (no previous specialist yet), so
	// PrevPredictor/PrevStateVector must be armed for cycle 2 to train.
	if engine.PrevPredictor == nil || engine.PrevStateVector == nil {
		t.Fatalf("FAIL: expected PrevPredictor/PrevStateVector to be cached after cycle 1")
	}

	res2, err := engine.RunPredictiveCycle(ctx, "color5-cell1-1 south", "goal", true)
	if err != nil {
		t.Fatalf("FAIL: unexpected error on cycle 2: %v", err)
	}
	if !approxEqual(res2.PredictionError, res2.ForecastError) {
		t.Fatalf("FAIL: expected PredictionError (training loss on the realized transition) to equal ForecastError (cached forecast vs. reality) -- the forward-model signature -- got predErr=%.6f forecastErr=%.6f",
			res2.PredictionError, res2.ForecastError)
	}
	// Sanity: it must actually be a nonzero error here (the two observations
	// differ), otherwise the identity above could hold trivially at 0.
	if res2.ForecastError == 0 {
		t.Fatalf("FAIL: expected a nonzero forecast error between two different observations, got 0")
	}
}
