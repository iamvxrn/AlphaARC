package pipeline

import (
	"context"
	"testing"
)

// TestActionConditioningLearnsActionDependentTransitions is the point of
// branch A: a forward model p(next | state) cannot learn a world where the
// SAME state leads to DIFFERENT next states depending on the action (one
// input, two targets -- unlearnable). p(next | state, action) can.
//
// World: from state S, action A -> outcome P, action B -> outcome Q; then a
// step back to S. The action alternates each visit to S, so an unconditioned
// model sees S -> P and S -> Q interchangeably and can't predict the outcome;
// a model conditioned on the action (ConditionForecastOnAction) sees
// (S,A) -> P and (S,B) -> Q and learns to predict each. Success = the
// conditioned engine's outcome-forecast error converges far below the
// unconditioned one's.
func TestActionConditioningLearnsActionDependentTransitions(t *testing.T) {
	ctx := context.Background()
	const S, P, Q = "sss-cell0-0", "ppp-cell1-1", "qqq-cell2-2"
	const actA, actB, actStep = "actiona", "actionb", "actionstep"

	run := func(conditioned bool) (early, late float64) {
		e := NewEngine()
		var outErrs []float64
		useA := true
		for visit := 0; visit < 80; visit++ {
			// At S: run the cycle, then (conditioned) commit to the action.
			if _, err := e.RunPredictiveCycle(ctx, S, "goal", true); err != nil {
				t.Fatalf("FAIL: cycle at S: %v", err)
			}
			action, outcome := actA, P
			if !useA {
				action, outcome = actB, Q
			}
			if conditioned {
				e.ConditionForecastOnAction(action)
			}
			// Outcome cycle: forecast error here is the S(+action) -> outcome
			// prediction we care about.
			res, err := e.RunPredictiveCycle(ctx, outcome, "goal", true)
			if err != nil {
				t.Fatalf("FAIL: cycle at outcome: %v", err)
			}
			outErrs = append(outErrs, res.ForecastError)
			if conditioned {
				e.ConditionForecastOnAction(actStep)
			}
			useA = !useA
		}
		return mean(outErrs[2:12]), mean(outErrs[len(outErrs)-10:])
	}

	condEarly, condLate := run(true)
	uncEarly, uncLate := run(false)
	t.Logf("outcome forecast error -- conditioned: early=%.4f late=%.4f | unconditioned: early=%.4f late=%.4f", condEarly, condLate, uncEarly, uncLate)

	if !(condLate < uncLate*0.5) {
		t.Fatalf("FAIL: action conditioning did not learn the action-dependent transition: conditioned late %.4f not < half of unconditioned late %.4f", condLate, uncLate)
	}
}
