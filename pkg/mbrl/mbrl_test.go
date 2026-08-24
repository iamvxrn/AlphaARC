package mbrl

import (
	"testing"

	"alphaarc/pkg/actuate"
)

// moveEnv: avatar (colour 2) moved by actions 1=up 2=down 3=left 4=right; goal
// (colour 8) fixed; reward when the avatar reaches the goal.
type moveEnv struct{ ar, ac int }

const startR, startC, goalR, goalC = 1, 1, 5, 5

func (e *moveEnv) render() actuate.Grid {
	g := make(actuate.Grid, 8)
	for r := range g {
		g[r] = make([]int, 8)
	}
	g[goalR][goalC] = 8
	g[e.ar][e.ac] = 2
	return g
}
func (e *moveEnv) Reset() actuate.Grid { e.ar, e.ac = startR, startC; return e.render() }
func (e *moveEnv) Step(c actuate.Control) (actuate.Grid, bool) {
	if c.Kind == "action" {
		nr, nc := e.ar, e.ac
		switch c.ActionID {
		case 1:
			nr--
		case 2:
			nr++
		case 3:
			nc--
		case 4:
			nc++
		}
		if nr >= 0 && nr < 8 && nc >= 0 && nc < 8 {
			e.ar, e.ac = nr, nc
		}
	}
	return e.render(), e.ar == goalR && e.ac == goalC
}

var moveActions = []actuate.Control{
	{Kind: "action", ActionID: 1}, {Kind: "action", ActionID: 2},
	{Kind: "action", ActionID: 3}, {Kind: "action", ActionID: 4},
}

// The whole MBRL loop composes: budget spent LEARNING the model (4 probes), the
// solution SYNTHESISED in imagination (free), then executed to reach the goal.
func TestMBRL_LoopComposesEndToEnd(t *testing.T) {
	res := Solve(&moveEnv{}, 0, moveActions, 60, nil)
	if !res.Won {
		t.Fatalf("MBRL loop should reach the goal, didn't: %+v", res)
	}
	if res.ExploreActions > 4 {
		t.Fatalf("should learn the model in <=4 probes (one per action), spent %d", res.ExploreActions)
	}
	if res.PlanLen != 8 { // manhattan (1,1)->(5,5)
		t.Fatalf("synthesised plan should be the optimal 8 steps, got %d", res.PlanLen)
	}
	if res.Executed != res.PlanLen {
		t.Fatalf("executed (%d) should equal plan length (%d)", res.Executed, res.PlanLen)
	}
	t.Logf("MBRL: %d probes to learn model, %d-step plan synthesised in imagination, executed to win", res.ExploreActions, res.PlanLen)
}

// The learned world model PREDICTS reality: Predict matches the env's actual move.
func TestMBRL_WorldModelPredictsReality(t *testing.T) {
	env := &moveEnv{}
	g := env.Reset()
	m := NewWorldModel(0)
	for _, a := range moveActions {
		before := env.Reset()
		after, _ := env.Step(a)
		m.Observe(before, after, a)
		// prediction from `before` must equal the real `after`
		pred := m.Predict(before, a)
		for r := range after {
			for c := range after[r] {
				if pred[r][c] != after[r][c] {
					t.Fatalf("model mispredicts action %d at (%d,%d): pred=%d real=%d", a.ActionID, r, c, pred[r][c], after[r][c])
				}
			}
		}
	}
	if m.AvatarColor != 2 {
		t.Fatalf("model should identify avatar colour 2, got %d", m.AvatarColor)
	}
	_ = g
}

// Info-gain: unseen controls are surprising; once all observed, exploration stops.
func TestMBRL_InfoGainExhausts(t *testing.T) {
	m := NewWorldModel(0)
	seen := map[int]bool{}
	for i := 0; i < len(moveActions); i++ {
		c, ok := m.NextExploratoryAction(moveActions)
		if !ok {
			t.Fatalf("should still have unseen actions at step %d", i)
		}
		seen[c.ActionID] = true
		m.seen[key(c)] = true
	}
	if _, ok := m.NextExploratoryAction(moveActions); ok {
		t.Fatalf("all actions observed -> no more info to gain, but got one")
	}
	if len(seen) != 4 {
		t.Fatalf("should have explored 4 distinct actions, got %d", len(seen))
	}
}
