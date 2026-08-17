package macro

import (
	"alphaarc/pkg/environment"
)

// IntentCompiler transforms an abstract macro-program into a queue of API-compliant
// discrete pixel actions (Action6) that exactly match the program's output grid.
type IntentCompiler struct{}

// Compile compares the current grid to the target grid produced by a macro program
// and returns a queue of Action6 clicks to reconcile the differences.
func (c *IntentCompiler) Compile(before, after [][]int) []environment.Action {
	var queue []environment.Action
	
	// Fast diff generation
	for r := 0; r < len(after); r++ {
		for c := 0; c < len(after[r]); c++ {
			// If the grid grew, or pixel differs, we must click it.
			if r >= len(before) || c >= len(before[r]) || before[r][c] != after[r][c] {
				// We append an Action6. Note: ARC-AGI-3 coloring usually requires
				// color selection first via Action1-5, but for baseline translation,
				// we enqueue the pixel location. The interactive agent handles color.
				queue = append(queue, environment.Action{
					ID: environment.Action6,
					X:  c,
					Y:  r,
				})
			}
		}
	}
	return queue
}
