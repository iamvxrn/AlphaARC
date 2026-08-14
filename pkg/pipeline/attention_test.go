package pipeline

import (
	"context"
	"testing"
)

// TestRegisterForecastErrorIgnoresColdStartLargeErrors is the core guard the
// user's own live log motivated: an untrained forward model produces large
// errors that ARE the norm, not anomalies. During warmup no error, however
// large, may be flagged acute -- otherwise the agent would tunnel-vision on
// startup noise. Only a genuine spike ABOVE a settled running norm counts.
func TestRegisterForecastErrorIgnoresColdStartLargeErrors(t *testing.T) {
	e := NewEngine()

	// Warmup: minForecastSamplesForSurprise consecutive large errors. Even
	// though each is huge, none may fire -- they're establishing the norm,
	// not deviating from it, and warmup isn't satisfied yet.
	for i := 0; i < minForecastSamplesForSurprise; i++ {
		if acute, _ := e.registerForecastError(5.0, true); acute {
			t.Fatalf("FAIL: cold-start large error #%d flagged acute during warmup", i+1)
		}
	}

	// Still 5.0 -- past warmup now, but 5.0 is exactly the running norm, not a
	// spike above it, so it must NOT be acute.
	if acute, _ := e.registerForecastError(5.0, true); acute {
		t.Fatalf("FAIL: an error equal to the running norm was flagged acute")
	}
}

// TestRegisterForecastErrorFlagsGenuineSpike: after the norm settles LOW, an
// error well above it is a real surprise and must be flagged.
func TestRegisterForecastErrorFlagsGenuineSpike(t *testing.T) {
	e := NewEngine()

	// Settle the running norm near 0.05 with several small errors (past warmup).
	for i := 0; i < minForecastSamplesForSurprise+3; i++ {
		if acute, _ := e.registerForecastError(0.05, true); acute {
			t.Fatalf("FAIL: a steady small error #%d was flagged acute", i+1)
		}
	}

	// A spike an order of magnitude above the settled norm must fire.
	if acute, _ := e.registerForecastError(1.0, true); !acute {
		t.Fatalf("FAIL: expected a spike (1.0) far above the settled norm (~0.05) to be flagged acute, got false (EMA=%.4f)", e.ForecastErrorEMA)
	}
}

// TestRegisterForecastErrorNoForecastNeverAcute: a cycle with no prior
// prediction (the very first cycle) can't be a surprise and mustn't advance
// the sample count.
func TestRegisterForecastErrorNoForecastNeverAcute(t *testing.T) {
	e := NewEngine()
	if acute, _ := e.registerForecastError(0.0, false); acute {
		t.Fatalf("FAIL: a cycle with no prior forecast was flagged acute")
	}
	if e.ForecastSamples != 0 {
		t.Fatalf("FAIL: a no-forecast cycle advanced ForecastSamples to %d", e.ForecastSamples)
	}
}

// TestRegisterForecastErrorAbsoluteFloorKillsTinyOscillationFalsePositive is
// the regression for the 1000-action live run: a dead 2-state loop whose
// error oscillates between two tiny values (0.0011 / 0.0045) flagged "acute
// surprise" on the high phase ~445 times, because 0.0045 > 1.5x the blended
// norm even though both errors are microscopic. The absolute floor must
// suppress that: an error above the relative threshold but below the floor is
// NOT a surprise.
func TestRegisterForecastErrorAbsoluteFloorKillsTinyOscillationFalsePositive(t *testing.T) {
	e := NewEngine()
	// Reproduce the loop: alternate 0.0011 / 0.0045 for a while past warmup.
	for i := 0; i < minForecastSamplesForSurprise+2; i++ {
		lo, _ := e.registerForecastError(0.0011, true)
		hi, _ := e.registerForecastError(0.0045, true)
		if lo || hi {
			t.Fatalf("FAIL: a tiny 2-state oscillation (0.0011/0.0045) flagged acute surprise at iter %d -- absolute floor not applied", i)
		}
	}
	// Sanity that the floor isn't just always-false: a real spike well above
	// the floor still fires.
	if acute, _ := e.registerForecastError(1.0, true); !acute {
		t.Fatalf("FAIL: a genuine 1.0 spike above both the norm and the floor did not fire")
	}
}

// TestRegisterForecastErrorSettledIsRelativeToNorm is the regression for the
// A-broke-B interaction: wiring structural tokens raised every forecast error,
// and an ABSOLUTE predictability cutoff (0.05) then sat below 87% of them, so
// the epistemic escape stopped firing and the agent re-locked. With a relative
// "settled" signal, an error well below its running norm counts as settled
// even when that norm is high -- and an error sitting at the norm does not.
func TestRegisterForecastErrorSettledIsRelativeToNorm(t *testing.T) {
	e := NewEngine()
	// Establish a HIGH norm (structure-inflated errors), past warmup.
	for i := 0; i < minForecastSamplesForSurprise+3; i++ {
		e.registerForecastError(0.30, true)
	}
	// 0.10 is well above the old absolute 0.05 cutoff, but well below the ~0.30
	// running norm -> must be settled now.
	if _, settled := e.registerForecastError(0.10, true); !settled {
		t.Fatalf("FAIL: error 0.10 far below the running norm (EMA=%.4f) not flagged settled -- relativization not applied", e.ForecastErrorEMA)
	}
	// An error at/above the norm is not settled.
	if _, settled := e.registerForecastError(0.40, true); settled {
		t.Fatalf("FAIL: error 0.40 above the running norm was flagged settled")
	}
}

