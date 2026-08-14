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
	"strings"
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
	// ForecastError is the REAL cross-cycle prediction error (Step 0b): how
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
	// AcuteSurprise is true when ForecastError spiked above its recent
	// running norm (past warmup) -- the trigger that narrowed this cycle's
	// attention (Step 1a) and bumped Cortisol (Step 7c). SeededConcepts is
	// how many concept nodes were actually seeded this cycle: equal to the
	// frame's blob count normally, but SMALLER when an acute surprise
	// narrowed seeding to the locus of change, so a live log can see the
	// narrowing happen instead of inferring it.
	AcuteSurprise       bool
	// Predictable is the mirror of AcuteSurprise: the forward model predicts
	// this cycle notably below its running norm (or absolutely tiny) -- a
	// settled, well-understood spot. bridge.ChooseClickAction's epistemic
	// escape reads it to stop exploiting an action there's nothing left to
	// learn from. Relative to the norm, so a baseline shift can't disable it.
	Predictable bool
	// SeededConcepts is how many concept nodes were ACTIVATED this cycle;
	// SeededConceptsFull is how many the full frame would activate without
	// narrowing. Equal normally; SeededConcepts < SeededConceptsFull exactly
	// when an acute surprise narrowed activation to the locus of change. (The
	// pair replaces comparing SeededConcepts against the blob count, which
	// went wrong once structural tokens joined the observation -- seeds then
	// exceeded the blob count and every cycle looked "narrowed".)
	SeededConcepts     int
	SeededConceptsFull int
	// LearningProgress is the intrinsic competence-gain drive (branch B):
	// positive where the forward model is actively improving, ~0 when mastered
	// or on noise. An intrinsic reward the agent can pursue with no external
	// reward signal.
	LearningProgress    float64
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
	// real predictive-coding comparison (see RunPredictiveCycle's Step 0b).
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

	// ForecastErrorEMA / ForecastSamples track the running norm of forecast
	// error so an ACUTE surprise can be defined RELATIVE to what's normal
	// lately, not as an absolute threshold. This is the fix for the cold-
	// start trap: an untrained forward model produces large errors that ARE
	// the norm, not anomalies -- narrowing attention on those would just
	// tunnel-vision on startup noise. Because the EMA tracks whatever errors
	// are typical right now, a large error during cold start doesn't exceed
	// its own running average, so it isn't flagged; only a spike above a
	// settled baseline is. See registerForecastError.
	ForecastErrorEMA float64
	ForecastSamples  int

	// PrevObservation is last cycle's full observation string, kept so this
	// cycle can compute the locus of change (tokens present now but not
	// then) -- the region attention narrows to under an acute surprise. See
	// changedTokens and RunPredictiveCycle Step 1a.
	PrevObservation string

	// CompressionThreshold is the mean intra-cluster edge weight at/above
	// which the sleep step collapses a cluster into one abstraction node
	// (Stage 3). Configurable (was a hardcoded 0.75) so the level can be
	// studied against the transfer thermometer -- organic co-activation edges
	// start at 0.4, so 0.75 is rarely reached and nothing compresses.
	CompressionThreshold float64

	// PrevForecastError / LearningProgress implement an intrinsic drive
	// (branch B, Oudeyer-style intrinsic motivation): LearningProgress is a
	// smoothed measure of how fast the forward model is GETTING BETTER
	// (forecast error decreasing over time). Unlike surprise (high error) or
	// settled (low error), this is the DERIVATIVE -- it's high precisely where
	// there is something learnable being learned, and ~0 both when a spot is
	// mastered (error flat low) and when it's unlearnable noise (error flat
	// high). That gives the agent something to want with no external reward:
	// competence gain.
	PrevForecastError float64
	LearningProgress  float64
}

const (
	// forecastEMAAlpha weights the newest forecast error in the running-norm
	// EMA (~4-cycle memory at 0.25).
	forecastEMAAlpha = 0.25
	// forecastSurpriseFactor: a forecast error counts as ACUTE only when it
	// exceeds the running norm by this multiple -- a genuine spike above
	// what's normal lately, not merely a large absolute value.
	forecastSurpriseFactor = 1.5
	// minForecastSamplesForSurprise is a warmup gate: no acute surprise can
	// fire until the EMA has seen at least this many real forecasts, so the
	// very first (necessarily untrained, necessarily large) errors can never
	// trigger narrowing before there's any baseline to be surprised against.
	minForecastSamplesForSurprise = 5
	// minAbsoluteForecastSurprise is an absolute floor on what counts as
	// acute, ON TOP of the relative test. A 1000-action live run exposed the
	// need: once the forward model perfectly predicts a dead 2-state loop, its
	// error oscillates between tiny values (e.g. 0.0011 and 0.0045), and the
	// high phase, though microscopic, is >1.5x the blended norm -- so the pure
	// relative test flagged "acute surprise" on ~half of a perfectly-predicted
	// dead loop (445/1000 actions). A surprise on a 0.0045 error is not a
	// surprise; requiring the error to also clear this floor kills that false
	// alarm while leaving genuine spikes (observed 0.1-1.4) untouched. It
	// doubles as the absolute-low floor for "settled" (see below).
	minAbsoluteForecastSurprise = 0.02
	// learningProgressAlpha smooths the LearningProgress signal (branch B):
	// the intrinsic competence-gain drive. ~4-cycle memory at 0.25.
	learningProgressAlpha = 0.25
	// forecastSettledFactor: a forecast error counts as SETTLED (the model
	// predicts this spot notably better than its recent norm -- a well-
	// understood place with nothing to learn) when it's below this fraction
	// of the running norm. Relative, for the same reason acute is: a baseline
	// shift (e.g. structural tokens raising every error) must not silently
	// disable it -- which is exactly what an absolute threshold did to the
	// epistemic escape once structure was wired in (87% of actions sat above
	// the old fixed 0.05, so the escape stopped firing and the agent re-locked).
	forecastSettledFactor = 0.6
)

