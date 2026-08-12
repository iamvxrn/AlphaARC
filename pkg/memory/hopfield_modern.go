package memory

import (
	"protaxon/pkg/core"
	"math"
)

type ModernHopfield struct {
	N       int
	Beta    float64
	Memory  [][]float64 // K stored pattern vectors of dimension N
}

func NewModernHopfield(n int, beta float64) *ModernHopfield {
	return &ModernHopfield{
		N:      n,
		Beta:   beta,
		Memory: make([][]float64, 0),
	}
}

// StorePattern adds a pattern to stored memories matrix X.
func (mh *ModernHopfield) StorePattern(sys *core.System, pattern []float64) {
	sys.OnlineGuard("ModernHopfield.StorePattern")
	cp := make([]float64, mh.N)
	copy(cp, pattern)
	mh.Memory = append(mh.Memory, cp)
}

// Recall performs one-step Softmax Attention retrieval:
// Recall(q) = X * Softmax(beta * X^T * q)
func (mh *ModernHopfield) Recall(sys *core.System, query []float64) []float64 {
	sys.OnlineGuard("ModernHopfield.Recall")
	K := len(mh.Memory)
	if K == 0 {
		out := make([]float64, mh.N)
		copy(out, query)
		return out
	}

	// Step 1: Compute inner products X^T * q
	scores := make([]float64, K)
	maxScore := math.Inf(-1)
	for k := 0; k < K; k++ {
		dot := 0.0
		for i := 0; i < mh.N; i++ {
			dot += mh.Memory[k][i] * query[i]
		}
		score := mh.Beta * dot
		scores[k] = score
		if score > maxScore {
			maxScore = score
		}
	}

	// Step 2: Softmax over scores
	weights := make([]float64, K)
	sumExp := 0.0
	for k := 0; k < K; k++ {
		expV := math.Exp(scores[k] - maxScore)
		weights[k] = expV
		sumExp += expV
	}
	for k := 0; k < K; k++ {
		weights[k] /= sumExp
	}

	// Step 3: Linear combination of stored memory patterns X * weights
	retrieved := make([]float64, mh.N)
	for k := 0; k < K; k++ {
		for i := 0; i < mh.N; i++ {
			retrieved[i] += mh.Memory[k][i] * weights[k]
		}
	}

	return retrieved
}
