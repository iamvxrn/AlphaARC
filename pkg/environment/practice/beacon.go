// Package practice provides small, local, offline implementations of
// environment.Environment for developing and testing against before wiring
// up the real ARC-AGI-3 network service (which needs an ARC_API_KEY).
package practice

import "protaxon/pkg/environment"

// Beacon is the simplest possible ARC-AGI-3-shaped game: a marker (color 1)
// must reach a beacon (color 2) on an otherwise-blank 64x64 grid (color 0),
// using the four directional actions. It exists purely to validate the
// Reset/Step/Frame contract end to end, not to be an interesting puzzle.
type Beacon struct {
	agentX, agentY   int
	targetX, targetY int
	startAX, startAY int
	step             int
	maxSteps         int
	state            environment.GameState
}

// NewBeacon creates a Beacon game with the given fixed start/target
// positions, deterministic (no randomness) so episodes are reproducible.
func NewBeacon(startX, startY, targetX, targetY, maxSteps int) *Beacon {
	return &Beacon{
		startAX:  startX,
		startAY:  startY,
		targetX:  targetX,
		targetY:  targetY,
		maxSteps: maxSteps,
		state:    environment.StateNotPlayed,
	}
}

func (b *Beacon) Reset() (environment.Frame, error) {
	b.agentX, b.agentY = b.startAX, b.startAY
	b.step = 0
	b.state = environment.StateNotFinished
	if b.agentX == b.targetX && b.agentY == b.targetY {
		b.state = environment.StateWin
	}
	return b.frame(), nil
}

// Step applies a directional action (Action1=north/-Y, Action2=south/+Y,
// Action3=west/-X, Action4=east/+X; matching the real SDK's convention that
// ACTION1-4 are directional moves). Any other action ID is a no-op move
// (still consumes a step). Movement is clamped to the grid.
func (b *Beacon) Step(action environment.Action) (environment.Frame, error) {
	if b.state != environment.StateNotFinished {
		return b.frame(), nil
	}

	switch action.ID {
	case environment.Action1: // north
		if b.agentY > 0 {
			b.agentY--
		}
	case environment.Action2: // south
		if b.agentY < environment.GridSize-1 {
			b.agentY++
		}
	case environment.Action3: // west
		if b.agentX > 0 {
			b.agentX--
		}
	case environment.Action4: // east
		if b.agentX < environment.GridSize-1 {
			b.agentX++
		}
	}
	b.step++

	if b.agentX == b.targetX && b.agentY == b.targetY {
		b.state = environment.StateWin
	} else if b.step >= b.maxSteps {
		b.state = environment.StateGameOver
	}

	return b.frame(), nil
}

// AgentPosition and TargetPosition expose the current state for tests and
// for the bridge's observation-description logic, without requiring callers
// to decode the rendered grid.
func (b *Beacon) AgentPosition() (int, int)  { return b.agentX, b.agentY }
func (b *Beacon) TargetPosition() (int, int) { return b.targetX, b.targetY }

func (b *Beacon) frame() environment.Frame {
	grid := make([][]int, environment.GridSize)
	for y := range grid {
		grid[y] = make([]int, environment.GridSize)
	}
	grid[b.targetY][b.targetX] = 2
	grid[b.agentY][b.agentX] = 1

	available := []environment.ActionID{environment.ActionReset}
	if b.state == environment.StateNotFinished {
		available = append(available,
			environment.Action1, environment.Action2, environment.Action3, environment.Action4)
	}

	return environment.Frame{
		GameID:           "beacon",
		Grid:             grid,
		State:            b.state,
		LevelsCompleted:  winCount(b.state),
		AvailableActions: available,
	}
}

func winCount(s environment.GameState) int {
	if s == environment.StateWin {
		return 1
	}
	return 0
}
