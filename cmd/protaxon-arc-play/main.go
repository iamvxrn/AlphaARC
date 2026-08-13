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

	"protaxon/pkg/environment"
	"protaxon/pkg/environment/bridge"
	"protaxon/pkg/environment/remote"
	"protaxon/pkg/pipeline"
)

func main() {
	gameID := flag.String("game", "", "game_id to play (default: first game from /api/games)")
	maxActions := flag.Int("actions", 20, "hard cap on real Step() calls before giving up")
	maxBlobs := flag.Int("maxblobs", 5, "max blobs perceived per frame")
	cols := flag.Int("cols", 16, "grid-cell lattice columns for ChooseClickAction")
	rows := flag.Int("rows", 16, "grid-cell lattice rows for ChooseClickAction")
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
	actionsTaken := 0

	for actionsTaken < *maxActions {
		if frame.State == environment.StateWin || frame.State == environment.StateGameOver {
			break
		}
		if !hasAction6(frame.AvailableActions) {
			fmt.Printf("game does not offer ACTION6 (available: %v) -- ChooseClickAction only proposes clicks, stopping\n", frame.AvailableActions)
			break
		}

		action, res, err := bridge.ChooseClickAction(ctx, engine, frame.Grid, "solve the puzzle", *maxBlobs, *cols, *rows)
		if err != nil {
			fmt.Printf("action %d: choose action failed: %v\n", actionsTaken, err)
			break
		}

		newFrame, stepErr := sess.Step(action)
		actionsTaken++
		if stepErr != nil {
			fmt.Printf("action %d: step failed: %v\n", actionsTaken, stepErr)
			break
		}
		frame = newFrame
		fmt.Printf("action %d: click (%d,%d) -> state=%s levels_completed=%d (cohesion this cycle=%.4f)\n",
			actionsTaken, action.X, action.Y, frame.State, frame.LevelsCompleted, res.MaxCohesionObserved)
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
