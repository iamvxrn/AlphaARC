package graph

import (
	"testing"
)

func TestDeadAgentReversalFix(t *testing.T) {
	wA, wB := 0.6, 0.55
	penalty := 0.15
	boost := 0.05

	inA := []float64{0.90, 0.85, 0.80, 0.10, 0.10, 0.10, 0.10}
	inB := []float64{0.40, 0.35, 0.30, 1.00, 1.00, 1.00, 1.00}

	bWonAfterReversal := false

	for r := 0; r < len(inA); r++ {
		actA := wA * inA[r]
		actB := wB * inB[r]

		UpdateCandidateWeights(&wA, &wB, inA[r], inB[r], boost, penalty)

		if actB > actA && r >= 3 {
			bWonAfterReversal = true
		}
	}

	if !bWonAfterReversal {
		t.Fatalf("FAIL: Candidate B failed to recover after input reversal! Final wA=%.4f wB=%.4f", wA, wB)
	}

	if wB <= MinWeightFloor {
		t.Fatalf("FAIL: Candidate B weight did not grow after winning post-reversal rounds. wB=%.4f", wB)
	}

	t.Logf("SUCCESS: Candidate B recovered after reversal! Final wA=%.4f, wB=%.4f", wA, wB)
}

func TestRecoverySpeedFromFloor(t *testing.T) {
	wA := 1.0
	wB := MinWeightFloor
	penalty := 0.15
	boost := 0.05

	inA := 0.01
	inB := 1.00

	activationParityRound := -1
	weightParityRound := -1

	for round := 1; round <= 50; round++ {
		actA := wA * inA
		actB := wB * inB

		if actB > actA && activationParityRound == -1 {
			activationParityRound = round
		}
		if wB > wA && weightParityRound == -1 {
			weightParityRound = round
		}

		t.Logf("Round %2d: inA=%.2f inB=%.2f | actA=%.6f actB=%.6f | wA=%.4f wB=%.4f",
			round, inA, inB, actA, actB, wA, wB)

		if weightParityRound != -1 && activationParityRound != -1 {
			break
		}

		UpdateCandidateWeights(&wA, &wB, inA, inB, boost, penalty)
	}

	t.Logf("Recovery Metrics from floor (wB_start=%.4f):", MinWeightFloor)
	t.Logf("  Rounds to Activation Parity (actB > actA): %d", activationParityRound)
	t.Logf("  Rounds to Weight Parity     (wB > wA)   : %d", weightParityRound)

	if activationParityRound < 1 || activationParityRound > 3 {
		t.Fatalf("FAIL: Activation parity took too long or failed! Got round %d", activationParityRound)
	}

	if weightParityRound < 1 || weightParityRound > 15 {
		t.Fatalf("FAIL: Candidate B failed to achieve weight parity within 15 rounds! Got round %d", weightParityRound)
	}
}

func TestRecoverySpeedLowStimulus(t *testing.T) {
	penalty := 0.15
	boost := 0.05
	inA := 0.01
	stimuli := []float64{0.50, 0.20, 0.15, 0.10, 0.05, 0.01}

	t.Logf("=== LOW STIMULUS RECOVERY MATRIX (wB_start=%.4f, wA_start=1.0) ===", MinWeightFloor)

	for _, inB := range stimuli {
		wA := 1.0
		wB := MinWeightFloor
		actParityRound := -1
		wtParityRound := -1

		for round := 1; round <= 100; round++ {
			actA := wA * inA
			actB := wB * inB

			if actB > actA && actParityRound == -1 {
				actParityRound = round
			}
			if wB > wA && wtParityRound == -1 {
				wtParityRound = round
			}

			if actParityRound != -1 && wtParityRound != -1 {
				break
			}

			UpdateCandidateWeights(&wA, &wB, inA, inB, boost, penalty)
		}

		t.Logf("Stimulus inB=%.2f | ActParityRound: %3d | WtParityRound: %3d | Final wA=%.4f wB=%.4f",
			inB, actParityRound, wtParityRound, wA, wB)
	}
}
