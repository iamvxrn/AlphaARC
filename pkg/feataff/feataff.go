// Package feataff is the bridge between goal selection and actuation: a
// FEATURE-level affordance model. Where pkg/actuate maps controls to CELL
// changes, this maps controls to per-FEATURE deltas (compression, correspondence,
// mismatch, ...) and, crucially, implements goalsel.Experimenter -- so the Goal
// Selector's causal disambiguation runs against real controls.
//
// Because features are entangled (no control isolates one), Probe does NOT try
// to isolate. It selects the most SEPARATING recorded interventions -- the ones
// whose confounded-feature delta vectors are most angularly spread (a greedy
// D-optimal-style pick) plus a mix of reward/no-reward outcomes -- so the
// selector's difference-of-means attribution is well-conditioned.
package feataff

import (
	"math"

	"alphaarc/pkg/actuate"
	"alphaarc/pkg/goalsel"
)

// Feature is a named scalar readout of a grid (compression savings, a mismatch,
// etc.). The library of these is the innate side; which one is the goal is
// learned by goalsel.
type Feature struct {
	Name string
	Eval func(actuate.Grid) float64
}

// Env applies a control and reports the resulting grid plus whether the
// environment's (sparse) reward fired.
type Env interface {
	Reset() actuate.Grid
	Step(actuate.Control) (grid actuate.Grid, reward bool)
}

type observation struct {
	ctrl          actuate.Control
	deltas        map[string]float64 // per-feature change this control caused
	reward        bool
	before, after actuate.Grid // kept so newly-grown features can be back-filled for free
}

// FeatureMapper records control -> per-feature-delta (+reward) and answers the
// selector's Probe with separating interventions.
type FeatureMapper struct {
	features []Feature
	obs      []observation
}

func New(features []Feature) *FeatureMapper {
	return &FeatureMapper{features: append([]Feature(nil), features...)}
}

// Explore applies each control once from a fresh reset and records its
// per-feature effect and reward.
func (m *FeatureMapper) Explore(env Env, controls []actuate.Control) {
	for _, c := range controls {
		before := env.Reset()
		bvals := m.eval(before)
		after, reward := env.Step(c)
		avals := m.eval(after)
		d := make(map[string]float64, len(m.features))
		for _, f := range m.features {
			d[f.Name] = avals[f.Name] - bvals[f.Name]
		}
		m.obs = append(m.obs, observation{ctrl: c, deltas: d, reward: reward, before: before, after: after})
	}
}

// ExploreSequential explores from a SINGLE reset, stepping through the controls
// in order and recording each one's INCREMENTAL per-feature delta (from the grid
// just before that step) plus whether the reward fired. Cost is 1 Reset + N Steps
// instead of Explore's N x (Reset+Step) -- essential on live games with a bounded
// action budget where a reset-per-candidate sweep exhausts the session before it
// can pursue (observed on ft09's ~150-action cap). Deltas are under accumulation
// (state carries over, like real play), which is fine for DISCOVERY: a control
// that raises a feature still shows a positive incremental delta. Returns true
// (and the reward step) if the sparse reward fired during the sweep itself.
func (m *FeatureMapper) ExploreSequential(env Env, controls []actuate.Control) (rewarded bool, atStep int) {
	prev := env.Reset()
	bvals := m.eval(prev)
	for i, c := range controls {
		after, reward := env.Step(c)
		avals := m.eval(after)
		d := make(map[string]float64, len(m.features))
		for _, f := range m.features {
			d[f.Name] = avals[f.Name] - bvals[f.Name]
		}
		m.obs = append(m.obs, observation{ctrl: c, deltas: d, reward: reward, before: prev, after: after})
		bvals = avals
		prev = after
		if reward {
			return true, i
		}
	}
	return false, -1
}

// ObserveStep records one played step (before -> after via ctrl, with reward) into
// the same store Explore uses, computing the per-feature deltas. This is the live
// single-episode driver's primitive: it plays ONE continuous game (one Reset) and
// feeds each real step here, so the affordance model is learned within the action
// budget instead of by a reset-per-candidate sweep that exhausts it.
func (m *FeatureMapper) ObserveStep(before, after actuate.Grid, ctrl actuate.Control, reward bool) {
	bvals, avals := m.eval(before), m.eval(after)
	d := make(map[string]float64, len(m.features))
	for _, f := range m.features {
		d[f.Name] = avals[f.Name] - bvals[f.Name]
	}
	m.obs = append(m.obs, observation{ctrl: ctrl, deltas: d, reward: reward, before: before, after: after})
}

