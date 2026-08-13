package pipeline

import (
	"protaxon/pkg/agent"
	"protaxon/pkg/conflict"
	"protaxon/pkg/core"
	"protaxon/pkg/goals"
	"protaxon/pkg/graph"
	"protaxon/pkg/homeostasis"
	"protaxon/pkg/memory"
	"protaxon/pkg/offline"
	"context"
	"fmt"
	"math"
	"sort"
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
	// ForecastError is the REAL cross-cycle prediction error (Step 1e): how
	// far the previous cycle's forecast of this cycle's stateVector was
	// from what actually happened -- a genuine expectation-vs-reality
	// mismatch across time. Since the forward-model fix, PredictionError
	// (mlpLoss) is the training loss of that same forecast against the
	// realized transition and numerically coincides with ForecastError
	// while the forecaster's Hopfield is empty; the two are kept as separate
	// fields because they can diverge once the Hopfield blend kicks in.
	// Always 0.0 on a cold Engine's first cycle, since nothing was forecast
	// yet to compare against.
	ForecastError       float64
	DriveError          float64
	MLPTrainLoss        float64
	SleepTriggered      bool
	AbstractionsCreated int
	NodesAbsorbed       int
	MaxCohesionObserved float64
	SpecialistCluster   int
	PredictorPoolSize   int
	GoalStackDepth      int
	CurrentGoal         string
	ConflictDetected    bool
	ConflictCandidates  int
	ConflictWinner      string
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

	// Goals is the Stage 4 piece 2 hierarchical goal stack (CONCEPT.md
	// Section 8). RunPredictiveCycle auto-bootstraps a root External goal
	// from the goal string the first time the stack is empty, then tags
	// whichever goal is on top with every graph ClusterID active that
	// cycle -- callers who want real hierarchy (sub-goals for a deficit,
	// meta-goals, etc.) push/pop it explicitly via e.Goals.
	Goals *goals.Stack

	// RecentConflicts is the Stage 4 piece 3 conflict memory (CONCEPT.md
	// Section 16): whenever more than one graph cluster is simultaneously
	// active, every competing specialist's prediction is recorded here --
	// winner AND losers -- instead of silently discarding all but the
	// primary cluster's. Bounded to the most recent maxRecentConflicts.
	RecentConflicts []*conflict.Record

	StepCounter int

	// PendingPrediction is the previous cycle's specialist's forecast of
	// THIS cycle's stateVector -- the cross-cycle "expectation" half of a
	// real predictive-coding comparison (see RunPredictiveCycle's Step 1e).
	// nil before the first cycle ever runs, since there's nothing to have
	// predicted yet.
	PendingPrediction []float64

	// PrevStateVector and PrevPredictor are the OTHER half of the forward
	// model: the input the previous cycle fed its predictor, and the
	// specialist that made that forecast. Next cycle trains THAT specialist
	// on (PrevStateVector -> the observation that actually arrived), so the
	// network learns genuine (state_t -> state_{t+1}) transitions instead of
	// autoencoding a single frame. Both nil before the first cycle. See
	// RunPredictiveCycle Step 5 for why this one-cycle lag is what makes
	// PendingPrediction an actual forecast rather than a reconstruction.
	PrevStateVector []float64
	PrevPredictor   *agent.PredictorAgent
}

