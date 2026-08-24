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
	cgButtonX, cgButtonY = 0, 14 // trigger button (row 14, col 0)
	cgMissR, cgMissC     = 11, 8 // copy interior cell (copy top-left is (9,6))
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

// rot180Env: the reward tracks 180-degree ROTATIONAL symmetry -- a regularity
// the FIXED library (reflect=mirror, translate=period, count, correspondence)
// does not capture. A button completes the point-symmetry; direct clicks inert.
// Feature-growth must invent the discovered-transform feature to win.
type rot180Env struct{ filled bool }

func (e *rot180Env) render() actuate.Grid {
	g := make(actuate.Grid, 8)
	for r := range g {
		g[r] = make([]int, 8)
	}
	// A pattern that is rot180-symmetric ONLY once the partners are filled; it is
	// NOT mirror-symmetric or periodic (so reflect/translate stay ~0).
	g[1][1], g[1][2], g[2][1] = 1, 2, 3 // top-left marks
	if e.filled {
		g[6][6], g[6][5], g[5][6] = 1, 2, 3 // rot180 partners: (r,c)->(7-r,7-c)
	}
	g[7][0] = 7 // the trigger button
	return g
}

func (e *rot180Env) Reset() actuate.Grid { e.filled = false; return e.render() }

func (e *rot180Env) Step(c actuate.Control) (actuate.Grid, bool) {
	newly := false
	if c.Kind == "click" && c.X == 0 && c.Y == 7 && !e.filled {
		e.filled = true
		newly = true
	}
	return e.render(), newly
}

// Piece (4) capstone: the fixed library CANNOT pursue this game (no feature
// moves), but growing the discovered-transform feature closes the gap and the
// loop wins via it -- feature growth reaching a goal the fixed set couldn't.
func TestGrowth_InventsFeatureToWinWhenLibraryStuck(t *testing.T) {
	button := actuate.Control{Kind: "click", X: 0, Y: 7}
	direct := actuate.Control{Kind: "click", X: 6, Y: 6}
	controls := []actuate.Control{button, direct}

	// 1) FIXED library is stuck: nothing to pursue.
	fixed := DefaultFeatures()
	env1 := &rot180Env{}
	fm1 := New(fixed)
	fm1.Explore(env1, controls)
	sel1 := goalsel.New(featureNames(fixed), 3, 0.5)
	won, _, steps := PursueToReward(env1, fixed, fm1, sel1, "translate", 20)
	if won {
		t.Fatal("fixed library should NOT be able to win the rot180 game")
	}
	if steps != 0 {
		t.Fatalf("fixed library should be stuck immediately (no pursuable feature), pursued %d", steps)
	}

	// 2) GROW features on the grid, retry -> now it wins via the invented feature.
	env2 := &rot180Env{}
	grown := append(DefaultFeatures(), GrowFeatures(env2.render())...)
	hasGrown := false
	for _, f := range grown {
		if f.Name == "discovered-transform" {
			hasGrown = true
		}
	}
	if !hasGrown {
		t.Fatal("GrowFeatures should have invented the discovered-transform feature")
	}
	fm2 := New(grown)
	fm2.Explore(env2, controls)
	sel2 := goalsel.New(featureNames(grown), 3, 0.5)
	won2, via, steps2 := PursueToReward(env2, grown, fm2, sel2, "translate", 20)
	if !won2 {
		t.Fatalf("with the grown feature the loop should win, didn't (steps=%d)", steps2)
	}
	if via != "discovered-transform" {
		t.Fatalf("should win via the invented feature, got via=%q", via)
	}
	t.Logf("feature growth closed the gap: won via %q after fixed library was stuck", via)
}

// colorPermEnv: reward tracks symmetry UP TO A COLOUR SWAP. Left marks {1,2};
// a button fills the mirror partners RECOLOURED to {5,6} (1<->5, 2<->6), so the
// grid becomes reflect-symmetric only after recolouring. Exact reflect and the
// discovered-transform (exact involutions) stay 0; only color-perm-symmetry rises.
type colorPermEnv struct{ filled bool }

func (e *colorPermEnv) render() actuate.Grid {
	g := make(actuate.Grid, 6)
	for r := range g {
		g[r] = make([]int, 6)
	}
	// left block (cols 1,2)
	g[1][1], g[1][2] = 1, 2
	g[2][1], g[2][2] = 2, 1
	if e.filled {
		// mirror partners (cols 4,3) recoloured 1->5, 2->6
		g[1][4], g[1][3] = 5, 6
		g[2][4], g[2][3] = 6, 5
	}
	g[5][0] = 7 // button
	return g
}

func (e *colorPermEnv) Reset() actuate.Grid { e.filled = false; return e.render() }

func (e *colorPermEnv) Step(c actuate.Control) (actuate.Grid, bool) {
	newly := false
	if c.Kind == "click" && c.X == 0 && c.Y == 5 && !e.filled {
		e.filled = true
		newly = true
	}
	return e.render(), newly
}

