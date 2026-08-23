// Package goalsel grows the GOAL from raw interaction: it discovers WHICH
// internal feature the environment's (sparse) reward actually tracks, rather
// than hardcoding a per-game goal. Features (compression, correspondence,
// mismatch-to-discovered-regions, ...) are innate readouts; which one IS the
// goal is learned from the reward.
//
// The hard part it handles is CORRELATION vs CAUSATION. If a distractor feature
// happened to move in lock-step with the true goal feature right before a
// reward, naive credit assignment glues them together forever. Instead, when a
// reward credits several features at once (a confound set), the selector runs an
// active experiment: ask the Experimenter to move ONE feature while holding the
// others fixed, and see whether the reward still follows. Only the feature whose
// ISOLATED movement reproduces the reward is the real goal (Pearl's do-operator,
// not correlation).
//
// The selector is decoupled: it never computes features or touches controls. It
// receives feature VALUES + a reward flag (passive phase) and delegates
// interventions to an Experimenter (active phase). This keeps it isolated and
// unit-testable; the feature library and the causal/actuation layer plug in.
package goalsel

import (
	"math"
	"sort"
	"strings"
)

// Hypothesis: "the goal is to move Feature in direction Dir (+1 up, -1 down)."
// Credit is accumulated evidence (from rewards and, decisively, interventions).
type Hypothesis struct {
	Feature string
	Dir     int
	Credit  float64
}

// Intervention is one active experiment's outcome: the ACTUAL per-feature change
// a control produced (features are entangled, so this is a vector, not an
// isolated move) plus whether the reward fired.
type Intervention struct {
	Deltas map[string]float64
	Reward bool
}

// Experimenter runs active experiments to disentangle a confounded feature set.
// In real grids features are coupled -- a step moves distance AND compression
// AND symmetry -- so perfect isolation is usually impossible. Probe therefore
// returns the best SEPARATING controls it can (varied coupling ratios / most
// informative), each with its real feature-delta vector + reward, and the
// selector attributes causality across them. Empty slice = cannot act yet.
type Experimenter interface {
	Probe(confounded []string) []Intervention
}

// GoalSelector maintains candidate goal-hypotheses over a fixed feature set and
// grows belief about which is the real goal from interaction.
type GoalSelector struct {
	features []string
	hyp      map[string]*Hypothesis
	traj     map[string][]float64
	window   int        // steps to look back when attributing a reward
	minDelta float64    // minimum feature move (over the window) to count as "moved"
	credited []string   // features credited together on the most recent reward
	merged   [][]string // inseparable feature groups collapsed into one goal
}

// New builds a selector over the given feature names. window is how many recent
// steps a reward is attributed over; minDelta is the movement threshold.
func New(features []string, window int, minDelta float64) *GoalSelector {
	s := &GoalSelector{
		features: append([]string(nil), features...),
		hyp:      make(map[string]*Hypothesis, len(features)),
		traj:     make(map[string][]float64, len(features)),
		window:   window,
		minDelta: minDelta,
	}
	for _, f := range features {
		s.hyp[f] = &Hypothesis{Feature: f}
	}
	return s
}

// Observe feeds one step of feature values and whether the reward fired this
// step. On a reward it credits every feature that moved past minDelta over the
// look-back window (direction = sign of the move) -- producing a confound set
// when several moved together.
func (s *GoalSelector) Observe(values map[string]float64, reward bool) {
	for _, f := range s.features {
		s.traj[f] = append(s.traj[f], values[f])
	}
	if !reward {
		return
	}
	s.credited = s.credited[:0]
	for _, f := range s.features {
		d := s.windowDelta(f)
		if math.Abs(d) >= s.minDelta {
			h := s.hyp[f]
			h.Credit += 1
			h.Dir = sign(d)
			s.credited = append(s.credited, f)
		}
	}
}

// windowDelta is the feature's change over the last `window` steps.
func (s *GoalSelector) windowDelta(f string) float64 {
	t := s.traj[f]
	if len(t) < 2 {
		return 0
	}
	i := len(t) - 1 - s.window
	if i < 0 {
		i = 0
	}
	return t[len(t)-1] - t[i]
}

// Confounded reports the set of features that a reward credited together and
// that remain tied in credit -- i.e. the selector cannot yet tell which is the
// real cause. ok is true only when >=2 features are so tied.
func (s *GoalSelector) Confounded() ([]string, bool) {
	if len(s.credited) < 2 {
		return nil, false
	}
	// tied iff their credits are within 0.5 of the max among the credited set.
	top := 0.0
	for _, f := range s.credited {
		if c := s.hyp[f].Credit; c > top {
			top = c
		}
	}
	var tied []string
	for _, f := range s.credited {
		if top-s.hyp[f].Credit <= 0.5 {
			tied = append(tied, f)
		}
	}
	if len(tied) < 2 {
		return nil, false
	}
	return tied, true
}

// entangleTol: how close two delta-vectors must be (in cross-ratio) to count as
// "parallel" -- i.e. the features moved in the same proportion.
const entangleTol = 1e-6

