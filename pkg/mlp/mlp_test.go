package mlp

import (
	"math"
	"testing"
)

func TestTinyMLPOnlineLearning(t *testing.T) {
	// 4-Input -> 8-Hidden -> 4-Output tiny MLP
	net := NewMLP(4, 8, 4, 0.05, 42)

	input := []float64{1.0, -0.5, 0.2, 0.8}
	target := []float64{-1.0, 0.5, -0.2, -0.8}

	initialLoss := net.Train(input, target)

	t.Logf("Tiny MLP Online Learning Test:")
	t.Logf("  Initial MSE Loss: %.6f", initialLoss)

	finalLoss := 0.0
	for step := 0; step < 100; step++ {
		finalLoss = net.Train(input, target)
	}

	t.Logf("  Final MSE Loss after 100 steps: %.6f", finalLoss)

	if finalLoss >= initialLoss {
		t.Fatalf("FAIL: Tiny MLP failed to reduce MSE loss! Initial=%.6f, Final=%.6f", initialLoss, finalLoss)
	}

	_, output := net.Forward(input)
	for i := range target {
		diff := math.Abs(output[i] - target[i])
		if diff > 0.2 {
			t.Fatalf("FAIL: Output[%d] = %.4f differs from Target[%d] = %.4f by %.4f (>0.2)", i, output[i], i, target[i], diff)
		}
	}

	t.Logf("Tiny MLP Test PASS: Loss reduced from %.6f to %.6f", initialLoss, finalLoss)
}
