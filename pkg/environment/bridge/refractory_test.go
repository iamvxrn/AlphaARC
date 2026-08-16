package bridge

import "testing"

// TestRefractoryLocksOutThenExpires: a refracted label is locked out for the
// requested number of steps (each Record = one step), then becomes selectable
// again -- Inhibition of Return with a finite cooldown, not a permanent ban.
func TestRefractoryLocksOutThenExpires(t *testing.T) {
	m := NewOutcomeMemory()
	if m.IsRefractory("color5-cell12-7") {
		t.Fatal("a fresh label should not be refractory")
	}
	m.Refract("color5-cell12-7", 3)
	if !m.IsRefractory("color5-cell12-7") {
		t.Fatal("label should be locked out immediately after Refract")
	}
	// three steps elapse (Record ticks the cooldown once per call)
	for i := 0; i < 3; i++ {
		if !m.IsRefractory("color5-cell12-7") {
			t.Fatalf("label should still be locked out at step %d", i)
		}
		m.Record("other", true)
	}
	if m.IsRefractory("color5-cell12-7") {
		t.Fatal("label should be selectable again after its cooldown elapses")
	}
}

// TestRefractExtendsNotShortens: re-confirming a label is dead extends the
// lockout to the larger value rather than resetting it to a smaller one.
func TestRefractExtendsNotShortens(t *testing.T) {
	m := NewOutcomeMemory()
	m.Refract("x", 5)
	m.Refract("x", 2) // shorter request must not win -- lockout stays 5
	// 5 steps to expire: still locked through the first 4, gone after the 5th.
	for i := 0; i < 4; i++ {
		if !m.IsRefractory("x") {
			t.Fatalf("longer lockout should still hold at step %d", i)
		}
		m.Record("y", true)
	}
	m.Record("y", true) // fifth step
	if m.IsRefractory("x") {
		t.Fatal("lockout should finally expire")
	}
}

// TestLooksLikeBlobLabelObjTokens: Fix 3 makes obj-id identity tokens the click
// vocabulary. looksLikeBlobLabel must accept "obj<N>-color<C>" (a clickable
// candidate that IS in the graph) while still rejecting the "obj<N>-<dir>"
// motion tokens and "nobj<N>" count tokens that share the observation but are
// not click targets -- otherwise winningBlobLabel could bind the graph winner
// to a non-clickable node.
func TestLooksLikeBlobLabelObjTokens(t *testing.T) {
	accept := []string{"obj4-color5", "obj12-color0", "color2-cell3-2"}
	reject := []string{"obj4-left", "obj7-up", "nobj11", "tdist2", "cmult3"}
	for _, w := range accept {
		if !looksLikeBlobLabel(w) {
			t.Errorf("expected %q to be recognized as a clickable label", w)
		}
	}
	for _, w := range reject {
		if looksLikeBlobLabel(w) {
			t.Errorf("expected %q to be rejected as a clickable label", w)
		}
	}
}
