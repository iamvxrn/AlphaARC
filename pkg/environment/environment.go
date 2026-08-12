// Package environment defines Protaxon's side of an ARC-AGI-3-shaped
// interaction loop: reset a level, submit one action, get back a frame.
//
// The types here deliberately mirror the OFFICIAL ARC-AGI-3 Python SDK
// (arcprize/ARC-AGI-3-Agents, package `arcengine`) rather than an invented
// shape -- confirmed 2026-08-12 against that repository's real source
// (agents/agent.go equivalent: agents/agent.py, agents/templates/random_agent.py,
// tests/unit/test_core.py) rather than guessed: a 64x64 grid of 16 colors,
// GameState with exactly 4 values (NOT_PLAYED, NOT_FINISHED, GAME_OVER,
// WIN), and GameAction IDs 0 (RESET) through 7 (undo), with ID 6 being the
// one "complex" action requiring explicit X/Y coordinates.
//
// No real network client exists yet -- that needs an ARC_API_KEY the user
// has to obtain from arcprize.org. This package only defines the interface
// and the types; pkg/environment/practice provides one local, offline
// implementation for developing and testing against before wiring up the
// real service.
package environment

// GameState mirrors arcengine.GameState.
type GameState string

const (
	StateNotPlayed   GameState = "NOT_PLAYED"
	StateNotFinished GameState = "NOT_FINISHED"
	StateGameOver    GameState = "GAME_OVER"
	StateWin         GameState = "WIN"
)

// ActionID mirrors arcengine.GameAction's numeric IDs.
type ActionID int

const (
	ActionReset ActionID = 0
	Action1     ActionID = 1
	Action2     ActionID = 2
	Action3     ActionID = 3
	Action4     ActionID = 4
	Action5     ActionID = 5
	Action6     ActionID = 6 // "complex": requires X/Y
	Action7Undo ActionID = 7
)

// GridSize and NumColors match the real ARC-AGI-3 environment exactly.
const (
	GridSize  = 64
	NumColors = 16
)

// Action is one submitted move. X/Y are only meaningful when ID==Action6
// (the one action requiring explicit coordinates), each in [0, GridSize).
type Action struct {
	ID ActionID
	X  int
	Y  int
}

// IsComplex reports whether this action requires X/Y data (only Action6,
// per the real SDK's GameAction.is_complex()).
func (a Action) IsComplex() bool {
	return a.ID == Action6
}

// Frame is one observation, returned after Reset or Step.
type Frame struct {
	GameID           string
	Grid             [][]int // GridSize x GridSize, values in [0, NumColors)
	State            GameState
	LevelsCompleted  int
	AvailableActions []ActionID
}

// Environment is Protaxon's side of the ARC-AGI-3 interaction loop.
// Implementations: a real network client (not built yet), or a local
// practice environment (pkg/environment/practice) for offline development.
type Environment interface {
	// Reset starts or restarts the current level, returning the first frame.
	Reset() (Frame, error)
	// Step submits one action and returns the resulting frame.
	Step(action Action) (Frame, error)
}
