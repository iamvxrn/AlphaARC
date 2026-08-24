package macro

// --- The general GOAL-DERIVER (half ① of the reverse-campaign synthesis) ---
//
// The reverse campaign found a common goal representation across ARC-AGI-3 games:
// drive the WORKSPACE region to match a REFERENCE encoded in the grid. Grounded in
// the compression thesis, that reference is the grid's own dominant regularity, and
// the goal is "make the cells that BREAK it conform to what it PREDICTS." So the
// goal is not a hand-set target -- it is derived: find the best tiling, take the
// per-class majority as the reference, and every cell that differs from its class
// majority becomes a target (cell -> the colour that would complete the regularity).
//
// This turns residual-as-attention into residual-as-GOAL: not just "these cells are
// anomalous" but "these cells should become THIS to shorten the description."

// GoalTarget is one derived goal: cell (R,C) should become colour Want.
type GoalTarget struct{ R, C, Want int }

const goalMaxPeriod = 16

// tilingAgreement is the fraction of cells equal to their (pr,pc)-class majority.
func tilingAgreement(grid [][]int, h, w, pr, pc int) float64 {
	maj := majorityTile(grid, h, w, pr, pc)
	same, total := 0, 0
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			total++
			if grid[r][c] == maj[r%pr][c%pc] {
				same++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(same) / float64(total)
}

// majorityTile returns the pr x pc pattern where each cell is the most common
// colour over its residue class (r%pr, c%pc).
func majorityTile(grid [][]int, h, w, pr, pc int) [][]int {
	counts := make([]map[int]int, pr*pc)
	for i := range counts {
		counts[i] = map[int]int{}
	}
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			counts[(r%pr)*pc+(c%pc)][grid[r][c]]++
		}
	}
	maj := make([][]int, pr)
	for i := range maj {
		maj[i] = make([]int, pc)
		for j := range maj[i] {
			best, bn := 0, -1
			for col, n := range counts[i*pc+j] {
				if n > bn || (n == bn && col < best) {
					best, bn = col, n
				}
			}
			maj[i][j] = best
		}
	}
	return maj
}

// ContentBox is the bounding box of non-background cells: the crude workspace
// segmentation. ok is false when the grid is all background.
func ContentBox(grid [][]int, bg int) (r0, c0, r1, c1 int, ok bool) {
	h, w := rectDims(grid)
	r0, c0, r1, c1 = h, w, -1, -1
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			if grid[r][c] != bg {
				if r < r0 {
					r0 = r
				}
				if r > r1 {
					r1 = r
				}
				if c < c0 {
					c0 = c
				}
				if c > c1 {
					c1 = c
				}
			}
		}
	}
	return r0, c0, r1, c1, r1 >= 0
}

// GoalTargets derives the goal for a grid: the cells that break its dominant tiling
// and the colour each should take to complete it. Returns nil when no non-trivial
// tiling explains the grid well enough. Guards against the degenerate "background
// is the reference" reading: it refuses a majority tile that is mostly background,
// and never emits a target that would ERASE content (Want == bg).
func GoalTargets(grid [][]int, bg int) []GoalTarget {
	h, w := rectDims(grid)
	return goalTargetsIn(grid, bg, 0, 0, h-1, w-1)
}

