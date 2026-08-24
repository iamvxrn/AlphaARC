package feataff

import (
	"testing"

	"alphaarc/pkg/actuate"
	"alphaarc/pkg/macro"
)

// templateEnv is a reference<->workspace completion game with an INDIRECT protocol
// (like ft09/lp85, where clicking the hole itself is inert): a 4x4 tiling of motif
// [[1,2],[3,4]] with two holes; clicking the matching REFERENCE cell copies its
// colour into the far hole. Clicking anything else is inert. This exercises BOTH
// halves: (1) derive the goal (which cells, what colour) and (2) learn the protocol
// (which control achieves each cell change) and plan it.
type templateEnv struct{ a, b bool }

func (e *templateEnv) grid() actuate.Grid {
	g := actuate.Grid{
		{1, 2, 1, 0}, // hole (0,3) -> want 2
		{3, 4, 3, 4},
		{1, 2, 1, 2},
		{3, 4, 0, 4}, // hole (3,2) -> want 3
	}
	if e.a {
		g[0][3] = 2
	}
	if e.b {
		g[3][2] = 3
	}
	return g
}
func (e *templateEnv) Reset() actuate.Grid { e.a, e.b = false, false; return e.grid() }
func (e *templateEnv) Step(c actuate.Control) actuate.Grid {
	if c.X == 1 && c.Y == 0 { // reference cell (row0,col1)=2 -> fills hole (0,3)
		e.a = true
	}
	if c.X == 0 && c.Y == 3 { // reference cell (row3,col0)=3 -> fills hole (3,2)
		e.b = true
	}
	return e.grid()
}
func (e *templateEnv) complete() bool { return e.a && e.b }

// The full reverse-campaign synthesis loop: derive goal (macro.GoalTargets) ->
// learn the actuation protocol (actuate.CausalMapper) -> plan -> execute -> done.
func TestGoalDerive_DeriveLearnPlanExecute(t *testing.T) {
	bg := 0

	// (1) DERIVE the goal from the grid's own regularity.
	start := (&templateEnv{}).Reset()
	targets := macro.GoalTargets(start, bg)
	if len(targets) != 2 {
		t.Fatalf("goal-deriver should find 2 targets, got %d: %v", len(targets), targets)
	}

	// (2) LEARN the protocol: explore all cell-clicks, record control->effect.
	env := &templateEnv{}
	var cands []actuate.Control
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			cands = append(cands, actuate.Control{Kind: "click", X: c, Y: r})
		}
	}
	m := &actuate.CausalMapper{}
	m.Explore(env, cands)

	// plan the derived goal into the learned (indirect) controls.
	desired := make([]actuate.CellChange, 0, len(targets))
	for _, gt := range targets {
		desired = append(desired, actuate.CellChange{R: gt.R, C: gt.C, To: gt.Want})
	}
	plan, unmet := m.Plan(desired)
	if len(unmet) != 0 {
		t.Fatalf("protocol learning failed to actuate targets: unmet=%v", unmet)
	}

	// (3) EXECUTE the plan from a fresh episode; the workspace must complete.
	exec := &templateEnv{}
	exec.Reset()
	for _, c := range plan {
		exec.Step(c)
	}
	if !exec.complete() {
		t.Fatalf("executing the plan did not complete the template (plan=%v)", plan)
	}
	// the plan must use the INDIRECT reference clicks, not clicks on the holes.
	for _, c := range plan {
		if (c.X == 3 && c.Y == 0) || (c.X == 2 && c.Y == 3) {
			t.Fatalf("plan clicked a (inert) hole directly -- protocol not truly learned: %v", c)
		}
	}
	t.Logf("derive->learn->plan->execute closed: targets=%v plan=%v", targets, plan)
}
