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
	"math/rand"

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
	actionsTaken := 0
	prevObservation := ""
	prevClickedLabel := ""
	prevLevelsCompleted := frame.LevelsCompleted

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

		labeled := perception.RankedLabeledBlobs(frame.Grid, *maxBlobs, *cols, *rows)
		labels := make([]string, len(labeled))
		for i, lb := range labeled {
			labels[i] = lb.Label
		}
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

		action, observation, clickedLabel, res, err := bridge.ChooseClickAction(ctx, engine, frame.Grid, "solve the puzzle", *maxBlobs, *cols, *rows, prevObservation, levelsCompletedIncreased, *curiosityStep, rand.Float64(), memory, prevClickedLabel)
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
		fmt.Printf("action %d: perceives %q (changed since last frame: %v, levels_completed_increased: %v, curiosity=%.4f)\n",
			actionsTaken+1, observation, changedSinceLastFrame, levelsCompletedIncreased, engine.Homeostasis.Curiosity)
		// Diagnostic: real internal graph state for THIS cycle's candidate
		// categories -- cluster assignment, activation, and edges between
		// them -- so diversification in the click choice can be attributed
		// to an actual mechanism (Hebbian edge weight vs. Louvain
		// re-clustering) instead of guessed from external behavior alone.
		fmt.Printf("action %d: graph state: %s\n", actionsTaken+1, bridge.DescribeCategoryGraphState(engine.Graph, labels))
		// Diagnostic: what OutcomeMemory has accumulated for the category
		// about to be clicked -- distinguishes "picked because it's proven"
		// from "picked because WTA/exploration said so with zero evidence."
		if rate, attempts := memory.SuccessRate(clickedLabel); attempts > 0 {
			fmt.Printf("action %d: clicking %q, prior record: %d/%d successful\n", actionsTaken+1, clickedLabel, int(rate*float64(attempts)+0.5), attempts)
		} else {
			fmt.Printf("action %d: clicking %q, no prior record\n", actionsTaken+1, clickedLabel)
		}
		prevObservation = observation
		prevClickedLabel = clickedLabel
		prevLevelsCompleted = frame.LevelsCompleted

		newFrame, stepErr := sess.Step(action)
		actionsTaken++
		if stepErr != nil {
			fmt.Printf("action %d: step failed: %v\n", actionsTaken, stepErr)
			break
		}
		frame = newFrame
		fmt.Printf("action %d: click (%d,%d) -> state=%s levels_completed=%d (cohesion this cycle=%.4f)\n",
			actionsTaken, action.X, action.Y, frame.State, frame.LevelsCompleted, res.MaxCohesionObserved)

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
			if _, finalObs, _, _, regErr := bridge.ChooseClickAction(ctx, engine, frame.Grid, "solve the puzzle", *maxBlobs, *cols, *rows, prevObservation, terminalLevelsCompletedIncreased, *curiosityStep, rand.Float64(), memory, prevClickedLabel); regErr != nil {
				fmt.Printf("terminal state %s: outcome registration failed: %v\n", frame.State, regErr)
			} else {
				fmt.Printf("terminal state %s registered (changed since last frame: %v, levels_completed_increased: %v)\n", frame.State, finalObs != prevObservation, terminalLevelsCompletedIncreased)
			}
			break
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
