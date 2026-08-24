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
	"sort"

	"alphaarc/pkg/actuate"
	"alphaarc/pkg/environment"
	"alphaarc/pkg/environment/perception"
	"alphaarc/pkg/environment/remote"
	"alphaarc/pkg/feataff"
	"alphaarc/pkg/goalsel"
)

// liveEnv adapts a remote game session to feataff.Env.
type liveEnv struct {
	sess *remote.Session
	prev int
}

func (e *liveEnv) Reset() actuate.Grid {
	f, err := e.sess.Reset()
	if err != nil {
		fmt.Println("reset:", err)
		return actuate.Grid{{0}}
	}
	e.prev = f.LevelsCompleted
	return f.Grid
}

func (e *liveEnv) Step(c actuate.Control) (actuate.Grid, bool) {
	f, err := e.sess.Step(environment.Action{ID: environment.Action6, X: c.X, Y: c.Y})
	if err != nil {
		fmt.Println("step:", err)
		return actuate.Grid{{0}}, false
	}
	reward := f.LevelsCompleted > e.prev
	e.prev = f.LevelsCompleted
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

	// (1) live feature readouts on the real grid.
	feats := feataff.DefaultFeatures()
	fmt.Print("live features: ")
	for _, f := range feats {
		fmt.Printf("%s=%.0f  ", f.Name, f.Eval(grid))
	}
	fmt.Println()

	// (2) real candidate controls from perception (residual).
	controls := feataff.ResidualControls(grid, 12)
	fmt.Printf("residual controls: %d\n", len(controls))
	if len(controls) == 0 {
		fmt.Println("no candidate controls -- nothing to explore")
		return
	}

	// Run the feature-affordance exploration on the live game.
	fm := feataff.New(feats)
	fm.Explore(env, controls)

	// Report the actual per-feature effect of each real control.
	rewards := 0
	fmt.Println("control -> per-feature deltas (reward):")
	for _, r := range fm.Records() {
		if r.Reward {
			rewards++
		}
		names := make([]string, 0, len(r.Deltas))
		for n := range r.Deltas {
			names = append(names, n)
		}
		sort.Strings(names)
		fmt.Printf("  click(%2d,%2d):", r.Control.X, r.Control.Y)
		for _, n := range names {
			if d := r.Deltas[n]; d != 0 {
				fmt.Printf(" %s%+.0f", n, d)
			}
		}
		if r.Reward {
			fmt.Print("  [REWARD]")
		}
		fmt.Println()
	}
	fmt.Printf("explored %d controls, %d produced a reward (level_up)\n", len(fm.Records()), rewards)

	// ===== Piece (3): PURSUIT =====
	// Bootstrap the provisional goal from the intrinsic drive (compression), then
	// repeatedly actuate the control that most increases it -- from the live
	// (post-reset) state, letting the effect accumulate toward the sparse reward.
	// Feed every step to goalsel; on a reward, confirm/select the goal causally.
	names := make([]string, 0, len(feats))
	for _, f := range feats {
		names = append(names, f.Name)
	}
	sel := goalsel.New(names, 5, 5.0)
	goalFeature := "compression" // provisional intrinsic goal (grow compressibility)

	env.Reset()
	won := false
	const budget = 60
	step := 0
	for ; step < budget; step++ {
		c, gain, ok := fm.BestControlFor(goalFeature, +1)
		if !ok || gain <= 0 {
			fmt.Printf("pursuit: no positive-gain control for %s -- stop at step %d\n", goalFeature, step)
			break
		}
		grid, reward := env.Step(c)
		vals := make(map[string]float64, len(feats))
		for _, f := range feats {
			vals[f.Name] = f.Eval(grid)
		}
		sel.Observe(vals, reward)
		if reward {
			won = true
			fmt.Printf("*** LEVEL_UP at pursuit step %d via click(%d,%d) pursuing %q ***\n", step, c.X, c.Y, goalFeature)
			break
		}
	}

	if won {
		// Confirm the goal causally: the reward credited several coupled features;
		// disambiguate via the feature mapper (interventions) -- inseparable ones
		// merge, honestly.
		sel.Disambiguate(fm)
		if f, d, ok := sel.Goal(); ok {
			fmt.Printf("goalsel confirmed goal: %s dir=%+d  merged=%v\n", f, d, sel.Merged())
		}
		fmt.Println("RESULT: vc33 L1 solved through the Path-B stack (features -> pursuit -> reward -> causal goal).")
	} else {
		fmt.Printf("no level_up within %d pursuit steps\n", budget)
	}
}
