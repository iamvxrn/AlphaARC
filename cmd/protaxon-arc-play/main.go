// Command protaxon-arc-play actually PLAYS one real ARC-AGI-3 game end to
// end, using bridge.ChooseClickAction to pick every action -- unlike
// cmd/protaxon-arc-smoke, which deliberately never calls Step().
//
// Only games whose available_actions include ACTION6 are supported --
// ChooseClickAction only ever proposes a click, so anything else isn't
// handled honestly here; the loop stops immediately if a game doesn't
// offer ACTION6, rather than silently sending an action the game never
// asked for.
//
// Defaults are deliberately conservative for a first live run against
// completely untested territory (a real game ChooseClickAction has never
// seen before): -actions 20, not the reference Python SDK's 80, so a bad
// decision loop can't quietly burn a large number of real API calls
// before anyone notices; and -cols/-rows 16 (a fine lattice) prioritizes
// click precision over the graph-cohesion stability a coarser lattice
// would give -- see ARCHITECTURE.md's coarse-vs-fine cohesion experiment:
// an imprecise click fails immediately and hard, while lower cohesion
// degrades gracefully and needs many more cycles than a short first run
// gets to even matter.
//
// Requires ARC_API_KEY in the environment (see .env.example). Run:
//
//	go run ./cmd/protaxon-arc-play
//	go run ./cmd/protaxon-arc-play -game <game_id> -actions 40
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"

	"protaxon/pkg/environment"
	"protaxon/pkg/environment/bridge"
	"protaxon/pkg/environment/perception"
	"protaxon/pkg/environment/remote"
	"protaxon/pkg/pipeline"
)

