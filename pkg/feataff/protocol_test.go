package feataff

import (
	"testing"

	"alphaarc/pkg/actuate"
	"alphaarc/pkg/macro"
)

// paletteTemplateEnv is a STATEFUL reference<->workspace game: to fill a hole you
// must ARM the right colour first (click any reference cell of that colour, which
// lights a visible indicator), THEN click the hole. Clicking a hole unarmed, or
// with the wrong colour armed, is inert. Rows 0-3 are the 2x2-tiled workspace (with
// two holes); row 4 col 0 is the selection indicator.
type paletteTemplateEnv struct {
	armed int
	a, b  bool
}

func (e *paletteTemplateEnv) motif(r, c int) int { return [][]int{{1, 2}, {3, 4}}[r%2][c%2] }

func (e *paletteTemplateEnv) grid() actuate.Grid {
	g := make(actuate.Grid, 5)
	for r := 0; r < 4; r++ {
		g[r] = make([]int, 4)
		for c := 0; c < 4; c++ {
			g[r][c] = e.motif(r, c)
		}
	}
	g[0][3], g[3][2] = 0, 0 // holes
	if e.a {
		g[0][3] = 2
	}
	if e.b {
		g[3][2] = 3
	}
	g[4] = []int{e.armed, 0, 0, 0} // selection indicator (visible arm state)
	return g
}

func (e *paletteTemplateEnv) isHole(x, y int) bool {
	return (x == 3 && y == 0) || (x == 2 && y == 3)
}

func (e *paletteTemplateEnv) Reset() actuate.Grid { e.armed, e.a, e.b = 0, false, false; return e.grid() }

func (e *paletteTemplateEnv) Step(c actuate.Control) (actuate.Grid, bool) {
	if c.Y < 4 && c.X < 4 {
		if e.isHole(c.X, c.Y) {
			if c.X == 3 && c.Y == 0 && e.armed == 2 {
				e.a = true
			}
			if c.X == 2 && c.Y == 3 && e.armed == 3 {
				e.b = true
			}
		} else {
			e.armed = e.motif(c.Y, c.X) // arm the clicked reference cell's colour
		}
	}
	return e.grid(), e.a && e.b
}

// (1) stateful protocol learning + (①) goal derivation, end to end: derive the
// goal from the workspace tiling, plan the ARM->PLACE sequence for each target,
// execute, and complete -- a protocol single-step planning cannot express.
func TestProtocol_StatefulArmThenPlace(t *testing.T) {
	bg := 0

	// ① derive the goal from the cropped workspace (rows 0-3), ignoring the indicator row.
	start := (&paletteTemplateEnv{}).Reset()
	work := start[:4]
	targets := macro.GoalTargets(work, bg)
	if len(targets) != 2 {
		t.Fatalf("goal-deriver should find 2 holes, got %d: %v", len(targets), targets)
	}

	var cands []actuate.Control
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			cands = append(cands, actuate.Control{Kind: "click", X: c, Y: r})
		}
	}

	// (1) plan a STATEFUL sequence for each target; single clicks can't do it.
	var full []actuate.Control
	for _, gt := range targets {
		// a single step cannot achieve it (arming is required first)
		if _, ok := PlanStatefulActuation(&paletteTemplateEnv{}, actuate.CellChange{R: gt.R, C: gt.C, To: gt.Want}, cands, 1); ok {
			t.Fatalf("depth-1 must NOT achieve target %+v (arm is required)", gt)
		}
		seq, ok := PlanStatefulActuation(&paletteTemplateEnv{}, actuate.CellChange{R: gt.R, C: gt.C, To: gt.Want}, cands, 2)
		if !ok || len(seq) != 2 {
			t.Fatalf("stateful planner should find a 2-step arm->place for %+v, got ok=%v seq=%v", gt, ok, seq)
		}
		full = append(full, seq...)
	}

	// (3) execute all sequences in one episode -> template completes.
	exec := &paletteTemplateEnv{}
	exec.Reset()
	var reward bool
	for _, c := range full {
		_, reward = exec.Step(c)
	}
	if !reward {
		t.Fatalf("executing the stateful plan did not complete the template (plan=%v)", full)
	}
	t.Logf("stateful derive->plan->execute closed: targets=%v plan=%v", targets, full)
}
