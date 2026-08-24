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

	// ===== Piece (3): generalized PURSUIT =====
	// Chase whichever feature has the strongest actuatable gain (not fixed to any
	// one) until the sparse reward; feed goalsel each step; on reward confirm the
	// goal causally. `translate` only breaks ties as an exploration prior.
	names := make([]string, 0, len(feats))
	for _, f := range feats {
		names = append(names, f.Name)
	}
	sel := goalsel.New(names, 5, 5.0)
	won, via, steps := feataff.PursueToReward(env, feats, fm, sel, "translate", 60)

	// STUCK -> expand on TWO axes before retrying:
	//   (reachability) the perceptual control set may miss the real actuator (the
	//     trigger isn't salient: vc33's off-pattern button, ft09's inert block) --
	//     discover controls by CAUSATION (coarse sweep, keep clicks that change the
	//     board), and
	//   (feature growth, piece 4) the fixed library may lack the rewarded feature
	//     -- GROW new candidate families (discovered-transform, color-perm-symmetry).
	if !won && steps == 0 {
		grown := feataff.GrowFeatures(grid)
		feats = append(feats, grown...)
		names2 := make([]string, 0, len(feats))
		for _, f := range feats {
			names2 = append(names2, f.Name)
		}
		// Budget-fitting causal discovery: ONE sequential coarse sweep from a single
		// reset (1 Reset + N Steps), NOT a reset-per-candidate sweep -- the latter
		// exhausts a bounded live budget (ft09's ~150-action cap kills it mid-sweep).
		// The sweep both finds real actuators (a swept control that moves a grown
		// feature) and can catch the sparse reward mid-sweep if a control triggers it.
		fm = feataff.New(feats)
		sweep := feataff.SweepControls(grid, 8)
		fmt.Printf("stuck -- grew %d feature(s), budget-fitting sequential sweep of %d controls\n", len(grown), len(sweep))
		if rew, at := fm.ExploreSequential(env, sweep); rew {
			won, via, steps = true, "sweep-trigger", at
		} else {
			sel = goalsel.New(names2, 5, 5.0)
			won, via, steps = feataff.PursueToReward(env, feats, fm, sel, "translate", 40)
		}
	}

	if won {
		fmt.Printf("*** LEVEL_UP at pursuit step %d, pursued via %q ***\n", steps, via)
		sel.Disambiguate(fm)
		if f, d, ok := sel.Goal(); ok {
			fmt.Printf("goalsel confirmed goal: %s dir=%+d  merged=%v\n", f, d, sel.Merged())
		}
		fmt.Printf("RESULT: %s L1 solved through the Path-B stack (features -> pursuit -> reward -> causal goal).\n", game)
	} else {
		fmt.Printf("no level_up within budget (pursued %d steps)\n", steps)
	}
}
