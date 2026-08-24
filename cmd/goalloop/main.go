// Command goalloop is a live diagnostic for the causal goal-discovery loop's
// first two live pieces: (1) real grid-derived feature readouts and (2) real
// candidate controls from perception. It connects to a real game (free API),
// computes the feature library on the live grid, generates residual controls,
// and runs FeatureMapper.Explore -- printing the actual per-feature deltas each
// control caused and whether the sparse reward fired.
//
//	GAME=vc33-5430563c go run ./cmd/goalloop   (needs ARC_API_KEY)
//
// It does NOT yet close the full loop live: the sparse reward (level_up) rarely
// fires under bounded exploration, so goal SELECTION needs the exploration
// engine (piece 3) before it can act on a live game. This proves (1)+(2) run on
// real data.
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"alphaarc/pkg/actuate"
	"alphaarc/pkg/environment"
	"alphaarc/pkg/environment/perception"
	"alphaarc/pkg/environment/remote"
	"alphaarc/pkg/feataff"
	"alphaarc/pkg/goalsel"
)

// liveEnv adapts a remote game session to feataff.Env. It tracks a `dead` flag:
// once the session errors (expired / GAME_NOT_STARTED after budget exhaustion),
// further Step/Reset stop hitting the API and return the last good grid with no
// reward, so a running pursuit ends cleanly on "nothing changes" instead of
// spewing 60 identical API errors and recording garbage 1x1 grids.
type liveEnv struct {
	sess *remote.Session
	prev int
	last actuate.Grid
	dead bool
	errN int
}

func (e *liveEnv) lastOr1x1() actuate.Grid {
	if e.last == nil {
		return actuate.Grid{{0}}
	}
	return e.last
}

func (e *liveEnv) Reset() actuate.Grid {
	if e.dead {
		return e.lastOr1x1()
	}
	f, err := e.sess.Reset()
	if err != nil {
		fmt.Println("reset:", err)
		e.dead = true
		return e.lastOr1x1()
	}
	e.prev = f.LevelsCompleted
	e.last = f.Grid
	return f.Grid
}

func (e *liveEnv) Step(c actuate.Control) (actuate.Grid, bool) {
	if e.dead {
		return e.lastOr1x1(), false
	}
	f, err := e.sess.Step(environment.Action{ID: environment.Action6, X: c.X, Y: c.Y})
	if err != nil {
		e.errN++
		if e.errN <= 1 {
			fmt.Println("step:", err, "(marking session dead; suppressing further errors)")
		}
		e.dead = true
		return e.lastOr1x1(), false
	}
	reward := f.LevelsCompleted > e.prev
	e.prev = f.LevelsCompleted
	e.last = f.Grid
	return f.Grid, reward
}

