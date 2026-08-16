package perception

import (
	"fmt"
	"sort"
	"strings"
)

// Object Delta Grammar, RELATIONAL edition. The prior version learned only what
// happened to the CLICKED object's own cells -- so a game whose click changes
// something ELSEWHERE (the norm: switch here toggles a light there) taught it
// nothing (affordances stuck at 1 on vc33). Here the affordance is the full,
// possibly non-local delta a click produces, recorded RELATIVE to the trigger so
// it generalizes across a class: "clicking a green object flips the cell 5 to
// its right" transfers to every green object. This is the relational dynamics
// Action(O_trigger) -> ΔO_target the discrete-synthesis view of ARC demands --
// without it a 1-step (or any) counterfactual is blind to consequences.
//
// HONEST limits: the delta is stored relative to the trigger centroid, so it
// generalizes for spatially-consistent mechanics but mispredicts a truly
// ABSOLUTE target (a switch that always toggles the same fixed lock); object
// class is color only; and a cascade/animation delta is too large to be a
// clean operator, so it's ignored rather than mislearned.

// CellChange is one cell the click altered, as an offset from the trigger's
// centroid plus the resulting color.
type CellChange struct {
	DX, DY, Color int
}

// AffordanceTable maps an object class (color) to the relational delta templates
// observed for clicking it, with support counts so the most-confirmed wins.
type AffordanceTable struct {
	ops map[int]map[string]templateEntry
}

type templateEntry struct {
	changes []CellChange
	count   int
}

// maxChangeCells caps how large a delta can be and still be treated as a clean
// operator -- a bigger change is a cascade/animation, not a learnable rule.
const maxChangeCells = 40

// NewAffordanceTable returns an empty table.
func NewAffordanceTable() *AffordanceTable {
	return &AffordanceTable{ops: map[int]map[string]templateEntry{}}
}

// ObserveClick learns from one real click: it finds the trigger object at (x,y)
// in the BEFORE grid, computes the FULL delta to the AFTER grid relative to the
// trigger centroid, and credits that relational template to the trigger's class.
// A no-change or cascade-sized delta is ignored (nothing clean to learn).
func (t *AffordanceTable) ObserveClick(before, after [][]int, x, y int) {
	bg := BackgroundColor(before)
	trig, ok := blobAt(before, bg, x, y)
	if !ok {
		return
	}
	var changes []CellChange
	for r := range after {
		for c := range after[r] {
			if r < len(before) && c < len(before[r]) && before[r][c] == after[r][c] {
				continue
			}
			changes = append(changes, CellChange{DX: c - trig.Centroid.X, DY: r - trig.Centroid.Y, Color: after[r][c]})
		}
	}
	if len(changes) == 0 || len(changes) > maxChangeCells {
		return
	}
	key := templateKey(changes)
	if t.ops[trig.Color] == nil {
		t.ops[trig.Color] = map[string]templateEntry{}
	}
	e := t.ops[trig.Color][key]
	e.changes = changes
	e.count++
	t.ops[trig.Color][key] = e
}

// Operator returns the most-confirmed relational delta for a color class.
func (t *AffordanceTable) Operator(color int) ([]CellChange, bool) {
	m := t.ops[color]
	if len(m) == 0 {
		return nil, false
	}
	var best []CellChange
	bestN := 0
	for _, e := range m {
		if e.count > bestN {
			bestN, best = e.count, e.changes
		}
	}
	return best, true
}

// Known reports whether the agent has learned what clicking this color does.
func (t *AffordanceTable) Known(color int) bool {
	_, ok := t.Operator(color)
	return ok
}

// KnownCount is how many object classes have a learned affordance.
func (t *AffordanceTable) KnownCount() int { return len(t.ops) }

// Predict applies obj's learned relational template to grid (offsets anchored at
// obj's centroid), returning the realistic next state; (nil,false) when obj's
// class has no known affordance.
func (t *AffordanceTable) Predict(grid [][]int, obj Blob) ([][]int, bool) {
	changes, ok := t.Operator(obj.Color)
	if !ok {
		return nil, false
	}
	out := make([][]int, len(grid))
	for r := range grid {
		row := make([]int, len(grid[r]))
		copy(row, grid[r])
		out[r] = row
	}
	for _, ch := range changes {
		nr, nc := obj.Centroid.Y+ch.DY, obj.Centroid.X+ch.DX
		if nr >= 0 && nr < len(out) && nc >= 0 && nc < len(out[nr]) {
			out[nr][nc] = ch.Color
		}
	}
	return out, true
}

// LookaheadValue is the short-horizon planner (the fix for the non-monotonic
// landscape): the best scoped satisfaction reachable within `depth` clicks that
// START with clicking `first`, simulating each step through the learned
// affordances. depth>=2 lets a preparatory move with Δsat<=0 count as valuable
// when a FOLLOW-UP move then closes the invariant -- stepping over the valleys a
// greedy 1-step agent gets stuck in. known=false when first's affordance is
// unknown (can't roll out -> caller treats it as epistemic). `others` are the
// candidate second-step objects (applied at their current centroids on the
// simulated state -- a one-frame-stale approximation kept cheap on purpose).
func (t *AffordanceTable) LookaheadValue(grid [][]int, first Blob, others []Blob, score func([][]int) float64, depth int) (float64, bool) {
	s1, ok := t.Predict(grid, first)
	if !ok {
		return 0, false
	}
	best := scopedScore(s1, score)
	if depth >= 2 {
		for _, b := range others {
			if s2, ok := t.Predict(s1, b); ok {
				if v := scopedScore(s2, score); v > best {
					best = v
				}
			}
		}
	}
	return best, true
}

func templateKey(changes []CellChange) string {
	cp := append([]CellChange(nil), changes...)
	sort.Slice(cp, func(i, j int) bool {
		if cp[i].DY != cp[j].DY {
			return cp[i].DY < cp[j].DY
		}
		if cp[i].DX != cp[j].DX {
			return cp[i].DX < cp[j].DX
		}
		return cp[i].Color < cp[j].Color
	})
	var b strings.Builder
	for _, ch := range cp {
		fmt.Fprintf(&b, "%d,%d,%d;", ch.DX, ch.DY, ch.Color)
	}
	return b.String()
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
