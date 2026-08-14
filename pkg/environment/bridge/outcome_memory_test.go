package bridge

import "testing"

func TestOutcomeMemorySuccessRateOnNeverSeenLabel(t *testing.T) {
	m := NewOutcomeMemory()
	rate, attempts := m.SuccessRate("color2-cell0-0")
	if attempts != 0 {
		t.Fatalf("FAIL: expected 0 attempts for a never-recorded label, got %d", attempts)
	}
	if rate != 0 {
		t.Fatalf("FAIL: expected rate 0 for a never-recorded label, got %.4f", rate)
	}
}

func TestOutcomeMemoryRecordIgnoresBlankLabel(t *testing.T) {
	m := NewOutcomeMemory()
	m.Record("", true)
	m.Record("", false)
	if _, attempts := m.SuccessRate(""); attempts != 0 {
		t.Fatalf("FAIL: expected blank label to never be recorded, got %d attempts", attempts)
	}
}

func TestOutcomeMemorySuccessRateComputation(t *testing.T) {
	m := NewOutcomeMemory()
	m.Record("color2-cell0-0", true)
	m.Record("color2-cell0-0", true)
	m.Record("color2-cell0-0", false)
	m.Record("color2-cell0-0", true)

	rate, attempts := m.SuccessRate("color2-cell0-0")
	if attempts != 4 {
		t.Fatalf("FAIL: expected 4 attempts, got %d", attempts)
	}
	wantRate := 3.0 / 4.0
	if diff := rate - wantRate; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("FAIL: expected rate %.4f (3 successes / 4 attempts), got %.4f", wantRate, rate)
	}
}

func TestOutcomeMemoryTracksLabelsIndependently(t *testing.T) {
	m := NewOutcomeMemory()
	m.Record("color2-cell0-0", true)
	m.Record("color2-cell0-0", true)
	m.Record("color5-cell1-1", false)

	if rate, attempts := m.SuccessRate("color2-cell0-0"); attempts != 2 || rate != 1.0 {
		t.Fatalf("FAIL: expected color2-cell0-0 at rate=1.0 attempts=2, got rate=%.4f attempts=%d", rate, attempts)
	}
	if rate, attempts := m.SuccessRate("color5-cell1-1"); attempts != 1 || rate != 0.0 {
		t.Fatalf("FAIL: expected color5-cell1-1 at rate=0.0 attempts=1, got rate=%.4f attempts=%d", rate, attempts)
	}
}

// TestReinforceLevelCompletionCreditsRecentSequenceMostAndProves is branch C:
// a sparse level-completion reward must flow back along the recent click
// sequence with recency weighting -- the last click credited most, earlier
// ones less -- and lift them all to "proven", so the winning path sticks.
func TestReinforceLevelCompletionCreditsRecentSequenceMostAndProves(t *testing.T) {
	m := NewOutcomeMemory()
	// Three clicks in order, none a proxy-success on its own.
	for _, l := range []string{"L1", "L2", "L3"} {
		m.Record(l, false)
	}
	// Level completes right after L3.
	m.ReinforceLevelCompletion(5.0)

	successesOf := func(label string) int {
		rate, attempts := m.SuccessRate(label)
		return int(rate*float64(attempts) + 0.5)
	}
	s1, s2, s3 := successesOf("L1"), successesOf("L2"), successesOf("L3")
	if !(s3 > s2 && s2 > s1) {
		t.Fatalf("FAIL: level-completion credit not recency-weighted: L3=%d L2=%d L1=%d (want L3>L2>L1)", s3, s2, s1)
	}
	// The winning path must now read as proven (>= minProvenAttempts, rate > 0.5).
	for _, l := range []string{"L1", "L2", "L3"} {
		rate, attempts := m.SuccessRate(l)
		if attempts < minProvenAttempts || rate <= 0.5 {
			t.Fatalf("FAIL: %s not proven after level completion: rate=%.2f attempts=%d", l, rate, attempts)
		}
	}
}
