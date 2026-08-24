// Package mbrl is the Model-Based RL core: the architectural pivot away from a flat
// catalogue of detectors (a hidden Path A) toward a real meta-algorithm.
//
// The key insight that dissolves "search vs a ~150-action budget": the budget is
// spent LEARNING A WORLD MODEL, not searching. Program synthesis (the search for a
// solution) then runs IN IMAGINATION -- against the learned model -- for free, and
// only the found program is executed for real.
//
// Four pieces, composed by Solve:
//  1. Object-oriented World Model -- a scene graph (entities from perception) with
//     learned transitions, NOT pixel dynamics.
//  2. Active exploration -- real actions chosen to maximise info-gain (resolve what
//     the model can't yet predict), not random/exhaustive sweeps.
//  3. Imagination synthesis -- goal-directed search over the learned model.
//  4. Execution -- run the synthesised program in the real environment.
//
// This first cut models the OBJECT-TRANSLATION class (an avatar moved by actions);
// richer transition kinds (fill / toggle / stateful protocols) are later rungs. The
// existing detectors (framing, tiling, correspondence, motor, CausalMapper) become
// the primitive library this substrate composes -- they are not discarded.
package mbrl

import "alphaarc/pkg/actuate"

// Grid and Control reuse the shared types.
type Grid = actuate.Grid
type Control = actuate.Control

// Env is the real environment: one continuous episode (Reset once, then Step).
type Env interface {
	Reset() Grid
	Step(Control) (Grid, bool)
}

// --- 1. Object-oriented World Model (scene graph + learned transitions) ---

// Entity is one object in the scene graph: a colour with its size and centroid.
type Entity struct {
	Color, Size, CR, CC int // centroid row/col
}

// Scene extracts the scene graph: one entity per non-background colour.
func Scene(g Grid, bg int) map[int]Entity {
	type acc struct{ n, sr, sc int }
	a := map[int]*acc{}
	for r := range g {
		for c := range g[r] {
			v := g[r][c]
			if v == bg {
				continue
			}
			if a[v] == nil {
				a[v] = &acc{}
			}
			a[v].n++
			a[v].sr += r
			a[v].sc += c
		}
	}
	out := make(map[int]Entity, len(a))
	for col, x := range a {
		out[col] = Entity{Color: col, Size: x.n, CR: x.sr / x.n, CC: x.sc / x.n}
	}
	return out
}

// WorldModel holds the learned object-translation dynamics: which action rigidly
// displaces the avatar (the entity that moves) by how much.
type WorldModel struct {
	Bg          int
	AvatarColor int
	Disp        map[int][2]int  // ActionID -> (drow, dcol)
	seen        map[string]bool // control keys already observed (info-gain: unseen = surprising)
}

func NewWorldModel(bg int) *WorldModel {
	return &WorldModel{Bg: bg, Disp: map[int][2]int{}, seen: map[string]bool{}}
}

func key(c Control) string {
	if c.Kind == "action" {
		return "a" + string(rune('0'+c.ActionID))
	}
	return "c"
}

// Known reports whether the model has already observed this control's effect.
func (m *WorldModel) Known(c Control) bool { return m.seen[key(c)] }

// Observe updates the model from one real transition: if a single colour rigidly
// translated, learn it as the avatar displacement for this action.
func (m *WorldModel) Observe(before, after Grid, c Control) {
	m.seen[key(c)] = true
	sb, sa := Scene(before, m.Bg), Scene(after, m.Bg)
	best, bestN := -1, 0
	var bdr, bdc int
	for col, eb := range sb {
		ea, ok := sa[col]
		if !ok || ea.Size != eb.Size {
			continue // vanished / resized -> not a rigid translation
		}
		dr, dc := ea.CR-eb.CR, ea.CC-eb.CC
		if dr == 0 && dc == 0 {
			continue
		}
		if eb.Size > bestN {
			best, bestN, bdr, bdc = col, eb.Size, dr, dc
		}
	}
	if best >= 0 && c.Kind == "action" {
		m.AvatarColor = best
		m.Disp[c.ActionID] = [2]int{bdr, bdc}
	}
}

// Predict simulates a control in imagination (no real budget): translate the avatar
// by the learned displacement, clamped to the grid; unknown/blocked -> unchanged.
func (m *WorldModel) Predict(g Grid, c Control) Grid {
	d, ok := m.Disp[c.ActionID]
	if !ok || c.Kind != "action" || m.AvatarColor == 0 {
		return g
	}
	h, w := len(g), 0
	if h > 0 {
		w = len(g[0])
	}
	// gather avatar cells
	var cells [][2]int
	for r := range g {
		for col := range g[r] {
			if g[r][col] == m.AvatarColor {
				cells = append(cells, [2]int{r, col})
			}
		}
	}
	for _, p := range cells { // blocked if any cell would leave the grid
		nr, nc := p[0]+d[0], p[1]+d[1]
		if nr < 0 || nr >= h || nc < 0 || nc >= w {
			return g
		}
	}
	out := make(Grid, h)
	for r := range g {
		out[r] = append([]int(nil), g[r]...)
	}
	for _, p := range cells {
		out[p[0]][p[1]] = m.Bg
	}
	for _, p := range cells {
		out[p[0]+d[0]][p[1]+d[1]] = m.AvatarColor
	}
	return out
}

