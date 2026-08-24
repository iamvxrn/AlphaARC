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

// GoalTargets derives the goal for a grid: the cells that break its dominant tiling
// and the colour each should take to complete it. Returns nil when no non-trivial
// tiling explains the grid well enough (agreement < 0.6) -- i.e. there is no
// reference regularity to conform to. Periods are bounded (small tiles); the
// smallest period that maximises agreement wins (Occam).
func GoalTargets(grid [][]int, bg int) []GoalTarget {
	h, w := rectDims(grid)
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
			a := tilingAgreement(grid, h, w, pr, pc)
			// strictly better, or equally good with a smaller tile (Occam)
			if a > bestAgree+1e-9 || (a > bestAgree-1e-9 && pr*pc < bestPr*bestPc) {
				bestAgree, bestPr, bestPc = a, pr, pc
			}
		}
	}
	// need a real regularity that is not already perfect (nothing to fix) and not noise
	if bestPr == 0 || bestAgree < 0.6 || bestAgree >= 1.0 {
		return nil
	}
	maj := majorityTile(grid, h, w, bestPr, bestPc)
	var out []GoalTarget
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			if want := maj[r%bestPr][c%bestPc]; grid[r][c] != want {
				out = append(out, GoalTarget{R: r, C: c, Want: want})
			}
		}
	}
	return out
}
