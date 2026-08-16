package perception

// Goal inference by HYPOTHESIS-AND-TEST -- the first mechanism here that is
// actually goal inference rather than a proxy for it. The environment gives
// almost no reward (only a rare level-completion), so the agent cannot LEARN
// the goal from reward; it must GUESS candidate goals from structural priors,
// act to REALIZE each guess, and TEST it against the one ground truth it has
// (did the level complete). This file is the guessing half: a small ordered set
// of candidate goals drawn from Core Knowledge (Chollet/Spelke: objectness,
// geometry/symmetry, correspondence), each a scorer in [0,1] saying how well a
// grid currently satisfies that hypothesized objective. The pragmatic loop then
// treats the PURSUED hypothesis's score as its prior preference and climbs it;
// a real completion CONFIRMS the guess (RememberGoalState bootstraps from there),
// a plateau or a reset ROTATES to the next.
//
// HONEST scope: four structural hypotheses is a starting vocabulary, not a
// claim to cover ARC-AGI-3's goal space (Chollet built games to resist any
// fixed prior). The VALUE is the architecture -- generate/realize/test/rotate --
// which is the shape goal inference has to take under near-zero reward; the
// specific hypotheses are swappable and meant to grow.

// Hypothesis is a candidate GOAL: a named scorer over a grid returning how well
// the grid currently satisfies this hypothesized objective, in [0,1] (1 = met).
type Hypothesis struct {
	Name  string
	Score func(grid [][]int) float64
}

// GoalHypotheses is the ordered candidate-goal vocabulary the agent tests. It's
// a set of Core-Knowledge structural invariants (Chollet/Spelke): color, geometry
// (symmetry/correspondence), and topology (connectivity/enclosure/gravity). The
// vocabulary is meant to GROW -- s5i5 showed the climb machinery works but climbs
// the wrong invariant when the real goal isn't in this list.
func GoalHypotheses() []Hypothesis {
	return []Hypothesis{
		{"all-one-color", hypAllOneColor},
		{"horizontal-symmetry", hypHorizontalSymmetry},
		{"vertical-symmetry", hypVerticalSymmetry},
		{"halves-match", hypHalvesMatch},
		{"connectivity", hypConnectivity},
		{"enclosure", hypEnclosure},
		{"gravity", hypGravity},
	}
}

// hypAllOneColor: the goal is consolidation -- every non-background cell the
// same color. Score is the largest single-color share of the non-background
// cells (1.0 when all foreground is one color). 0 for an empty foreground.
func hypAllOneColor(grid [][]int) float64 {
	bg := BackgroundColor(grid)
	counts := map[int]int{}
	total := 0
	for _, row := range grid {
		for _, c := range row {
			if c == bg {
				continue
			}
			counts[c]++
			total++
		}
	}
	if total == 0 {
		return 0
	}
	best := 0
	for _, n := range counts {
		if n > best {
			best = n
		}
	}
	return float64(best) / float64(total)
}

// hypHorizontalSymmetry: the goal is left-right mirror symmetry. Measured only
// over cells where at least one side is foreground (so a mostly-background grid
// isn't trivially "symmetric"): the fraction of those cell-pairs that match.
func hypHorizontalSymmetry(grid [][]int) float64 {
	return mirrorSatisfaction(grid, func(r, c, h, w int) (int, int) { return r, w - 1 - c })
}

// hypVerticalSymmetry: top-bottom mirror symmetry, same measure.
func hypVerticalSymmetry(grid [][]int) float64 {
	return mirrorSatisfaction(grid, func(r, c, h, w int) (int, int) { return h - 1 - r, c })
}

