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
