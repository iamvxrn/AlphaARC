package feataff

import (
	"testing"

	"alphaarc/pkg/actuate"
	"alphaarc/pkg/goalsel"
)

// entangledEnv: clicking a control adds fixed amounts to two ENTANGLED features
// (fk, fdist) from a fresh reset; the HIDDEN reward fires iff fk's increment
// reaches the threshold. No control isolates a feature, but the controls have
// different fk:fdist ratios.
type entangledEnv struct {
	dfk, dfdist  map[actuate.Control]float64
	thr          float64
	fkVal, fdVal float64
}

func (e *entangledEnv) Reset() actuate.Grid {
	e.fkVal, e.fdVal = 0, 0
	// grids are unused here (features are read via the closures below), return 1x1
	return actuate.Grid{{0}}
}

func (e *entangledEnv) Step(c actuate.Control) (actuate.Grid, bool) {
	e.fkVal += e.dfk[c]
	e.fdVal += e.dfdist[c]
	return actuate.Grid{{0}}, e.fkVal >= e.thr
}

// Bridge capstone: FeatureMapper implements goalsel.Experimenter, and the Goal
// Selector uses it to resolve causation among ENTANGLED features on a live env.
func TestBridge_FeatureMapperDrivesCausalDisambiguation(t *testing.T) {
	cBoth := actuate.Control{Kind: "click", X: 1, Y: 1}  // Δfk=5, Δfdist=5, reward
	cFdist := actuate.Control{Kind: "click", X: 2, Y: 2} // Δfk=1, Δfdist=5, no reward
	cFk := actuate.Control{Kind: "click", X: 3, Y: 3}    // Δfk=5, Δfdist=1, reward
	env := &entangledEnv{
		dfk:    map[actuate.Control]float64{cBoth: 5, cFdist: 1, cFk: 5},
		dfdist: map[actuate.Control]float64{cBoth: 5, cFdist: 5, cFk: 1},
		thr:    5,
	}

	// Features read the env's running feature values (a stand-in for real
	// grid-derived readouts; the point here is the bridge/causal loop).
	feats := []Feature{
		{Name: "fk", Eval: func(actuate.Grid) float64 { return env.fkVal }},
		{Name: "fdist", Eval: func(actuate.Grid) float64 { return env.fdVal }},
	}
	fm := New(feats)
	fm.Explore(env, []actuate.Control{cBoth, cFdist, cFk})

	// Selector: passive confound (fk & fdist rose together into a reward), then
	// disambiguate THROUGH the feature mapper (the bridge).
	s := goalsel.New([]string{"fk", "fdist"}, 5, 3.0)
	for step := 1; step <= 5; step++ {
		s.Observe(map[string]float64{"fk": float64(step), "fdist": float64(step)}, step == 5)
	}
	if _, ok := s.Confounded(); !ok {
		t.Fatal("fk/fdist should be confounded before disambiguation")
	}
	if !s.Disambiguate(fm) {
		t.Fatal("Disambiguate should run via the FeatureMapper's Probe")
	}
	feat, dir, ok := s.Goal()
	if !ok || feat != "fk" || dir != 1 {
		t.Fatalf("bridge should resolve the goal to fk(+1), got %q dir=%d hyps=%v", feat, dir, s.Hypotheses())
	}
}

// BestControlFor picks the control that most increases the provisional goal
// feature -- the pursuit engine's actuator selection.
func TestBridge_BestControlForGoal(t *testing.T) {
	cGrow := actuate.Control{Kind: "click", X: 1, Y: 1}
	cShrink := actuate.Control{Kind: "click", X: 2, Y: 2}
	env := &entangledEnv{
		dfk:    map[actuate.Control]float64{cGrow: 5, cShrink: -5},
		dfdist: map[actuate.Control]float64{cGrow: 0, cShrink: 0},
		thr:    1e9, // never reward here; we only test control selection
	}
	feats := []Feature{{Name: "fk", Eval: func(actuate.Grid) float64 { return env.fkVal }}}
	fm := New(feats)
	fm.Explore(env, []actuate.Control{cGrow, cShrink})
	c, gain, ok := fm.BestControlFor("fk", +1)
	if !ok || c != cGrow || gain <= 0 {
		t.Fatalf("should pick the grow control to increase fk, got %+v gain=%.1f ok=%v", c, gain, ok)
	}
	// and the opposite direction picks shrink
	c2, _, ok2 := fm.BestControlFor("fk", -1)
	if !ok2 || c2 != cShrink {
		t.Fatalf("decreasing fk should pick shrink, got %+v", c2)
	}
}

