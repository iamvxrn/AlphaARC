package memory

import (
	"alphaarc/pkg/core"
	"math/rand"
	"testing"
)

func TestModernHopfieldCapacity(t *testing.T) {
	sys := core.NewSystem()
	rng := rand.New(rand.NewSource(99))

	N := 20
	patternCounts := []int{2, 5, 8, 12, 15, 20}

	t.Logf("=== HOPFIELD CAPACITY COMPARISON (Dim N=%d, 10%% Noise / 2 Bit Flips) ===", N)

	for _, K := range patternCounts {
		classical := NewClassicalHopfield(N)
		modern := NewModernHopfield(N, 10.0)

		patterns := make([][]float64, K)
		for p := 0; p < K; p++ {
			pat := make([]float64, N)
			for i := 0; i < N; i++ {
				if rng.Float64() < 0.5 {
					pat[i] = 1.0
				} else {
					pat[i] = -1.0
				}
			}
			patterns[p] = pat
			classical.StorePattern(sys, pat)
			modern.StorePattern(sys, pat)
		}

		classCorrect := 0
		modCorrect := 0

		for p := 0; p < K; p++ {
			noisy := append([]float64{}, patterns[p]...)
			for _, idx := range rng.Perm(N)[:2] {
				noisy[idx] = -noisy[idx]
			}

			recClass := classical.Recall(sys, noisy, 20)
			recMod := modern.Recall(sys, noisy)

			if isBipolarEqual(recClass, patterns[p]) {
				classCorrect++
			}
			if isBipolarEqualThreshold(recMod, patterns[p]) {
				modCorrect++
			}
		}

		t.Logf("Patterns K=%2d | Classical (1982): %2d/%2d (%5.1f%%) | Modern (2020): %2d/%2d (%5.1f%%)",
			K, classCorrect, K, float64(classCorrect)/float64(K)*100.0,
			modCorrect, K, float64(modCorrect)/float64(K)*100.0)

		if K <= 15 && modCorrect < K {
			t.Fatalf("FAIL: Modern Hopfield failed to achieve 100%% recall at K=%d!", K)
		}
	}
}

func TestModernHopfieldStressTestKN(t *testing.T) {
	sys := core.NewSystem()
	rng := rand.New(rand.NewSource(123))

	N := 20
	stressPatternCounts := []int{10, 20, 40, 60, 80, 100, 150, 200}

	t.Logf("=== MODERN HOPFIELD STRESS TEST K >> N (Dim N=%d, Beta=10.0, 10%% Noise) ===", N)

	for _, K := range stressPatternCounts {
		classical := NewClassicalHopfield(N)
		modern := NewModernHopfield(N, 10.0)

		patterns := make([][]float64, K)
		for p := 0; p < K; p++ {
			pat := make([]float64, N)
			for i := 0; i < N; i++ {
				if rng.Float64() < 0.5 {
					pat[i] = 1.0
				} else {
					pat[i] = -1.0
				}
			}
			patterns[p] = pat
			classical.StorePattern(sys, pat)
			modern.StorePattern(sys, pat)
		}

		classCorrect := 0
		modCorrect := 0

		for p := 0; p < K; p++ {
			noisy := append([]float64{}, patterns[p]...)
			for _, idx := range rng.Perm(N)[:2] {
				noisy[idx] = -noisy[idx]
			}

			recClass := classical.Recall(sys, noisy, 20)
			recMod := modern.Recall(sys, noisy)

			if isBipolarEqual(recClass, patterns[p]) {
				classCorrect++
			}
			if isBipolarEqualThreshold(recMod, patterns[p]) {
				modCorrect++
			}
		}

		t.Logf("Stress K=%3d (K/N=%4.1fx) | Classical: %3d/%3d (%5.1f%%) | Modern: %3d/%3d (%5.1f%%)",
			K, float64(K)/float64(N), classCorrect, K, float64(classCorrect)/float64(K)*100.0,
			modCorrect, K, float64(modCorrect)/float64(K)*100.0)
	}
}

func isBipolarEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isBipolarEqualThreshold(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		signA := 1.0
		if a[i] < 0 {
			signA = -1.0
		}
		if signA != b[i] {
			return false
		}
	}
	return true
}