// ControlForCellChange returns a recorded control observed to set cell (r,c) to
// colour `to`, from the kept before/after grids -- the cell-level protocol query
// the goal-deriver needs to actuate a GoalTarget. ok is false if nothing observed
// achieved it (on stateful games no single step will, which is the honest signal).
func (m *FeatureMapper) ControlForCellChange(r, c, to int) (actuate.Control, bool) {
	for _, o := range m.obs {
		if o.before == nil || o.after == nil {
			continue
		}
		if r < len(o.after) && c < len(o.after[r]) && o.after[r][c] == to &&
			(r >= len(o.before) || c >= len(o.before[r]) || o.before[r][c] != to) {
			return o.ctrl, true
		}
	}
	return actuate.Control{}, false
}

// AddFeatures grows the library WITHOUT spending any actions: it appends the new
// features and back-fills their deltas on every stored observation from the kept
// before/after grids. So "grow when stuck" costs nothing on a live budget -- the
// grown families are re-scored against everything already seen this episode.
func (m *FeatureMapper) AddFeatures(newFeats []Feature) {
	for _, nf := range newFeats {
		m.features = append(m.features, nf)
		for i := range m.obs {
			o := &m.obs[i]
			if o.before == nil || o.after == nil {
				continue
			}
			o.deltas[nf.Name] = nf.Eval(o.after) - nf.Eval(o.before)
		}
	}
}

// Record is one recorded control->effect observation (for inspection/telemetry).
type Record struct {
	Control actuate.Control
	Deltas  map[string]float64
	Reward  bool
}

// Records exposes what Explore observed (control, per-feature deltas, reward).
func (m *FeatureMapper) Records() []Record {
	out := make([]Record, 0, len(m.obs))
	for _, o := range m.obs {
		out = append(out, Record{Control: o.ctrl, Deltas: o.deltas, Reward: o.reward})
	}
	return out
}

func (m *FeatureMapper) eval(g actuate.Grid) map[string]float64 {
	v := make(map[string]float64, len(m.features))
	for _, f := range m.features {
		v[f.Name] = f.Eval(g)
	}
	return v
}

// BestControlFor returns the recorded control that most moves `goal` in
// direction `dir` (+1 up / -1 down), and its gain. ok is false if no recorded
// control moves it that way -- the pursuit engine uses this to actuate a
// provisional goal-feature before any reward has confirmed it.
func (m *FeatureMapper) BestControlFor(goal string, dir int) (actuate.Control, float64, bool) {
	var best actuate.Control
	bestGain, found := 0.0, false
	for _, o := range m.obs {
		if g := float64(dir) * o.deltas[goal]; g > bestGain {
			bestGain, best, found = g, o.ctrl, true
		}
	}
	return best, bestGain, found
}

// PursueToReward closes the loop generically: each step it actuates the control
// with the strongest positive gain over ANY feature (not fixed to compression),
// so the goal it chases is whichever feature the world lets it push -- and it
// switches feature freely. Every step feeds goalsel; on the sparse reward it
// returns which feature's pursuit reached it. `intrinsicFirst` (e.g.
// "compression") only breaks ties, as an exploration prior, not a hardcoded goal.
func PursueToReward(env Env, feats []Feature, fm *FeatureMapper, sel goalselObserver, intrinsicFirst string, budget int) (won bool, via string, steps int) {
	// Baseline: give goalsel the pre-pursuit feature values so a reward on the
	// very first step still has a trajectory to attribute the movement over.
	base := env.Reset()
	bvals := make(map[string]float64, len(feats))
	for _, f := range feats {
		bvals[f.Name] = f.Eval(base)
	}
	sel.Observe(bvals, false)

	for steps = 0; steps < budget; steps++ {
		bestFeat, bestGain := "", 0.0
		var bestCtrl actuate.Control
		for _, f := range feats {
			c, g, ok := fm.BestControlFor(f.Name, +1)
			if !ok || g <= 0 {
				continue
			}
			// prefer strictly larger gain; on a near-tie prefer the intrinsic prior
			if g > bestGain+1e-9 || (g > bestGain-1e-9 && f.Name == intrinsicFirst) {
				bestGain, bestFeat, bestCtrl = g, f.Name, c
			}
		}
		if bestFeat == "" {
			return false, "", steps // nothing actuatable -> stuck
		}
		grid, reward := env.Step(bestCtrl)
		vals := make(map[string]float64, len(feats))
		for _, f := range feats {
			vals[f.Name] = f.Eval(grid)
		}
		sel.Observe(vals, reward)
		if reward {
			return true, bestFeat, steps
		}
	}
	return false, "", steps
}

