// Package bridge connects Protaxon's graph/router cognition
// (pkg/pipeline.Engine) to an environment.Environment: Observe -> Decide ->
// Act, closing the loop that Stage 4 built the pieces for but never drove
// against a real interactive task.
package bridge

import (
	"context"
	"fmt"
	"math"
	"protaxon/pkg/environment"
	"protaxon/pkg/environment/perception"
	"protaxon/pkg/graph"
	"protaxon/pkg/pipeline"
	"strings"
)

// DescribeFrame converts an agent/target position pair into a short
// textual observation Protaxon's graph-based cognition can act on: just the
// bare direction words needed, nothing else. (Deliberately no filler words
// like "target" -- EnsureConceptNodes would create a node for that word
// too, and it would compete with the real direction concepts for the
// router's attention on every single cycle, since it's a fixed word that
// always appears.) This is a deliberately simple perception stand-in -- a
// real vision pipeline over the raw 64x64 grid is future work; this exists
// to validate the Observe -> Decide -> Act loop end to end, not to solve
// perception.
func DescribeFrame(agentX, agentY, targetX, targetY int) string {
	words := make([]string, 0, 2)
	if targetY < agentY {
		words = append(words, "north")
	} else if targetY > agentY {
		words = append(words, "south")
	}
	if targetX < agentX {
		words = append(words, "west")
	} else if targetX > agentX {
		words = append(words, "east")
	}
	if len(words) == 0 {
		return "aligned"
	}
	return strings.Join(words, " ")
}

var directionActions = map[string]environment.ActionID{
	"north": environment.Action1,
	"south": environment.Action2,
	"west":  environment.Action3,
	"east":  environment.Action4,
}

// ChooseAction runs one predictive cycle against engine using observation,
// then reads back which direction concept actually won the Stage 2 router's
// competition (res.ActiveNodeIDs) and maps that label to a game action.
//
// This deliberately reuses Protaxon's existing graph/router machinery as
// the decision mechanism rather than adding a separate hand-written policy
// -- but it is NOT a claim that the choice is "intelligent." With no prior
// training on this specific task, which direction wins is governed by
// Winner-Takes-All tie-breaking among concepts (activation magnitude, and
// node-creation order on exact ties), not a learned preference. What this
// validates is that the full loop actually wires together end to end.
//
// actualSuccess is passed as true on every intermediate step -- there is no
// clear per-step reward signal from the environment yet (only a terminal
// WIN/GAME_OVER), so this is a known, documented simplification for this
// first version, not a claim that every step was actually "successful."
func ChooseAction(ctx context.Context, engine *pipeline.Engine, observation, goal string) (environment.Action, *pipeline.CycleResult, error) {
	res, err := engine.RunPredictiveCycle(ctx, observation, goal, true)
	if err != nil {
		return environment.Action{}, nil, fmt.Errorf("predictive cycle: %w", err)
	}

	for _, nodeID := range res.ActiveNodeIDs {
		word := labelForNode(engine.Graph, nodeID)
		if actionID, ok := directionActions[word]; ok {
			return environment.Action{ID: actionID}, res, nil
		}
	}

	// No recognized direction concept won this cycle -- shouldn't happen
	// once DescribeFrame emits at least one direction word, but stay safe
	// rather than returning a zero-value action that could hang the caller.
	return environment.Action{ID: environment.Action2}, res, nil
}

func labelForNode(g *graph.Graph, nodeID int) string {
	for keyword, ids := range g.Labels {
		for _, id := range ids {
			if id == nodeID {
				return keyword
			}
		}
	}
	return ""
}

