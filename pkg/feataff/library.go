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
	// Each PRIMITIVE is a separate feature (not lumped into one "compression"
	// max) so the Goal Selector can discover WHICH regularity the reward tracks.
	// Lumping them (DrivePreference = max) wrongly subsumes correspondence under
	// compression and makes relational goals inseparable from self-regularity.
	return []Feature{
		mk("reflect", macro.SymmetryPreference),
		mk("translate", macro.TranslatePreference),
		mk("count", macro.NumerosityPreference),
		mk("correspondence", macro.CorrespondencePreference),
	}
}

// GrowFeatures is feature-growth rung 1: when the fixed library explains a grid
// poorly / has nothing to pursue, INVENT new candidate features that capture
// structure the fixed primitives miss -- discovered from the grid, not
// hardcoded. Rung 1 offers a self-parameterizing "discovered-transform" feature
// (macro.DiscoverTransform searches involutions -- rot180/transpose/
// antitranspose -- that Reflect/Translate/Count don't), included only when it
// actually finds a regularity on this grid. Deeper rungs grow color-permutation,
// legend-mapping, and scale-aware relational features.
func GrowFeatures(g actuate.Grid) []Feature {
	// Offer the discovered-transform family unconditionally: its value may be 0 on
	// the CURRENT (e.g. incomplete) grid yet become pursuable once a control moves
	// toward the structure -- the mapper measures that during exploration. The
	// caller grows parsimoniously by only invoking this when the fixed library is
	// STUCK (nothing pursuable), not by gating on the current reading.
	return []Feature{{Name: "discovered-transform", Eval: func(gg actuate.Grid) float64 {
		return float64(macro.DiscoverTransformPreference(gg, perception.BackgroundColor(gg)))
	}}}
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
