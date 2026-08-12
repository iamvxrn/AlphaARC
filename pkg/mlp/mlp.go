package mlp

import (
	"math"
	"math/rand"
)

// MLP represents a lightweight, high-performance Multi-Layer Perceptron (Input -> Hidden -> Output)
// designed for real-time online continuous learning on low-power CPU/GPU hardware.
type MLP struct {
	InDim     int
	HiddenDim int
	OutDim    int
	W1        [][]float64 // HiddenDim x InDim
	B1        []float64   // HiddenDim
	W2        [][]float64 // OutDim x HiddenDim
	B2        []float64   // OutDim
	LR        float64
}

func NewMLP(inDim, hiddenDim, outDim int, lr float64, seed int64) *MLP {
	rng := rand.New(rand.NewSource(seed))

	// Xavier / He initialization
	w1 := make([][]float64, hiddenDim)
	scale1 := math.Sqrt(2.0 / float64(inDim))
	for i := 0; i < hiddenDim; i++ {
		w1[i] = make([]float64, inDim)
		for j := 0; j < inDim; j++ {
			w1[i][j] = (rng.Float64()*2.0 - 1.0) * scale1
		}
	}

	w2 := make([][]float64, outDim)
	scale2 := math.Sqrt(2.0 / float64(hiddenDim))
	for i := 0; i < outDim; i++ {
		w2[i] = make([]float64, hiddenDim)
		for j := 0; j < hiddenDim; j++ {
			w2[i][j] = (rng.Float64()*2.0 - 1.0) * scale2
		}
	}

	return &MLP{
		InDim:     inDim,
		HiddenDim: hiddenDim,
		OutDim:    outDim,
		W1:        w1,
		B1:        make([]float64, hiddenDim),
		W2:        w2,
		B2:        make([]float64, outDim),
		LR:        lr,
	}
}

func relu(x float64) float64 {
	if x > 0 {
		return x
	}
	return 0.01 * x // Leaky ReLU
}

func reluDeriv(x float64) float64 {
	if x > 0 {
		return 1.0
	}
	return 0.01
}

// Forward passes input through hidden Leaky-ReLU layer and linear output layer.
func (m *MLP) Forward(input []float64) (hidden []float64, output []float64) {
	hidden = make([]float64, m.HiddenDim)
	for i := 0; i < m.HiddenDim; i++ {
		sum := m.B1[i]
		for j := 0; j < m.InDim; j++ {
			sum += m.W1[i][j] * input[j]
		}
		hidden[i] = relu(sum)
	}

	output = make([]float64, m.OutDim)
	for i := 0; i < m.OutDim; i++ {
		sum := m.B2[i]
		for j := 0; j < m.HiddenDim; j++ {
			sum += m.W2[i][j] * hidden[j]
		}
		output[i] = sum
	}

	return hidden, output
}

// Train executes single-step backpropagation & online weight update, returning MSE loss.
func (m *MLP) Train(input, target []float64) float64 {
	hidden, output := m.Forward(input)

	// Step 1: Compute output errors e_k = output_k - target_k
	outGrad := make([]float64, m.OutDim)
	mseLoss := 0.0
	for k := 0; k < m.OutDim; k++ {
		diff := output[k] - target[k]
		outGrad[k] = diff
		mseLoss += diff * diff
	}
	mseLoss /= float64(m.OutDim)

	// Step 2: Compute hidden layer gradients
	hiddenGrad := make([]float64, m.HiddenDim)
	for j := 0; j < m.HiddenDim; j++ {
		sum := 0.0
		for k := 0; k < m.OutDim; k++ {
			sum += outGrad[k] * m.W2[k][j]
		}
		hiddenGrad[j] = sum * reluDeriv(hidden[j])
	}

	// Step 3: Update W2 and B2
	for k := 0; k < m.OutDim; k++ {
		m.B2[k] -= m.LR * outGrad[k]
		for j := 0; j < m.HiddenDim; j++ {
			m.W2[k][j] -= m.LR * outGrad[k] * hidden[j]
		}
	}

	// Step 4: Update W1 and B1
	for j := 0; j < m.HiddenDim; j++ {
		m.B1[j] -= m.LR * hiddenGrad[j]
		for i := 0; i < m.InDim; i++ {
			m.W1[j][i] -= m.LR * hiddenGrad[j] * input[i]
		}
	}

	return mseLoss
}
