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

// TestRunPredictiveCycleDopamineRisesWithAccurateForecastVsInaccurate is a
// differential test: two engines run an identical bootstrap cycle 1 (same
// deterministic seeds, same observation, same actualSuccess -- NewEngine
// gives every fresh engine the same seeded MLP weights, so both evolve
// identically). Before cycle 2, one engine's PendingPrediction is
// overridden to exactly match cycle 2's real ObservationVector (a perfect
// forecast, ForecastError=0) and the other's is overridden to something
// wildly far away (an inaccurate forecast, large ForecastError) -- isolating
// the forecastError -> Dopamine wiring from whatever the real Predictor MLP
// happened to output, which neither engine's cycle-2 Dopamine computation
// otherwise depends on differently (both go through an identical
// drive-error-delta contribution from UpdateHormones, since Energy/
// Curiosity/Stress evolve identically on both engines). The accurate
// forecast must leave Dopamine strictly higher than the inaccurate one.
func TestRunPredictiveCycleDopamineRisesWithAccurateForecastVsInaccurate(t *testing.T) {
	ctx := context.Background()
	engineAccurate := NewEngine()
	engineInaccurate := NewEngine()

	obs1 := "color3-cell0-0 north"
	if _, err := engineAccurate.RunPredictiveCycle(ctx, obs1, "goal", true); err != nil {
		t.Fatalf("FAIL: unexpected error (accurate engine, cycle 1): %v", err)
	}
	if _, err := engineInaccurate.RunPredictiveCycle(ctx, obs1, "goal", true); err != nil {
		t.Fatalf("FAIL: unexpected error (inaccurate engine, cycle 1): %v", err)
	}

	obs2 := "color5-cell1-1 south"
	engineAccurate.PendingPrediction = ObservationVector(obs2)
	far := make([]float64, ObservationVectorDim)
	for i := range far {
		far[i] = 100.0
	}
	engineInaccurate.PendingPrediction = far

	if _, err := engineAccurate.RunPredictiveCycle(ctx, obs2, "goal", true); err != nil {
		t.Fatalf("FAIL: unexpected error (accurate engine, cycle 2): %v", err)
	}
	if _, err := engineInaccurate.RunPredictiveCycle(ctx, obs2, "goal", true); err != nil {
		t.Fatalf("FAIL: unexpected error (inaccurate engine, cycle 2): %v", err)
	}

	if engineAccurate.Homeostasis.Dopamine <= engineInaccurate.Homeostasis.Dopamine {
		t.Fatalf("FAIL: expected an accurate forecast to leave Dopamine higher than a wildly inaccurate one, got accurate=%.4f inaccurate=%.4f",
			engineAccurate.Homeostasis.Dopamine, engineInaccurate.Homeostasis.Dopamine)
	}
}
