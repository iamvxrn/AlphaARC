package agent

import (
	"alphaarc/pkg/mlp"
	"context"
	"fmt"
)

type AssociatorAgent struct {
	BaseAgent
	MLP *mlp.MLP
}

func NewAssociatorAgent(id string, dim int) *AssociatorAgent {
	// Tiny Autoencoder MLP: dim inputs -> dim/2 bottleneck hidden -> dim outputs
	tinyMLP := mlp.NewMLP(dim, dim/2, dim, 0.05, 202)
	return &AssociatorAgent{
		BaseAgent: NewBaseAgent(id, RoleAssociator),
		MLP:       tinyMLP,
	}
}

func (ass *AssociatorAgent) Process(ctx context.Context, payload ContextPayload) (AgentResponse, error) {
	reconstructed := make([]float64, len(payload.StateVector))

	if len(payload.StateVector) > 0 {
		_, reconstructed = ass.MLP.Forward(payload.StateVector)
	}

	content := fmt.Sprintf("Micro-MLP Associator: Extracted bottleneck features across %d nodes", len(payload.ActiveNodes))

	return AgentResponse{
		AgentID:     ass.ID(),
		Role:        ass.Role(),
		Content:     content,
		ValueVector: reconstructed,
		Confidence:  0.88,
		TrustScore:  ass.TrustScore(),
	}, nil
}

func (ass *AssociatorAgent) TrainStep(input []float64) float64 {
	if ass.MLP != nil && len(input) > 0 {
		return ass.MLP.Train(input, input) // Autoencoder reconstruction training
	}
	return 0.0
}