// hypHalvesMatch: the goal is the left half equal to the right half (a copy/
// correspondence task, not a mirror). Compares (r,c) with (r, c+w/2) over the
// left half, again only where at least one side is foreground.
func hypHalvesMatch(grid [][]int) float64 {
	if len(grid) == 0 || len(grid[0]) == 0 {
		return 0
	}
	h, w := len(grid), len(grid[0])
	half := w / 2
	if half == 0 {
		return 0
	}
	bg := BackgroundColor(grid)
	match, total := 0, 0
	for r := 0; r < h; r++ {
		for c := 0; c < half; c++ {
			a, b := grid[r][c], grid[r][c+half]
			if a == bg && b == bg {
				continue
			}
			total++
			if a == b {
				match++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(match) / float64(total)
}

// mirrorSatisfaction is the shared core of the symmetry hypotheses: over every
// cell whose mirror (given by mirrorOf) makes at least one side foreground,
// the fraction whose colors match. Each pair is visited once from each side, so
// the ratio is unaffected; background-background pairs are skipped so an empty
// grid scores 0, not a trivial 1.
func mirrorSatisfaction(grid [][]int, mirrorOf func(r, c, h, w int) (int, int)) float64 {
	if len(grid) == 0 || len(grid[0]) == 0 {
		return 0
	}
	h, w := len(grid), len(grid[0])
	bg := BackgroundColor(grid)
	match, total := 0, 0
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			mr, mc := mirrorOf(r, c, h, w)
			if mr < 0 || mr >= h || mc < 0 || mc >= w {
				continue
			}
			a, b := grid[r][c], grid[mr][mc]
			if a == bg && b == bg {
				continue
			}
			total++
			if a == b {
				match++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(match) / float64(total)
}

// hypConnectivity: the goal is to JOIN the foreground into one body. Score =
// largest connected foreground component (any non-background color, 4-
// connectivity) / total foreground cells; 1.0 when all foreground is a single
// connected piece. Rewards bridging gaps between objects.
func hypConnectivity(grid [][]int) float64 {
	bg := BackgroundColor(grid)
	h := len(grid)
	if h == 0 || len(grid[0]) == 0 {
		return 0
	}
	w := len(grid[0])
	seen := make([][]bool, h)
	for i := range seen {
		seen[i] = make([]bool, w)
	}
	total, largest := 0, 0
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			if grid[r][c] != bg {
				total++
			}
		}
	}
	if total == 0 {
		return 0
	}
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			if grid[r][c] == bg || seen[r][c] {
				continue
			}
			sz := 0
			q := []Point{{X: c, Y: r}}
			seen[r][c] = true
			for len(q) > 0 {
				p := q[len(q)-1]
				q = q[:len(q)-1]
				sz++
				for _, d := range neighbors4 {
					ny, nx := p.Y+d[0], p.X+d[1]
					if ny >= 0 && ny < h && nx >= 0 && nx < w && !seen[ny][nx] && grid[ny][nx] != bg {
						seen[ny][nx] = true
						q = append(q, Point{X: nx, Y: ny})
					}
				}
			}
			if sz > largest {
				largest = sz
			}
		}
	}
	return float64(largest) / float64(total)
}

// hypEnclosure: the goal is to CLOSE contours -- foreground walls surrounding
// background. Score = background cells NOT reachable from the border (enclosed)
// / total background; 1.0 when the foreground encloses all interior background,
// 0 when nothing is closed off. Rewards forming loops/boundaries.
func hypEnclosure(grid [][]int) float64 {
	bg := BackgroundColor(grid)
	h := len(grid)
	if h == 0 || len(grid[0]) == 0 {
		return 0
	}
	w := len(grid[0])
	reach := make([][]bool, h)
	for i := range reach {
		reach[i] = make([]bool, w)
	}
	var q []Point
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			if (r == 0 || r == h-1 || c == 0 || c == w-1) && grid[r][c] == bg && !reach[r][c] {
				reach[r][c] = true
				q = append(q, Point{X: c, Y: r})
			}
		}
	}
	for len(q) > 0 {
		p := q[len(q)-1]
		q = q[:len(q)-1]
		for _, d := range neighbors4 {
			ny, nx := p.Y+d[0], p.X+d[1]
			if ny >= 0 && ny < h && nx >= 0 && nx < w && !reach[ny][nx] && grid[ny][nx] == bg {
				reach[ny][nx] = true
				q = append(q, Point{X: nx, Y: ny})
			}
		}
	}
	totalBg, enclosed := 0, 0
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			if grid[r][c] == bg {
				totalBg++
				if !reach[r][c] {
					enclosed++
				}
			}
		}
	}
	if totalBg == 0 {
		return 0
	}
	return float64(enclosed) / float64(totalBg)
}

