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

import "math"

// Hypothesis: "the goal is to move Feature in direction Dir (+1 up, -1 down)."
// Credit is accumulated evidence (from rewards and, decisively, interventions).
type Hypothesis struct {
	Feature string
	Dir     int
	Credit  float64
}

// Experimenter runs an isolating intervention: move `move` while holding `keep`
// approximately fixed, and report whether the environment's reward fired. ok is
// false when no control that isolates `move` is known yet (needs more
// affordance learning). This is the causal probe the selector uses to break
// spurious correlations.
type Experimenter interface {
	Intervene(move string, keep []string) (reward bool, ok bool)
}

// GoalSelector maintains candidate goal-hypotheses over a fixed feature set and
// grows belief about which is the real goal from interaction.
type GoalSelector struct {
	features []string
	hyp      map[string]*Hypothesis
	traj     map[string][]float64
	window   int      // steps to look back when attributing a reward
	minDelta float64  // minimum feature move (over the window) to count as "moved"
	credited []string // features credited together on the most recent reward
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

// Disambiguate breaks a confound by active experiment: for each tied feature,
// ask the Experimenter to move it ALONE and check whether the reward follows.
// Isolated movement that reproduces the reward is strong CAUSAL evidence
// (credit up); isolated movement that does NOT is evidence the feature was a
// spurious correlate (credit down). This is the correlation->causation step.
func (s *GoalSelector) Disambiguate(exp Experimenter) bool {
	tied, ok := s.Confounded()
	if !ok {
		return false
	}
	acted := false
	for _, f := range tied {
		keep := others(tied, f)
		reward, ran := exp.Intervene(f, keep)
		if !ran {
			continue // can't isolate this feature yet
		}
		acted = true
		if reward {
			s.hyp[f].Credit += 2 // isolated cause reproduced the reward
		} else {
			s.hyp[f].Credit -= 1 // moved alone, no reward -> spurious correlate
		}
	}
	if acted {
		s.credited = s.credited[:0] // confound resolved for this round
	}
	return acted
}

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

func others(set []string, exclude string) []string {
	var out []string
	for _, x := range set {
		if x != exclude {
			out = append(out, x)
		}
	}
	return out
}