// FramedRegions finds the INTERIORS of rectangular frames -- a border of a single
// non-background colour enclosing a differently-coloured region. This is the
// workspace-segmentation device most ARC-AGI-3 puzzles use (a box drawn around the
// play area). Returns interior boxes [r0,c0,r1,c1] (inclusive).
func FramedRegions(grid [][]int, bg int) [][4]int {
	h, w := rectDims(grid)
	// colour -> bounding box
	type bbox struct{ r0, c0, r1, c1, n int }
	boxes := map[int]*bbox{}
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			v := grid[r][c]
			if v == bg {
				continue
			}
			b := boxes[v]
			if b == nil {
				boxes[v] = &bbox{r, c, r, c, 1}
				continue
			}
			if r < b.r0 {
				b.r0 = r
			}
			if r > b.r1 {
				b.r1 = r
			}
			if c < b.c0 {
				b.c0 = c
			}
			if c > b.c1 {
				b.c1 = c
			}
			b.n++
		}
	}
	var out [][4]int
	for f, b := range boxes {
		if b.r1-b.r0 < 2 || b.c1-b.c0 < 2 {
			continue // too small to enclose an interior
		}
		perimTot, perimF, intTot, intF := 0, 0, 0, 0
		for r := b.r0; r <= b.r1; r++ {
			for c := b.c0; c <= b.c1; c++ {
				onBorder := r == b.r0 || r == b.r1 || c == b.c0 || c == b.c1
				if onBorder {
					perimTot++
					if grid[r][c] == f {
						perimF++
					}
				} else {
					intTot++
					if grid[r][c] == f {
						intF++
					}
				}
			}
		}
		// a frame: perimeter mostly the frame colour, interior mostly not.
		if perimTot > 0 && perimF*5 >= perimTot*4 && (intTot == 0 || intF*2 < intTot) {
			out = append(out, [4]int{b.r0 + 1, b.c0 + 1, b.r1 - 1, b.c1 - 1})
		}
	}
	return out
}

// SegmentedGoalTargets segments the workspace then derives targets there, mapped
// back to original coordinates. It prefers a FRAMED region's interior (the puzzle
// box) -- robust to border decorations that would blow up the content bbox -- and
// falls back to the non-background content box when no frame is found.
func SegmentedGoalTargets(grid [][]int, bg int) []GoalTarget {
	for _, fr := range FramedRegions(grid, bg) {
		if t := goalTargetsIn(grid, bg, fr[0], fr[1], fr[2], fr[3]); len(t) > 0 {
			return t
		}
	}
	r0, c0, r1, c1, ok := ContentBox(grid, bg)
	if !ok {
		return nil
	}
	return goalTargetsIn(grid, bg, r0, c0, r1, c1)
}

// goalTargetsIn derives goal targets within the inclusive box [r0..r1, c0..c1] of
// grid, returning targets in ORIGINAL grid coordinates.
func goalTargetsIn(grid [][]int, bg, r0, c0, r1, c1 int) []GoalTarget {
	sub := crop(grid, r0, c0, r1, c1)
	h, w := rectDims(sub)
	if h < 2 || w < 2 {
		return nil
	}
	bestPr, bestPc := 0, 0
	bestAgree := -1.0
	for pr := 1; pr <= h/2 && pr <= goalMaxPeriod; pr++ {
		for pc := 1; pc <= w/2 && pc <= goalMaxPeriod; pc++ {
			if pr == 1 && pc == 1 {
				continue
			}
			a := tilingAgreement(sub, h, w, pr, pc)
			if a > bestAgree+1e-9 || (a > bestAgree-1e-9 && pr*pc < bestPr*bestPc) {
				bestAgree, bestPr, bestPc = a, pr, pc
			}
		}
	}
	if bestPr == 0 || bestAgree < 0.6 || bestAgree >= 1.0 {
		return nil
	}
	maj := majorityTile(sub, h, w, bestPr, bestPc)
	// reject "background is the reference": the majority tile must be mostly non-bg.
	nonbg := 0
	for i := range maj {
		for j := range maj[i] {
			if maj[i][j] != bg {
				nonbg++
			}
		}
	}
	if nonbg*2 < bestPr*bestPc {
		return nil
	}
	var out []GoalTarget
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			want := maj[r%bestPr][c%bestPc]
			if want == bg { // never erase content to reach the "reference"
				continue
			}
			if sub[r][c] != want {
				out = append(out, GoalTarget{R: r0 + r, C: c0 + c, Want: want})
			}
		}
	}
	return out
}

// crop returns the inclusive sub-grid [r0..r1, c0..c1].
func crop(grid [][]int, r0, c0, r1, c1 int) [][]int {
	out := make([][]int, 0, r1-r0+1)
	for r := r0; r <= r1; r++ {
		row := make([]int, 0, c1-c0+1)
		for c := c0; c <= c1; c++ {
			row = append(row, grid[r][c])
		}
		out = append(out, row)
	}
	return out
}
