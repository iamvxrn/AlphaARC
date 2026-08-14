package pipeline

import (
	"context"
	"fmt"
	"testing"
)

// TestLearningProgressMarksLearnableRegime is the point of branch B: an
// intrinsic drive that gives the agent something to want with no external
// reward. LearningProgress (the forward model's rate of improvement) must be
// clearly positive precisely while something learnable is being learned, and
// fall to ~0 both once it's mastered (error flat low) and on unlearnable
// noise (error flat high) -- distinguishing it from surprise (high error) and
// settled (low error), which can't tell "still learning" from "stuck".
func TestLearningProgressMarksLearnableRegime(t *testing.T) {
	ctx := context.Background()

	// Learnable: an alternating A/B stream the model learns to predict.
	const A, B = "aaa-cell0-0", "bbb-cell1-1"
	learn := NewEngine()
	var lpLearning, lpMastered []float64
	for i := 0; i < 70; i++ {
		obs := A
		if i%2 == 1 {
			obs = B
		}
		res, err := learn.RunPredictiveCycle(ctx, obs, "goal", true)
		if err != nil {
			t.Fatalf("FAIL: %v", err)
		}
		if i >= 4 && i < 18 { // during active learning
			lpLearning = append(lpLearning, res.LearningProgress)
		}
		if i >= 55 { // long after mastery
			lpMastered = append(lpMastered, res.LearningProgress)
		}
	}

	// Noise: a fresh, distinct observation every cycle -- nothing to learn.
	noise := NewEngine()
	var lpNoise []float64
	for i := 0; i < 70; i++ {
		obs := fmt.Sprintf("noise%d-cell%d-%d", i, i%7, i%5)
		res, err := noise.RunPredictiveCycle(ctx, obs, "goal", true)
		if err != nil {
			t.Fatalf("FAIL: %v", err)
		}
		if i >= 4 {
			lpNoise = append(lpNoise, res.LearningProgress)
		}
	}

	learning, mastered, noiseLP := mean(lpLearning), mean(lpMastered), mean(lpNoise)
	t.Logf("LearningProgress -- learning=%.5f  mastered=%.5f  noise=%.5f", learning, mastered, noiseLP)

	if learning <= 0.001 {
		t.Fatalf("FAIL: no positive learning progress during active learning (%.5f)", learning)
	}
	if !(learning > 3*mastered) {
		t.Fatalf("FAIL: learning progress did not decay after mastery: learning=%.5f not > 3x mastered=%.5f", learning, mastered)
	}
	if !(learning > 3*noiseLP) {
		t.Fatalf("FAIL: learning progress did not distinguish learnable from noise: learning=%.5f not > 3x noise=%.5f", learning, noiseLP)
	}
}