// Disambiguate breaks a confound WITHOUT requiring clean isolation (features are
// entangled). It Probes for a set of separating interventions and:
//  1. if the tied features moved in a constant proportion across EVERY
//     intervention (no control ever changed their ratio) they are operationally
//     indistinguishable -> MERGE them into one goal-feature (honest limit, no
//     infinite loop);
//  2. otherwise attribute causality by DIFFERENCE OF MEANS: the feature whose
//     delta is systematically larger on reward-yielding interventions than on
//     non-reward ones is the cause. Entanglement is beaten by the VARIETY of
//     coupling ratios across interventions, not by one perfect lever.
func (s *GoalSelector) Disambiguate(exp Experimenter) bool {
	tied, ok := s.Confounded()
	if !ok {
		return false
	}
	ivs := exp.Probe(tied)
	if len(ivs) == 0 {
		return false
	}

	if parallelSet(tied, ivs) {
		s.merge(tied)
		s.credited = s.credited[:0]
		return true
	}

	// Difference-of-means attribution (magnitude-aware, entanglement-robust).
	haveReward, haveNo := false, false
	for _, iv := range ivs {
		if iv.Reward {
			haveReward = true
		} else {
			haveNo = true
		}
	}
	for _, f := range tied {
		var sr, sn float64
		var nr, nn int
		for _, iv := range ivs {
			if iv.Reward {
				sr += iv.Deltas[f]
				nr++
			} else {
				sn += iv.Deltas[f]
				nn++
			}
		}
		var score float64
		if haveReward && haveNo {
			score = sr/float64(nr) - sn/float64(nn) // Δf bigger when rewarded => cause
		} else {
			// no reward contrast yet: weak fallback, favour the consistent mover
			score = 0
			for _, iv := range ivs {
				score += float64(sign(iv.Deltas[f]))
			}
			score /= float64(len(ivs))
		}
		s.hyp[f].Credit += score
	}
	s.credited = s.credited[:0]
	return true
}

// parallelSet reports whether the tied features' delta sub-vectors are all
// scalar multiples of one another across the interventions (a constant coupling
// ratio) -- i.e. no experiment separated them. Uses pairwise cross-ratios
// against the first intervention with a non-zero tied-vector.
func parallelSet(tied []string, ivs []Intervention) bool {
	if len(tied) < 2 || len(ivs) < 2 {
		// with <2 interventions we cannot claim separability either way; treat as
		// NOT provably parallel so we prefer attribution over premature merge.
		return len(ivs) >= 2 && allParallelPairs(tied, ivs)
	}
	return allParallelPairs(tied, ivs)
}

func allParallelPairs(tied []string, ivs []Intervention) bool {
	// For every pair of features (a,b), the ratio Δa:Δb must be constant across
	// all interventions (cross product ~0 for every pair of interventions).
	for ai := 0; ai < len(tied); ai++ {
		for bi := ai + 1; bi < len(tied); bi++ {
			a, b := tied[ai], tied[bi]
			for i := 0; i < len(ivs); i++ {
				for j := i + 1; j < len(ivs); j++ {
					cross := ivs[i].Deltas[a]*ivs[j].Deltas[b] - ivs[j].Deltas[a]*ivs[i].Deltas[b]
					if math.Abs(cross) > entangleTol {
						return false // this pair's ratio changed -> separable
					}
				}
			}
		}
	}
	return true
}

// merge collapses an inseparable set into one effective goal-feature (sorted
// join). Its credit takes the group's best, and members are demoted so the
// merged feature wins Goal(). Recorded in Merged for inspection.
func (s *GoalSelector) merge(group []string) {
	name := mergedName(group)
	best := 0.0
	for _, f := range group {
		if c := s.hyp[f].Credit; c > best {
			best = c
		}
	}
	if s.hyp[name] == nil {
		s.hyp[name] = &Hypothesis{Feature: name, Dir: 1}
		s.features = append(s.features, name)
	}
	s.hyp[name].Credit = best + 1
	for _, f := range group {
		s.hyp[f].Credit = 0 // subsumed by the merged feature
	}
	s.merged = append(s.merged, append([]string(nil), group...))
}

// Merged returns the inseparable feature groups discovered so far.
func (s *GoalSelector) Merged() [][]string { return s.merged }

// Goal returns the current best goal hypothesis (highest credit, must be
// positive). ok is false before any evidence.
func (s *GoalSelector) Goal() (feature string, dir int, ok bool) {
	best := -math.MaxFloat64
	for _, f := range s.features {
		if h := s.hyp[f]; h.Credit > best {
			best, feature, dir = h.Credit, h.Feature, h.Dir
		}
	}
	if best <= 0 {
		return "", 0, false
	}
	return feature, dir, true
}

// Hypotheses exposes the current hypotheses (for inspection/telemetry).
func (s *GoalSelector) Hypotheses() []Hypothesis {
	out := make([]Hypothesis, 0, len(s.features))
	for _, f := range s.features {
		out = append(out, *s.hyp[f])
	}
	return out
}

func sign(x float64) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

func mergedName(group []string) string {
	g := append([]string(nil), group...)
	sort.Strings(g)
	return strings.Join(g, "+")
}
