package bridge

import (
	"context"
	"alphaarc/pkg/environment"
	"alphaarc/pkg/environment/practice"
	"alphaarc/pkg/pipeline"
	"testing"
)

func TestDescribeFrame(t *testing.T) {
	cases := []struct {
		name                       string
		ax, ay, tx, ty             int
		want                       string
	}{
		{"target directly north", 5, 5, 5, 2, "north"},
		{"target directly south", 5, 5, 5, 8, "south"},
		{"target directly east", 5, 5, 8, 5, "east"},
		{"target directly west", 5, 5, 2, 5, "west"},
		{"target north-east (diagonal)", 5, 5, 8, 2, "north east"},
		{"target south-west (diagonal)", 5, 5, 2, 8, "south west"},
		{"already aligned", 5, 5, 5, 5, "aligned"},
	}
	for _, c := range cases {
		got := DescribeFrame(c.ax, c.ay, c.tx, c.ty)
		if got != c.want {
			t.Fatalf("FAIL [%s]: expected %q, got %q", c.name, c.want, got)
		}
	}
}

func TestChooseActionMapsWinningLabelToAction(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()

	action, res, err := ChooseAction(ctx, engine, "east", "reach the beacon")
	if err != nil {
		t.Fatalf("ChooseAction failed: %v", err)
	}
	if action.ID != environment.Action4 {
		t.Fatalf("FAIL: expected Action4 (east) for observation 'east', got %d", action.ID)
	}
	if len(res.ActiveNodeIDs) == 0 {
		t.Fatalf("FAIL: expected a non-empty ActiveNodeIDs behind the decision")
	}
}

// TestBeaconEpisodeSingleAxisReachesWin fully hand-traces the simplest
// possible case: the target is directly north, so DescribeFrame emits only
// "north" every step -- a single candidate concept with no competitor, so
// it wins trivially every cycle regardless of activation dynamics. This is
// the base case that proves the Observe -> Decide -> Act loop can complete
// a real episode end to end.
func TestBeaconEpisodeSingleAxisReachesWin(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	game := practice.NewBeacon(5, 5, 5, 2, 20) // target 3 cells north
	game.Reset()

	steps := runEpisode(t, ctx, engine, game, 20)

	if steps != 3 {
		t.Fatalf("FAIL: expected exactly 3 steps (north x3, single-candidate every cycle), got %d", steps)
	}
	ax, ay := game.AgentPosition()
	if ax != 5 || ay != 2 {
		t.Fatalf("FAIL: expected final agent position (5,2), got (%d,%d)", ax, ay)
	}
}

// TestBeaconEpisodeDiagonalReachesWin hand-traces the richer case where two
// direction concepts compete every cycle (south AND east both needed).
// Because DescribeFrame always emits the vertical word before the
// horizontal one, EnsureConceptNodes always creates the vertical concept's
// node first (lower ID) the first time both appear together, and the
// Stage 2 router's Winner-Takes-All tie-break favors the lower ID on an
// exact activation tie (both stay flat at 1.0 forever, since the router
// narrows to 1 winner every cycle before FormCoActivationEdges could ever
// wire them together) -- so the agent deterministically moves vertically
// until aligned, THEN horizontally (the only remaining candidate) until it
// reaches the target. Hand-traced: south,south,south,east,east -- 5 steps.
func TestBeaconEpisodeDiagonalReachesWin(t *testing.T) {
	ctx := context.Background()
	engine := pipeline.NewEngine()
	game := practice.NewBeacon(0, 0, 2, 3, 20) // 3 south, 2 east
	game.Reset()

	steps := runEpisode(t, ctx, engine, game, 20)

	if steps != 5 {
		t.Fatalf("FAIL: expected exactly 5 steps (south x3 then east x2), got %d", steps)
	}
	ax, ay := game.AgentPosition()
	if ax != 2 || ay != 3 {
		t.Fatalf("FAIL: expected final agent position (2,3), got (%d,%d)", ax, ay)
	}
}

// runEpisode drives the Observe -> Decide -> Act loop until the game
// reaches a terminal state or maxIterations is hit, asserting WIN and
// returning the number of steps taken.
func runEpisode(t *testing.T, ctx context.Context, engine *pipeline.Engine, game *practice.Beacon, maxIterations int) int {
	t.Helper()

	steps := 0
	for i := 0; i < maxIterations; i++ {
		ax, ay := game.AgentPosition()
		tx, ty := game.TargetPosition()
		obs := DescribeFrame(ax, ay, tx, ty)

		action, _, err := ChooseAction(ctx, engine, obs, "reach the beacon")
		if err != nil {
			t.Fatalf("ChooseAction failed at step %d: %v", steps, err)
		}

		frame, err := game.Step(action)
		if err != nil {
			t.Fatalf("Step failed at step %d: %v", steps, err)
		}
		steps++

		if frame.State != environment.StateNotFinished {
			if frame.State != environment.StateWin {
				t.Fatalf("FAIL: episode ended in %s, not WIN, after %d steps", frame.State, steps)
			}
			return steps
		}
	}

	t.Fatalf("FAIL: episode did not terminate within %d iterations", maxIterations)
	return steps
}
