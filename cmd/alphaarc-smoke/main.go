// Command alphaarc-arc-smoke is a minimal, safe connectivity check against
// the real ARC-AGI-3 service: list games, open a scorecard, reset the
// first (or a chosen) game, print what a real frame actually looks like
// through pkg/environment/perception, then close the scorecard.
//
// It deliberately never calls Step() -- choosing a real action for an
// arbitrary game is a separate, still-open problem (see ARCHITECTURE.md
// Stage 5). This only checks that auth, the endpoint wire format, and
// perception's grid parsing all hold up against the real server, not just
// against this repo's own httptest mocks.
//
// Requires ARC_API_KEY in the environment (see .env.example). Run:
//
//	go run ./cmd/alphaarc-arc-smoke
//	go run ./cmd/alphaarc-arc-smoke -game <game_id>
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"alphaarc/pkg/environment/perception"
	"alphaarc/pkg/environment/remote"
)

func main() {
	gameID := flag.String("game", "", "game_id to reset (default: first game from /api/games)")
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
	fmt.Printf("GET /api/games -> %d games\n", len(games))
	for i, g := range games {
		if i >= 5 {
			fmt.Printf("  ... and %d more\n", len(games)-5)
			break
		}
		fmt.Printf("  %s\n", g.GameID)
	}
	if len(games) == 0 {
		log.Fatalf("no games returned -- nothing to reset")
	}

	target := *gameID
	if target == "" {
		target = games[0].GameID
	}

	cardID, err := client.OpenScorecard(ctx, []string{"alphaarc-smoke-test"})
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
	fmt.Printf("RESET %s -> state=%s levels_completed=%d available_actions=%v\n",
		target, frame.State, frame.LevelsCompleted, frame.AvailableActions)

	fmt.Printf("grid: %d rows", len(frame.Grid))
	if len(frame.Grid) > 0 {
		fmt.Printf(" x %d cols", len(frame.Grid[0]))
	}
	fmt.Println()
	fmt.Printf("perception.DescribeGrid -> %q\n", perception.DescribeGrid(frame.Grid, 5))

	if summary, err := client.CloseScorecard(ctx, cardID); err != nil {
		log.Printf("close scorecard: %v (scorecard %s left open on the server)", err, cardID)
	} else {
		fmt.Printf("scorecard closed: %+v\n", summary)
	}
}
