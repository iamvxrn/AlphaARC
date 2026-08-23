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
	ctrl   actuate.Control
	deltas map[string]float64 // per-feature change this control caused
	reward bool
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
		m.obs = append(m.obs, observation{ctrl: c, deltas: d, reward: reward})
	}
}

func (m *FeatureMapper) eval(g actuate.Grid) map[string]float64 {
	v := make(map[string]float64, len(m.features))
	for _, f := range m.features {
		v[f.Name] = f.Eval(g)
	}
	return v
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
