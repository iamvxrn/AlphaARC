package memory

import "protaxon/pkg/core"

type ClassicalHopfield struct {
	N int
	W [][]float64
}

func NewClassicalHopfield(n int) *ClassicalHopfield {
	w := make([][]float64, n)
	for i := range w {
		w[i] = make([]float64, n)
	}
	return &ClassicalHopfield{
		N: n,
		W: w,
	}
}

func (ch *ClassicalHopfield) StorePattern(sys *core.System, pattern []float64) {
	sys.OnlineGuard("ClassicalHopfield.StorePattern")
	for i := 0; i < ch.N; i++ {
		for j := 0; j < ch.N; j++ {
			if i == j {
				continue
			}
			ch.W[i][j] += (pattern[i] * pattern[j]) / float64(ch.N)
		}
	}
}

func (ch *ClassicalHopfield) Recall(sys *core.System, input []float64, maxIter int) []float64 {
	sys.OnlineGuard("ClassicalHopfield.Recall")
	state := append([]float64{}, input...)
	for it := 0; it < maxIter; it++ {
		next := make([]float64, ch.N)
		changed := false
		for i := 0; i < ch.N; i++ {
			sum := 0.0
			for j := 0; j < ch.N; j++ {
				sum += ch.W[i][j] * state[j]
			}
			if sum >= 0 {
				next[i] = 1.0
			} else {
				next[i] = -1.0
			}
			if next[i] != state[i] {
				changed = true
			}
		}
		state = next
		if !changed {
			break
		}
	}
	return state
}