// hypGravity: the goal is for bodies to SETTLE -- nothing floating. Score =
// foreground cells that are supported (on the bottom edge, or with foreground
// directly below) / total foreground; 1.0 when everything rests. Rewards
// dropping/packing objects downward.
func hypGravity(grid [][]int) float64 {
	bg := BackgroundColor(grid)
	h := len(grid)
	if h == 0 || len(grid[0]) == 0 {
		return 0
	}
	w := len(grid[0])
	total, supported := 0, 0
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			if grid[r][c] == bg {
				continue
			}
			total++
			if r == h-1 || grid[r+1][c] != bg {
				supported++
			}
		}
	}
	if total == 0 {
		return 0
	}
	return float64(supported) / float64(total)
}

var neighbors4 = [4][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}

// hypRotatePatience is how many steps the pursued hypothesis may go without
// improving its best-seen satisfaction before the tester gives up on it and
// rotates to the next candidate. Sized against a typical level's action budget
// (tens) so a single attempt can try several hypotheses.
const hypRotatePatience = 12

// hypMaxPursue caps how many steps a SINGLE hypothesis may be pursued before
// rotating regardless of whether it's still climbing. Without it, a hypothesis
// that slowly climbs toward a low ceiling (all-one-color reaching 0.44 on s5i5
// but never completing) never plateaus, so the tester stays glued to that one
// "wrong hill" and never samples the others. This forces the agent to try the
// whole invariant vocabulary within an attempt -- realizing a wrong invariant
// to its ceiling without a win IS evidence it's the wrong goal.
const hypMaxPursue = 12

// HypothesisTester pursues one candidate goal at a time and rotates when it
// stalls. The pursued hypothesis's Satisfaction is what the pragmatic loop uses
// as prior preference (so the agent acts to REALIZE the current guess); Observe
// tracks plateau and rotates on a stall, Refute rotates on a reset/loss, and
// Confirm locks the current hypothesis once a real completion validates it.
type HypothesisTester struct {
	hyps     []Hypothesis
	idx      int
	best     float64 // best satisfaction seen for the current hypothesis this attempt
	stale    int     // steps since best improved
	pursued  int     // steps spent on the current hypothesis this attempt
	patience int
	wins     int // levels completed while a hypothesis was being pursued
}

// NewHypothesisTester starts on the first candidate goal.
func NewHypothesisTester() *HypothesisTester {
	return &HypothesisTester{hyps: GoalHypotheses(), patience: hypRotatePatience}
}

// Current is the hypothesis being pursued right now.
func (t *HypothesisTester) Current() Hypothesis { return t.hyps[t.idx] }

// Wins is how many levels completed so far under this tester.
func (t *HypothesisTester) Wins() int { return t.wins }

// Satisfaction is how well grid meets the current hypothesis, in [0,1], scored
// within the foreground bounding box (the ActiveMask) so the symmetry axis is
// centered on the actual objects rather than the whole canvas.
func (t *HypothesisTester) Satisfaction(grid [][]int) float64 {
	return scopedScore(grid, t.hyps[t.idx].Score)
}

// Observe folds this frame into plateau tracking and returns true if it JUST
// rotated to a new hypothesis (the current one stalled without improving).
func (t *HypothesisTester) Observe(grid [][]int) bool {
	t.pursued++
	// Rotate if the hypothesis has plateaued OR if it's been pursued too long
	// without a win (a slowly-climbing wrong hill would otherwise never plateau).
	if t.pursued >= hypMaxPursue {
		t.rotate()
		return true
	}
	s := t.Satisfaction(grid)
	if s > t.best+1e-9 {
		t.best = s
		t.stale = 0
		return false
	}
	t.stale++
	if t.stale >= t.patience {
		t.rotate()
		return true
	}
	return false
}

// Refute abandons the current hypothesis (e.g. after a reset or GAME_OVER with
// no completion) and moves to the next -- so a fresh attempt tests a fresh guess.
func (t *HypothesisTester) Refute() { t.rotate() }

// WarmStart is called on a LEVEL COMPLETION: the current hypothesis just won, so
// move it to the head of the queue (Bayesian warm start -- the mechanic may
// carry over even though the level's geometry changed) and reset plateau
// tracking so it gets a fresh full patience window on the new level. Rotation
// stays OPEN, unlike a permanent lock: each level may have a different goal, so
// if the warm-started hypothesis stalls on the new level it rotates onward.
func (t *HypothesisTester) WarmStart() {
	t.wins++
	win := t.hyps[t.idx]
	reordered := make([]Hypothesis, 0, len(t.hyps))
	reordered = append(reordered, win)
	for i, h := range t.hyps {
		if i != t.idx {
			reordered = append(reordered, h)
		}
	}
	t.hyps = reordered
	t.idx = 0
	t.best = 0
	t.stale = 0
	t.pursued = 0
}