// ChooseClickAction runs one predictive cycle over a real ARC-AGI-3 grid
// and returns a concrete ACTION6 click (X, Y).
//
// The graph/router (Stage 2/4) competes over blob CATEGORIES --
// perception.DescribeGridCells's composite tokens like "color2-cell3-2" --
// exactly the way ChooseAction competes over direction words for Beacon.
// Category identity is stable across frames via the same label-reuse
// mechanism EnsureConceptNodes already gives "north"/"south" (same graph
// node ID reused every time the label recurs), so Hebbian weight and
// eligibility traces on a category node accumulate real, comparable
// signal over time -- unlike a fresh per-frame blob object/ID, which
// would never accumulate anything at all.
//
// A category node has no pixel coordinates of its own, though: WHICH blob
// currently carries that label has to be re-derived every call from the
// frame actually in hand -- that's the bind step. After RunPredictiveCycle,
// this reads off the true WTA winner (the active node with the highest
// post-inhibition Activation -- ActiveNodeIDs itself is just ascending
// sorted IDs, see graph.ExtractActiveSubgraph, not ranked by strength),
// and looks up which of THIS frame's blobs (perception.RankedLabeledBlobs,
// the same ranking DescribeGridCells used to build the observation)
// currently carries that label. If the winning label isn't a recognized
// blob-category token, or doesn't match any blob actually present now
// (e.g. spreading activation surfaced a node from past experience that
// this frame doesn't have), it falls back to the frame's single
// highest-ranked blob rather than hanging the caller on a zero-value
// action.
//
// This does NOT claim to solve action selection for arbitrary ARC-AGI-3
// games -- it wires the existing Stage 2/4 competition machinery into a
// concrete click instead of leaving it unconnected. Games where the
// correct target isn't "the most-reinforced category of colored region"
// (most of them, honestly) will not be well served by this.
//
// previousObservation is the observation string ChooseClickAction returned
// last call (empty string on the very first call, e.g. right after
// Reset()) -- used to compute actualSuccess as "did the grid visibly
// change since the action that produced THIS frame" (see
// actionSucceeded), instead of unconditionally claiming success. This is
// a real, if cheap, signal: confirmed 2026-08-13 against the live service
// that without it, the router just keeps re-clicking whatever category won
// once, with nothing to push it away from a click that provably does
// nothing (one location got clicked 14 of 20 actions with zero observed
// effect, while a different, rarely-tried location visibly changed the
// grid twice and was never reinforced over the dead one). "Grid changed"
// is not the same as "score went up" -- a strictly better signal (real
// score/levels_completed) is a natural next upgrade, not a claim that
// this one is sufficient on its own.
//
// Returns the observation this call computed, so the caller can pass it
// back in as previousObservation next call without recomputing it.
//
// curiosityStep is how much engine.Homeostasis.Curiosity (setpoint 0.0 per
// pkg/homeostasis, otherwise unwired into any real cycle before this) moves
// per call: up on failure, down on success, clamped to [0,1] -- unresolved
// "why didn't that work" tension that exploration below discharges.
//
// explorationRoll is a caller-supplied number in [0,1) (production: a real
// random draw; tests: a fixed value for determinism, same philosophy as
// this codebase's explicit MLP seeds). When explorationRoll < Curiosity,
// this deliberately picks a DIFFERENT blob than the one WTA/fallback would
// have chosen -- excluding that default choice rather than sampling over
// everything means an exploration roll always actually changes behavior,
// never coincidentally reselects the same dead click. This is the fix for
// a real failure mode confirmed 2026-08-13: 123 real actions, only 5
// distinct points ever tried, because nothing ever pushed the choice away
// from whatever won once.
//
// memory and previousClickedLabel add a first, deliberately modest step
// toward tracking real outcomes instead of relying purely on graph
// dynamics: previousClickedLabel (the label THIS caller clicked last call,
// "" on the first) gets its outcome recorded into memory using THIS
// cycle's actualSuccess, and if any of the CURRENT frame's candidates has
// a proven track record (see minProvenAttempts), that candidate is
// preferred over both the WTA default and any exploration pick -- real
// accumulated evidence outranks structural graph state, which a live run
// showed can get scrambled by Louvain re-clustering even when nothing
// about the world actually changed. Returns the label actually clicked
// this call, for the caller to pass back in as previousClickedLabel next
// time.
//
// levelsCompletedIncreased is real ground truth from the game server (the
// caller compares environment.Frame.LevelsCompleted across calls) and, when
// true, forces actualSuccess regardless of what actionSucceeded's grid-diff
// proxy says. This closes a real gap confirmed 2026-08-13: a 300-action
// live run had OutcomeMemory correctly, mechanically lock onto a click that
// reliably satisfied the grid-changed proxy (96% of the time) while
// levels_completed and score stayed at 0 the entire run -- proof the proxy
// and the actual goal aren't the same thing, not a bug in OutcomeMemory
// itself. levelsCompletedIncreased is deliberately an OR-strengthener, not
// a replacement: real level completion is rare (most individual actions
// won't trigger one even when making real progress), so the grid-changed
// proxy still has to carry most cycles -- this only means "if the game
// itself just confirmed real progress, believe that over anything else."
func ChooseClickAction(ctx context.Context, engine *pipeline.Engine, grid [][]int, goal string, maxBlobs, cols, rows int, previousObservation string, levelsCompletedIncreased bool, curiosityStep, explorationRoll float64, memory *OutcomeMemory, previousClickedLabel string) (environment.Action, string, string, *pipeline.CycleResult, error) {
	if memory == nil {
		// A caller that forgot to construct one loses cross-call persistence
		// (this fresh instance dies with the call), but that's a silent
		// capability downgrade, not a nil-pointer panic on memory.Record/
		// SuccessRate below.
		memory = NewOutcomeMemory()
	}

	labeled := perception.RankedLabeledBlobs(grid, maxBlobs, cols, rows)
	if len(labeled) == 0 {
		return environment.Action{}, "", "", nil, fmt.Errorf("bridge: no blobs found in grid, nothing to click")
	}

	// Structural observation: per-blob surface tokens PLUS surface-agnostic
	// structural tokens (RelationalTokens). The forward model and graph now
	// see structure that can transfer across surface-different domains; the
	// structural tokens aren't blob-shaped so they never become click targets.
	observation := perception.DescribeGridStructural(grid, maxBlobs, cols, rows)
	actualSuccess := levelsCompletedIncreased || actionSucceeded(previousObservation, observation)
	memory.Record(previousClickedLabel, actualSuccess)

	if actualSuccess {
		engine.Homeostasis.Curiosity = math.Max(0.0, engine.Homeostasis.Curiosity-curiosityStep)
	} else {
		engine.Homeostasis.Curiosity = math.Min(1.0, engine.Homeostasis.Curiosity+curiosityStep)
	}

	res, err := engine.RunPredictiveCycle(ctx, observation, goal, actualSuccess)
	if err != nil {
		return environment.Action{}, observation, "", nil, fmt.Errorf("predictive cycle: %w", err)
	}

	// Epistemic drive (active inference): when the forward model already
	// predicts the current dynamics well, the scene offers nothing new to
	// learn -- exploiting a "proven" action here is the dead-loop trap a
	// 1000-action live run exposed (one pixel clicked 976/1000 times). So when
	// the outcome is predictable, push exploration UP (countering the proxy-
	// driven curiosity collapse that caused the lock-in) and, below, refuse to
	// let the proven-outcome override re-lock onto the epistemically exhausted
	// action. "Predictable" is res.Predictable -- error notably below its
	// RUNNING NORM (pipeline Step 0b), deliberately relative rather than an
	// absolute threshold: wiring structural tokens into the observation raised
	// every forecast error, and an earlier absolute cutoff silently stopped
	// firing (87% of actions sat above it) so the agent re-locked. The relative
	// signal survives that baseline shift.
	predictable := res.Predictable
	if predictable {
		engine.Homeostasis.Curiosity = math.Min(1.0, engine.Homeostasis.Curiosity+curiosityStep)
	}

	defaultIndex := 0
	if winner := winningBlobLabel(engine.Graph, res.ActiveNodeIDs); winner != "" {
		for i, lb := range labeled {
			if lb.Label == winner {
				defaultIndex = i
				break
			}
		}
	}
	chosenIndex := defaultIndex

	if curiosity := engine.Homeostasis.Curiosity; explorationRoll < curiosity && len(labeled) > 1 {
		others := make([]int, 0, len(labeled)-1)
		for i := range labeled {
			if i != defaultIndex {
				others = append(others, i)
			}
		}
		normalizedRoll := explorationRoll / curiosity // curiosity > explorationRoll >= 0, so curiosity > 0 here
		idx := int(normalizedRoll * float64(len(others)))
		if idx >= len(others) {
			idx = len(others) - 1
		}
		chosenIndex = others[idx]
	}

	bestProvenIndex := -1
	bestProvenRate := 0.0
	for i, lb := range labeled {
		if rate, attempts := memory.SuccessRate(lb.Label); attempts >= minProvenAttempts && rate > bestProvenRate {
			bestProvenRate = rate
			bestProvenIndex = i
		}
	}
	// The proven-outcome override yields when the outcome is predictable: an
	// action the forward model can already forecast has no epistemic value,
	// and continuing to exploit it (on the grid-changed proxy, which we know
	// is misaligned -- real progress would have fired levelsCompletedIncreased)
	// is exactly the reward-hacking loop. Suppressing it here lets the
	// exploration/WTA choice above stand, breaking the single-point lock-in.
	if bestProvenIndex >= 0 && !predictable {
		chosenIndex = bestProvenIndex
	}

	clickedLabel := labeled[chosenIndex].Label
	return clickAction(labeled[chosenIndex].Blob), observation, clickedLabel, res, nil
}

