package perception

// Object Delta Grammar: a discrete, online model of what the agent's actions
// actually DO, learned from observation instead of hard-guessed. The prior
// counterfactual assumed every click removed/recolored its target -- a phantom
// unrelated to the game's real physics -- so the estimated Δsat was noise and
// the agent could never climb a hypothesis. Here, after each click, the agent
// DIFFS the object before and after and parameterizes the change into one of a
// few Core-Knowledge transform operators; it accumulates these into an
// affordance table (object class -> operator) that gives the counterfactual a
// REALISTIC next state. Efference-copy / sensorimotor-contingency learning, cut
// to what a 60-step budget allows: no heavy approximator, just 1-2-shot
// operator induction over discrete object transformations.

// OpKind is one learned transform an action can apply to an object.
type OpKind int

const (
	OpUnknown   OpKind = iota // effect not yet identified -- worth probing (epistemic)
	OpRecolor                 // the object's cells changed color (state/toggle cycle)
	OpDespawn                 // the object vanished (cells -> background)
	OpTranslate               // the object shifted by (DX,DY)
)

// TransformOperator is a parameterized, applicable effect.
type TransformOperator struct {
	Kind   OpKind
	Color  int // OpRecolor: the new color
	DX, DY int // OpTranslate: the shift
}

// AffordanceTable maps an object class (its color -- the simplest class that
// gives 1-shot generalization: "clicking a green square shifts it" transfers to
// every green square) to the transform operators observed for it, with support
// counts so the most-confirmed operator wins.
type AffordanceTable struct {
	ops map[int]map[TransformOperator]int
}

// NewAffordanceTable returns an empty table.
func NewAffordanceTable() *AffordanceTable {
	return &AffordanceTable{ops: map[int]map[TransformOperator]int{}}
}

// ObserveClick learns from one real click: it finds the object clicked at (x,y)
// in the BEFORE grid, classifies how it changed in the AFTER grid, and (if the
// change is a recognized operator) credits that operator to the object's class.
// Unknown/ambiguous changes are not recorded -- noise must not pollute the table.
func (t *AffordanceTable) ObserveClick(before, after [][]int, x, y int) {
	bg := BackgroundColor(before)
	clicked, ok := blobAt(before, bg, x, y)
	if !ok {
		return
	}
	op := classifyEffect(after, bg, clicked)
	if op.Kind == OpUnknown {
		return
	}
	if t.ops[clicked.Color] == nil {
		t.ops[clicked.Color] = map[TransformOperator]int{}
	}
	t.ops[clicked.Color][op]++
}

// Operator returns the most-confirmed transform for a color class, if any.
func (t *AffordanceTable) Operator(color int) (TransformOperator, bool) {
	m := t.ops[color]
	if len(m) == 0 {
		return TransformOperator{}, false
	}
	best, bestN := TransformOperator{}, 0
	for op, n := range m {
		if n > bestN {
			bestN, best = n, op
		}
	}
	return best, true
}

// Known reports whether the agent has learned what clicking this color does.
func (t *AffordanceTable) Known(color int) bool {
	_, ok := t.Operator(color)
	return ok
}

// KnownCount is how many object classes have a learned affordance -- how much of
// the environment's physics the agent has mapped so far.
func (t *AffordanceTable) KnownCount() int { return len(t.ops) }

// Predict applies obj's learned operator to grid, returning the realistic next
// state and true; (nil,false) when obj's class has no known affordance yet.
func (t *AffordanceTable) Predict(grid [][]int, obj Blob) ([][]int, bool) {
	op, ok := t.Operator(obj.Color)
	if !ok {
		return nil, false
	}
	bg := BackgroundColor(grid)
	switch op.Kind {
	case OpRecolor:
		return mutateCells(grid, obj.Cells, op.Color), true
	case OpDespawn:
		return mutateCells(grid, obj.Cells, bg), true
	case OpTranslate:
		return translateCells(grid, obj.Cells, obj.Color, bg, op.DX, op.DY), true
	}
	return nil, false
}

