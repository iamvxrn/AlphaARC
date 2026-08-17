package practice

import (
	"alphaarc/pkg/environment"
	"testing"
)

func TestBeaconResetPlacesAgentAndTarget(t *testing.T) {
	b := NewBeacon(5, 5, 10, 8, 100)
	frame, err := b.Reset()
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
	if frame.State != environment.StateNotFinished {
		t.Fatalf("FAIL: expected NOT_FINISHED after reset, got %s", frame.State)
	}
	if frame.Grid[5][5] != 1 {
		t.Fatalf("FAIL: expected agent marker (1) at (5,5), got %d", frame.Grid[5][5])
	}
	if frame.Grid[8][10] != 2 {
		t.Fatalf("FAIL: expected target marker (2) at (10,8), got %d", frame.Grid[8][10])
	}
	if len(frame.Grid) != environment.GridSize || len(frame.Grid[0]) != environment.GridSize {
		t.Fatalf("FAIL: expected a %dx%d grid, got %dx%d", environment.GridSize, environment.GridSize, len(frame.Grid), len(frame.Grid[0]))
	}
}

func TestBeaconResetOnAlreadyAlignedStartIsImmediateWin(t *testing.T) {
	b := NewBeacon(3, 3, 3, 3, 100)
	frame, _ := b.Reset()
	if frame.State != environment.StateWin {
		t.Fatalf("FAIL: expected immediate WIN when start==target, got %s", frame.State)
	}
}

func TestBeaconStepMovesInEachDirection(t *testing.T) {
	b := NewBeacon(5, 5, 50, 50, 100)
	b.Reset()

	frame, _ := b.Step(environment.Action{ID: environment.Action1}) // north
	if x, y := b.AgentPosition(); x != 5 || y != 4 {
		t.Fatalf("FAIL: expected (5,4) after north, got (%d,%d)", x, y)
	}
	if frame.State != environment.StateNotFinished {
		t.Fatalf("FAIL: expected NOT_FINISHED mid-episode, got %s", frame.State)
	}

	b.Step(environment.Action{ID: environment.Action2}) // south (back to 5)
	if x, y := b.AgentPosition(); x != 5 || y != 5 {
		t.Fatalf("FAIL: expected (5,5) after south, got (%d,%d)", x, y)
	}

	b.Step(environment.Action{ID: environment.Action4}) // east
	if x, y := b.AgentPosition(); x != 6 || y != 5 {
		t.Fatalf("FAIL: expected (6,5) after east, got (%d,%d)", x, y)
	}

	b.Step(environment.Action{ID: environment.Action3}) // west (back to 5)
	if x, y := b.AgentPosition(); x != 5 || y != 5 {
		t.Fatalf("FAIL: expected (5,5) after west, got (%d,%d)", x, y)
	}
}

func TestBeaconMovementClampsAtGridBoundary(t *testing.T) {
	b := NewBeacon(0, 0, 50, 50, 100)
	b.Reset()
	b.Step(environment.Action{ID: environment.Action1}) // north, already at y=0
	b.Step(environment.Action{ID: environment.Action3}) // west, already at x=0
	if x, y := b.AgentPosition(); x != 0 || y != 0 {
		t.Fatalf("FAIL: expected position clamped at (0,0), got (%d,%d)", x, y)
	}
}

func TestBeaconReachingTargetTriggersWin(t *testing.T) {
	b := NewBeacon(0, 0, 2, 0, 100)
	b.Reset()
	b.Step(environment.Action{ID: environment.Action4})
	frame, _ := b.Step(environment.Action{ID: environment.Action4})
	if frame.State != environment.StateWin {
		t.Fatalf("FAIL: expected WIN on reaching target, got %s", frame.State)
	}
	if frame.LevelsCompleted != 1 {
		t.Fatalf("FAIL: expected LevelsCompleted==1 on WIN, got %d", frame.LevelsCompleted)
	}
}

func TestBeaconExceedingMaxStepsTriggersGameOver(t *testing.T) {
	b := NewBeacon(0, 0, 63, 63, 3) // far target, only 3 steps allowed
	b.Reset()
	var frame environment.Frame
	for i := 0; i < 3; i++ {
		frame, _ = b.Step(environment.Action{ID: environment.Action4})
	}
	if frame.State != environment.StateGameOver {
		t.Fatalf("FAIL: expected GAME_OVER after exceeding maxSteps, got %s", frame.State)
	}
}

func TestBeaconStepAfterTerminalStateIsNoOp(t *testing.T) {
	b := NewBeacon(0, 0, 1, 0, 100)
	b.Reset()
	frame, _ := b.Step(environment.Action{ID: environment.Action4}) // reaches target -> WIN
	if frame.State != environment.StateWin {
		t.Fatalf("setup failed: expected WIN, got %s", frame.State)
	}

	again, _ := b.Step(environment.Action{ID: environment.Action3}) // should be ignored
	if x, y := b.AgentPosition(); x != 1 || y != 0 {
		t.Fatalf("FAIL: expected position to stay at (1,0) after a post-WIN step, got (%d,%d)", x, y)
	}
	if again.State != environment.StateWin {
		t.Fatalf("FAIL: expected state to remain WIN after a post-terminal step, got %s", again.State)
	}
}

func TestBeaconStepBeforeResetIsNoOp(t *testing.T) {
	b := NewBeacon(0, 0, 5, 5, 100)
	frame, err := b.Step(environment.Action{ID: environment.Action4})
	if err != nil {
		t.Fatalf("Step before Reset returned an error: %v", err)
	}
	if frame.State != environment.StateNotPlayed {
		t.Fatalf("FAIL: expected NOT_PLAYED for a Step before any Reset, got %s", frame.State)
	}
}