// minProvenAttempts is how many recorded attempts a label needs before
// OutcomeMemory's accumulated success rate is trusted enough to override
// the WTA/exploration choice -- low enough to matter within a real run's
// action budget (tens, not thousands, of actions), high enough that a
// single lucky/unlucky outcome can't flip the decision.
const minProvenAttempts = 3

// actionSucceeded reports whether the grid visibly changed since the
// previous frame's observation -- the cheapest honestly-available proxy
// for "did the last action accomplish anything." An empty
// previousObservation (no prior frame, e.g. the very first call after
// Reset) counts as success: there is no action of ours to judge yet, so
// this is a bootstrap default, not a real judgment.
func actionSucceeded(previousObservation, observation string) bool {
	return previousObservation == "" || observation != previousObservation
}

func clickAction(b perception.Blob) environment.Action {
	return environment.Action{ID: environment.Action6, X: b.Centroid.X, Y: b.Centroid.Y}
}

// winningBlobLabel finds the highest-Activation node among activeNodeIDs
// whose label looks like a DescribeGridCells composite token, and returns
// that label ("" if none qualify). Restricting to blob-shaped labels means
// this ignores any other vocabulary that might also be active in
// ActiveNodeIDs (defensive: not a concern with the current single-workload
// caller, but ChooseClickAction shouldn't silently misfire if this engine
// is ever reused alongside other observation sources).
//
// bestActivation starts at negative infinity, not a hardcoded -1.0: a real
// live run (2026-08-13) drove several nodes' Activation to roughly -4.12
// after a long actualSuccess=false streak (see MaxWeightMagnitude's doc
// comment) -- with a -1.0 sentinel, EVERY candidate would have failed the
// `Activation > bestActivation` check and this returned "" (silently
// forcing ChooseClickAction's fallback path), even though a real, correct
// winner (the least-negative candidate) existed. -Inf has no such blind
// spot regardless of how negative activations get.
func winningBlobLabel(g *graph.Graph, activeNodeIDs []int) string {
	bestActivation := math.Inf(-1)
	best := ""
	for _, id := range activeNodeIDs {
		node, exists := g.Nodes[id]
		if !exists {
			continue
		}
		word := labelForNode(g, id)
		if !looksLikeBlobLabel(word) {
			continue
		}
		if node.Activation > bestActivation {
			bestActivation = node.Activation
			best = word
		}
	}
	return best
}