// LearnedPragmaticValue is the honest counterfactual: how much clicking obj is
// PREDICTED to advance the current hypothesis, using the LEARNED effect rather
// than a guess. Returns (Δscoped-satisfaction, true) when obj's affordance is
// known, or (0,false) when it isn't -- the caller then treats an unknown-effect
// object as epistemically valuable (something to probe), which is what drives
// the early motor-babbling phase.
func LearnedPragmaticValue(grid [][]int, obj Blob, score func([][]int) float64, table *AffordanceTable) (float64, bool) {
	pred, ok := table.Predict(grid, obj)
	if !ok {
		return 0, false
	}
	minR, minC, maxR, maxC, hasFG := ForegroundBBox(grid)
	if !hasFG {
		return 0, true
	}
	base := score(cropTo(grid, minR, minC, maxR, maxC))
	return score(cropTo(pred, minR, minC, maxR, maxC)) - base, true
}

// classifyEffect parameterizes what happened to `clicked` (present in the BEFORE
// grid) by reading the AFTER grid at its cells.
func classifyEffect(after [][]int, bg int, clicked Blob) TransformOperator {
	colorsNow := map[int]int{}
	for _, p := range clicked.Cells {
		if p.Y >= 0 && p.Y < len(after) && p.X >= 0 && p.X < len(after[p.Y]) {
			colorsNow[after[p.Y][p.X]]++
		}
	}
	if len(colorsNow) != 1 {
		return TransformOperator{Kind: OpUnknown} // partial/mixed change -- don't guess
	}
	var now int
	for c := range colorsNow {
		now = c
	}
	switch {
	case now == bg:
		// cells cleared: the body either vanished or moved elsewhere.
		if dx, dy, ok := findTranslation(after, bg, clicked); ok {
			return TransformOperator{Kind: OpTranslate, DX: dx, DY: dy}
		}
		return TransformOperator{Kind: OpDespawn}
	case now != clicked.Color:
		return TransformOperator{Kind: OpRecolor, Color: now}
	default:
		// unchanged in place: maybe a same-color copy moved, else nothing learnable.
		if dx, dy, ok := findTranslation(after, bg, clicked); ok {
			return TransformOperator{Kind: OpTranslate, DX: dx, DY: dy}
		}
		return TransformOperator{Kind: OpUnknown}
	}
}

// findTranslation looks in the after-grid for a same-color, same-size body near
// where `clicked` was but shifted -- evidence the click pushed/slid the object.
func findTranslation(after [][]int, bg int, clicked Blob) (dx, dy int, ok bool) {
	for _, b := range FindBlobs(after, bg) {
		if b.Color != clicked.Color || len(b.Cells) != len(clicked.Cells) {
			continue
		}
		ddx, ddy := b.Centroid.X-clicked.Centroid.X, b.Centroid.Y-clicked.Centroid.Y
		if ddx == 0 && ddy == 0 {
			continue
		}
		if abs(ddx)+abs(ddy) <= 8 { // a plausible one-step shift, not a coincidental far body
			return ddx, ddy, true
		}
	}
	return 0, 0, false
}

// blobAt returns the blob containing (x,y), or the nearest-centroid blob.
func blobAt(grid [][]int, bg, x, y int) (Blob, bool) {
	blobs := FindBlobs(grid, bg)
	for _, b := range blobs {
		for _, p := range b.Cells {
			if p.X == x && p.Y == y {
				return b, true
			}
		}
	}
	best, bestD, found := Blob{}, 1<<30, false
	for _, b := range blobs {
		d := (b.Centroid.X-x)*(b.Centroid.X-x) + (b.Centroid.Y-y)*(b.Centroid.Y-y)
		if d < bestD {
			bestD, best, found = d, b, true
		}
	}
	return best, found
}

// translateCells returns a copy of grid with `cells` cleared to bg and the same
// shape re-drawn shifted by (dx,dy) in color.
func translateCells(grid [][]int, cells []Point, color, bg, dx, dy int) [][]int {
	out := mutateCells(grid, cells, bg)
	for _, p := range cells {
		nx, ny := p.X+dx, p.Y+dy
		if ny >= 0 && ny < len(out) && nx >= 0 && nx < len(out[ny]) {
			out[ny][nx] = color
		}
	}
	return out
}