func main() {
	gameID := flag.String("game", "", "game_id to play (default: first game from /api/games)")
	maxActions := flag.Int("actions", 20, "hard cap on real Step() calls before giving up")
	maxBlobs := flag.Int("maxblobs", 5, "max blobs perceived per frame")
	cols := flag.Int("cols", 16, "grid-cell lattice columns for ChooseClickAction")
	rows := flag.Int("rows", 16, "grid-cell lattice rows for ChooseClickAction")
	curiosityStep := flag.Float64("curiosity-step", 0.1, "how much Curiosity moves per action, up on failure, down on success")
	exploreActions := flag.Float64("explore-actions", 0.2, "probability of trying a random available SIMPLE action (ACTION1-5) instead of a click, so the agent explores the whole action space, not just clicks")
	flag.Parse()

	ctx := context.Background()

	client, err := remote.NewClientFromEnv()
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	fmt.Printf("base URL: %s\n", client.BaseURL())

	games, err := client.ListGames(ctx)
	if err != nil {
		log.Fatalf("list games: %v", err)
	}
	if len(games) == 0 {
		log.Fatalf("no games returned -- nothing to play")
	}
	target := *gameID
	if target == "" {
		target = games[0].GameID
	}

	cardID, err := client.OpenScorecard(ctx, []string{"protaxon-arc-play"})
	if err != nil {
		log.Fatalf("open scorecard: %v", err)
	}
	fmt.Printf("scorecard opened: %s\n", cardID)

	sess := remote.NewSession(client, target, cardID)
	frame, err := sess.Reset()
	if err != nil {
		client.CloseScorecard(ctx, cardID) //nolint:errcheck -- best-effort cleanup before exiting on the real error
		log.Fatalf("reset %s: %v (scorecard %s closed on a best-effort basis)", target, err, cardID)
	}
	fmt.Printf("RESET %s -> state=%s available_actions=%v\n", target, frame.State, frame.AvailableActions)

	engine := pipeline.NewEngine()
	memory := bridge.NewOutcomeMemory()
	// preferenceOf is the single source of truth for prior preference over a
	// state: learned goal + weak structural prior + spatial approach to the key,
	// MINUS a self-preservation term (resemblance to a state that got the agent
	// killed). One place so the in-loop, init, and post-reset values can't drift.
	preferenceOf := func(grid [][]int) float64 {
		vec := pipeline.ObservationVector(perception.DescribeGridStructural(grid, *maxBlobs, *cols, *rows))
		return engine.LearnedPreference(vec) + 0.1*perception.StructureScore(grid) + perception.ApproachPreference(grid) - dangerAvoidWeight*engine.DangerProximity(vec)
	}
	actionsTaken := 0
	prevObservation := ""
	prevClickedLabel := ""
	prevLevelsCompleted := frame.LevelsCompleted
	prevPreference := preferenceOf(frame.Grid)
	deadCount := map[string]float64{}                             // per action type: decaying count of "did nothing" (inhibition of return / taboo)
	tries := map[string]int{}                                     // per action type: how many times chosen (optimism for the under-tried)
	stagnation := 0                                               // consecutive actions with no frame change and no level progress
	lastActionTok := ""
	prevNumeric := ""                        // last frame's numeric progress tokens (nobj*/tdist*), for per-target inhibition of return
	tracker := perception.NewObjectTracker() // stable object identity + motion across frames (Core Knowledge)
	prevTopology := ""                       // last frame's object-topology signature, for conveyor-belt-aware stagnation

	for actionsTaken < *maxActions {
		if !hasAction6(frame.AvailableActions) {
			fmt.Printf("game does not offer ACTION6 (available: %v) -- ChooseClickAction only proposes clicks, stopping\n", frame.AvailableActions)
			break
		}

		// Diagnostic: total blobs actually present this frame, UNBOUNDED by
		// -maxblobs -- confirms or refutes, from a real count instead of a
		// guessed flag value, whether a real interactive element could be
		// getting cut off by the top-N-by-size ranking before it's ever
		// perceived at all.
		totalBlobs := len(perception.FindBlobs(frame.Grid, perception.BackgroundColor(frame.Grid)))
		fmt.Printf("action %d: %d total blobs in frame (using top %d)\n", actionsTaken+1, totalBlobs, *maxBlobs)

		// Real ground truth from the game server for whether the PREVIOUS
		// action actually completed a level, computed the same way as
		// changedSinceLastFrame below (comparing this frame's value against
		// the one recorded before that previous action). Passed into
		// ChooseClickAction as a strong override on top of the "did the
		// grid change" proxy -- see the doc comment on ChooseClickAction for
		// why this override was added (2026-08-13, a 300-action live run
		// exploiting the proxy while levels_completed never moved).
		levelsCompletedIncreased := frame.LevelsCompleted > prevLevelsCompleted

		// Core Knowledge brick 2: object MOTION since the previous frame -- a
		// cross-frame fact the single-frame grid can't express -- fed into the
		// observation so the forward model perceives that bodies moved.
		// Core Knowledge: stable object identity + motion via the tracker
		// (continuity-based, robust to pixel deformation -- unlike the brittle
		// shape hash). These tokens feed the observation so the graph gets one
		// persistent node per body. topo is the frame's object-level fingerprint
		// (which bodies exist), used below for conveyor-belt-aware stagnation.
		// Object identity + motion, PLUS the numeric-magnitude channel (object
		// count, quantized distance to the salient target) so the observation
		// carries a genuine sense of number, not only categorical tokens.
		numericObs := strings.Join(perception.NumericTokens(frame.Grid), " ")
		motionObs := strings.Join(append(tracker.Track(frame.Grid), numericObs), " ")
		topo := tracker.TopologySignature()
		topologyChanged := topo != prevTopology
		prevTopology = topo

		// Fix 3: click candidates in the graph's OWN vocabulary (obj-id labels +
		// centroids), computed after Track so ids are current. These become both
		// the selection candidates and the graph-state diagnostic's labels, so the
		// diagnostic finally shows real node/activation/edges instead of the
		// "not yet in graph" the cell-labels always produced.
		objCandidates := tracker.LabeledObjects()
		labels := make([]string, len(objCandidates))
		for i, lb := range objCandidates {
			labels[i] = lb.Label
		}

		// Inhibition of Return (fix 2): if the PREVIOUS click left the numeric
		// progress tokens (nobj*/tdist*) unchanged, it accomplished nothing
		// meaningful -- lock that target out for a few steps so the budget stops
		// draining into re-toggling an already-checked switch. Judged on the
		// numeric channel, not raw pixels: a switch that visually flips but moves
		// no body/nothing closer is exactly the dead click to refract. Applied
		// BEFORE the choice so this step can't re-pick it.
		if prevClickedLabel != "" && numericObs == prevNumeric {
			memory.Refract(prevClickedLabel, refractorySteps)
		}
		prevNumeric = numericObs

		action, observation, clickedLabel, res, err := bridge.ChooseClickAction(ctx, engine, frame.Grid, "solve the puzzle", *maxBlobs, *cols, *rows, prevObservation, levelsCompletedIncreased, *curiosityStep, rand.Float64(), memory, prevClickedLabel, motionObs, objCandidates)
		if err != nil {
			fmt.Printf("action %d: choose action failed: %v\n", actionsTaken, err)
			break
		}
		// actualSuccess fed into this call's predictive cycle was exactly
		// this comparison (see bridge.actionSucceeded) -- printed here so
		// it's visible whether the router just credited or blamed the
		// previous click for actually changing anything. Curiosity is
		// printed too since it's what determined whether this action was
		// an exploration override or the default WTA/fallback choice.
		changedSinceLastFrame := prevObservation == "" || observation != prevObservation

		// Inhibition of return + stagnation, judged on OBJECT TOPOLOGY, not raw
		// pixels (conveyor-belt fix): a body sliding under act-5 changes pixels
		// every step (changedSinceLastFrame stays true) but the set of bodies is
		// unchanged -- trivial simulation ticking, not progress. So "meaningful
		// progress" is a topology change (a body created/destroyed) OR a level
		// gain; otherwise the last action gets taboo'd and stagnation grows even
		// while pixels churn. When stagnation is sharp, spike curiosity rather
		// than let a predictable conveyor quench it.
		meaningfulProgress := topologyChanged || levelsCompletedIncreased
		if lastActionTok != "" {
			if !meaningfulProgress {
				deadCount[lastActionTok] = deadCount[lastActionTok]*0.9 + 1
				stagnation++
			} else {
				deadCount[lastActionTok] *= 0.5
				stagnation = 0
			}
		}
		if stagnation >= 4 {
			engine.Homeostasis.Curiosity = 1.0 // sharp anti-stagnation kick, not a quench
		}

		fmt.Printf("action %d: perceives %q (changed since last frame: %v, levels_completed_increased: %v, curiosity=%.4f, stagnation=%d)\n",
			actionsTaken+1, observation, changedSinceLastFrame, levelsCompletedIncreased, engine.Homeostasis.Curiosity, stagnation)
		// Diagnostic: the predictive-coding signals from THIS cycle's internal
		// forward model. forecast_error is how wrong the PREVIOUS cycle's
		// prediction of this frame turned out to be -- a real cross-cycle
		// surprise measured against a content-based observation embedding, NOT
		// the "did the grid change" proxy on the perceives line above. dopamine
		// is the global plasticity multiplier that surprise now drives (higher
		// on surprise). What to watch for on a live run, before this signal is
		// trusted with anything beyond plasticity: does forecast_error actually
		// rise when the grid visibly changes (changed since last frame: true)
		// and settle when it doesn't, and does dopamine track it? If instead it
		// stays flat regardless of what the frame does, the forward model isn't
		// perceiving the world and the whole predictive-coding layer is cosmetic.
		// seeded=%d/%d shows attentional narrowing directly: activated concepts
		// out of what the full frame would activate. Equal normally; the first
		// number drops below the second exactly when an acute surprise narrows
		// activation to the locus of change.
		fmt.Printf("action %d: predictive-coding: forecast_error=%.4f dopamine=%.4f acute_surprise=%v seeded=%d/%d cortisol=%.4f\n",
			actionsTaken+1, res.ForecastError, engine.Homeostasis.Dopamine, res.AcuteSurprise, res.SeededConcepts, res.SeededConceptsFull, engine.Homeostasis.Cortisol)
		// Diagnostic: real internal graph state for THIS cycle's candidate
		// categories -- cluster assignment, activation, and edges between
		// them -- so diversification in the click choice can be attributed
		// to an actual mechanism (Hebbian edge weight vs. Louvain
		// re-clustering) instead of guessed from external behavior alone.
		fmt.Printf("action %d: graph state: %s\n", actionsTaken+1, bridge.DescribeCategoryGraphState(engine.Graph, labels))
		// Diagnostic: what OutcomeMemory has accumulated for the category
		// about to be clicked -- distinguishes "picked because it's proven"
		// from "picked because WTA/exploration said so with zero evidence."
		// Action-space exploration: ChooseClickAction only ever proposes a
		// click (ACTION6), so the agent has been playing with 1 of the 3
		// available actions. A game may REQUIRE a simple control (ACTION1-5)
		// that no click can substitute for -- the most likely reason a
		// click-only agent stalls at score 0 forever. With probability
		// -explore-actions, send a random available simple action instead, so
		// the agent actually tries the whole action space and can discover
		// (via levels_completed / the forecast) what those controls do. Click
		// (6) and undo (7) are excluded from this exploration.
		// Goal-directed action selection (the A+B+C synergy, live): choose the
		// action TYPE by predicted competence gain -- BestCompetenceAction picks
		// the action the forward model is learning the most from -- with
		// epsilon-exploration (-explore-actions) so untried actions get tried at
		// all. "click" uses ChooseClickAction's chosen target; "act-N" sends the
		// simple control N. The forward model is then conditioned (branch A) on
		// the type actually taken, which is also what accrues the per-action
		// learning progress this selection reads next time.
		// PER-OBJECT action selection (the fix the uniform-graph run motivated).
		// Each object is its OWN click candidate -- token "click-<objlabel>" --
		// instead of a single "click". The forward model is then conditioned on
		// that per-object token, so ActionLearningProgress/ActionPreferenceGain
		// accrue PER OBJECT and the planner can prefer the object it's learning
		// most from / that yields the most preference gain -- breaking the
		// symmetry that made every toggle-node identical when "click" was one
		// token. act-N simple controls stay in the same balance.
		candidates := []string{}
		clickByToken := map[string]perception.LabeledBlob{}
		for _, ob := range objCandidates {
			tok := "click-" + ob.Label
			candidates = append(candidates, tok)
			clickByToken[tok] = ob
		}
		if len(candidates) == 0 {
			// No obj candidates (cell-label fallback path): a bare "click" that
			// uses ChooseClickAction's own chosen target.
			candidates = append(candidates, "click")
		}
		simpleByToken := map[string]environment.ActionID{}
		for _, a := range availableSimpleActions(frame.AvailableActions) {
			tok := fmt.Sprintf("act-%d", a)
			candidates = append(candidates, tok)
			simpleByToken[tok] = a
		}
		// Plan by expected free energy (preference gain + competence), with an
		// OPTIMISM bonus for under-tried candidates, a TABOO penalty (exponential
		// in recent deadness), and a small GRAPH-PRIOR bonus for the object the
		// reconnected graph's WTA actually chose -- so at cold start (all
		// per-object gains ~0) the graph decides, and the learned per-object signal
		// takes over as it accrues.
		graphPick := "click-" + clickedLabel
		chosenTok, bestVal := "", math.Inf(-1)
		for _, c := range candidates {
			efe := engine.ActionPreferenceGain[c] + 0.3*engine.ActionLearningProgress[c]
			optimism := 0.05 / float64(1+tries[c])
			taboo := 0.1 * (1 - math.Exp(-deadCount[c]))
			prior := 0.0
			if c == graphPick {
				prior = graphPriorBonus
			}
			if v := efe + optimism - taboo + prior; v > bestVal {
				bestVal, chosenTok = v, c
			}
		}
		exploring := rand.Float64() < *exploreActions
		if exploring || chosenTok == "" {
			chosenTok = candidates[rand.Intn(len(candidates))]
		}
		tries[chosenTok]++
		lastActionTok = chosenTok
		exploredAction := false
		if ob, ok := clickByToken[chosenTok]; ok {
			// Click THIS specific object's centroid (may differ from
			// ChooseClickAction's default when the per-object EFE prefers another).
			action = environment.Action{ID: environment.Action6, X: ob.Blob.Centroid.X, Y: ob.Blob.Centroid.Y}
			clickedLabel = ob.Label
		} else if id, ok := simpleByToken[chosenTok]; ok {
			action = environment.Action{ID: id}
			clickedLabel = "" // not a click -- don't credit a blob label to this step
			exploredAction = true
		}
		// (chosenTok == "click" bare fallback keeps ChooseClickAction's action/label.)
		engine.ConditionForecastOnAction(chosenTok)

		fmt.Printf("action %d: chose %q (%s) -- pref_gain=%+.4f competence=%+.4f%s\n",
			actionsTaken+1, chosenTok,
			map[bool]string{true: "exploring", false: "by expected free energy"}[exploring],
			engine.ActionPreferenceGain[chosenTok], engine.ActionLearningProgress[chosenTok],
			simpleActionGainString(engine.ActionPreferenceGain, simpleByToken))
		if exploredAction {
			fmt.Printf("action %d: sending simple action %v (not a click)\n", actionsTaken+1, action.ID)
		} else if rate, attempts := memory.SuccessRate(clickedLabel); attempts > 0 {
			fmt.Printf("action %d: clicking %q, prior record: %d/%d successful\n", actionsTaken+1, clickedLabel, int(rate*float64(attempts)+0.5), attempts)
		} else {
			fmt.Printf("action %d: clicking %q, no prior record\n", actionsTaken+1, clickedLabel)
		}
		prevObservation = observation
		prevClickedLabel = clickedLabel
		prevLevelsCompleted = frame.LevelsCompleted

		// The state this action is about to be taken FROM -- captured before the
		// grid is reassigned, so if the action turns out fatal we can remember
		// this "about to die" configuration as a danger exemplar.
		preStepGrid := frame.Grid
		newFrame, stepErr := sess.Step(action)
		actionsTaken++
		if stepErr != nil {
			fmt.Printf("action %d: step failed: %v\n", actionsTaken, stepErr)
			break
		}
		frame = newFrame

		// Pragmatic drive (source of meaning). LEARNED, not guessed (the dream):
		// if this action completed a level, remember the winning state as the
		// goal; thereafter preference = how much a state resembles a winning one
		// (LearnedPreference), plus a weak structural prior so there's some
		// gradient BEFORE the first win (until then, curiosity toward the unusual
		// does the finding). The winning cycle itself yields a big preference
		// jump -- exactly the reward that teaches which action mattered.
		frameVec := pipeline.ObservationVector(perception.DescribeGridStructural(frame.Grid, *maxBlobs, *cols, *rows))
		if frame.LevelsCompleted > prevLevelsCompleted {
			engine.RememberGoalState(frameVec)
		}
		// Composer + self-preservation: the spatial approach term makes preference
		// RISE as a body nears the salient target (so BestAction learns to navigate
		// toward the key before any level is completed), while the danger term (via
		// preferenceOf) LOWERS it near death-like states, so the same credit
		// machinery also steers away from what killed the agent.
		newPreference := preferenceOf(frame.Grid)
		preferenceDelta := newPreference - prevPreference
		preferenceIncreased := newPreference > prevPreference
		if preferenceIncreased && !exploredAction && clickedLabel != "" {
			memory.Record(clickedLabel, true)
		}
		// Plan: credit this action's realized preference change so BestAction
		// can steer toward the goal next time (the pragmatic half of expected
		// free energy). Attributed to the action just taken.
		engine.AttributePreferenceGain(preferenceDelta)
		fmt.Printf("action %d: preference=%.4f delta=%+.4f (learned_goal=%v)%s\n",
			actionsTaken, newPreference, newPreference-prevPreference, engine.HasLearnedGoal(),
			map[bool]string{true: " <- toward goal, reinforced", false: ""}[preferenceIncreased && !exploredAction && clickedLabel != ""])
		prevPreference = newPreference

		if exploredAction {
			fmt.Printf("action %d: sent %v -> state=%s levels_completed=%d\n",
				actionsTaken, action.ID, frame.State, frame.LevelsCompleted)
		} else {
			fmt.Printf("action %d: click (%d,%d) -> state=%s levels_completed=%d (cohesion this cycle=%.4f)\n",
				actionsTaken, action.X, action.Y, frame.State, frame.LevelsCompleted, res.MaxCohesionObserved)
		}

		if frame.State == environment.StateWin || frame.State == environment.StateGameOver {
			// Register the outcome of the action that just caused this
			// terminal state before stopping. Real bug this fixes: the loop
			// used to check for terminal state at the TOP, before ever
			// calling ChooseClickAction again -- so it broke on the NEXT
			// iteration without the credit-assignment machinery ever seeing
			// the final frame or judging the action that caused WIN/GAME_OVER.
			// The action this call proposes is discarded (the game is over,
			// nothing left to click); only the actualSuccess judgment it
			// computes internally -- whether the terminal frame's perception
			// differs from the pre-terminal one, OR whether this final action
			// completed a level (frame.LevelsCompleted was just reassigned
			// from newFrame above; prevLevelsCompleted still holds the
			// pre-this-action value) -- matters here.
			terminalLevelsCompletedIncreased := frame.LevelsCompleted > prevLevelsCompleted
			if _, finalObs, _, _, regErr := bridge.ChooseClickAction(ctx, engine, frame.Grid, "solve the puzzle", *maxBlobs, *cols, *rows, prevObservation, terminalLevelsCompletedIncreased, *curiosityStep, rand.Float64(), memory, prevClickedLabel, "", nil); regErr != nil {
				fmt.Printf("terminal state %s: outcome registration failed: %v\n", frame.State, regErr)
			} else {
				fmt.Printf("terminal state %s registered (changed since last frame: %v, levels_completed_increased: %v)\n", frame.State, finalObs != prevObservation, terminalLevelsCompletedIncreased)
			}

			// LEARN FROM THE LOSS, then RETRY: a GAME_OVER is a negative reward.
			// Penalize the action that ended the game and the recent click
			// sequence, then reset and keep playing with the SAME brain -- so
			// the loss actually teaches (die, learn what killed you, avoid it
			// next attempt), instead of throwing the episode away. WIN also
			// resets to keep going within the action budget.
			if frame.State == environment.StateGameOver {
				// Fix 1 -- separate a HAZARD death from a TIMEOUT/budget death. We
				// have no direct "collided with a deadly object" detector, so use
				// the sign of the fatal step's preference delta as the proxy: a
				// NEGATIVE delta means the action made things worse (hazard-like) --
				// penalize it and remember the danger state; a >= 0 delta means the
				// step was progress and the death was non-causal (ran out of the
				// level's step budget while improving, as vc33 did one click from
				// the win with delta=+0.0391) -- penalizing that would punish the
				// RIGHT move and, since click is the only tool here, collapse the
				// agent into negative-preference flailing. So only the harmful case
				// trains aversion.
				if preferenceDelta < 0 {
					engine.PenalizeLastAction(lossActionPenalty)
					memory.PenalizeSequence(lossSequenceStrength)
					engine.RememberDangerState(pipeline.ObservationVector(perception.DescribeGridStructural(preStepGrid, *maxBlobs, *cols, *rows)))
					fmt.Printf("GAME_OVER (hazard-like, delta=%+.4f): penalized fatal action %q + sequence, remembered danger, resetting\n", preferenceDelta, lastActionTok)
				} else {
					fmt.Printf("GAME_OVER (timeout-like, delta=%+.4f >= 0): fatal step was progress -- NOT penalized, resetting\n", preferenceDelta)
				}
			}
			resetFrame, resetErr := sess.Reset()
			if resetErr != nil {
				fmt.Printf("reset after %s failed: %v -- stopping\n", frame.State, resetErr)
				break
			}
			frame = resetFrame
			prevObservation, prevClickedLabel, lastActionTok, prevNumeric = "", "", "", ""
			tracker = perception.NewObjectTracker()
			prevTopology = ""
			prevLevelsCompleted = frame.LevelsCompleted
			prevPreference = preferenceOf(frame.Grid)
			stagnation = 0
			continue
		}
	}

	fmt.Printf("stopped after %d actions, final state=%s levels_completed=%d\n", actionsTaken, frame.State, frame.LevelsCompleted)

	if summary, err := client.CloseScorecard(ctx, cardID); err != nil {
		log.Printf("close scorecard: %v (scorecard %s left open on the server)", err, cardID)
	} else {
		fmt.Printf("scorecard closed: %+v\n", summary)
	}
}

