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

// GoalHypotheses is the ordered candidate-goal vocabulary the agent tests.
func GoalHypotheses() []Hypothesis {
	return []Hypothesis{
		{"all-one-color", hypAllOneColor},
		{"horizontal-symmetry", hypHorizontalSymmetry},
		{"vertical-symmetry", hypVerticalSymmetry},
		{"halves-match", hypHalvesMatch},
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

// hypRotatePatience is how many steps the pursued hypothesis may go without
// improving its best-seen satisfaction before the tester gives up on it and
// rotates to the next candidate. Sized against a typical level's action budget
// (tens) so a single attempt can try several hypotheses.
const hypRotatePatience = 12

// HypothesisTester pursues one candidate goal at a time and rotates when it
// stalls. The pursued hypothesis's Satisfaction is what the pragmatic loop uses
// as prior preference (so the agent acts to REALIZE the current guess); Observe
// tracks plateau and rotates on a stall, Refute rotates on a reset/loss, and
// Confirm locks the current hypothesis once a real completion validates it.
type HypothesisTester struct {
	hyps      []Hypothesis
	idx       int
	best      float64 // best satisfaction seen for the current hypothesis this attempt
	stale     int     // steps since best improved
	patience  int
	confirmed bool
}

// NewHypothesisTester starts on the first candidate goal.
func NewHypothesisTester() *HypothesisTester {
	return &HypothesisTester{hyps: GoalHypotheses(), patience: hypRotatePatience}
}

// Current is the hypothesis being pursued right now.
func (t *HypothesisTester) Current() Hypothesis { return t.hyps[t.idx] }

// Satisfaction is how well grid meets the current hypothesis, in [0,1].
func (t *HypothesisTester) Satisfaction(grid [][]int) float64 {
	return t.hyps[t.idx].Score(grid)
}

// Observe folds this frame into plateau tracking and returns true if it JUST
// rotated to a new hypothesis (the current one stalled without being met). A
// no-op once a hypothesis has been confirmed.
func (t *HypothesisTester) Observe(grid [][]int) bool {
	if t.confirmed {
		return false
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
// no completion) and moves to the next -- so a fresh attempt tests a fresh
// guess. A no-op once confirmed.
func (t *HypothesisTester) Refute() {
	if t.confirmed {
		return
	}
	t.rotate()
}

// Confirm locks the current hypothesis as validated (a level completed while
// pursuing it), so it stops rotating and keeps pursuing what actually worked.
func (t *HypothesisTester) Confirm() { t.confirmed = true }

// Confirmed reports whether a hypothesis has been validated by a real completion.
func (t *HypothesisTester) Confirmed() bool { return t.confirmed }

func (t *HypothesisTester) rotate() {
	t.idx = (t.idx + 1) % len(t.hyps)
	t.best = 0
	t.stale = 0
}
