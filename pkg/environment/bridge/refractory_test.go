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
