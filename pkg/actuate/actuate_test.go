package actuate

import "testing"

// triggerEnv: an INDIRECT-control world. A button at (buttonY,buttonX) paints a
// distant target cell; clicking the target cell (or anywhere else) does nothing.
// This is the vc33/ft09 shape: the actuator is not the cell you want to change.
type triggerEnv struct {
	h, w       int
	bg         int
	buttonX    int // col
	buttonY    int // row
	targetR    int
	targetC    int
	paintColor int
	painted    bool
}

func (e *triggerEnv) Reset() Grid {
	e.painted = false
	return e.render()
}

func (e *triggerEnv) render() Grid {
	g := make(Grid, e.h)
	for r := 0; r < e.h; r++ {
		g[r] = make([]int, e.w)
		for c := 0; c < e.w; c++ {
			g[r][c] = e.bg
		}
	}
	g[e.buttonY][e.buttonX] = 7 // the button is a distinct colour-7 mark
	if e.painted {
		g[e.targetR][e.targetC] = e.paintColor
	}
	return g
}

func (e *triggerEnv) Step(c Control) Grid {
	// Only clicking exactly on the button paints the target. Everything else --
	// including a direct click on the target cell -- does nothing.
	if c.Kind == "click" && c.X == e.buttonX && c.Y == e.buttonY {
		e.painted = true
	}
	return e.render()
}

// The capstone: learn the indirect mapping, then actuate a goalfind-style
// "cell (targetR,targetC) should become paintColor" via the BUTTON, not a direct
// click on the target.
func TestActuate_PaintsTargetViaIndirectTrigger(t *testing.T) {
	env := &triggerEnv{h: 10, w: 10, bg: 0,
		buttonX: 1, buttonY: 1, targetR: 7, targetC: 8, paintColor: 5}

	// Candidate controls the agent tries: the button, the target cell itself,
	// and a couple of inert cells.
	controls := []Control{
		{Kind: "click", X: 8, Y: 7}, // a direct click on the target cell -> inert
		{Kind: "click", X: 1, Y: 1}, // the button -> paints the target
		{Kind: "click", X: 4, Y: 4}, // inert
		{Kind: "click", X: 9, Y: 0}, // inert
	}

	m := &CausalMapper{}
	m.Explore(env, controls)

	// goalfind says: cell (7,8) must become colour 5. Ask the mapper how.
	ctrl, ok := m.ControlForCell(7, 8, 5)
	if !ok {
		t.Fatal("mapper failed to learn any control that paints the target")
	}
	// It must be the BUTTON, not a direct click on the target.
	if !(ctrl.Kind == "click" && ctrl.X == 1 && ctrl.Y == 1) {
		t.Fatalf("expected the indirect button click (1,1), got %+v", ctrl)
	}
	if ctrl.X == 8 && ctrl.Y == 7 {
		t.Fatal("actuator must NOT be a direct click on the target cell")
	}

	// Actuate it and confirm the goal is achieved.
	env.Reset()
	g := env.Step(ctrl)
	if g[7][8] != 5 {
		t.Fatalf("goal not actuated: target cell = %d, want 5", g[7][8])
	}
}

// A direct click on the target must be learned as USELESS (empty effect), so the
// planner never wastes the move on it.
func TestActuate_DirectClickLearnedUseless(t *testing.T) {
	env := &triggerEnv{h: 8, w: 8, bg: 0,
		buttonX: 0, buttonY: 0, targetR: 5, targetC: 5, paintColor: 9}
	m := &CausalMapper{}
	m.Explore(env, []Control{{Kind: "click", X: 5, Y: 5}}) // only the direct click
	if _, ok := m.ControlForCell(5, 5, 9); ok {
		t.Fatal("a direct click that does nothing must not be returned as an actuator")
	}
}

// Plan turns goalfind's desired changes into controls, and dedups when one
// control (the button) satisfies the change.
func TestActuate_PlanFromDesiredChanges(t *testing.T) {
	env := &triggerEnv{h: 8, w: 8, bg: 0,
		buttonX: 2, buttonY: 3, targetR: 6, targetC: 6, paintColor: 4}
	m := &CausalMapper{}
	m.Explore(env, []Control{{Kind: "click", X: 2, Y: 3}, {Kind: "click", X: 6, Y: 6}})

	desired := []CellChange{{R: 6, C: 6, From: 0, To: 4}}
	plan, unmet := m.Plan(desired)
	if len(unmet) != 0 {
		t.Fatalf("all desired changes should be actuatable, unmet=%v", unmet)
	}
	if len(plan) != 1 || !(plan[0].X == 2 && plan[0].Y == 3) {
		t.Fatalf("plan should be the single button click (2,3), got %+v", plan)
	}
}

// ControlForColor: the attribute-ish query "what turns SOME cell colour 5?"
func TestActuate_ControlForColor(t *testing.T) {
	env := &triggerEnv{h: 6, w: 6, bg: 0,
		buttonX: 1, buttonY: 1, targetR: 4, targetC: 4, paintColor: 5}
	m := &CausalMapper{}
	m.Explore(env, []Control{{Kind: "click", X: 1, Y: 1}})
	ctrl, ok := m.ControlForColor(5)
	if !ok || !(ctrl.X == 1 && ctrl.Y == 1) {
		t.Fatalf("ControlForColor(5) should be the button (1,1), got %+v ok=%v", ctrl, ok)
	}
}
