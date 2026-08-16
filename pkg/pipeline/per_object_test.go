package pipeline

import (
	"context"
	"testing"
)

// TestPerObjectTokensDifferentiate: conditioning the forward model on a
// per-object click token ("click-<objlabel>") makes preference gain accrue to
// THAT object, and BestAction prefers it over a sibling object that earned
// nothing. This is what breaks the uniform-graph symmetry: when every click was
// the single token "click", no per-object signal could exist.
func TestPerObjectTokensDifferentiate(t *testing.T) {
	e := NewEngine()
	ctx := context.Background()
	// Two cycles populate PrevPredictor/PrevStateVector so conditioning is live.
	if _, err := e.RunPredictiveCycle(ctx, "obj4-color5 obj7-color0", "goal", true); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if _, err := e.RunPredictiveCycle(ctx, "obj4-color5 obj7-color0", "goal", true); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}

	e.ConditionForecastOnAction("click-obj4-color5")
	if e.PendingActionToken != "click-obj4-color5" {
		t.Fatalf("expected pending token click-obj4-color5, got %q", e.PendingActionToken)
	}
	e.AttributePreferenceGain(0.5) // credit obj4 a real preference gain

	got := e.BestAction([]string{"click-obj4-color5", "click-obj7-color0"})
	if got != "click-obj4-color5" {
		t.Fatalf("BestAction should prefer the credited per-object token, got %q", got)
	}
	if e.ActionPreferenceGain["click-obj7-color0"] != 0 {
		t.Fatalf("the uncredited sibling object should have zero preference gain, got %.4f", e.ActionPreferenceGain["click-obj7-color0"])
	}
}
