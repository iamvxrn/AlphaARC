package agent

import (
	"alphaarc/pkg/mlp"
	"context"
	"fmt"
)

type ActorAgent struct {
	BaseAgent
	MLP *mlp.MLP
}

func NewActorAgent(id string, dim int) *ActorAgent {
	// Tiny MLP: dim inputs -> dim*2 hidden -> dim action vector outputs
	tinyMLP := mlp.NewMLP(dim, dim*2, dim, 0.05, 101)
	return &ActorAgent{
		BaseAgent: NewBaseAgent(id, RoleActor),
		MLP:       tinyMLP,
	}
}

func (a *ActorAgent) Process(ctx context.Context, payload ContextPayload) (AgentResponse, error) {
	actionVector := make([]float64, len(payload.StateVector))

	if len(payload.StateVector) > 0 {
		_, actionVector = a.MLP.Forward(payload.StateVector)
	}

	content := fmt.Sprintf("Micro-MLP Actor: Formulated optimal action plan vector (dim=%d)", len(actionVector))

	return AgentResponse{
		AgentID:     a.ID(),
		Role:        a.Role(),
		Content:     content,
		ValueVector: actionVector,
		Confidence:  0.90,
		TrustScore:  a.TrustScore(),
	}, nil
}

func (a *ActorAgent) TrainStep(input, targetAction []float64) float64 {
	if a.MLP != nil && len(input) > 0 && len(targetAction) > 0 {
		return a.MLP.Train(input, targetAction)
	}
	return 0.0
}
