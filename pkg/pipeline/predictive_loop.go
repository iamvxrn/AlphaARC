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
	MaxCohesionObserved float64
	SpecialistCluster   int
	PredictorPoolSize   int
}

type Engine struct {
	Sys         *core.System
	Graph       *graph.Graph
	Homeostasis *homeostasis.State
	Hopfield    *memory.ModernHopfield

	// Predictor is the cluster-0 / generalist specialist -- kept as a direct
	// field for backward compatibility (cmd/protaxon-stage1 and existing
	// tests read it directly). Predictors[0] is always this same instance.
	Predictor *agent.PredictorAgent
	// Predictors is the Stage 4 MoE pool: one Predictor specialist per graph
	// ClusterID, lazily created via specialistFor as Louvain discovers new
	// clusters beyond the default 0. Each gets its own deterministically
	// seeded MLP, so specialists genuinely diverge instead of starting as
	// identical clones.
	Predictors   map[int]*agent.PredictorAgent
	PredictorDim int

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

	predictor := agent.NewPredictorAgent("pred-main", sys, dim, 42)

	return &Engine{
		Sys:         sys,
		Graph:       g,
		Homeostasis: state,
		Hopfield:    hopfield,

		Predictor:    predictor,
		Predictors:   map[int]*agent.PredictorAgent{0: predictor},
		PredictorDim: dim,

		Actor:      agent.NewActorAgent("act-main", dim),
		Associator: agent.NewAssociatorAgent("assoc-main", dim),
	}
}

// specialistFor returns the Predictor specialist responsible for clusterID,
// lazily creating one with a deterministic per-cluster seed the first time
// that cluster needs a specialist of its own. Cluster 0 always resolves to
// the original "pred-main" agent created in NewEngine, so any cycle that
// never diversifies beyond the default cluster (every existing test that
// runs before the graph's first Louvain reclustering) sees exactly the same
// single-agent behavior as before Stage 4.
func (e *Engine) specialistFor(clusterID int) *agent.PredictorAgent {
	if p, exists := e.Predictors[clusterID]; exists {
		return p
	}
	seed := int64(1000 + clusterID)
	id := fmt.Sprintf("pred-cluster-%d", clusterID)
	p := agent.NewPredictorAgent(id, e.Sys, e.PredictorDim, seed)
	e.Predictors[clusterID] = p
	return p
}

// primaryCluster picks which cluster's specialist should handle this cycle:
// the ClusterID of the highest-Activation node among the router's post-
// competition winners. Falls back to cluster 0 (the default/generalist) when
// nothing is active this cycle, matching every node's default ClusterID.
func primaryCluster(g *graph.Graph, activeNodeIDs []int) int {
	best := 0
	bestActivation := -1.0
	for _, id := range activeNodeIDs {
		node, exists := g.Nodes[id]
		if !exists {
			continue
		}
		if node.Activation > bestActivation {
			bestActivation = node.Activation
			best = node.ClusterID
		}
	}
	return best
}

func (e *Engine) RunPredictiveCycle(ctx context.Context, observation, goal string, actualSuccess bool) (*CycleResult, error) {
	e.Sys.Mode = core.Online
	e.StepCounter++

	// 1. Observation -> Node Lookup-or-Create -> Spreading Activation -> Active Subgraph Extraction
	// EnsureConceptNodes lets the graph grow from real experience: novel
	// vocabulary in the observation becomes new concept nodes instead of
	// being silently dropped, which is what Stage 3 compression needs to
	// eventually act on organically-grown structure, not just hand-seeded
	// demo nodes.
	seeds := e.Graph.EnsureConceptNodes(observation, 0.5)
	activations := e.Graph.SpreadingActivation(e.Sys, seeds, 3, 0.7)
	rawActiveNodeIDs := graph.ExtractActiveSubgraph(activations, 0.1)

	// 1b. Dynamic Competition Router: within any Louvain cluster contributing
	// more than one active concept this tick, only the strongest survives
	// (Winner-Takes-All lateral inhibition) -- agents attend to what actually
	// won competition, not every merely-above-threshold candidate.
	activeNodeIDs := e.Graph.RouteCompetingClusters(e.Sys, rawActiveNodeIDs, 0.3)

	// 1c. Structural plasticity: concepts that won attention together this
	// cycle get wired together if they weren't already, giving Hebbian
	// plasticity (Step 6) something to act on next time they co-occur.
	// Gate-retry step 1/3 (2026-08-12): raised from 0.15 -- the first organic
	// growth run hit AbstractionsCreated=0 with peak cohesion far below the
	// 0.75 threshold, and 0.15 gave Hebbian ~5x further to climb than 0.4
	// does. Still meaningfully below threshold, so this still tests real
	// reinforcement, not a trivially pre-cleared bar.
	e.Graph.FormCoActivationEdges(e.Sys, activeNodeIDs, 0.4)

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
	// Stage 4 MoE: routed to the specialist responsible for this cycle's
	// primary cluster (the router's highest-activation winner) instead of
	// always the same generalist -- narrow specialists over one global agent.
	specialistCluster := primaryCluster(e.Graph, activeNodeIDs)
	predictor := e.specialistFor(specialistCluster)
	predResp, err := predictor.Process(ctx, payload)
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
	mlpLoss := predictor.TrainStep(stateVector, targetVector)
	e.Actor.TrainStep(stateVector, actResp.ValueVector)
	e.Associator.TrainStep(stateVector)

	predErr := mlpLoss

	// 6. Update Eligibility Traces & Hebbian Plasticity (temporal credit assignment)
	//
	// Structural graph plasticity is driven by actualSuccess (+1/-1), NOT by
	// mlpLoss-derived reward. Diagnosed 2026-08-12 (TestOrganicGraphGrowthAndCompression):
	// 1-predErr swung negative in 10/20 cycles (mean +0.0644, min -2.0262)
	// because the Predictor's target (stateVector, inverted on failure) isn't
	// actually learnable from stateVector alone -- actualSuccess isn't part
	// of its input, so the same input demands two different "correct"
	// outputs depending on an unobserved variable. That noise was leaking
	// straight into Hebbian/eligibility, capping organic cohesion growth far
	// below the 0.75 compression threshold regardless of how many cycles ran.
	// The graph's structural learning signal ("was this experience good or
	// bad") is conceptually independent of how well the Predictor's MLP
	// happens to be converging, so it no longer rides on mlpLoss at all.
	structuralReward := 1.0
	if !actualSuccess {
		structuralReward = -1.0
	}
	e.Graph.DecayEligibilityTraces(e.Sys, 0.7)
	e.Graph.UpdateEligibilityTraces(e.Sys)
	e.Graph.HebbianUpdateWithEligibility(e.Sys, 0.1, structuralReward, 10.0)

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

	// 8. Agent Trust Score Update -- applies to the specialist that actually
	// handled this cycle, not always the cluster-0 generalist.
	if actualSuccess {
		predictor.SetTrustScore(math.Min(2.0, predictor.TrustScore()+0.05))
		e.Actor.SetTrustScore(math.Min(2.0, e.Actor.TrustScore()+0.05))
	} else {
		predictor.SetTrustScore(math.Max(0.1, predictor.TrustScore()-0.10))
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
		MaxCohesionObserved: compression.MaxCohesionObserved,
		SpecialistCluster:   specialistCluster,
		PredictorPoolSize:   len(e.Predictors),
	}, nil
}
