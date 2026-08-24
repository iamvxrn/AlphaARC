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
	"strings"

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
	sess  *remote.Session
	prev  int
	last  actuate.Grid
	dead  bool
	errN  int
	avail []environment.ActionID // actions the game reports as available
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
	e.avail = f.AvailableActions
	return f.Grid
}

func (e *liveEnv) Step(c actuate.Control) (actuate.Grid, bool) {
	if e.dead {
		return e.lastOr1x1(), false
	}
	// A "click" control is Action6 (needs X/Y); an "action" control is a simple
	// non-coordinate action (Action1-5) the game exposes -- arrows, buttons, etc.
	act := environment.Action{ID: environment.Action6, X: c.X, Y: c.Y}
	if c.Kind == "action" {
		act = environment.Action{ID: environment.ActionID(c.ActionID)}
	}
	f, err := e.sess.Step(act)
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
	if len(f.AvailableActions) > 0 {
		e.avail = f.AvailableActions
	}
	return f.Grid, reward
}

// simpleActionControls turns the game's available non-click actions (Action1-5)
// into candidate controls. Reset/Undo and the click action are excluded (click is
// handled by the coordinate candidates). Games like su15/lf52/ls20 expose these
// and CANNOT be actuated by clicks alone.
func (e *liveEnv) simpleActionControls() []actuate.Control {
	var cs []actuate.Control
	for _, a := range e.avail {
		if a >= environment.Action1 && a <= environment.Action5 {
			cs = append(cs, actuate.Control{Kind: "action", ActionID: int(a)})
		}
	}
	return cs
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
	play := func(c actuate.Control) (reward, changed bool) {
		if env.dead {
			return false, false
		}
		before := cur
		after, r := env.Step(c)
		if env.dead {
			return false, false // the step itself failed; don't count it
		}
		actions++
		fm.ObserveStep(before, after, c, r)
		sel.Observe(featureVals(feats, after), r)
		changed = !gridsEqual(before, after)
		cur = after
		return r, changed
	}

	// Phase A -- probe the game's simple (non-click) actions FIRST, then salient
	// coordinate controls. Simple actions are few and are often the whole mechanic on
	// games that can't be actuated by clicks (su15/lf52/ls20).
	simpleActions := env.simpleActionControls()
	salient := append(simpleActions, feataff.ResidualControls(grid, 12)...)
	fmt.Printf("phase A: %d simple actions + salient = %d controls (budget %d)\n", len(simpleActions), len(salient), budget)
	for _, c := range salient {
		if actions >= budget || env.dead {
			break
		}
		if r, _ := play(c); r {
			won, via, steps = true, "salient-probe", actions
			break
		}
	}

	// Phase B -- single-step pursuit: press the strongest actuatable feature-gain
	// repeatedly (this is how vc33's grow button wins). A control that stops paying
	// off is blocked so we don't loop on it.
	if !won {
		blocked := map[actuate.Control]bool{}
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
				break
			}
			if r, _ := play(ctrl); r {
				won, via, steps = true, feat, actions
				break
			}
			recs := fm.Records()
			if last := recs[len(recs)-1]; last.Deltas[feat] <= 0 {
				blocked[ctrl] = true
			}
		}
	}

	// ===== Phase C: LIVE SEQUENCE ACTUATION (single-episode, forward pruning) =====
	// Single-step pursuit is exhausted -> the game is likely STATEFUL. One continuous
	// episode, no Reset (can't backtrack on a cumulative budget), so we search action
	// SEQUENCES greedily forward. Pruning by grid signature: an inert step leaves the
	// grid unchanged -> same signature -> never retried from that state (dead end
	// pruned); a state-changing step moves to a new signature -> actions become
	// available again, so arm->apply chains form naturally. At each state we EXPLOIT
	// the strongest recorded feature-gain if one is actuatable, else EXPLORE an untried
	// state-changer -- the "setup" moves (arm, pick-up) that unlock a later reward.
	if !won && actions < budget && !env.dead {
		fmt.Printf("phase C: single-step exhausted -- live sequence search\n")
	}
	changer := map[actuate.Control]bool{}            // changed the board at least once
	triedHere := map[string]map[actuate.Control]bool{} // per grid-signature: actions already tried
	viaOfLastStep := func() string {                 // feature that rose most on the last step
		recs := fm.Records()
		if len(recs) == 0 {
			return ""
		}
		best, bg := "", 0.0
		for n, d := range recs[len(recs)-1].Deltas {
			if d > bg {
				bg, best = d, n
			}
		}
		return best
	}
	sweepStepN := sweepStep(len(grid[0]), len(grid), budget)
	for actions < budget && !env.dead && !won {
		s := gridSig(cur)
		if triedHere[s] == nil {
			triedHere[s] = map[actuate.Control]bool{}
		}
		// candidate set from THIS state: salient controls + every known state-changer,
		// minus what we've already tried from this exact state.
		pool := env.simpleActionControls() // simple actions retried from every new state
		pool = append(pool, feataff.ResidualControls(cur, 12)...)
		pool = append(pool, feataff.SweepControls(cur, sweepStepN)...)
		for c := range changer {
			pool = append(pool, c)
		}
		// EXPLOIT: control with the largest recorded positive feature-gain, untried here.
		var chosen actuate.Control
		found := false
		bestGain := 0.0
		for _, r := range fm.Records() {
			if triedHere[s][r.Control] {
				continue
			}
			for _, d := range r.Deltas {
				if d > bestGain {
					bestGain, chosen, found = d, r.Control, true
				}
			}
		}
		// EXPLORE: else the first untried candidate, preferring known state-changers.
		if !found {
			for _, c := range pool {
				if !triedHere[s][c] && changer[c] {
					chosen, found = c, true
					break
				}
			}
		}
		if !found {
			for _, c := range pool {
				if !triedHere[s][c] {
					chosen, found = c, true
					break
				}
			}
		}
		if !found {
			fmt.Printf("dead end: every action from this state is exhausted/inert\n")
			break
		}
		triedHere[s][chosen] = true
		reward, changed := play(chosen)
		if changed {
			changer[chosen] = true
		}
		if reward {
			won, via, steps = true, viaOfLastStep(), actions
			if via == "" {
				via = "sequence"
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

// gridSig is a cheap content signature of a grid, used to prune: an action that
// leaves the grid unchanged keeps the same signature and is never retried there.
func gridSig(g actuate.Grid) string {
	var b strings.Builder
	for _, row := range g {
		for _, v := range row {
			b.WriteByte(byte('0' + v%64))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func gridsEqual(a, b actuate.Grid) bool {
	if len(a) != len(b) {
		return false
	}
	for r := range a {
		if len(a[r]) != len(b[r]) {
			return false
		}
		for c := range a[r] {
			if a[r][c] != b[r][c] {
				return false
			}
		}
	}
	return true
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
