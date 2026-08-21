package macro

// DrivePreference is the intrinsic goal signal for the closed sensorimotor loop:
// the number of bits saved by the best-compressing primitive on this grid. It is
// the agent's ONLY drive — there is no hand-authored goal. Maximizing it pulls
// the agent toward whatever regularity (symmetry, periodicity, repeated objects)
// this grid can be compressed by.
//
// This is exactly what the live cmd/alphaarc-play preferenceOf will consume, and
// what the sterile mock-loop (active_inference_test.go) optimizes to prove the
// drive's physics: because compression savings are second-order (a lone cell
// saves nothing until its partner exists), a 1-step gradient is FLAT from sparse
// states, and only planning (lookahead) crosses the valley. That planning gap is
// the Active Inference half of the loop — the static selector alone cannot act.
func DrivePreference(grid [][]int, bg int) int {
	_, savings := BestPrimitive(grid, bg)
	return savings
}

// DriveScore normalizes DrivePreference to [0,1] by the foreground mass, so it is
// scale-comparable to the old [0,1] hypothesis scores when it replaces them as the
// goal signal in the live loop (both in the state preference and as the score
// function the affordance rollout / class-macro value optimize). 0 on a blank grid.
func DriveScore(grid [][]int, bg int) float64 {
	fg := 0
	for _, row := range grid {
		for _, v := range row {
			if v != bg {
				fg++
			}
		}
	}
	if fg == 0 {
		return 0
	}
	return float64(DrivePreference(grid, bg)) / float64(fg)
}
