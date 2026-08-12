package agent

import (
	"protaxon/pkg/core"
	"context"
	"testing"
)

func TestMicroMLPAgentRoles(t *testing.T) {
	ctx := context.Background()
	sys := core.NewSystem()
	sys.Mode = core.Online
	dim := 20

	payload := ContextPayload{
		Observation: "Sensory stimulus",
		StateVector: []float64{1.0, -1.0, 1.0, -1.0, 1.0, -1.0, 1.0, -1.0, 1.0, -1.0, 1.0, -1.0, 1.0, -1.0, 1.0, -1.0, 1.0, -1.0, 1.0, -1.0},
		ActiveNodes: []int{1, 2, 3},
		Goal:        "Achieve homeostatic balance",
	}

	predictor := NewPredictorAgent("pred-1", sys, dim)
	actor := NewActorAgent("act-1", dim)
	associator := NewAssociatorAgent("assoc-1", dim)

	// Train predictor step
	loss := predictor.TrainStep(payload.StateVector, payload.StateVector)
	t.Logf("Initial Predictor MLP Online Train Loss: %.6f", loss)

	predResp, err := predictor.Process(ctx, payload)
	if err != nil {
		t.Fatalf("Predictor failed: %v", err)
	}
	if predResp.Role != RolePredictor {
		t.Fatalf("Expected RolePredictor, got %v", predResp.Role)
	}

	payload.RecallNotes = []string{predResp.Content}
	actResp, err := actor.Process(ctx, payload)
	if err != nil {
		t.Fatalf("Actor failed: %v", err)
	}

	assocResp, err := associator.Process(ctx, payload)
	if err != nil {
		t.Fatalf("Associator failed: %v", err)
	}

	t.Logf("Micro-MLP Agents Test PASS: Predictor=%s, Actor=%s, Associator=%s",
		predResp.AgentID, actResp.AgentID, assocResp.AgentID)
}
