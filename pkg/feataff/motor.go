package feataff

import (
	"alphaarc/pkg/actuate"
	"alphaarc/pkg/macro"
)

// --- Goal-directed MOTOR NAVIGATION ---
//
// Some games (ls20-class) have an AVATAR moved by simple (non-click) actions, and
// the reward needs it driven to a place -- not a single press. Blind action
// probing wastes the tiny budget wandering. Instead: (1) LEARN the motor model --
// which action displaces the avatar which way, discovered from observed moves, not
// hardcoded; (2) NAVIGATE greedily -- pick the action that most reduces the
// distance from the avatar to the target object. Both are derived from the grid;
// nothing about up/down/left/right or the avatar's colour is written in code.

// MotorModel maps each simple ActionID to the avatar displacement it causes, and
// records the avatar's (learned) colour.
type MotorModel struct {
	Disp        map[int][2]int // ActionID -> (drow, dcol)
	AvatarColor int
	known       bool
}

// Known reports whether at least one action's displacement was learned.
func (m MotorModel) Known() bool { return m.known }

// avatarShift infers the avatar's move between two grids, with a COHERENCE filter:
// a colour X counts as an avatar only if ALL of its cells translated by ONE shared
// vector -- i.e. the set of X-cells after equals the set before shifted by (dr,dc).
// A scattered/partial change (X-cells moving by different amounts, or some static)
// is NOT a rigid body and is rejected -- this kills the noisy false "avatars" that
// a centroid-only estimate produced on ls20. The avatar colour is the largest such
// coherently-translated blob. ok is false if no colour translates coherently.
func avatarShift(before, after actuate.Grid, bg int) (dr, dc, color int, ok bool) {
	cellsOf := func(g actuate.Grid, x int) map[[2]int]bool {
		s := map[[2]int]bool{}
		for r := range g {
			for c := range g[r] {
				if g[r][c] == x {
					s[[2]int{r, c}] = true
				}
			}
		}
		return s
	}
	centroid := func(s map[[2]int]bool) (int, int) {
		sr, sc := 0, 0
		for p := range s {
			sr += p[0]
			sc += p[1]
		}
		n := len(s)
		return sr / n, sc / n
	}
	setsEqual := func(a, b map[[2]int]bool) bool {
		if len(a) != len(b) {
			return false
		}
		for p := range a {
			if !b[p] {
				return false
			}
		}
		return true
	}

	// colours that appear anywhere (foreground only)
	colors := map[int]bool{}
	for _, g := range []actuate.Grid{before, after} {
		for r := range g {
			for c := range g[r] {
				if g[r][c] != bg {
					colors[g[r][c]] = true
				}
			}
		}
	}

	bestN := 0
	found := false
	for x := range colors {
		sb := cellsOf(before, x)
		sa := cellsOf(after, x)
		if len(sb) == 0 || len(sb) != len(sa) || setsEqual(sb, sa) {
			continue // gone/appeared/resized, or didn't move
		}
		// candidate rigid shift = centroid difference (exact for a true translation)
		br, bc := centroid(sb)
		ar, ac := centroid(sa)
		vr, vc := ar-br, ac-bc
		if vr == 0 && vc == 0 {
			continue
		}
		// COHERENCE: every X-cell must move by exactly (vr,vc).
		shifted := make(map[[2]int]bool, len(sb))
		for p := range sb {
			shifted[[2]int{p[0] + vr, p[1] + vc}] = true
		}
		if !setsEqual(shifted, sa) {
			continue // not a rigid translation -> not an avatar
		}
		if len(sb) > bestN {
			bestN, dr, dc, color, found = len(sb), vr, vc, x, true
		}
	}
	return dr, dc, color, found
}

// centroidOfColor returns the integer centroid of a colour's cells (and count).
func centroidOfColor(g actuate.Grid, color int) (r, c, n int) {
	sr, sc := 0, 0
	for i := range g {
		for j := range g[i] {
			if g[i][j] == color {
				sr += i
				sc += j
				n++
			}
		}
	}
	if n == 0 {
		return 0, 0, 0
	}
	return sr / n, sc / n, n
}

