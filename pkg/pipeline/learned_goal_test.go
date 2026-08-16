package pipeline

import (
	"math"
	"testing"
)

// TestLearnedPreferenceFromSuccess: before any success the agent has no goal
// (preference 0 everywhere); after remembering a winning state, states that
// resemble it score high and dissimilar ones low -- the goal is LEARNED from
// the success, not guessed.
func TestLearnedPreferenceFromSuccess(t *testing.T) {
	e := NewEngine()
	win := ObservationVector("won-cell0-0 clean-cell1-1 clean-cell1-1")
	other := ObservationVector("chaos-cell5-5 mess-cell7-3 junk-cell2-8")

	if e.HasLearnedGoal() {
		t.Fatalf("FAIL: a fresh engine should have no learned goal")
	}
	if e.LearnedPreference(win) != 0 {
		t.Fatalf("FAIL: preference should be 0 before any goal is learned")
	}

	e.RememberGoalState(win) // completed a level in state `win`

	if !e.HasLearnedGoal() {
		t.Fatalf("FAIL: should have a learned goal after remembering a win")
	}
	pWin := e.LearnedPreference(win)     // identical -> ~1
	pOther := e.LearnedPreference(other) // dissimilar -> lower
	if math.Abs(pWin-1.0) > 1e-9 {
		t.Fatalf("FAIL: the winning state should score ~1.0 against itself, got %.4f", pWin)
	}
	if !(pWin > pOther) {
		t.Fatalf("FAIL: a dissimilar state (%.4f) should score below the winning state (%.4f)", pOther, pWin)
	}
}

// TestDangerProximityFromDeath is the mirror of the goal test: before any death
// the agent has no aversion (proximity 0 everywhere); after remembering a state
// it died from, states resembling it score high (to be subtracted from
// preference) and dissimilar ones low -- learned self-preservation.
func TestDangerProximityFromDeath(t *testing.T) {
	e := NewEngine()
	death := ObservationVector("edge-cell9-9 spike-cell9-8 spike-cell9-8")
	safe := ObservationVector("open-cell1-1 calm-cell2-2 calm-cell2-2")

	if e.DangerProximity(death) != 0 {
		t.Fatalf("FAIL: danger proximity should be 0 before any death")
	}

	e.RememberDangerState(death) // an action from `death` caused GAME_OVER

	pDeath := e.DangerProximity(death)
	pSafe := e.DangerProximity(safe)
	if math.Abs(pDeath-1.0) > 1e-9 {
		t.Fatalf("FAIL: the death state should score ~1.0 against itself, got %.4f", pDeath)
	}
	if !(pDeath > pSafe) {
		t.Fatalf("FAIL: a dissimilar safe state (%.4f) should score below the death state (%.4f)", pSafe, pDeath)
	}
}