func TestChangedTokens(t *testing.T) {
	cases := []struct {
		name, cur, prev, want string
	}{
		{"one new token", "a b c", "a b", "c"},
		{"nothing new", "a b", "a b", ""},
		{"nothing new even if prev has extra", "a b", "a b x", ""},
		{"all new", "x y", "a b", "x y"},
		{"empty prev means all new", "a b", "", "a b"},
		{"order follows cur", "c a", "a", "c"},
	}
	for _, c := range cases {
		if got := changedTokens(c.cur, c.prev); got != c.want {
			t.Fatalf("FAIL [%s]: changedTokens(%q,%q)=%q, want %q", c.name, c.cur, c.prev, got, c.want)
		}
	}
}

// forceSurpriseReady arms an engine so the NEXT RunPredictiveCycle flags an
// acute surprise: past warmup, a settled-low norm, and a PendingPrediction
// wildly far from whatever the next observation will embed to (so
// forecastError is huge relative to the norm). PrevObservation is set so the
// changed-token diff is well-defined.
func forceSurpriseReady(e *Engine, prevObservation string) {
	e.ForecastSamples = minForecastSamplesForSurprise + 5
	e.ForecastErrorEMA = 0.01
	e.PrevObservation = prevObservation
	far := make([]float64, ObservationVectorDim)
	for i := range far {
		far[i] = 100.0
	}
	e.PendingPrediction = far
}

// TestRunPredictiveCycleNarrowsSeedsOnAcuteSurprise: with the engine armed
// for surprise and an observation whose only NEW token (vs PrevObservation)
// is one blob, attention must narrow seeding to exactly that locus of change
// -- one seeded concept, not the full three-blob frame.
func TestRunPredictiveCycleNarrowsSeedsOnAcuteSurprise(t *testing.T) {
	ctx := context.Background()
	e := NewEngine()
	forceSurpriseReady(e, "color1-cell0-0 color2-cell1-1 color3-cell2-2")

	// Two of the three tokens match PrevObservation; only color9-cell9-9 is new.
	res, err := e.RunPredictiveCycle(ctx, "color1-cell0-0 color2-cell1-1 color9-cell9-9", "goal", true)
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if !res.AcuteSurprise {
		t.Fatalf("FAIL: expected an acute surprise given the armed engine, got false")
	}
	if res.SeededConcepts != 1 {
		t.Fatalf("FAIL: expected attention to narrow seeding to the 1 changed blob, got %d seeded concepts", res.SeededConcepts)
	}
}

// TestRunPredictiveCycleNoNarrowingWithoutSurprise is the control: the same
// three-blob frame on a cold engine (no surprise) seeds the full frame.
func TestRunPredictiveCycleNoNarrowingWithoutSurprise(t *testing.T) {
	ctx := context.Background()
	e := NewEngine()

	res, err := e.RunPredictiveCycle(ctx, "color1-cell0-0 color2-cell1-1 color9-cell9-9", "goal", true)
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if res.AcuteSurprise {
		t.Fatalf("FAIL: a cold engine's first cycle can't be an acute surprise")
	}
	if res.SeededConcepts != 3 {
		t.Fatalf("FAIL: expected the full 3-blob frame to be seeded without narrowing, got %d", res.SeededConcepts)
	}
}

// TestRunPredictiveCycleAcuteSurpriseRaisesCortisol: cortisol (previously
// moved only by drive error) must rise when an acute surprise fires, giving
// it its first forecast-driven input.
func TestRunPredictiveCycleAcuteSurpriseRaisesCortisol(t *testing.T) {
	ctx := context.Background()
	e := NewEngine()
	before := e.Homeostasis.Cortisol
	forceSurpriseReady(e, "color1-cell0-0 color2-cell1-1")

	res, err := e.RunPredictiveCycle(ctx, "color1-cell0-0 color9-cell9-9", "goal", true)
	if err != nil {
		t.Fatalf("FAIL: unexpected error: %v", err)
	}
	if !res.AcuteSurprise {
		t.Fatalf("FAIL: expected acute surprise given the armed engine")
	}
	if e.Homeostasis.Cortisol <= before {
		t.Fatalf("FAIL: expected an acute surprise to raise Cortisol above %.4f, got %.4f", before, e.Homeostasis.Cortisol)
	}
}