// registerForecastError folds this cycle's forecast error into the running-
// norm EMA and returns two RELATIVE-to-norm judgments (both false until past
// warmup / when there was no forecast to score): acute, a genuine spike above
// the recent norm (and above an absolute floor -- see ForecastErrorEMA's
// field comment); and settled, its mirror image -- the model predicts this
// spot notably BELOW its norm (or absolutely tiny), a well-understood place
// with nothing to learn, which the epistemic escape in bridge.ChooseClickAction
// uses to stop exploiting. Both are relative so a baseline shift can't
// silently disable either; they're mutually exclusive by construction.
func (e *Engine) registerForecastError(forecastError float64, hadPrediction bool) (acute, settled bool) {
	if !hadPrediction {
		return false, false
	}
	if e.ForecastSamples >= minForecastSamplesForSurprise {
		acute = forecastError > minAbsoluteForecastSurprise &&
			forecastError > e.ForecastErrorEMA*forecastSurpriseFactor
		settled = forecastError < minAbsoluteForecastSurprise ||
			forecastError < e.ForecastErrorEMA*forecastSettledFactor
	}
	if e.ForecastSamples == 0 {
		e.ForecastErrorEMA = forecastError // seed the baseline to the first real error, not 0
	} else {
		e.ForecastErrorEMA = forecastEMAAlpha*forecastError + (1-forecastEMAAlpha)*e.ForecastErrorEMA
	}
	e.ForecastSamples++
	return acute, settled
}

// changedTokens returns the whitespace-joined subset of cur's tokens that do
// NOT appear in prev -- the locus of change since last cycle, which under an
// acute surprise is what attention narrows the graph seeding to. Returns ""
// when nothing is new (caller then keeps the full observation, since
// attention can't narrow to nothing).
func changedTokens(cur, prev string) string {
	prevSet := make(map[string]struct{})
	for _, t := range strings.Fields(prev) {
		prevSet[t] = struct{}{}
	}
	var changed []string
	for _, t := range strings.Fields(cur) {
		if _, seen := prevSet[t]; !seen {
			changed = append(changed, t)
		}
	}
	return strings.Join(changed, " ")
}

// actionBlendWeight is how strongly the chosen action perturbs the state
// representation the forward model conditions on (see ConditionForecastOnAction).
const actionBlendWeight = 1.0