// The NEW growable family closes a gap that neither the fixed library nor the
// first grown family (discovered-transform, exact involutions) can: symmetry up
// to a colour permutation.
func TestGrowth_ColorPermFamilyClosesTn36LikeGap(t *testing.T) {
	button := actuate.Control{Kind: "click", X: 0, Y: 5}
	direct := actuate.Control{Kind: "click", X: 4, Y: 1}
	controls := []actuate.Control{button, direct}

	// fixed library is stuck
	env1 := &colorPermEnv{}
	fixed := DefaultFeatures()
	fm1 := New(fixed)
	fm1.Explore(env1, controls)
	sel1 := goalsel.New(featureNames(fixed), 3, 0.5)
	if won, _, steps := PursueToReward(env1, fixed, fm1, sel1, "translate", 20); won || steps != 0 {
		t.Fatalf("fixed library should be stuck on the colour-perm game (won=%v steps=%d)", won, steps)
	}

	// grow -> wins specifically via color-perm-symmetry (not discovered-transform)
	env2 := &colorPermEnv{}
	grown := append(DefaultFeatures(), GrowFeatures(env2.render())...)
	fm2 := New(grown)
	fm2.Explore(env2, controls)
	sel2 := goalsel.New(featureNames(grown), 3, 0.5)
	won, via, steps := PursueToReward(env2, grown, fm2, sel2, "translate", 20)
	if !won {
		t.Fatalf("with the color-perm family the loop should win, didn't (steps=%d)", steps)
	}
	if via != "color-perm-symmetry" {
		t.Fatalf("must win via the color-perm family (not %q) -- it's the gap-closer here", via)
	}
	t.Logf("color-perm growth closed the tn36-like gap: won via %q", via)
}

// hiddenTriggerEnv: a partial reflect-symmetric grid whose actuator is a HIDDEN
// bg cell far from the pattern (like vc33's off-pattern grow button / ft09's
// inert mismatched block). Clicking the trigger completes the mirror -> reflect
// rises -> reward. Every other cell is inert. The trigger is NOT a residual or
// object centroid, so the perceptual control set can never offer it.
type hiddenTriggerEnv struct{ filled bool }

const htTrigX, htTrigY = 8, 8 // bg cell, off the pattern; hit by a step-4 sweep

func (e *hiddenTriggerEnv) render() actuate.Grid {
	g := make(actuate.Grid, 12)
	for r := range g {
		g[r] = make([]int, 12)
	}
	// left vertical bar col 1, rows 1..4 (colour 3)
	for r := 1; r <= 4; r++ {
		g[r][1] = 3
	}
	if e.filled {
		// mirror partner col 10 (reflectH: c <-> 11-c) -> exact reflect symmetry
		for r := 1; r <= 4; r++ {
			g[r][10] = 3
		}
	}
	return g
}

func (e *hiddenTriggerEnv) Reset() actuate.Grid { e.filled = false; return e.render() }

func (e *hiddenTriggerEnv) Step(c actuate.Control) (actuate.Grid, bool) {
	newly := false
	if c.Kind == "click" && c.X == htTrigX && c.Y == htTrigY && !e.filled {
		e.filled = true
		newly = true
	}
	return e.render(), newly
}

// Reachability capstone: the perceptual control set leaves the loop STUCK (the
// trigger isn't salient), but causal control discovery (sweep -> keep clicks that
// actually change the board) finds the real actuator and the loop wins.
func TestReachability_CausalControlDiscoveryWinsWhenTriggerNotSalient(t *testing.T) {
	feats := DefaultFeatures()

	// The trigger must NOT be in the perceptual control set (the crux).
	env0 := &hiddenTriggerEnv{}
	unfilled := env0.Reset()
	for _, c := range ResidualControls(unfilled, 12) {
		if c.X == htTrigX && c.Y == htTrigY {
			t.Fatalf("test invalid: trigger IS a perceptual control; it must be hidden")
		}
	}

	// (A) perceptual controls only -> stuck
	envA := &hiddenTriggerEnv{}
	fmA := New(feats)
	fmA.Explore(envA, ResidualControls(envA.Reset(), 12))
	selA := goalsel.New(featureNames(feats), 3, 0.5)
	if won, _, steps := PursueToReward(envA, feats, fmA, selA, "reflect", 20); won || steps != 0 {
		t.Fatalf("perceptual controls should leave the loop stuck (won=%v steps=%d)", won, steps)
	}

	// (B) causal control discovery finds the hidden trigger -> win
	envB := &hiddenTriggerEnv{}
	reach := ReachableControls(envB, SweepControls(envB.Reset(), 4))
	if len(reach) == 0 {
		t.Fatalf("causal discovery found no reachable control")
	}
	foundTrigger := false
	for _, c := range reach {
		if c.X == htTrigX && c.Y == htTrigY {
			foundTrigger = true
		}
	}
	if !foundTrigger {
		t.Fatalf("causal discovery missed the trigger; reach=%v", reach)
	}
	fmB := New(feats)
	fmB.Explore(envB, reach)
	selB := goalsel.New(featureNames(feats), 3, 0.5)
	won, via, steps := PursueToReward(envB, feats, fmB, selB, "reflect", 20)
	if !won {
		t.Fatalf("with causal control discovery the loop should win, didn't (steps=%d)", steps)
	}
	t.Logf("reachability closed it: %d reachable of swept, won via %q at step %d", len(reach), via, steps)
}