// MotorModel builds the motor model from observations ALREADY recorded (e.g. the
// Phase A action probes) -- no extra actions spent. It reads each action-control's
// stored before/after grids and infers the avatar displacement.
func (m *FeatureMapper) MotorModel(bg int) MotorModel {
	mm := MotorModel{Disp: map[int][2]int{}}
	for _, o := range m.obs {
		if o.ctrl.Kind != "action" || o.before == nil || o.after == nil {
			continue
		}
		if dr, dc, color, ok := avatarShift(o.before, o.after, bg); ok {
			mm.Disp[o.ctrl.ActionID] = [2]int{dr, dc}
			mm.AvatarColor = color
			mm.known = true
		}
	}
	return mm
}

// DiscoverMotor learns the motor model by trying each action once from a fresh
// reset (offline: reset is free and gives clean per-action attribution).
func DiscoverMotor(env Env, bg int, actions []actuate.Control) MotorModel {
	m := MotorModel{Disp: map[int][2]int{}}
	for _, a := range actions {
		if a.Kind != "action" {
			continue
		}
		before := env.Reset()
		after, _ := env.Step(a)
		if dr, dc, color, ok := avatarShift(before, after, bg); ok {
			m.Disp[a.ActionID] = [2]int{dr, dc}
			m.AvatarColor = color
			m.known = true
		}
	}
	return m
}

// nearestOtherObject returns the centroid of the object (a non-bg, non-avatar
// blob) nearest to (ar,ac) -- the navigation target.
func nearestOtherObject(g actuate.Grid, bg, avatarColor, ar, ac int) (tr, tc int, ok bool) {
	best := 1 << 30
	for _, p := range macro.ObjectTargets(g, bg, 32) {
		if g[p.Y][p.X] == avatarColor {
			continue
		}
		d := abs(p.Y-ar) + abs(p.X-ac)
		if d < best {
			best, tr, tc, ok = d, p.Y, p.X, true
		}
	}
	return tr, tc, ok
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// NextAction returns the action that most reduces the avatar's Manhattan distance
// to the nearest other object, from the given current grid. ok is false if there
// is no avatar, no target, or no learned action -- the live driver calls this each
// step so it can apply the move through its own budget/record/reward accounting.
func (m MotorModel) NextAction(cur actuate.Grid, bg int) (actuate.Control, bool) {
	if !m.known {
		return actuate.Control{}, false
	}
	ar, ac, n := centroidOfColor(cur, m.AvatarColor)
	if n == 0 {
		return actuate.Control{}, false
	}
	tr, tc, ok := nearestOtherObject(cur, bg, m.AvatarColor, ar, ac)
	if !ok {
		return actuate.Control{}, false
	}
	bestA, bestDist := -1, 1<<30
	for aid, d := range m.Disp {
		nd := abs(ar+d[0]-tr) + abs(ac+d[1]-tc)
		if nd < bestDist {
			bestA, bestDist = aid, nd
		}
	}
	if bestA < 0 {
		return actuate.Control{}, false
	}
	return actuate.Control{Kind: "action", ActionID: bestA}, true
}

// NavigateToTarget drives the avatar toward the nearest other object using the
// motor model until the reward fires or the budget/limit is spent, stopping if a
// move is blocked (avatar didn't shift) to avoid pushing into a wall forever.
// Offline convenience (it Resets); the live driver uses NextAction step-by-step.
func (m MotorModel) NavigateToTarget(env Env, bg, budget int) (won bool, steps int) {
	if !m.known {
		return false, 0
	}
	cur := env.Reset()
	for steps = 0; steps < budget; steps++ {
		c, ok := m.NextAction(cur, bg)
		if !ok {
			return false, steps
		}
		after, reward := env.Step(c)
		if reward {
			return true, steps + 1
		}
		if gridsEqual(cur, after) {
			return false, steps + 1 // blocked
		}
		cur = after
	}
	return false, steps
}