// ConditionForecastOnAction makes the forward model action-conditioned:
// p(next | state, action) instead of p(next | state). Call it AFTER the action
// for this cycle is chosen (the action isn't known when RunPredictiveCycle
// runs). It blends the action's embedding into the state just cached, so both
// (a) the forecast of next cycle's observation and (b) the input the next cycle
// trains that forecaster on are conditioned on the action actually taken --
// which is what lets the model learn that the SAME state leads to DIFFERENT
// next states under different actions (unlearnable without conditioning: one
// input, two targets). No-op before the first cycle / on an empty token.
func (e *Engine) ConditionForecastOnAction(actionToken string) {
	if e.PrevPredictor == nil || e.PrevStateVector == nil || actionToken == "" {
		return
	}
	actVec := ObservationVector(actionToken)
	blended := make([]float64, len(e.PrevStateVector))
	norm := 0.0
	for i := range blended {
		blended[i] = e.PrevStateVector[i] + actionBlendWeight*actVec[i]
		norm += blended[i] * blended[i]
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range blended {
			blended[i] /= norm
		}
	}
	e.PrevStateVector = blended // next cycle trains (state+action -> realized next)
	_, out := e.PrevPredictor.MLP.Forward(blended)
	pending := make([]float64, len(out))
	copy(pending, out)
	e.PendingPrediction = pending // forecast now conditioned on the action
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

		CompressionThreshold: 0.75,
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
// forecast-vs-actual mismatch RunPredictiveCycle's Step 0b compares
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

	// 0b. Predictive-coding pre-pass. Computed BEFORE the graph is seeded
	// because an acute surprise narrows what gets seeded (Step 1a). The
	// content embedding stateVector is always the FULL frame regardless of
	// narrowing: the forward model must see whole, consistent frames to learn
	// (state_t -> state_{t+1}) transitions, so attention narrows what is
	// ATTENDED TO / acted on (the graph), never what is PREDICTED (the MLP) --
	// which is also the theoretically correct split (a full generative model,
	// with precision/attention selective on top), not a shortcut.
	stateVector := ObservationVector(observation)
	var forecastError float64
	hadPendingPrediction := e.PendingPrediction != nil
	if hadPendingPrediction {
		// Real cross-cycle prediction error: e.PendingPrediction is the
		// PREVIOUS cycle's specialist's forecast of THIS cycle's stateVector.
		// Comparing it against what actually arrived is a genuine
		// expectation-vs-reality mismatch across time (Step 5 trains the
		// forecaster on the realized transition, so it's a forward model, not
		// an autoencoder of the previous frame).
		forecastError = vectorMSE(e.PendingPrediction, stateVector)
	}
	acuteSurprise, predictable := e.registerForecastError(forecastError, hadPendingPrediction)

	// Intrinsic drive (branch B): smoothed rate at which the forward model is
	// improving here. Positive when this cycle's error fell below last
	// cycle's -- competence being gained -- and decays to ~0 both once a spot
	// is mastered and on unlearnable noise, so it uniquely marks "there's
	// something learnable here and I'm learning it".
	if hadPendingPrediction && e.ForecastSamples > 1 {
		delta := e.PrevForecastError - forecastError
		e.LearningProgress = learningProgressAlpha*delta + (1-learningProgressAlpha)*e.LearningProgress
	}
	if hadPendingPrediction {
		e.PrevForecastError = forecastError
	}

	// 1a. Attentional narrowing (precision-weighting / Easterbrook 1959 cue-
	// utilization: arousal narrows the range of cues attended). Under an
	// ACUTE surprise -- a forecast error spiking above the recent running
	// norm, deliberately NOT any large absolute error (see
	// registerForecastError for why a cold-start model's large-but-normal
	// errors must not trigger this) -- restrict the ACTIVATED seed set to the
	// locus of change, the blobs that appeared/moved since last cycle. Fewer
	// active seeds -> smaller spreading activation -> a tighter active subgraph
	// competing for the click. When the click just caused a local change, this
	// locus IS the click neighborhood. Node CREATION always covers the full
	// frame (below): attention narrows what the graph ATTENDS to this cycle,
	// never what it KNOWS exists -- so a blob first seen during a narrowed
	// cycle still gets its node.
	prevObservation := e.PrevObservation
	e.PrevObservation = observation

	// 1. Observation -> Node Lookup-or-Create -> Spreading Activation -> Active Subgraph Extraction
	// EnsureConceptNodes always runs on the FULL observation so the graph grows
	// from all real experience (novel vocabulary becomes nodes regardless of
	// narrowing). allSeeds is the full seed set; under an acute surprise the
	// ACTIVATED seeds are narrowed to the changed locus via LookupSeeds (the
	// nodes already exist from the full EnsureConceptNodes call, so the lookup
	// finds them). SeededConcepts vs len(allSeeds) then honestly shows whether
	// narrowing actually fired.
	allSeeds := e.Graph.EnsureConceptNodes(observation, 0.5)
	seeds := allSeeds
	if acuteSurprise {
		if focus := changedTokens(observation, prevObservation); focus != "" {
			seeds = e.Graph.LookupSeeds(focus)
		}
	}
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
	// (Step 0b above). Copied defensively rather than aliasing predResp's
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

	// 7b-ii. Intrinsic reward (branch B): competence gain -- the forward model
	// getting better here -- is rewarding in itself, with no external signal.
	// Positive LearningProgress raises Dopamine, giving the agent something to
	// pursue precisely where there is something learnable being learned (not
	// merely where it's surprised, and not where it's already mastered).
	if e.LearningProgress > 0 {
		lpReward := e.LearningProgress / (e.LearningProgress + 0.1) // saturates into [0,1)
		e.Homeostasis.Dopamine = math.Min(2.0, e.Homeostasis.Dopamine+lpReward*0.1)
	}

	// 7c. Acute forecast surprise raises Cortisol -- its first real
	// functional input (previously only drive-error moved it via
	// UpdateHormones, and nothing read it at all). This makes Cortisol the
	// observable "alarm from surprise" accumulator. Note the attentional
	// narrowing in Step 1a is gated on the acute-surprise signal DIRECTLY,
	// not on this Cortisol level, on purpose: UpdateHormones overwrites
	// Cortisol from drive error every cycle, and this same run showed
	// Dopamine's forecast contribution getting swamped by exactly that
	// homeostatic baseline -- gating attention on the hormone would repeat
	// that confound, so Cortisol reflects surprise here without being the
	// thing attention keys off.
	if acuteSurprise {
		e.Homeostasis.Cortisol = math.Min(2.0, e.Homeostasis.Cortisol+0.25)
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
		compression = offline.CompressGraphAbstractions(e.Sys, e.Graph, e.CompressionThreshold)
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
		AcuteSurprise:      acuteSurprise,
		Predictable:        predictable,
		SeededConcepts:     len(seeds),
		SeededConceptsFull: len(allSeeds),
		LearningProgress:   e.LearningProgress,
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
