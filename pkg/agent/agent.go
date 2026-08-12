package agent

import (
	"context"
)

type Role string

const (
	RolePredictor  Role = "Predictor"
	RoleActor      Role = "Actor"
	RoleAssociator Role = "Associator"
)

type ContextPayload struct {
	Observation  string    `json:"observation"`
	StateVector  []float64 `json:"state_vector"`
	ActiveNodes  []int     `json:"active_nodes"`
	RecallNotes  []string  `json:"recall_notes"`
	Goal         string    `json:"goal"`
	ProposedPlan string    `json:"proposed_plan,omitempty"`
}

type AgentResponse struct {
	AgentID     string    `json:"agent_id"`
	Role        Role      `json:"role"`
	Content     string    `json:"content"`
	ValueVector []float64 `json:"value_vector"`
	Confidence  float64   `json:"confidence"`
	TrustScore  float64   `json:"trust_score"`
}

type Agent interface {
	ID() string
	Role() Role
	TrustScore() float64
	SetTrustScore(score float64)
	Process(ctx context.Context, payload ContextPayload) (AgentResponse, error)
}

type BaseAgent struct {
	id         string
	role       Role
	trustScore float64
}

func NewBaseAgent(id string, role Role) BaseAgent {
	return BaseAgent{
		id:         id,
		role:       role,
		trustScore: 1.0,
	}
}

func (b *BaseAgent) ID() string {
	return b.id
}

func (b *BaseAgent) Role() Role {
	return b.role
}

func (b *BaseAgent) TrustScore() float64 {
	return b.trustScore
}

func (b *BaseAgent) SetTrustScore(score float64) {
	b.trustScore = score
}
