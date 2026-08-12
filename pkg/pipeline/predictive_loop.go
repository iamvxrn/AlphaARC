package pipeline

import (
	"protaxon/pkg/agent"
	"protaxon/pkg/core"
	"protaxon/pkg/graph"
	"protaxon/pkg/homeostasis"
	"protaxon/pkg/memory"
	"protaxon/pkg/offline"
	"context"
	"fmt"
	"math"
)

type CycleResult struct {
	StepIndex           int
	Observation         string
	Goal                string
	ActiveNodeIDs       []int
	HopfieldRecalled    bool
	Prediction          agent.AgentResponse
	ActionPlan          agent.AgentResponse
	AssociatorRecall    agent.AgentResponse
	ActualOutcome       string
	PredictionError     float64
	DriveError          float64
	MLPTrainLoss        float64
	SleepTriggered      bool
	AbstractionsCreated int
	NodesAbsorbed       int
}

type Engine struct {
	Sys         *core.System
	Graph       *graph.Graph
	Homeostasis *homeostasis.State
	Hopfield    *memory.ModernHopfield

	Predictor  *agent.PredictorAgent
	Actor      *agent.ActorAgent
	Associator *agent.AssociatorAgent

	StepCounter int
}

func NewEngine() *Engine {
	sys := core.NewSystem()
	g := graph.NewGraph()
	state := homeostasis.NewState()
	dim := 20
	hopfield := memory.NewModernHopfield(dim, 10.0)

	// Initial graph nodes (Concepts / State Neurons)
	g.AddNode(graph.NewNode(1, 0.8, 0))
	g.AddNode(graph.NewNode(2, 0.9, 0))
	g.AddNode(graph.NewNode(3, 0.4, 0))
	g.AddEdge(1, 2, 0.7, false)
	g.AddEdge(2, 3, 0.6, false)

	// Observation -> Node Lookup keyword index (Section 6: Active Subgraph)
	g.AddLabel("stimulus", 1)
	g.AddLabel("sensory", 1)
	g.AddLabel("anomaly", 2)
	g.AddLabel("error", 2)
	g.AddLabel("failure", 2)
	g.AddLabel("graph", 3)
	g.AddLabel("recall", 3)
	g.AddLabel("query", 3)

	return &Engine{
		Sys:         sys,
		Graph:       g,
		Homeostasis: state,
		Hopfield:    hopfield,

		Predictor:  agent.NewPredictorAgent("pred-main", sys, dim),
		Actor:      agent.NewActorAgent("act-main", dim),
		Associator: agent.NewAssociatorAgent("assoc-main", dim),
	}
}