// looksLikeBlobLabel reports whether word matches DescribeGridCells's
// "color<N>-cell<C>-<R>" shape.
func looksLikeBlobLabel(word string) bool {
	return strings.HasPrefix(word, "color") && strings.Contains(word, "-cell")
}

// DescribeCategoryGraphState is a diagnostic, not a decision function: for
// each of the given blob-category labels, it reports the graph node's ID,
// ClusterID, current Activation, and any edges to the OTHER given labels
// with their weights.
//
// Purpose: answer, from real internal graph state rather than guessing
// from external behavior, WHICH mechanism is actually diversifying
// ChooseClickAction's choices over repeated failed clicks -- (a) Hebbian
// edge-weight changes driven by actualSuccess (see ChooseClickAction),
// which only affect a node's activation via propagation on
// SpreadingActivation's last hop (seed activation itself is always a flat
// 1.0 from LookupSeeds, confirmed 2026-08-13 by reading pkg/graph/node.go
// and pkg/graph/spreading.go -- Hebbian reward cannot suppress a directly-
// seeded node's own activation, only what it propagates to others via an
// edge), or (b) SubconsciousSleep's Louvain re-clustering splitting
// labels into separate competing clusters, a structural process
// independent of reward. Both look identical from outside (clicks
// diversify); this tells them apart.
func DescribeCategoryGraphState(g *graph.Graph, labels []string) string {
	idFor := make(map[string]int, len(labels))
	for _, label := range labels {
		if ids := g.Labels[label]; len(ids) > 0 {
			idFor[label] = ids[0]
		}
	}

	var lines []string
	for _, label := range labels {
		id, ok := idFor[label]
		if !ok {
			lines = append(lines, fmt.Sprintf("%s: not yet in graph", label))
			continue
		}
		node, exists := g.Nodes[id]
		if !exists {
			lines = append(lines, fmt.Sprintf("%s: node %d missing from graph", label, id))
			continue
		}

		line := fmt.Sprintf("%s: node=%d cluster=%d activation=%.4f", label, id, node.ClusterID, node.Activation)
		var edgeParts []string
		for _, other := range labels {
			if other == label {
				continue
			}
			otherID, ok := idFor[other]
			if !ok {
				continue
			}
			if edge, exists := node.Edges[otherID]; exists {
				edgeParts = append(edgeParts, fmt.Sprintf("->%s(w=%.4f)", other, edge.Weight))
			}
		}
		if len(edgeParts) > 0 {
			line += " edges:" + strings.Join(edgeParts, ",")
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, " | ")
}
