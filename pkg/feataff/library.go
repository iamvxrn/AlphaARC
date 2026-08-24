package feataff

import (
	"alphaarc/pkg/actuate"
	"alphaarc/pkg/environment/perception"
	"alphaarc/pkg/macro"
)

// DefaultFeatures is the innate readout library over a live grid -- the
// candidate goal-features the Goal Selector chooses among. Each computes the
// grid's background itself so it is a pure function of the grid. These are NOT
// goals; which one the environment rewards is learned by goalsel.
func DefaultFeatures() []Feature {
	mk := func(name string, f func([][]int, int) int) Feature {
		return Feature{Name: name, Eval: func(g actuate.Grid) float64 {
			return float64(f(g, perception.BackgroundColor(g)))
		}}
	}
	return []Feature{
		mk("compression", macro.DrivePreference),
		mk("reflect", macro.SymmetryPreference),
		mk("translate", macro.TranslatePreference),
		mk("count", macro.NumerosityPreference),
		mk("correspondence", macro.CorrespondencePreference),
	}
}

// ResidualControls turns perception into REAL candidate controls: the union of
// residual-cluster centroids (the cells that break the best regularity) AND
// object centroids (distinct blobs / buttons). The interactive actuator is often
// a small object that is NOT the largest anomaly, so both sources are needed
// (residual-only missed vc33's grow button). Deduped; a bounded, salient control
// set instead of a blind 64x64 sweep.
func ResidualControls(g actuate.Grid, maxN int) []actuate.Control {
	bg := perception.BackgroundColor(g)
	var cs []actuate.Control
	seen := map[[2]int]bool{}
	add := func(pts []macro.ResidualPoint) {
		for _, p := range pts {
			k := [2]int{p.X, p.Y}
			if seen[k] {
				continue
			}
			seen[k] = true
			cs = append(cs, actuate.Control{Kind: "click", X: p.X, Y: p.Y})
		}
	}
	add(macro.ResidualTargets(g, bg, maxN))
	add(macro.ObjectTargets(g, bg, maxN))
	return cs
}
