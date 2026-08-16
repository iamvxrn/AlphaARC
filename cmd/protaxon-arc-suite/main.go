// Command protaxon-arc-suite probes MANY ARC-AGI-3 games with the full agent
// for a short budget each, and prints an outcome matrix:
//
//	SOLVED      -- a level was completed (levels_completed > 0)
//	INTERACTIVE -- the world responded structurally (object topology changed)
//	DEAD-END    -- nothing the agent did changed which objects exist
//
// The point (a strategic pivot away from grinding one game): find which games
// the current stack can even get traction on, instead of a mute 0 on a single
// possibly-unwinnable puzzle. Uses the same core the single-game player does --
// ChooseClickAction, the object tracker, expected-free-energy action choice --
// just quiet and repeated. Real API calls: ~ suite * actions; keep both modest.
//
//	go run ./cmd/protaxon-arc-suite -suite 12 -actions 60
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"protaxon/pkg/environment"
	"protaxon/pkg/environment/bridge"
	"protaxon/pkg/environment/perception"
	"protaxon/pkg/environment/remote"
	"protaxon/pkg/pipeline"
)

func main() {
	suite := flag.Int("suite", 12, "how many different games to probe (the first N) when -games is empty")
	gamesFilter := flag.String("games", "", "comma-separated game-id prefixes to probe INSTEAD of the first -suite (e.g. vc33,sp80,bp35) -- targeted, cheaper than a blind sweep")
	maxActions := flag.Int("actions", 60, "actions per game")
	maxBlobs := flag.Int("maxblobs", 300, "max blobs perceived per frame")
	cols := flag.Int("cols", 16, "grid-cell lattice columns")
	rows := flag.Int("rows", 16, "grid-cell lattice rows")
	exploreActions := flag.Float64("explore-actions", 0.25, "probability of a random action-type instead of the planned one")
	flag.Parse()

	ctx := context.Background()
	client, err := remote.NewClientFromEnv()
	if err != nil {
		log.Fatalf("client: %v", err)
	}
	games, err := client.ListGames(ctx)
	if err != nil {
		log.Fatalf("list games: %v", err)
	}
	if len(games) == 0 {
		log.Fatalf("no games returned")
	}
	// Pick which games to probe: a targeted -games list (prefix match, so short
	// ids like "vc33" work) when given, else the first -suite games.
	var selected []string
	if *gamesFilter != "" {
		wanted := strings.Split(*gamesFilter, ",")
		for _, g := range games {
			for _, w := range wanted {
				if w = strings.TrimSpace(w); w != "" && strings.HasPrefix(g.GameID, w) {
					selected = append(selected, g.GameID)
					break
				}
			}
		}
		if len(selected) == 0 {
			log.Fatalf("no games matched -games=%q (have %d games)", *gamesFilter, len(games))
		}
	} else {
		n := *suite
		if n > len(games) {
			n = len(games)
		}
		for i := 0; i < n; i++ {
			selected = append(selected, games[i].GameID)
		}
	}
	cardID, err := client.OpenScorecard(ctx, []string{"protaxon-arc-suite"})
	if err != nil {
		log.Fatalf("open scorecard: %v", err)
	}
	fmt.Printf("probing %d games, %d actions each (scorecard %s)\n\n", len(selected), *maxActions, cardID)

	type row struct {
		game                            string
		verdict                         string
		bestLevels, changed, topo, gOvr int
		actions                         int
	}
	var rows_ []row
	counts := map[string]int{}
	for _, g := range selected {
		o := probeGame(ctx, client, cardID, g, *maxActions, *maxBlobs, *cols, *rows, *exploreActions)
		verdict := classify(o)
		counts[verdict]++
		rows_ = append(rows_, row{g, verdict, o.bestLevels, o.changed, o.topo, o.gameOvers, o.actions})
		fmt.Printf("  %-14s %-11s  levels=%d changed=%d/%d topo_changes=%d game_overs=%d\n",
			g, verdict, o.bestLevels, o.changed, o.actions, o.topo, o.gameOvers)
	}

	fmt.Printf("\n=== MATRIX ===\n")
	fmt.Printf("SOLVED=%d  INTERACTIVE=%d  DEAD-END=%d  UNPLAYABLE=%d  (of %d)\n", counts["SOLVED"], counts["INTERACTIVE"], counts["DEAD-END"], counts["UNPLAYABLE"], len(selected))
	if summary, err := client.CloseScorecard(ctx, cardID); err != nil {
		log.Printf("close scorecard: %v", err)
	} else {
		fmt.Printf("scorecard closed: %+v\n", summary)
	}
}

type outcome struct {
	bestLevels, changed, topo, gameOvers, actions int
}

// dangerAvoidWeight scales how strongly resemblance to a death state lowers a
// state's preference -- the self-preservation instinct. Above spatialApproach
// (0.15) so it overrides rushing into a hazardous key, below a learned goal's
// full weight so it deters without freezing the agent (dark-room trap). Kept in
// sync with the live player's constant of the same name.
const dangerAvoidWeight = 0.3

// preferenceOf is the prior preference over a state, identical to the live
// player's: learned goal + weak structural prior + spatial approach to the key,
// MINUS the self-preservation term (resemblance to a state that got the agent
// killed). Single source of truth so init, loop, and post-reset can't drift.
func preferenceOf(engine *pipeline.Engine, grid [][]int, maxBlobs, cols, rows int) float64 {
	vec := pipeline.ObservationVector(perception.DescribeGridStructural(grid, maxBlobs, cols, rows))
	return engine.LearnedPreference(vec) + 0.1*perception.StructureScore(grid) + perception.ApproachPreference(grid) - dangerAvoidWeight*engine.DangerProximity(vec)
}

