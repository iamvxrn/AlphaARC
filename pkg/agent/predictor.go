package agent

import (
	"protaxon/pkg/core"
	"protaxon/pkg/memory"
	"protaxon/pkg/mlp"
	"context"
	"fmt"
)

type PredictorAgent struct {
	BaseAgent
	MLP      *mlp.MLP
	Hopfield *memory.ModernHopfield
	Sys      *core.System
}

func NewPredictorAgent(id string, sys *core.System, dim int) *PredictorAgent {
	// Tiny MLP: dim inputs -> 2*dim hidden -> dim predicted outputs
	tinyMLP := mlp.NewMLP(dim, dim*2, dim, 0.05, 42)
	return &PredictorAgent{
		BaseAgent: NewBaseAgent(id, RolePredictor),
		MLP:       tinyMLP,
		Hopfield:  memory.NewModernHopfield(dim, 10.0),
		Sys:       sys,
	}
}

func (p *PredictorAgent) Process(ctx context.Context, payload ContextPayload) (AgentResponse, error) {
	predictedVector := make([]float64, len(payload.StateVector))

	if len(payload.StateVector) > 0 {
		// Forward pass through tiny MLP neural network
		_, predictedVector = p.MLP.Forward(payload.StateVector)

		// Secondary associative projection via Modern Hopfield Attention
		if len(p.Hopfield.Memory) > 0 {
			hopfieldVec := p.Hopfield.Recall(p.Sys, payload.StateVector)
			for i := range predictedVector {
				predictedVector[i] = 0.7*predictedVector[i] + 0.3*hopfieldVec[i]
			}
		}
	}

	conf := 0.85
	if len(payload.ActiveNodes) == 0 {
		conf = 0.40
	}

	content := fmt.Sprintf("Micro-MLP Forecast: Neural state transition predicted across %d active nodes (conf=%.2f)",
		len(payload.ActiveNodes), conf)

	return AgentResponse{
		AgentID:     p.ID(),
		Role:        p.Role(),
		Content:     content,
		ValueVector: predictedVector,
		Confidence:  conf,
		TrustScore:  p.TrustScore(),
	}, nil
}

// TrainStep executes online continuous backpropagation on actual outcome
func (p *PredictorAgent) TrainStep(input, actualTarget []float64) float64 {
	if p.MLP != nil && len(input) > 0 && len(actualTarget) > 0 {
		loss := p.MLP.Train(input, actualTarget)
		p.Hopfield.StorePattern(p.Sys, actualTarget)
		return loss
	}
	return 0.0
}
