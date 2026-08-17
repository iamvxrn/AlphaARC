package macro

import (
	"alphaarc/pkg/environment"
)

// IntentCompiler transforms an abstract macro-program into a queue of API-compliant
// discrete pixel actions (Action6) that exactly match the program's output grid.
type IntentCompiler struct {
	Kinematics *MotorKinematics // If set and calibrated, generates button sequences instead of Action6
}

// Compile compares the current grid to the target grid produced by a macro program
// and returns a queue of actions to reconcile the differences.
func (comp *IntentCompiler) Compile(before, after [][]int) []environment.Action {
	var queue []environment.Action
	
	// Fast diff generation
	var targetPixels []Vector2D
	for r := 0; r < len(after); r++ {
		for c := 0; c < len(after[r]); c++ {
			// If the grid grew, or pixel differs, we must click/paint it.
			if r >= len(before) || c >= len(before[r]) || before[r][c] != after[r][c] {
				// Avoid treating the avatar's own current position as a required change
				if comp.Kinematics != nil && comp.Kinematics.IsCalibrated && after[r][c] == comp.Kinematics.AvatarColor {
					continue
				}
				targetPixels = append(targetPixels, Vector2D{DX: c, DY: r})
			}
		}
	}

	if comp.Kinematics != nil && comp.Kinematics.IsCalibrated {
		// PATHFINDING MODE (Action1-5)
		if len(targetPixels) == 0 {
			return queue
		}
		
		// Find starting position
		curX, curY, err := comp.Kinematics.LocateAvatar(before)
		if err != nil {
			return queue // Avatar lost, can't pathfind
		}

		// Since drawing changes the grid, we pathfind sequentially.
		// Note: we assume the avatar doesn't get blocked by what it draws (or it does, and A* works around it).
		currentGrid := make([][]int, len(before))
		for i := range before {
			currentGrid[i] = make([]int, len(before[i]))
			copy(currentGrid[i], before[i])
		}

		for _, target := range targetPixels {
			path := comp.Kinematics.AStar(currentGrid, curX, curY, target.DX, target.DY)
			if path != nil {
				actions := comp.Kinematics.CompilePath(path)
				queue = append(queue, actions...)
				curX, curY = target.DX, target.DY
			}
		}
	} else {
		// DIRECT CLICK MODE (Action6)
		for _, target := range targetPixels {
			queue = append(queue, environment.Action{
				ID: environment.Action6,
				X:  target.DX,
				Y:  target.DY,
			})
		}
	}
	
	return queue
}