func classify(o outcome) string {
	if o.actions == 0 {
		return "UNPLAYABLE" // never offered ACTION6 -- the agent couldn't act at all
	}
	if o.bestLevels > 0 {
		return "SOLVED"
	}
	// The world responded structurally (a body appeared/vanished) or changed on
	// a meaningful fraction of actions -- there's something to work with.
	if o.topo > 0 || o.changed*3 >= o.actions {
		return "INTERACTIVE"
	}
	return "DEAD-END"
}

func probeGame(ctx context.Context, client *remote.Client, cardID, gameID string, maxActions, maxBlobs, cols, rows int, exploreActions float64) outcome {
	sess := remote.NewSession(client, gameID, cardID)
	frame, err := sess.Reset()
	if err != nil {
		return outcome{}
	}
	engine := pipeline.NewEngine()
	memory := bridge.NewOutcomeMemory()
	tracker := perception.NewObjectTracker()
	var o outcome
	prevObs, prevClicked, prevTopology := "", "", ""
	prevLevels := frame.LevelsCompleted
	prevPref := preferenceOf(engine, frame.Grid, maxBlobs, cols, rows)

	for o.actions < maxActions {
		if !hasAction6(frame.AvailableActions) {
			break
		}
		levelsInc := frame.LevelsCompleted > prevLevels
		if frame.LevelsCompleted > o.bestLevels {
			o.bestLevels = frame.LevelsCompleted
		}
		motion := strings.Join(append(tracker.Track(frame.Grid), perception.NumericTokens(frame.Grid)...), " ")
		objCandidates := tracker.LabeledObjects() // Fix 3: click candidates in the graph's obj-id vocabulary
		if topo := tracker.TopologySignature(); topo != prevTopology {
			if prevTopology != "" {
				o.topo++
			}
			prevTopology = topo
		}

		candidates := []string{"click"}
		simpleByTok := map[string]environment.ActionID{}
		for _, a := range simpleActions(frame.AvailableActions) {
			t := fmt.Sprintf("act-%d", a)
			candidates = append(candidates, t)
			simpleByTok[t] = a
		}
		tok := engine.BestAction(candidates)
		if rand.Float64() < exploreActions || tok == "" {
			tok = candidates[rand.Intn(len(candidates))]
		}
		engine.ConditionForecastOnAction(tok)

		action, obs, clicked, _, cerr := bridge.ChooseClickAction(ctx, engine, frame.Grid, "solve the puzzle", maxBlobs, cols, rows, prevObs, levelsInc, 0.1, rand.Float64(), memory, prevClicked, motion, objCandidates)
		if cerr != nil {
			break
		}
		if id, ok := simpleByTok[tok]; ok {
			action = environment.Action{ID: id}
			clicked = ""
		}
		if prevObs != "" && obs != prevObs {
			o.changed++
		}
		prevObs, prevClicked, prevLevels = obs, clicked, frame.LevelsCompleted

		preStepGrid := frame.Grid // state the action is taken from, for danger memory
		newFrame, serr := sess.Step(action)
		o.actions++
		if serr != nil {
			break
		}
		frame = newFrame
		if frame.LevelsCompleted > o.bestLevels {
			o.bestLevels = frame.LevelsCompleted
		}
		// Composer: credit the spatial approach (+ any learned goal) so BestAction
		// steers toward the salient target -- the same drive the live player uses.
		frameVec := pipeline.ObservationVector(perception.DescribeGridStructural(frame.Grid, maxBlobs, cols, rows))
		if frame.LevelsCompleted > prevLevels {
			engine.RememberGoalState(frameVec)
		}
		newPref := preferenceOf(engine, frame.Grid, maxBlobs, cols, rows)
		engine.AttributePreferenceGain(newPref - prevPref)
		prevPref = newPref
		if frame.State == environment.StateWin || frame.State == environment.StateGameOver {
			if frame.State == environment.StateGameOver {
				o.gameOvers++
				engine.PenalizeLastAction(0.05)
				memory.PenalizeSequence(3.0)
				// Self-preservation: learn the "about to die" state (mirror of the
				// goal machinery), so DangerProximity steers the planner away from it.
				engine.RememberDangerState(pipeline.ObservationVector(perception.DescribeGridStructural(preStepGrid, maxBlobs, cols, rows)))
			}
			rf, rerr := sess.Reset()
			if rerr != nil {
				break
			}
			frame = rf
			tracker = perception.NewObjectTracker()
			prevObs, prevClicked, prevTopology = "", "", ""
			prevLevels = frame.LevelsCompleted
			prevPref = preferenceOf(engine, frame.Grid, maxBlobs, cols, rows)
		}
	}
	return o
}

func hasAction6(actions []environment.ActionID) bool {
	for _, a := range actions {
		if a == environment.Action6 {
			return true
		}
	}
	return false
}

func simpleActions(actions []environment.ActionID) []environment.ActionID {
	var s []environment.ActionID
	for _, a := range actions {
		if a != environment.Action6 && a != environment.ActionReset {
			s = append(s, a)
		}
	}
	return s
}