// Probe returns a spread of separating interventions (varied ratios) with a
// reward contrast, from the recorded controls.
func TestBridge_ProbeReturnsSeparatingContrast(t *testing.T) {
	cBoth := actuate.Control{Kind: "click", X: 1, Y: 1}
	cFdist := actuate.Control{Kind: "click", X: 2, Y: 2}
	cFk := actuate.Control{Kind: "click", X: 3, Y: 3}
	env := &entangledEnv{
		dfk:    map[actuate.Control]float64{cBoth: 5, cFdist: 1, cFk: 5},
		dfdist: map[actuate.Control]float64{cBoth: 5, cFdist: 5, cFk: 1},
		thr:    5,
	}
	feats := []Feature{
		{Name: "fk", Eval: func(actuate.Grid) float64 { return env.fkVal }},
		{Name: "fdist", Eval: func(actuate.Grid) float64 { return env.fdVal }},
	}
	fm := New(feats)
	fm.Explore(env, []actuate.Control{cBoth, cFdist, cFk})
	ivs := fm.Probe([]string{"fk", "fdist"})
	if len(ivs) < 2 {
		t.Fatalf("probe should return multiple interventions, got %d", len(ivs))
	}
	haveT, haveF := false, false
	for _, iv := range ivs {
		if iv.Reward {
			haveT = true
		} else {
			haveF = true
		}
	}
	if !haveT || !haveF {
		t.Fatalf("probe should include a reward contrast (T and F), got ivs=%v", ivs)
	}
}

// corrGoalEnv: the reward tracks CORRESPONDENCE, not compression. An exemplar +
// an incomplete copy; a trigger BUTTON fills the copy's missing cell (raising
// correspondence toward a match); reward fires when the copy is completed. A
// direct click on the missing cell is inert. A "goal != compression" world.
type corrGoalEnv struct{ filled bool }

var cgBox = [][]int{
	{2, 2, 2, 2, 2},
	{2, 4, 3, 4, 2},
	{2, 3, 4, 3, 2},
	{2, 4, 3, 4, 2},
	{2, 2, 2, 2, 2},
}

const (
	cgBG                 = 9
	cgButtonX, cgButtonY = 0, 14   // trigger button (row 14, col 0)
	cgMissR, cgMissC     = 11, 8   // copy interior cell (copy top-left is (9,6))
)

// Boxes are placed OFFSET (exemplar (1,1), copy (9,6)) with NO global
// periodicity/symmetry, so completing the copy raises CORRESPONDENCE only --
// reflect/translate/count stay flat (see TestEmergence_SelectsCorrespondence).
func (e *corrGoalEnv) render() actuate.Grid {
	g := make(actuate.Grid, 16)
	for r := range g {
		g[r] = make([]int, 16)
		for c := range g[r] {
			g[r][c] = cgBG
		}
	}
	place := func(top, left int) {
		for r := range cgBox {
			for c := range cgBox[r] {
				g[top+r][left+c] = cgBox[r][c]
			}
		}
	}
	place(1, 1)
	place(9, 6)
	if !e.filled {
		g[cgMissR][cgMissC] = cgBG
	}
	g[cgButtonY][cgButtonX] = 7
	return g
}

func (e *corrGoalEnv) Reset() actuate.Grid { e.filled = false; return e.render() }

func (e *corrGoalEnv) Step(c actuate.Control) (actuate.Grid, bool) {
	newly := false
	if c.Kind == "click" && c.X == cgButtonX && c.Y == cgButtonY && !e.filled {
		e.filled = true
		newly = true
	}
	return e.render(), newly
}

func featureNames(feats []Feature) []string {
	out := make([]string, 0, len(feats))
	for _, f := range feats {
		out = append(out, f.Name)
	}
	return out
}

// GENERALIZATION: the Path-B loop must win a game whose goal is NOT compression,
// selecting a non-compression feature via an indirect trigger -- proving the
// selector isn't hardwired to the compression bootstrap.
func TestPursuit_GeneralizesToNonCompressionGoal(t *testing.T) {
	env := &corrGoalEnv{}
	feats := DefaultFeatures()
	controls := []actuate.Control{
		{Kind: "click", X: cgButtonX, Y: cgButtonY}, // fill button (indirect)
		{Kind: "click", X: cgMissC, Y: cgMissR},     // direct click on the cell (inert)
	}
	fm := New(feats)
	fm.Explore(env, controls)

	sel := goalsel.New(featureNames(feats), 3, 0.5)
	won, via, steps := PursueToReward(env, feats, fm, sel, "translate", 20)
	if !won {
		t.Fatalf("Path-B loop should win the correspondence-goal game, didn't (steps=%d)", steps)
	}
	// The relational goal must be pursued via correspondence, not a self-
	// regularity feature -- proving the selector generalizes beyond compression.
	if via != "correspondence" {
		t.Fatalf("goal is relational; must win via correspondence, got via=%q", via)
	}
	feat, _, ok := sel.Goal()
	if !ok || feat != "correspondence" {
		t.Fatalf("goalsel should select correspondence, got %q ok=%v hyps=%v", feat, ok, sel.Hypotheses())
	}
	t.Logf("won via %q, goalsel goal=%q, steps=%d", via, feat, steps)
}
