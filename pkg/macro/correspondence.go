package macro

import "fmt"

// Primitive #4: Correspondence / Copy (RELATIONAL compression).
//
// Reflect/Translate/Count measure SELF-regularity (a region regular within
// itself). Correspondence measures regularity BETWEEN regions: region A described
// as "a copy of region B, up to a simple transform, plus a residual". This closes
// the copy/match/template class (s5i5's template boxes, tn36's legend, vc33-L2's
// mirror flip) -- puzzles whose solution is ANTI-compression to a self-regularity
// drive (filling a template raises A's own complexity) but compression-POSITIVE
// relationally (A becomes describable as B, so the residual shrinks).
//
// Search stays O(objects), no NP explosion:
//   - ANCHORS are repetitions, found by the SAME component grouping as Count but
//     with a LOOSE signature (bbox WxH + border/component colour) so regions with
//     different interiors still cluster (V1: exact WxH, no scale).
//   - only WITHIN-class members are compared (k small), never all-pairs.
//   - a CONSTANT transform set {Identity, ColorSwap, Reflect(H/V)} -- ColorSwap's
//     mapping is derived by an O(area) majority vote, not searched.
//
// Saving(class) = sum over non-reference members A of
//     area - residual(A, ref) - pointerCost
// so filling one correct cell of a partial pattern drops the residual by 1 and
// lifts the saving by 1 -- the dopamine click on DriveScore.

const correspondencePointerCost = 1 // bits to name the template + anchor (small constant)
const correspondenceMinArea = 4     // a real region, not a dot/line (degenerate-cheat guard)

func cellsBBox(cells []Cell) (minR, minC, maxR, maxC int) {
	minR, minC = cells[0].R, cells[0].C
	maxR, maxC = cells[0].R, cells[0].C
	for _, cl := range cells {
		if cl.R < minR {
			minR = cl.R
		}
		if cl.R > maxR {
			maxR = cl.R
		}
		if cl.C < minC {
			minC = cl.C
		}
		if cl.C > maxC {
			maxC = cl.C
		}
	}
	return
}

// subgridOf extracts the full rectangular content (all colours) of a bbox.
func subgridOf(grid [][]int, minR, minC, maxR, maxC int) [][]int {
	out := make([][]int, maxR-minR+1)
	for r := minR; r <= maxR; r++ {
		row := make([]int, maxC-minC+1)
		for c := minC; c <= maxC; c++ {
			if r < len(grid) && c < len(grid[r]) {
				row[c-minC] = grid[r][c]
			}
		}
		out[r-minR] = row
	}
	return out
}

func distinctColorCount(g [][]int) int {
	seen := map[int]struct{}{}
	for _, row := range g {
		for _, v := range row {
			seen[v] = struct{}{}
		}
	}
	return len(seen)
}

func residualIdentity(a, b [][]int) int {
	res := 0
	for i := range a {
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				res++
			}
		}
	}
	return res
}

func residualReflectH(a, b [][]int) int {
	res := 0
	for i := range a {
		w := len(a[i])
		for j := 0; j < w; j++ {
			if a[i][j] != b[i][w-1-j] {
				res++
			}
		}
	}
	return res
}

func residualReflectV(a, b [][]int) int {
	res := 0
	h := len(a)
	for i := 0; i < h; i++ {
		for j := range a[i] {
			if a[i][j] != b[h-1-i][j] {
				res++
			}
		}
	}
	return res
}

// residualColorSwap derives the majority colour map b->a in O(area) (no search)
// and counts the residual under it. Subsumes Identity when the map is the
// identity, and catches a pure recolouring of the same pattern. bg is pinned to
// bg -- a background hole is a deletion, NOT a recolouring, so ColorSwap may not
// remap bg to a foreground colour (which would hide a missing cell).
func residualColorSwap(a, b [][]int, bg int) int {
	tally := map[int]map[int]int{}
	for i := range a {
		for j := range a[i] {
			bc, ac := b[i][j], a[i][j]
			if bc == bg {
				continue // bg is fixed to bg, not remapped
			}
			if tally[bc] == nil {
				tally[bc] = map[int]int{}
			}
			tally[bc][ac]++
		}
	}
	m := map[int]int{bg: bg}
	for bc, counts := range tally {
		best, bn := bc, -1
		for ac, n := range counts {
			if ac == bg {
				continue // a foreground template colour may not map onto bg
			}
			if n > bn || (n == bn && ac < best) {
				best, bn = ac, n
			}
		}
		m[bc] = best
	}
	res := 0
	for i := range a {
		for j := range a[i] {
			if a[i][j] != m[b[i][j]] {
				res++
			}
		}
	}
	return res
}

// minResidual is the fewest cells A needs to differ from B under any allowed
// transform {Identity, ColorSwap, Reflect-H, Reflect-V}.
func minResidual(a, b [][]int, bg int) int {
	best := residualIdentity(a, b)
	for _, r := range []int{residualColorSwap(a, b, bg), residualReflectH(a, b), residualReflectV(a, b)} {
		if r < best {
			best = r
		}
	}
	return best
}

// nonBgCount counts non-background cells in a region.
func nonBgCount(g [][]int, bg int) int {
	n := 0
	for _, row := range g {
		for _, v := range row {
			if v != bg {
				n++
			}
		}
	}
	return n
}

// CorrespondenceSavings sums the description saved by describing repeated
// same-size regions as copies of one representative (up to a simple transform).
func CorrespondenceSavings(grid [][]int, bg int) int {
	comps := components(grid, bg)
	if len(comps) < 2 {
		return 0
	}
	// Group component bboxes by (W, H, colour) -- the loose anchor signature.
	type region struct{ minR, minC, maxR, maxC int }
	groups := map[string][]region{}
	for _, cp := range comps {
		minR, minC, maxR, maxC := cellsBBox(cp.cells)
		w, h := maxC-minC+1, maxR-minR+1
		if w*h < correspondenceMinArea {
			continue
		}
		key := fmt.Sprintf("%dx%d:c%d", w, h, cp.color)
		groups[key] = append(groups[key], region{minR, minC, maxR, maxC})
	}

	savings := 0
	for _, members := range groups {
		if len(members) < 2 {
			continue
		}
		// Extract each member's full bbox content; the reference is the FULLEST
		// member (most non-bg cells) so a partial copy is scored against the
		// template, not vice-versa (a bg hole must not become the reference and get
		// remapped away).
		grids := make([][][]int, len(members))
		refIdx, refFull := 0, -1
		for i, m := range members {
			grids[i] = subgridOf(grid, m.minR, m.minC, m.maxR, m.maxC)
			if full := nonBgCount(grids[i], bg); full > refFull {
				refFull, refIdx = full, i
			}
		}
		ref := grids[refIdx]
		if distinctColorCount(ref) < 2 {
			continue // reference must be a real pattern, not a solid block (cheat guard)
		}
		area := len(ref) * len(ref[0])
		for i, g := range grids {
			if i == refIdx {
				continue
			}
			s := area - minResidual(g, ref, bg) - correspondencePointerCost
			if s > 0 {
				savings += s
			}
		}
	}
	return savings
}

// CorrespondencePreference is the compression saving under the Copy primitive.
func CorrespondencePreference(grid [][]int, bg int) int {
	return CorrespondenceSavings(grid, bg)
}