func hasAction6(actions []environment.ActionID) bool {
	for _, a := range actions {
		if a == environment.Action6 {
			return true
		}
	}
	return false
}

// availableSimpleActions returns the offered "simple" actions (ACTION1-5, no
// X/Y) -- the controls the click-only agent never tries. Click (ACTION6) and
// undo (ACTION7) are deliberately excluded: 6 is what ChooseClickAction
// already handles, and randomly firing undo would mostly just reverse
// progress rather than explore a control.
// simpleActionGainString formats a per-action gain map (preference or
// competence) for the available simple actions in the live diagnostic, so a
// run shows the two signals the planner chooses on.
func simpleActionGainString(gains map[string]float64, simpleByToken map[string]environment.ActionID) string {
	out := ""
	for tok := range simpleByToken {
		out += fmt.Sprintf(" %s=%+.4f", tok, gains[tok])
	}
	return out
}

// lossActionPenalty nudges the fatal action's pragmatic value down on a
// GAME_OVER (moderate -- comparable to the other selection terms, so the
// planner avoids it without permanently killing a usually-useful control).
// lossSequenceStrength is how hard the loss drops the recent click sequence.
const (
	lossActionPenalty    = 0.05
	lossSequenceStrength = 3.0
	// dangerAvoidWeight scales how strongly resemblance to a death state lowers a
	// state's preference (DangerProximity is a cosine in [0,1]). Set ABOVE
	// spatialApproachWeight (0.15) so self-preservation overrides the pull toward
	// a hazardous key -- don't rush into death to reach the goal -- but well below
	// a learned goal's full weight so it deters without freezing the agent into
	// the dark-room "never act" corner (the anti-stagnation drive is the other
	// counterweight). A guess, tunable against a live run.
	dangerAvoidWeight = 0.3
	// refractorySteps is how many steps a target is locked out after a click
	// there changed nothing meaningful (Inhibition of Return). ~4 is the middle
	// of the "3-5 moves" the vc33 post-mortem called for: long enough to break a
	// hammering loop, short enough that a target worth revisiting comes back.
	refractorySteps = 4
	// graphPriorBonus nudges per-object selection toward the object the
	// reconnected graph's WTA chose. Comparable to the optimism bonus (max 0.05)
	// so at cold start (per-object gains ~0) the graph breaks the tie, but small
	// enough that a real learned per-object preference/competence signal overrides
	// it once accrued. A guess, tunable against a live run.
	graphPriorBonus = 0.03
)

// availableSimpleActions returns every offered non-click action (ACTION1-5 and
// ACTION7) -- the keyboard controls the click-only agent never tries. Only
// ACTION6 (the click, which ChooseClickAction handles) and ACTIONRESET are
// excluded. ACTION7 is INCLUDED, not assumed to be useless undo: in a novel
// game we don't know what a control does until we try it.
func availableSimpleActions(actions []environment.ActionID) []environment.ActionID {
	simple := make([]environment.ActionID, 0, len(actions))
	for _, a := range actions {
		if a != environment.Action6 && a != environment.ActionReset {
			simple = append(simple, a)
		}
	}
	return simple
}
