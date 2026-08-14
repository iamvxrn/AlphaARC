package pipeline

import (
	"context"
	"math"
	"testing"
)

// TestStage6TransferThermometer is the falsifiable Stage-6 experiment
// (ARCHITECTURE.md Stage 6): does an abstraction compressed on domain A
// measurably ACCELERATE learning a structurally identical, surface-different
// domain B, versus a control that never saw A?
//
// A "domain" is an alternating stream X,Y,X,Y -- the structure to learn is
// "after X comes Y". A and B share that structure exactly but use disjoint
// surface tokens. Positive transfer = the transfer engine (which learned +
// compressed A first) converges on B faster than a fresh control.
//
// This is a DIAGNOSTIC instrument, not a spec: it logs the thermometer
// reading and only fails on something actually broken (non-finite / negative
// error, wrong curve length), NOT on the transfer direction -- so it stays a
// valid measuring tool to re-run after a representation change, whichever way
// the reading then goes.
//
// Reading as of 2026-08-14 (label-bound abstraction implementation): NO
// positive transfer, slight interference (transfer ~1.1x WORSE early on B).
// Concrete cause: abstractions are nodes bound to specific token labels, so
// B's disjoint tokens make fresh nodes A's compressed abstraction never
// touches, and the only cross-talk is incidental hash-dimension overlap in
// the MLP, which mildly interferes rather than helps. This does NOT falsify
// the thesis (compression -> transferable abstraction) in principle; it
// pinpoints that THIS implementation abstracts surface labels, not relational
// structure, which is the gap a structure-aware representation would have to
// close for the reading to move.
func TestStage6TransferThermometer(t *testing.T) {
	ctx := context.Background()

	aX, aY := "aaa1-cell0-0 aaa2-cell1-1", "aaa3-cell3-3 aaa4-cell4-4"
	bX, bY := "bbb1-cell0-0 bbb2-cell1-1", "bbb3-cell3-3 bbb4-cell4-4"

	runAlt := func(e *Engine, x, y string, cycles int) []float64 {
		errs := make([]float64, cycles)
		for i := 0; i < cycles; i++ {
			obs := x
			if i%2 == 1 {
				obs = y
			}
			res, err := e.RunPredictiveCycle(ctx, obs, "goal", true)
			if err != nil {
				t.Fatalf("FAIL: cycle %d error: %v", i+1, err)
			}
			if math.IsNaN(res.ForecastError) || math.IsInf(res.ForecastError, 0) || res.ForecastError < 0 {
				t.Fatalf("FAIL: non-finite/negative forecast error %.5f at cycle %d", res.ForecastError, i+1)
			}
			errs[i] = res.ForecastError
		}
		return errs
	}

	control := NewEngine()
	controlB := runAlt(control, bX, bY, 30)

	transfer := NewEngine()
	_ = runAlt(transfer, aX, aY, 25) // learn + compress domain A first
	transferB := runAlt(transfer, bX, bY, 30)

	// Early-window mean on B (cycles 2..9, skipping the cold first cycle).
	cEarly, tEarly := mean(controlB[1:9]), mean(transferB[1:9])

	t.Logf("Stage-6 transfer thermometer:")
	t.Logf("  control  early-B forecast error (cycles 2-9) = %.5f", cEarly)
	t.Logf("  transfer early-B forecast error (cycles 2-9) = %.5f", tEarly)
	if tEarly < cEarly {
		t.Logf("  => POSITIVE transfer: %.2fx faster start on B", cEarly/tEarly)
	} else {
		t.Logf("  => NO positive transfer: transfer/control = %.2fx (prior A-domain did not help)", tEarly/cEarly)
	}
}