// --- 2. Active exploration (info-gain) ---

// NextExploratoryAction returns the candidate whose effect the model has not yet
// observed -- the maximally surprising probe. ok is false once all are known.
func (m *WorldModel) NextExploratoryAction(cands []Control) (Control, bool) {
	for _, c := range cands {
		if !m.Known(c) {
			return c, true
		}
	}
	return Control{}, false
}

// --- 3. Imagination synthesis (goal-directed search in the model) ---

// SynthesizeToTarget searches, IN THE MODEL, an action sequence that brings the
// avatar's centroid to (tr,tc). BFS over centroid positions using the learned
// displacements -- zero real budget. Returns the plan and whether the goal is
// reachable in the model.
func (m *WorldModel) SynthesizeToTarget(g Grid, tr, tc int, maxSteps int) ([]Control, bool) {
	if m.AvatarColor == 0 || len(m.Disp) == 0 {
		return nil, false
	}
	h, w := len(g), 0
	if h > 0 {
		w = len(g[0])
	}
	start := Scene(g, m.Bg)[m.AvatarColor]
	type node struct{ r, c int }
	startN := node{start.CR, start.CC}
	if startN.r == tr && startN.c == tc {
		return nil, true
	}
	prev := map[node]node{}
	act := map[node]Control{}
	seen := map[node]bool{startN: true}
	queue := []node{startN}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for aid, d := range m.Disp {
			nr, nc := cur.r+d[0], cur.c+d[1]
			if nr < 0 || nr >= h || nc < 0 || nc >= w {
				continue
			}
			nn := node{nr, nc}
			if seen[nn] {
				continue
			}
			seen[nn] = true
			prev[nn] = cur
			act[nn] = Control{Kind: "action", ActionID: aid}
			if nn.r == tr && nn.c == tc {
				// reconstruct
				var rev []Control
				for n := nn; n != startN; n = prev[n] {
					rev = append(rev, act[n])
					if len(rev) > maxSteps {
						return nil, false
					}
				}
				for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
					rev[i], rev[j] = rev[j], rev[i]
				}
				return rev, true
			}
			queue = append(queue, nn)
		}
	}
	return nil, false
}

// --- 4. The MBRL main loop ---

// Result reports how Solve went.
type Result struct {
	Won            bool
	ExploreActions int // real actions spent learning the model
	PlanLen        int // synthesised program length
	Executed       int // real actions spent executing
}

// Solve runs the full loop: learn the world model by info-gain exploration within
// budget, derive the goal (the nearest non-avatar object), synthesise a program in
// imagination, then execute it for real. targetFn lets the caller supply the goal;
// if nil, the nearest other object's centroid is used.
func Solve(env Env, bg int, cands []Control, budget int, targetFn func(Grid, *WorldModel) (int, int, bool)) Result {
	res := Result{}
	g := env.Reset()
	m := NewWorldModel(bg)

	// (2) real-world loop: spend budget resolving what the model can't predict.
	for res.ExploreActions+res.Executed < budget {
		c, ok := m.NextExploratoryAction(cands)
		if !ok {
			break // model has observed every control -> no more info to gain
		}
		before := g
		after, reward := env.Step(c)
		res.ExploreActions++
		m.Observe(before, after, c)
		g = after
		if reward {
			res.Won = true
			return res
		}
	}

	// derive the goal target.
	var tr, tc int
	var ok bool
	if targetFn != nil {
		tr, tc, ok = targetFn(g, m)
	} else {
		tr, tc, ok = nearestOther(g, bg, m.AvatarColor)
	}
	if !ok {
		return res
	}

	// (3) imagination synthesis against the learned model (free).
	plan, ok := m.SynthesizeToTarget(g, tr, tc, budget)
	if !ok {
		return res
	}
	res.PlanLen = len(plan)

	// (4) execute the synthesised program for real.
	for _, c := range plan {
		if res.ExploreActions+res.Executed >= budget {
			break
		}
		_, reward := env.Step(c)
		res.Executed++
		if reward {
			res.Won = true
			return res
		}
	}
	return res
}

// nearestOther returns the centroid of the non-avatar object nearest the avatar.
func nearestOther(g Grid, bg, avatar int) (int, int, bool) {
	s := Scene(g, bg)
	av, ok := s[avatar]
	if !ok {
		return 0, 0, false
	}
	best := 1 << 30
	var br, bc int
	found := false
	for col, e := range s {
		if col == avatar {
			continue
		}
		d := abs(e.CR-av.CR) + abs(e.CC-av.CC)
		if d < best {
			best, br, bc, found = d, e.CR, e.CC, true
		}
	}
	return br, bc, found
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
