package environment

import "testing"

func TestActionIsComplexOnlyForAction6(t *testing.T) {
	complex := Action{ID: Action6, X: 10, Y: 20}
	if !complex.IsComplex() {
		t.Fatalf("FAIL: expected Action6 to be complex")
	}

	simple := []ActionID{ActionReset, Action1, Action2, Action3, Action4, Action5, Action7Undo}
	for _, id := range simple {
		a := Action{ID: id}
		if a.IsComplex() {
			t.Fatalf("FAIL: expected action ID %d to NOT be complex", id)
		}
	}
}

func TestActionIDValuesMatchOfficialSDK(t *testing.T) {
	// Confirmed 2026-08-12 against arcprize/ARC-AGI-3-Agents'
	// tests/unit/test_core.py: RESET=0, ACTION6=6, ACTION1..5 in between.
	if ActionReset != 0 {
		t.Fatalf("FAIL: expected ActionReset==0, got %d", ActionReset)
	}
	if Action6 != 6 {
		t.Fatalf("FAIL: expected Action6==6, got %d", Action6)
	}
	if Action7Undo != 7 {
		t.Fatalf("FAIL: expected Action7Undo==7, got %d", Action7Undo)
	}
}

func TestGridDimensionsMatchRealARCAGI3(t *testing.T) {
	if GridSize != 64 {
		t.Fatalf("FAIL: expected GridSize==64, got %d", GridSize)
	}
	if NumColors != 16 {
		t.Fatalf("FAIL: expected NumColors==16, got %d", NumColors)
	}
}