func main() {
	ctx := context.Background()
	game := os.Getenv("GAME")
	if game == "" {
		game = "vc33-5430563c"
	}
	client, err := remote.NewClientFromEnv()
	if err != nil {
		fmt.Println("client:", err)
		os.Exit(1)
	}
	card, err := client.OpenScorecard(ctx, []string{"alphaarc-goalloop"})
	if err != nil {
		fmt.Println("scorecard:", err)
		os.Exit(1)
	}
	defer client.CloseScorecard(ctx, card)

	env := &liveEnv{sess: remote.NewSession(client, game, card)}
	grid := env.Reset()
	bg := perception.BackgroundColor(grid)
	fmt.Printf("=== %s  %dx%d bg=%d ===\n", game, len(grid), len(grid[0]), bg)

	// SINGLE-EPISODE, BUDGETED driver. The live wall isn't the mechanism, it's
	// exploration economy: budget-capped games (ft09 ~150 actions, NOT refreshed by
	// RESET) die if the driver resets per candidate. So we play ONE continuous game
	// from ONE reset, feeding every real step to the affordance model, and never
	// reset again. Feature growth is FREE (pure functions of the grid), so the full
	// family library -- fixed primitives + grown families (discovered-transform,
	// color-perm-symmetry, periodic-color-perm) -- is present from the start.
	feats := append(feataff.DefaultFeatures(), feataff.GrowFeatures(grid)...)
	names := make([]string, 0, len(feats))
	fmt.Print("live features: ")
	for _, f := range feats {
		fmt.Printf("%s=%.0f  ", f.Name, f.Eval(grid))
		names = append(names, f.Name)
	}
	fmt.Println()

	budget := 100
	if b := os.Getenv("BUDGET"); b != "" {
		if v, err := strconv.Atoi(b); err == nil && v > 0 {
			budget = v
		}
	}

	fm := feataff.New(feats)
	sel := goalsel.New(names, 5, 5.0)
	sel.Observe(featureVals(feats, grid), false) // baseline for first-step attribution

	cur := grid
	actions := 0
	won, via, steps := false, "", 0
	// play applies one control from the CURRENT state (no reset), records it, and
	// feeds goalsel. Returns whether the sparse reward fired. If the session has
	// died (budget exhausted server-side), it is a no-op and does NOT advance the
	// action counter -- so `actions` reflects REAL API steps and reveals the game's
	// true live budget.
	play := func(c actuate.Control) bool {
		if env.dead {
			return false
		}
		before := cur
		after, reward := env.Step(c)
		if env.dead {
			return false // the step itself failed; don't count it
		}
		actions++
		fm.ObserveStep(before, after, c, reward)
		sel.Observe(featureVals(feats, after), reward)
		cur = after
		return reward
	}

	// Phase A -- probe salient (residual + object) controls sequentially.
	salient := feataff.ResidualControls(grid, 12)
	fmt.Printf("phase A: probing %d salient controls (budget %d)\n", len(salient), budget)
	for _, c := range salient {
		if actions >= budget || env.dead {
			break
		}
		if play(c) {
			won, via, steps = true, "salient-probe", actions
			break
		}
	}

	// Phase B -- generalized pursuit: chase whichever feature has the strongest
	// actuatable gain seen so far, applying it from the CURRENT state. A control
	// that stops paying off (path-dependent effects in a shared episode) is blocked
	// so we don't loop on it.
	if !won {
		blocked := map[actuate.Control]bool{}
		// pick scans everything observed for the max positive per-feature gain among
		// controls not yet blocked.
		pick := func() (string, actuate.Control, float64) {
			best, bg := "", 0.0
			var bc actuate.Control
			for _, r := range fm.Records() {
				if blocked[r.Control] {
					continue
				}
				for name, d := range r.Deltas {
					if d > bg {
						bg, best, bc = d, name, r.Control
					}
				}
			}
			return best, bc, bg
		}
		for actions < budget && !env.dead {
			feat, ctrl, gain := pick()
			if feat == "" || gain <= 0 {
				break // nothing pursuable -> fall through to the sweep
			}
			if play(ctrl) {
				won, via, steps = true, feat, actions
				break
			}
			recs := fm.Records()
			if last := recs[len(recs)-1]; last.Deltas[feat] <= 0 {
				blocked[ctrl] = true // re-applying it no longer helps this feature
			}
		}
	}

	// Phase C -- if still stuck and budget remains, a budgeted coarse sweep in the
	// SAME episode: it both discovers real actuators and can trip the reward.
	if !won && actions < budget {
		remaining := budget - actions
		step := sweepStep(len(grid[0]), len(grid), remaining)
		sweep := feataff.SweepControls(grid, step)
		fmt.Printf("phase C: stuck -- budgeted sweep step=%d (%d controls, %d actions left)\n", step, len(sweep), remaining)
		for _, c := range sweep {
			if actions >= budget || env.dead {
				break
			}
			if play(c) {
				won, via, steps = true, "sweep-trigger", actions
				break
			}
		}
	}

	if env.dead {
		fmt.Printf("session ended (server budget) after %d real actions\n", actions)
	}
	fmt.Printf("used %d/%d actions\n", actions, budget)
	if won {
		fmt.Printf("*** LEVEL_UP at action %d, via %q ***\n", steps, via)
		sel.Disambiguate(fm)
		if f, d, ok := sel.Goal(); ok {
			fmt.Printf("goalsel confirmed goal: %s dir=%+d  merged=%v\n", f, d, sel.Merged())
		}
		fmt.Printf("RESULT: %s L1 solved through the single-episode Path-B stack.\n", game)
	} else {
		fmt.Printf("no level_up within the action budget (single episode)\n")
	}
}

// featureVals reads every feature on a grid into a name->value map.
func featureVals(feats []feataff.Feature, g actuate.Grid) map[string]float64 {
	v := make(map[string]float64, len(feats))
	for _, f := range feats {
		v[f.Name] = f.Eval(g)
	}
	return v
}

// sweepStep picks a sweep spacing so the coarse grid fits within `budget` clicks.
func sweepStep(w, h, budget int) int {
	if budget < 1 {
		budget = 1
	}
	s := 1
	for (w/s+1)*(h/s+1) > budget {
		s++
	}
	return s
}
