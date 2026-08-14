package pipeline

import (
	"context"
	"fmt"
	"testing"
)

// TestSynergyChoosesActionWithMostCompetenceGain is the payoff of merging the
// three branches: goal-directed action selection with no external reward,
// reachable only with BOTH the action-conditioned forward model (A) and the
// learning-progress drive (B). One action ("learnable") always leads to a
// predictable outcome the model masters (competence gain); another ("noisy")
// leads to a fresh random outcome every time (nothing to learn). After
// experiencing both, the agent must prefer the learnable one -- it pursues
// the action it is learning the most from.
func TestSynergyChoosesActionWithMostCompetenceGain(t *testing.T) {
	ctx := context.Background()
	e := NewEngine()
	const learnable, noisy = "learnable", "noisy"
	const P = "ppp-cell0-0" // the learnable action's fixed, predictable outcome

	cur := "start-cell2-2"
	useLearnable := true
	for i := 0; i < 120; i++ {
		if _, err := e.RunPredictiveCycle(ctx, cur, "goal", true); err != nil {
			t.Fatalf("FAIL: %v", err)
		}
		if useLearnable {
			e.ConditionForecastOnAction(learnable)
			cur = P // predictable outcome -> the model learns it -> competence gain
		} else {
			e.ConditionForecastOnAction(noisy)
			cur = fmt.Sprintf("rr%d-cell1-1", i) // fresh noise -> no learning
		}
		useLearnable = !useLearnable
	}

	lpLearn := e.ActionLearningProgress[learnable]
	lpNoise := e.ActionLearningProgress[noisy]
	t.Logf("per-action learning progress -- learnable=%.5f noisy=%.5f", lpLearn, lpNoise)

	if got := e.BestCompetenceAction([]string{learnable, noisy}); got != learnable {
		t.Fatalf("FAIL: goal-directed selection picked %q, not the action with more competence gain (%q): learnable=%.5f noisy=%.5f", got, learnable, lpLearn, lpNoise)
	}
	if !(lpLearn > lpNoise) {
		t.Fatalf("FAIL: learnable action did not accrue more learning progress than noise: %.5f vs %.5f", lpLearn, lpNoise)
	}
}
