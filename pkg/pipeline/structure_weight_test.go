package pipeline

import (
	"context"
	"strings"
	"testing"

	"alphaarc/pkg/environment/perception"
)

// TestStructureWeightImprovesTransfer is variant-A step (2)'s real lever: the
// SHARE of the observation given to surface-agnostic structure, not graph
// compression, is what drives transfer. Repeating the structural tokens (so
// they occupy more of the hashed observation vector) makes the forward model
// lean on the part that transfers across surface-different domains.
//
// Reading (2026-08-14): transfer/control ~0.97x at weight 1 vs ~0.69x at
// weight 3 -- a large gain from simply giving structure more weight. (Past
// ~6 it reverses as structure drowns the surface detail needed to predict a
// frame; hence perception.structuralWeight = 3.) Asserts the directional
// claim (weight 3 transfers meaningfully better than weight 1).
func TestStructureWeightImprovesTransfer(t *testing.T) {
	ctx := context.Background()

	blobsGrid := func(color, n int) [][]int {
		g := make([][]int, n)
		for i := range g {
			g[i] = make([]int, n*2)
		}
		for i := 0; i < n; i++ {
			g[i][i*2] = color
		}
		return g
	}
	obs := func(grid [][]int, weight int) string {
		s := perception.DescribeGridCells(grid, 10, 8, 8)
		rel := strings.Join(perception.RelationalTokens(grid), " ")
		for i := 0; i < weight; i++ {
			s += " " + rel
		}
		return s
	}
	runAlt := func(e *Engine, x, y string, cycles int) []float64 {
		errs := make([]float64, cycles)
		for i := 0; i < cycles; i++ {
			o := x
			if i%2 == 1 {
				o = y
			}
			res, err := e.RunPredictiveCycle(ctx, o, "goal", true)
			if err != nil {
				t.Fatalf("FAIL: cycle %d: %v", i+1, err)
			}
			errs[i] = res.ForecastError
		}
		return errs
	}
	ratio := func(weight int) float64 {
		aX, aY := obs(blobsGrid(5, 3), weight), obs(blobsGrid(6, 2), weight)
		bX, bY := obs(blobsGrid(1, 3), weight), obs(blobsGrid(2, 2), weight)
		control := NewEngine()
		cB := runAlt(control, bX, bY, 30)
		transfer := NewEngine()
		_ = runAlt(transfer, aX, aY, 25)
		tB := runAlt(transfer, bX, bY, 30)
		return mean(tB[1:9]) / mean(cB[1:9])
	}

	light := ratio(1)
	heavy := ratio(3)
	t.Logf("transfer/control -- structural weight 1 = %.3fx, weight 3 = %.3fx (lower = better transfer)", light, heavy)
	if !(heavy < light*0.9) {
		t.Fatalf("FAIL: weighting structure did not meaningfully improve transfer: weight-3 %.3fx not < 0.9 * weight-1 %.3fx", heavy, light)
	}
}