func NewEngine() *Engine {
	sys := core.NewSystem()
	g := graph.NewGraph()
	state := homeostasis.NewState()
	// Tied to ObservationVectorDim, not redeclared as its own magic 20:
	// vectorMSE(e.PendingPrediction, stateVector) in RunPredictiveCycle
	// requires every specialist's Predictor MLP output length to exactly
	// match ObservationVector's output length, or it panics on mismatch.
	// Two independently-declared 20s would silently drift apart the moment
	// either one changed.
	dim := ObservationVectorDim
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

		Goals: goals.NewStack(),
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
//
// bestActivation starts at math.Inf(-1), not a hardcoded -1.0: the same bug
// class already found and fixed once in winningBlobLabel (pkg/environment/
// bridge/bridge.go) -- real node activations were observed live to go well
// below -1.0 (down to roughly -4.12) before edge weights got a floor, and
// nothing bounds Node.Activation itself even now, only Hebbian edge
// weights. A hardcoded -1.0 sentinel here would silently misroute every
// cycle where every active node's Activation is more negative than that,
// always falling through to cluster 0 regardless of which cluster actually
// won.
func primaryCluster(g *graph.Graph, activeNodeIDs []int) int {
	best := 0
	bestActivation := math.Inf(-1)
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

// distinctClusters returns the sorted, deduplicated list of ClusterIDs
// represented among activeNodeIDs. Since RouteCompetingClusters (Stage 2)
// already guarantees at most one winner per cluster, this is normally just
// "which clusters won at all" -- but it's written generically rather than
// assuming that invariant holds forever.
func distinctClusters(g *graph.Graph, activeNodeIDs []int) []int {
	seen := make(map[int]bool)
	clusters := make([]int, 0, len(activeNodeIDs))
	for _, id := range activeNodeIDs {
		node, exists := g.Nodes[id]
		if !exists || seen[node.ClusterID] {
			continue
		}
		seen[node.ClusterID] = true
		clusters = append(clusters, node.ClusterID)
	}
	sort.Ints(clusters)
	return clusters
}

// vectorMSE returns the mean squared error between a and b -- the real
// forecast-vs-actual mismatch RunPredictiveCycle's Step 1e compares
// e.PendingPrediction against the current cycle's actual ObservationVector
// with. Panics on length mismatch rather than silently truncating or
// zero-padding: a and b are always both ObservationVectorDim-length by
// construction (ObservationVector's fixed output size, and the Predictor
// MLP's OutDim==InDim==dim guarantee -- see pkg/agent), so a mismatch here
// would mean something upstream is already broken, not a normal input to
// tolerate quietly.
func vectorMSE(a, b []float64) float64 {
	if len(a) != len(b) {
		panic(fmt.Sprintf("vectorMSE: length mismatch %d vs %d", len(a), len(b)))
	}
	sum := 0.0
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return sum / float64(len(a))
}

// maxRecentConflicts bounds Engine.RecentConflicts so it doesn't grow
// unbounded over a long-running engine.
const maxRecentConflicts = 20

func (e *Engine) RunPredictiveCycle(ctx context.Context, observation, goal string, actualSuccess bool) (*CycleResult, error) {
	e.Sys.Mode = core.Online
	e.StepCounter++

	// 0. Hierarchical Goal Stack (Stage 4 piece 2): auto-bootstrap a root
	// External goal from the flat `goal` string the first time the stack is
	// empty, so every caller gets a real (if shallow) stack for free.
	// Callers that want genuine hierarchy push/pop sub-goals themselves via
	// e.Goals -- this cycle's activity then gets recorded against whatever
	// goal is actually on top, not always the root.
	if e.Goals.Top() == nil {
		e.Goals.Push(goal, goals.TypeExternal, 1.0, e.StepCounter)
	}

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

	// 1b-ii. Tag the current goal (whatever's on top of the stack) with
	// every distinct ClusterID actually active this cycle -- its real,
	// accumulating "multi-layered subgraph" footprint. The same distinct-
	// cluster list also drives Step 4's conflict resolution below.
	activeClusters := distinctClusters(e.Graph, activeNodeIDs)
	for _, cID := range activeClusters {
		e.Goals.RecordScope(cID)
	}

	// 1c. Structural plasticity: concepts that won attention together this
	// cycle get wired together if they weren't already, giving Hebbian
	// plasticity (Step 6) something to act on next time they co-occur.
	// Gate-retry step 1/3 (2026-08-12): raised from 0.15 -- the first organic
	// growth run hit AbstractionsCreated=0 with peak cohesion far below the
	// 0.75 threshold, and 0.15 gave Hebbian ~5x further to climb than 0.4
	// does. Still meaningfully below threshold, so this still tests real
	// reinforcement, not a trivially pre-cleared bar.
	e.Graph.FormCoActivationEdges(e.Sys, activeNodeIDs, 0.4)

	// 1d. Content-based observation embedding (replaces a pseudo-random
	// placeholder derived purely from e.StepCounter's parity, which carried
	// zero information about what was actually observed -- see
	// ObservationVector's doc comment in observation_vector.go for why that
	// made real forecasting impossible: the same observation never produced
	// the same vector twice, and different observations were
	// indistinguishable to the Predictor MLP by construction).
	stateVector := ObservationVector(observation)

	// 1e. Real, cross-cycle prediction error (Active Inference / predictive
	// coding, CONCEPT.md's north-star framing): e.PendingPrediction is
	// whatever the PREVIOUS cycle's specialist forecast THIS cycle's
	// stateVector would look like (set below, after this cycle's own
	// predictor runs). Comparing it against the actual stateVector just
	// computed is a genuine expectation-vs-reality mismatch across time.
	// Step 5 then trains the specialist that made that forecast on the
	// transition that actually happened, so PendingPrediction is a real
	// forward-model forecast rather than an autoencoded copy of the previous
	// frame (which is what the first cut of this did -- see Step 5's comment
	// for the autoencoder pitfall that collapsed forecastError into a mere
	// "did the observation change" proxy). forecastError is 0 on the very
	// first cycle (nothing was predicted yet to compare against).
	var forecastError float64
	hadPendingPrediction := e.PendingPrediction != nil
	if hadPendingPrediction {
		forecastError = vectorMSE(e.PendingPrediction, stateVector)
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

	// Cache this cycle's forecast for next cycle's forecastError comparison
	// (Step 1e above). Copied defensively rather than aliasing predResp's
	// slice directly, since predictor.Process's Hopfield-blended output is
	// an internal buffer this function doesn't own the lifetime of.
	pending := make([]float64, len(predResp.ValueVector))
	copy(pending, predResp.ValueVector)
	e.PendingPrediction = pending

	// 4b. Conflict resolution (Stage 4 piece 3, CONCEPT.md Section 16): when
	// more than one cluster is simultaneously active, every competing
	// specialist gets to predict, and the resulting conflict -- winner AND
	// losers -- is recorded rather than silently discarding everything but
	// the primary cluster's opinion. Process() is a pure forward pass (no
	// training, no trust-score mutation), so querying extra specialists here
	// has zero effect on anything Stage 1-3 already verified.
	var conflictRecord *conflict.Record
	if len(activeClusters) > 1 {
		candidates := make([]conflict.Candidate, 0, len(activeClusters))
		for _, cID := range activeClusters {
			specialist := e.specialistFor(cID)
			resp, cErr := specialist.Process(ctx, payload)
			if cErr != nil {
				continue
			}
			candidates = append(candidates, conflict.Candidate{
				Source:     specialist.ID(),
				ClusterID:  cID,
				Value:      resp.ValueVector,
				TrustScore: specialist.TrustScore(),
				Confidence: resp.Confidence,
			})
		}
		conflictRecord = conflict.Resolve(e.StepCounter, candidates)
		if conflictRecord != nil {
			e.RecentConflicts = append(e.RecentConflicts, conflictRecord)
			if len(e.RecentConflicts) > maxRecentConflicts {
				e.RecentConflicts = e.RecentConflicts[len(e.RecentConflicts)-maxRecentConflicts:]
			}
		}
	}

	// 5. Forward-model training (the fix that makes PendingPrediction a real
	// forecast). Train the specialist that made LAST cycle's forecast on the
	// transition it actually observed: its input then (e.PrevStateVector)
	// mapped to the observation that actually arrived now (stateVector).
	// That teaches the network genuine (state_t -> state_{t+1}) transitions.
	//
	// The old version trained THIS cycle's predictor on
	// (stateVector -> stateVector), i.e. to autoencode the current frame,
	// inverting the target on failure. That had two problems: (a) a network
	// trained to reproduce its input can't forecast the next input -- its
	// "prediction" was just a lossy copy of the current frame, so the
	// resulting forecastError collapsed to "how much did the observation
	// change," the very grid-changed proxy this work set out to move past;
	// and (b) the failure-inversion made the same input demand two different
	// outputs depending on actualSuccess (not part of the input), which is
	// the unlearnable-noise incident the Step 6 comment records. Both are
	// gone: the target is now simply the realized next observation, and
	// "did the last action succeed" stays a separate channel (structuralReward
	// below, Curiosity in bridge.go) rather than being folded into what the
	// world is predicted to look like.
	mlpLoss := 0.0
	if e.PrevPredictor != nil {
		mlpLoss = e.PrevPredictor.TrainStep(e.PrevStateVector, stateVector)
	}
	predErr := mlpLoss

	// Snapshot this cycle's input and specialist so next cycle can train them
	// on next cycle's realized observation -- mirrors the PendingPrediction
	// snapshot above, which froze this cycle's forecast for next cycle's
	// forecastError. Copied for the same lifetime-safety reason.
	e.PrevStateVector = append([]float64(nil), stateVector...)
	e.PrevPredictor = predictor

	// Actor/Associator keep their pre-existing same-cycle training (out of
	// scope for the forward-model change, which is specifically about the
	// Predictor's temporal forecast).
	e.Actor.TrainStep(stateVector, actResp.ValueVector)
	e.Associator.TrainStep(stateVector)

	// 6. Update Eligibility Traces & Hebbian Plasticity (temporal credit assignment)
	//
	// Structural graph plasticity is driven by actualSuccess (+1/-1), NOT by
	// any predictor-derived reward -- and deliberately kept that way even now
	// that the predictor is a real forward model. The original reason this
	// separation was introduced (2026-08-12, TestOrganicGraphGrowthAndCompression):
	// the old same-cycle target (stateVector, inverted on failure) wasn't
	// learnable from stateVector alone -- actualSuccess, an unobserved
	// variable, decided the target -- so the same input demanded two
	// different outputs and 1-predErr swung negative in 10/20 cycles (mean
	// +0.0644, min -2.0262), noise that leaked into Hebbian/eligibility and
	// capped cohesion growth. That specific pathology is now fixed at the
	// source (the forward-model target is the realized next observation, no
	// inversion). The separation still stands on principle, though: "was this
	// experience good or bad" (what grows structure) is a different question
	// from "how surprising was the world" (predErr/forecastError, which now
	// modulates Dopamine/plasticity in Step 7b) -- conflating them is exactly
	// how this burned once, so forecast error stays off structural learning
	// until there's live evidence it behaves.
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

	// 7b. Forecast surprise modulates Dopamine directly (Homeostasis's own
	// doc comment already labels it "Reward / Prediction Error" --
	// hormones.go:11 -- but nothing fed it anything resembling a real
	// prediction error until now). Mutated directly from here rather than
	// folded into UpdateHormones's own signature, the same way bridge.go
	// already mutates Homeostasis.Curiosity directly from outside the
	// package -- an established pattern in this codebase, not a new one.
	//
	// Direction: surprise RAISES Dopamine, accurate prediction lowers it.
	// Dopamine here is the global plasticity multiplier (hormones.go:11), and
	// the predictive-coding principle is that a prediction error is precisely
	// the signal worth learning from -- when the world violates the forward
	// model, turn plasticity UP; when it's already well predicted, there's
	// nothing new to learn, so turn it down. The first cut of this had the
	// sign inverted (rewarding low error), which rewarded stasis: it turned
	// plasticity up exactly when nothing changed and down on novelty -- the
	// "dark room" failure mode, and a quiet way to make the agent prefer a
	// dead, predictable corner. Note this is deliberately NOT "reward =
	// accurate prediction": accurately predicting a bad outcome is not good
	// news, so prediction accuracy drives learning-rate here, not reward --
	// reward/goal signal stays on actualSuccess (structuralReward, Curiosity).
	//
	// Still NOT wired into Hebbian/structuralReward or Curiosity -- see Step
	// 6's comment for why forecast error stays off structural learning until
	// there's live evidence it behaves; Dopamine is the lowest-stakes place
	// to let a not-yet-live-tested signal prove itself first.
	if hadPendingPrediction {
		// surprise saturates smoothly into [0,1) as forecastError grows,
		// instead of scaling Dopamine by a raw unbounded MSE value (whose
		// magnitude depends on how many tokens an observation happens to
		// have -- see ObservationVector's doc comment).
		surprise := forecastError / (forecastError + 1.0)
		dopamineDelta := (surprise - 0.5) * 0.2
		e.Homeostasis.Dopamine = math.Min(2.0, math.Max(0.1, e.Homeostasis.Dopamine+dopamineDelta))
	}

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
		ForecastError:    forecastError,
		DriveError:       newDriveErr,
		MLPTrainLoss:     mlpLoss,
		SleepTriggered:   sleepTriggered,

		AbstractionsCreated: compression.AbstractionsCreated,
		NodesAbsorbed:       compression.NodesAbsorbed,
		MaxCohesionObserved: compression.MaxCohesionObserved,
		SpecialistCluster:   specialistCluster,
		PredictorPoolSize:   len(e.Predictors),
		GoalStackDepth:      e.Goals.Depth(),
		CurrentGoal:         currentGoalDescription(e.Goals),
		ConflictDetected:    conflictRecord != nil,
		ConflictCandidates:  conflictCandidateCount(conflictRecord),
		ConflictWinner:      conflictWinnerSource(conflictRecord),
	}, nil
}

func conflictCandidateCount(r *conflict.Record) int {
	if r == nil {
		return 0
	}
	return len(r.Candidates)
}

func conflictWinnerSource(r *conflict.Record) string {
	if r == nil {
		return ""
	}
	return r.Winner().Source
}

// currentGoalDescription returns the description of whichever goal is on
// top of the stack, or "" if the stack is somehow empty (shouldn't happen
// after RunPredictiveCycle's Step 0 bootstrap, but stay panic-safe).
func currentGoalDescription(s *goals.Stack) string {
	if top := s.Top(); top != nil {
		return top.Description
	}
	return ""
}
