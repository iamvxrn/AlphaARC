// Package actuate is the Actuation layer (Affordance Model), rung 1: an
// empirical Action -> Effect map that bridges KNOWING a goal (goalfind: "cell X
// should become colour k") and DOING it, when controls are INDIRECT (the thing
// that changes cell X is a button/tile elsewhere, not a click on X itself --
// vc33's grow button, ls20's changer tiles, ft09's trigger != cell).
//
// It is control-agnostic: a Control is anything the agent can do (a coordinate
// click, or a simple action); the mapper records what each one CHANGED, then
// answers the inverse query the planner needs: "which control makes cell (r,c)
// become colour `to`?" -- so the planner actuates via the real (possibly
// distant) control instead of clicking the target cell blindly.
//
// Rung 1 = single-control cell effects, attributed from a fresh reset. Deferred
// (harder) rungs: conditional effects (a changer that only fires when the avatar
// stands on it), attribute effects (the avatar's own colour/shape/rotation
// rather than a fixed cell), and multi-step chains.
package actuate

// Grid is a rectangular colour grid (values are colour ids).
type Grid = [][]int

// Control is one thing the agent can do.
type Control struct {
	Kind     string // "click" | "action"
	X, Y     int    // click coordinates (X=col, Y=row) when Kind=="click"
	ActionID int    // simple-action id when Kind=="action"
}

func (c Control) equal(o Control) bool {
	return c.Kind == o.Kind && c.X == o.X && c.Y == o.Y && c.ActionID == o.ActionID
}

// CellChange records one cell whose colour changed From -> To at (R,C).
type CellChange struct{ R, C, From, To int }

// Effect is everything a control changed on the board.
type Effect struct{ Changes []CellChange }

func (e Effect) empty() bool { return len(e.Changes) == 0 }

// Observation pairs a control with the effect it produced (from the reset state).
type Observation struct {
	Control Control
	Effect  Effect
}

// Env is the minimal interaction surface the mapper explores. Step applies a
// control and returns the resulting grid; Reset returns the start grid.
type Env interface {
	Reset() Grid
	Step(Control) Grid
}

// diff lists the cells whose colour changed between before and after (same dims).
func diff(before, after Grid) Effect {
	var e Effect
	for r := range before {
		if r >= len(after) {
			break
		}
		for c := range before[r] {
			if c >= len(after[r]) {
				break
			}
			if before[r][c] != after[r][c] {
				e.Changes = append(e.Changes, CellChange{R: r, C: c, From: before[r][c], To: after[r][c]})
			}
		}
	}
	return e
}

// CausalMapper holds what has been learned about control -> effect.
type CausalMapper struct {
	Obs []Observation
}

// Explore tries each candidate control ONCE from a fresh reset (clean
// attribution: the effect is that control's alone) and records what changed.
// Controls that change nothing are still recorded (empty effect) so the planner
// knows a direct click on an inert cell is useless.
func (m *CausalMapper) Explore(env Env, controls []Control) {
	for _, ctrl := range controls {
		before := env.Reset()
		after := env.Step(ctrl)
		m.Obs = append(m.Obs, Observation{Control: ctrl, Effect: diff(before, after)})
	}
}

// Observe records a single control->effect pair the agent saw during normal
// play (not only during a dedicated exploration sweep).
func (m *CausalMapper) Observe(before, after Grid, ctrl Control) {
	m.Obs = append(m.Obs, Observation{Control: ctrl, Effect: diff(before, after)})
}

// ControlForCell returns a control observed to set cell (r,c) to colour `to`.
// This is the inverse query the planner uses: goalfind says "cell (r,c) needs
// colour k" -> ask which control causes exactly that. Prefers the control with
// the SMALLEST side-effect footprint (fewest other cells changed), so the
// planner picks the most precise actuator available.
func (m *CausalMapper) ControlForCell(r, c, to int) (Control, bool) {
	best, found := Control{}, false
	bestFootprint := 1 << 30
	for _, o := range m.Obs {
		for _, ch := range o.Effect.Changes {
			if ch.R == r && ch.C == c && ch.To == to {
				if fp := len(o.Effect.Changes); fp < bestFootprint {
					best, found, bestFootprint = o.Control, true, fp
				}
				break
			}
		}
	}
	return best, found
}

// ControlForColor returns a control observed to set SOME cell to colour `to`
// (a "colour changer" whose target cell may vary) -- the attribute-ish query.
func (m *CausalMapper) ControlForColor(to int) (Control, bool) {
	for _, o := range m.Obs {
		for _, ch := range o.Effect.Changes {
			if ch.To == to {
				return o.Control, true
			}
		}
	}
	return Control{}, false
}

// Plan turns a list of desired cell changes (e.g. from goalfind's per-cell
// targets) into the controls that actuate them, deduped in order. A change with
// no known actuator is skipped (returned in `unmet`) so the caller can explore
// more before retrying.
func (m *CausalMapper) Plan(desired []CellChange) (plan []Control, unmet []CellChange) {
	for _, d := range desired {
		ctrl, ok := m.ControlForCell(d.R, d.C, d.To)
		if !ok {
			unmet = append(unmet, d)
			continue
		}
		dup := false
		for _, p := range plan {
			if p.equal(ctrl) {
				dup = true
				break
			}
		}
		if !dup {
			plan = append(plan, ctrl)
		}
	}
	return plan, unmet
}
