package macro

// --- Operator GROWTH, rung 1: a discovered-transform meta-operator ---
//
// Every primitive so far (Reflect/Translate/Count/Correspondence) is a
// hand-authored regularity DETECTOR: a human decided "look for horizontal
// mirror symmetry", "look for a period", etc. That hand-authoring is the
// un-brain-like part -- the reason the agent has to be reverse-engineered per
// game. The thesis step is to make the operator layer GROW: instead of fixing
// WHICH regularity to look for, search a family and let the DATA pick.
//
// This is the smallest honest first rung: the operator no longer knows in
// advance which symmetry a grid has -- it searches a family of involutions
// {reflectH, reflectV, rot180, transpose, antitranspose} and reports the one
// that explains the most cells by bits. reflectH/reflectV reproduce the
// hand-authored Reflect primitive; rot180/transpose/antitranspose are
// regularities the hand-authored set never looked for, and *which* one wins is
// derived from the grid, not written in code. Later rungs grow the family
// itself (compositions) and cache discovered instances as reusable named
// operators; this rung proves the mechanism.
//
// NOT registered in Primitives yet: per the project's own discipline, prove the
// mechanism offline first, then wire it in behind the generalization gate.
//
// Savings accounting matches SymmetrySavings: an involution T pairs each cell i
// with T(i); a matched foreground pair (i != T(i)) means the second cell is
// determined by the first => 1 bit saved, counted once. Fixed points (i==T(i))
// and the identity are excluded (identity is the degenerate cheat: it "explains"
// every cell while saying nothing).

// involution is a shape-preserving self-inverse coordinate map on an h x w grid:
// it maps (r,c) to (r2,c2) with the same cell value expected. name is for
// introspection (which regularity was discovered).
type involution struct {
	name    string
	applies func(h, w int) bool
	mapRC   func(r, c, h, w int) (int, int)
}

var discoverInvolutions = []involution{
	{
		name:    "reflectH",
		applies: func(h, w int) bool { return w >= 2 },
		mapRC:   func(r, c, h, w int) (int, int) { return r, w - 1 - c },
	},
	{
		name:    "reflectV",
		applies: func(h, w int) bool { return h >= 2 },
		mapRC:   func(r, c, h, w int) (int, int) { return h - 1 - r, c },
	},
	{
		name:    "rot180",
		applies: func(h, w int) bool { return h >= 2 || w >= 2 },
		mapRC:   func(r, c, h, w int) (int, int) { return h - 1 - r, w - 1 - c },
	},
	{
		name:    "transpose", // main-diagonal mirror; square only (shape-preserving)
		applies: func(h, w int) bool { return h == w && h >= 2 },
		mapRC:   func(r, c, h, w int) (int, int) { return c, r },
	},
	{
		name:    "antitranspose", // anti-diagonal mirror; square only
		applies: func(h, w int) bool { return h == w && h >= 2 },
		mapRC:   func(r, c, h, w int) (int, int) { return w - 1 - c, h - 1 - r },
	},
}

// rectDims returns (h, w) treating the grid as rectangular by its min row length;
// ragged rows are clipped so mapRC never indexes out of range.
func rectDims(grid [][]int) (int, int) {
	h := len(grid)
	if h == 0 {
		return 0, 0
	}
	return h, minRowLen(grid)
}

// involutionSavings counts foreground cells explained by one involution: matched
// pairs (i, T(i)) with i != T(i), counted once (canonical: the lexicographically
// smaller coordinate owns the pair). Fixed points contribute nothing.
func involutionSavings(grid [][]int, bg int, inv involution) int {
	h, w := rectDims(grid)
	if h == 0 || w == 0 || !inv.applies(h, w) {
		return 0
	}
	saved := 0
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			r2, c2 := inv.mapRC(r, c, h, w)
			if r2 == r && c2 == c {
				continue // fixed point: no pair
			}
			// Count each pair once: only from the canonical (smaller) endpoint.
			if r2 < r || (r2 == r && c2 < c) {
				continue
			}
			if grid[r][c] != bg && grid[r][c] == grid[r2][c2] {
				saved++
			}
		}
	}
	return saved
}

// DiscoverTransform searches the involution family and returns the best-saving
// transform's name and its savings. name is "" and savings 0 when no non-trivial
// regularity is found (blank / no symmetry). This is the meta-operator: the
// specific regularity is discovered from the grid, not hand-picked.
func DiscoverTransform(grid [][]int, bg int) (name string, savings int) {
	for _, inv := range discoverInvolutions {
		if s := involutionSavings(grid, bg, inv); s > savings {
			savings, name = s, inv.name
		}
	}
	return name, savings
}

// DiscoverTransformPreference is the compression saving under the discovered
// best involution -- the meta-operator's scalar drive, shaped like the other
// primitives' *Preference so it can later join Primitives behind the gate.
func DiscoverTransformPreference(grid [][]int, bg int) int {
	_, s := DiscoverTransform(grid, bg)
	return s
}

// --- Growable family #2: symmetry UP TO A COLOUR PERMUTATION ---
//
// The fixed Reflect/DiscoverTransform require the mirrored cell to be the SAME
// colour. Many grids are symmetric only after a consistent RECOLOURING (a legend
// maps one colour set to another -- the tn36 class). This family discovers, for
// each involution, the majority colour map sigma between paired cells and counts
// the foreground pairs matched under it. It subsumes exact symmetry (sigma =
// identity) and additionally captures colour-swapped symmetry that the exact
// primitives score as 0. Guards: bg is pinned (a hole is a deletion, not a
// recolour), pairs counted once, blank -> 0.

func colorPermSavings(grid [][]int, bg int, inv involution) int {
	h, w := rectDims(grid)
	if h == 0 || w == 0 || !inv.applies(h, w) {
		return 0
	}
	type pair struct{ a, b int }
	var pairs []pair
	tally := map[int]map[int]int{} // a -> (b -> count), majority vote for sigma(a)
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			r2, c2 := inv.mapRC(r, c, h, w)
			if r2 == r && c2 == c {
				continue // fixed point
			}
			if r2 < r || (r2 == r && c2 < c) {
				continue // count each pair once from the canonical endpoint
			}
			a, b := grid[r][c], grid[r2][c2]
			pairs = append(pairs, pair{a, b})
			if a != bg {
				if tally[a] == nil {
					tally[a] = map[int]int{}
				}
				tally[a][b]++
			}
		}
	}
	sigma := map[int]int{bg: bg}
	for a, counts := range tally {
		best, bn := a, -1
		for b, n := range counts {
			if b == bg {
				continue // a foreground colour may not map onto bg
			}
			if n > bn || (n == bn && b < best) {
				best, bn = b, n
			}
		}
		sigma[a] = best
	}
	saved := 0
	for _, p := range pairs {
		if p.a != bg && sigma[p.a] == p.b {
			saved++
		}
	}
	return saved
}

// ColorPermSymmetry is the best colour-permutation symmetry saving over the
// involution family -- a growable feature capturing "symmetric once recoloured",
// which the exact-match primitives miss.
func ColorPermSymmetry(grid [][]int, bg int) int {
	best := 0
	for _, inv := range discoverInvolutions {
		if s := colorPermSavings(grid, bg, inv); s > best {
			best = s
		}
	}
	return best
}