func (t *HypothesisTester) rotate() {
	t.idx = (t.idx + 1) % len(t.hyps)
	t.best = 0
	t.stale = 0
	t.pursued = 0
}

// ForegroundBBox is the ActiveMask: the inclusive bounding box of all non-
// background cells. Scoring a hypothesis within it (rather than the whole
// canvas) centers the symmetry axis on the actual objects and stops a mostly-
// empty canvas from diluting the measure. ok=false when there is no foreground.
func ForegroundBBox(grid [][]int) (minR, minC, maxR, maxC int, ok bool) {
	bg := BackgroundColor(grid)
	for r := range grid {
		for c := range grid[r] {
			if grid[r][c] == bg {
				continue
			}
			if !ok {
				minR, maxR, minC, maxC, ok = r, r, c, c, true
				continue
			}
			if r < minR {
				minR = r
			}
			if r > maxR {
				maxR = r
			}
			if c < minC {
				minC = c
			}
			if c > maxC {
				maxC = c
			}
		}
	}
	return minR, minC, maxR, maxC, ok
}

// cropTo returns the [minR..maxR]x[minC..maxC] sub-grid (rows shared, read-only).
func cropTo(grid [][]int, minR, minC, maxR, maxC int) [][]int {
	out := make([][]int, 0, maxR-minR+1)
	for r := minR; r <= maxR; r++ {
		out = append(out, grid[r][minC:maxC+1])
	}
	return out
}

// scopedScore evaluates a hypothesis within the foreground bounding box.
func scopedScore(grid [][]int, score func([][]int) float64) float64 {
	if minR, minC, maxR, maxC, ok := ForegroundBBox(grid); ok {
		return score(cropTo(grid, minR, minC, maxR, maxC))
	}
	return score(grid)
}

// PragmaticValue is the COUNTERFACTUAL 1-step lookahead that closes sat onto the
// click choice. For a candidate object it emulates a small set of plausible
// local mutations (remove it; recolor it to the majority foreground color) and
// returns the best resulting Δ(scoped satisfaction) -- an estimate of how much
// clicking this object could advance the CURRENT hypothesis, injected straight
// into the object's action score so a genuinely goal-advancing click spikes
// above exploration noise instead of waiting for slow categorical credit to
// accrue. Both base and counterfactual are scored in the SAME fixed window (the
// current grid's foreground bbox) so they compare like-for-like. 0 when no
// mutation helps (or there is no foreground).
func PragmaticValue(grid [][]int, obj Blob, score func([][]int) float64) float64 {
	minR, minC, maxR, maxC, ok := ForegroundBBox(grid)
	if !ok {
		return 0
	}
	base := score(cropTo(grid, minR, minC, maxR, maxC))
	bg := BackgroundColor(grid)
	best := 0.0
	for _, color := range []int{bg, majorityForeground(grid, bg)} {
		m := mutateCells(grid, obj.Cells, color)
		if d := score(cropTo(m, minR, minC, maxR, maxC)) - base; d > best {
			best = d
		}
	}
	return best
}

// majorityForeground returns the most common non-background color (bg itself if
// there is no foreground). Deterministic tie-break toward the lower color value.
func majorityForeground(grid [][]int, bg int) int {
	counts := map[int]int{}
	for _, row := range grid {
		for _, c := range row {
			if c != bg {
				counts[c]++
			}
		}
	}
	best, bestN := bg, 0
	for c, n := range counts {
		if n > bestN || (n == bestN && c < best) {
			best, bestN = c, n
		}
	}
	return best
}

// mutateCells returns a deep copy of grid with the given cells set to color.
func mutateCells(grid [][]int, cells []Point, color int) [][]int {
	out := make([][]int, len(grid))
	for r := range grid {
		row := make([]int, len(grid[r]))
		copy(row, grid[r])
		out[r] = row
	}
	for _, p := range cells {
		if p.Y >= 0 && p.Y < len(out) && p.X >= 0 && p.X < len(out[p.Y]) {
			out[p.Y][p.X] = color
		}
	}
	return out
}