// goalselObserver is the slice of goalsel the pursuit needs (kept as an interface
// so feataff need not import goalsel just for this signature).
type goalselObserver interface {
	Observe(values map[string]float64, reward bool)
}

// cand is one recorded control reduced to its confounded-feature delta vector.
type cand struct {
	vec    []float64
	reward bool
	deltas map[string]float64
}

// Probe implements goalsel.Experimenter: return separating interventions over
// the confounded features. Greedy angular spread over the recorded controls'
// confounded-delta vectors, and it forces in a reward-varying mix when available
// so the selector can attribute by difference-of-means.
func (m *FeatureMapper) Probe(confounded []string) []goalsel.Intervention {
	if len(m.obs) == 0 {
		return nil
	}
	var cands []cand
	for _, o := range m.obs {
		vec := make([]float64, len(confounded))
		nonzero := false
		for i, f := range confounded {
			vec[i] = o.deltas[f]
			if vec[i] != 0 {
				nonzero = true
			}
		}
		if !nonzero {
			continue
		}
		sub := make(map[string]float64, len(confounded))
		for _, f := range confounded {
			sub[f] = o.deltas[f]
		}
		cands = append(cands, cand{vec: vec, reward: o.reward, deltas: sub})
	}
	if len(cands) == 0 {
		return nil
	}

	maxK := len(confounded) + 2
	// greedy: start with the largest-norm vector, then repeatedly add the
	// candidate whose max |cosine| to already-picked is smallest (most new
	// direction). Ensure both reward classes if present.
	picked := []int{argmaxNorm(cands)}
	used := map[int]bool{picked[0]: true}
	for len(picked) < maxK && len(picked) < len(cands) {
		best, bestScore := -1, math.MaxFloat64
		for i := range cands {
			if used[i] {
				continue
			}
			maxCos := 0.0
			for _, p := range picked {
				if cs := math.Abs(cosine(cands[i].vec, cands[p].vec)); cs > maxCos {
					maxCos = cs
				}
			}
			if maxCos < bestScore {
				best, bestScore = i, maxCos
			}
		}
		if best < 0 {
			break
		}
		picked = append(picked, best)
		used[best] = true
	}
	// force in a reward contrast if the greedy pick is single-class
	ensureRewardContrast(cands, used, &picked)

	out := make([]goalsel.Intervention, 0, len(picked))
	for _, i := range picked {
		out = append(out, goalsel.Intervention{Deltas: cands[i].deltas, Reward: cands[i].reward})
	}
	return out
}

func ensureRewardContrast(cands []cand, used map[int]bool, picked *[]int) {
	haveT, haveF := false, false
	for _, i := range *picked {
		if cands[i].reward {
			haveT = true
		} else {
			haveF = true
		}
	}
	add := func(want bool) {
		for i := range cands {
			if !used[i] && cands[i].reward == want {
				*picked = append(*picked, i)
				used[i] = true
				return
			}
		}
	}
	if !haveT {
		add(true)
	}
	if !haveF {
		add(false)
	}
}

func argmaxNorm(cands []cand) int {
	best, bestN := 0, -1.0
	for i := range cands {
		if n := norm(cands[i].vec); n > bestN {
			best, bestN = i, n
		}
	}
	return best
}

func norm(v []float64) float64 {
	s := 0.0
	for _, x := range v {
		s += x * x
	}
	return math.Sqrt(s)
}

func cosine(a, b []float64) float64 {
	dot, na, nb := 0.0, 0.0, 0.0
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
