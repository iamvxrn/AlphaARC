package pipeline

import "testing"

// TestBestActionPlansTowardPreference is the deliberative pragmatic loop: the
// agent must CHOOSE the action expected to move the world toward the preferred
// (meaningful) state -- not merely be rewarded after one happens to. One
// action ("orderly") reliably raises prior preference; another ("meh") never
// does. After experiencing both, BestAction must plan the orderly one.
func TestBestActionPlansTowardPreference(t *testing.T) {
	e := NewEngine()
	for i := 0; i < 20; i++ {
		e.PendingActionToken = "orderly"
		e.AttributePreferenceGain(0.1) // moves toward the goal
		e.PendingActionToken = "meh"
		e.AttributePreferenceGain(0.0) // does nothing
	}

	if e.ActionPreferenceGain["orderly"] <= e.ActionPreferenceGain["meh"] {
		t.Fatalf("FAIL: preference gain not attributed per action: orderly=%.4f meh=%.4f",
			e.ActionPreferenceGain["orderly"], e.ActionPreferenceGain["meh"])
	}
	if got := e.BestAction([]string{"orderly", "meh"}); got != "orderly" {
		t.Fatalf("FAIL: planner did not steer toward the preference-raising action, chose %q", got)
	}

	// A small epistemic (competence) signal breaks ties but doesn't hijack a
	// clear goal: give meh a modest learning-progress that, at the epistemic
	// weight, stays below orderly's pragmatic lead.
	e.ActionLearningProgress["meh"] = 0.2 // 0.3*0.2 = 0.06 < orderly's ~0.1 pragmatic
	if got := e.BestAction([]string{"orderly", "meh"}); got != "orderly" {
		t.Fatalf("FAIL: a modest epistemic value hijacked the goal, chose %q", got)
	}
}