func (e *Engine) RunPredictiveCycle(ctx context.Context, observation, goal string, actualSuccess bool) (*CycleResult, error) {
	e.Sys.Mode = core.Online
	e.StepCounter++

	// 1. Observation -> Node Lookup -> Spreading Activation -> Active Subgraph Extraction
	seeds := e.Graph.LookupSeeds(observation)
	activations := e.Graph.SpreadingActivation(e.Sys, seeds, 3, 0.7)
	rawActiveNodeIDs := graph.ExtractActiveSubgraph(activations, 0.1)

	// 1b. Dynamic Competition Router: within any Louvain cluster contributing
	// more than one active concept this tick, only the strongest survives
	// (Winner-Takes-All lateral inhibition) -- agents attend to what actually
	// won competition, not every merely-above-threshold candidate.
	activeNodeIDs := e.Graph.RouteCompetingClusters(e.Sys, rawActiveNodeIDs, 0.3)

	stateVector := make([]float64, 20)
	for i := 0; i < 20; i++ {
		if (i+e.StepCounter)%2 == 0 {
			stateVector[i] = 1.0
		} else {
			stateVector[i] = -1.0
		}
	}

	payload := agent.ContextPayload{
		Observation: observation,
		StateVector: stateVector,
		ActiveNodes: activeNodeIDs,
		Goal:        goal,
	}

	// 2. Associator Agent Call (Micro-MLP Autoencoder Feature Extraction)
	assocResp, err := e.Associator.Process(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("associator error: %w", err)
	}
	payload.RecallNotes = []string{assocResp.Content}

	// 3. Actor Agent Call (Micro-MLP Action Vector Formulation)
	actResp, err := e.Actor.Process(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("actor error: %w", err)
	}
	payload.ProposedPlan = actResp.Content

	// 4. Predictor Agent Call (Micro-MLP Neural Forecast + Hopfield Attention)
	predResp, err := e.Predictor.Process(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("predictor error: %w", err)
	}

	// 5. Compute Target Outcome Vector & Prediction Error
	targetVector := make([]float64, len(stateVector))
	copy(targetVector, stateVector)
	if !actualSuccess {
		for i := range targetVector {
			targetVector[i] = -targetVector[i] // Invert state vector on failure shock
		}
	}

	// Train Predictor Micro-MLP via Online Backpropagation!
	mlpLoss := e.Predictor.TrainStep(stateVector, targetVector)
	e.Actor.TrainStep(stateVector, actResp.ValueVector)
	e.Associator.TrainStep(stateVector)

	predErr := mlpLoss

	// 6. Update Eligibility Traces & Hebbian Plasticity (temporal credit assignment)
	reward := 1.0 - predErr
	e.Graph.DecayEligibilityTraces(e.Sys, 0.7)
	e.Graph.UpdateEligibilityTraces(e.Sys)
	e.Graph.HebbianUpdateWithEligibility(e.Sys, 0.1, reward, 10.0)

	// 7. Update Homeostasis & Hormones
	prevDriveErr := e.Homeostasis.DriveError()
	e.Homeostasis.Energy += (1.0 - e.Homeostasis.Energy) * 0.1
	if actualSuccess {
		e.Homeostasis.Stress -= e.Homeostasis.Stress * 0.15
	} else {
		e.Homeostasis.Stress += (1.0 - e.Homeostasis.Stress) * 0.30
	}
	newDriveErr := e.Homeostasis.DriveError()
	e.Homeostasis.UpdateHormones(prevDriveErr, newDriveErr)

	// 8. Agent Trust Score Update
	if actualSuccess {
		e.Predictor.SetTrustScore(math.Min(2.0, e.Predictor.TrustScore()+0.05))
		e.Actor.SetTrustScore(math.Min(2.0, e.Actor.TrustScore()+0.05))
	} else {
		e.Predictor.SetTrustScore(math.Max(0.1, e.Predictor.TrustScore()-0.10))
		e.Actor.SetTrustScore(math.Max(0.1, e.Actor.TrustScore()-0.10))
	}

	// 9. Subconscious Sleep Engine Trigger (every 5 steps or high stress)
	sleepTriggered := false
	compression := offline.CompressionStats{}
	if e.StepCounter%5 == 0 || e.Homeostasis.Stress > 0.8 {
		e.Sys.Mode = core.Offline
		offline.SubconsciousSleep(e.Sys, e.Graph, 0.05)
		// Stage 3: collapse clusters cohesive enough (>=0.75 mean intra-cluster
		// weight) into a single abstraction node. Conservative on purpose: the
		// engine's small seed graph (chain edges at 0.6-0.7) stays below this
		// bar and is untouched until real experience grows denser clusters.
		compression = offline.CompressGraphAbstractions(e.Sys, e.Graph, 0.75)
		sleepTriggered = true
		e.Sys.Mode = core.Online
	}

	actualOutcome := "Task Executed Successfully"
	if !actualSuccess {
		actualOutcome = "Task Execution Failed / Error Encountered"
	}

	return &CycleResult{
		StepIndex:        e.StepCounter,
		Observation:      observation,
		Goal:             goal,
		ActiveNodeIDs:    activeNodeIDs,
		HopfieldRecalled: true,
		Prediction:       predResp,
		ActionPlan:       actResp,
		AssociatorRecall: assocResp,
		ActualOutcome:    actualOutcome,
		PredictionError:  predErr,
		DriveError:       newDriveErr,
		MLPTrainLoss:     mlpLoss,
		SleepTriggered:   sleepTriggered,

		AbstractionsCreated: compression.AbstractionsCreated,
		NodesAbsorbed:       compression.NodesAbsorbed,
	}, nil
}
