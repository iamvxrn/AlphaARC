package pipeline

import (
	"context"
	"strings"
	"testing"

	"alphaarc/pkg/environment/perception"
)

// TestRelationalStructureImprovesTransfer is variant A's payoff measurement:
// does sharing a surface-agnostic STRUCTURAL token (perception.RelationalTokens)
// enable the cross-domain transfer that label-only observations did NOT
// (TestStage6TransferThermometer)?
//
// Setup: the X and Y phases of the alternating stream have DIFFERENT structure
// (a 3-blob group -> "cmult3" vs a 2-blob group -> "cmult2"); domains A and B
// share that structure but are painted in disjoint colors. Measured two ways:
// with only surface labels (the baseline that showed interference), and with
// the structural tokens appended.
//
// Reading (2026-08-14): label-only transfer/control ~1.18x (interference, as
// the thermometer showed), structural ~0.97x (mild POSITIVE transfer) -- so
// the structural token flips the sign. The effect is small because structure
// is still just one token among many surface ones; the direction, not the
// magnitude, is the point. This asserts only the DIRECTIONAL claim (structure
// transfers better than surface labels), the falsifiable core of the
// representation fix; the reading may evolve as more of the representation
// becomes structural.
func TestRelationalStructureImprovesTransfer(t *testing.T) {
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
	obs := func(grid [][]int, withStruct bool) string {
		s := perception.DescribeGridCells(grid, 10, 8, 8)
		if withStruct {
			s += " " + strings.Join(perception.RelationalTokens(grid), " ")
		}
		return s
	}

	aX3, aY2 := blobsGrid(5, 3), blobsGrid(6, 2)
	bX3, bY2 := blobsGrid(1, 3), blobsGrid(2, 2)

	runAlt := func(e *Engine, x, y string, cycles int) []float64 {
		errs := make([]float64, cycles)
		for i := 0; i < cycles; i++ {
			o := x
			if i%2 == 1 {
				o = y
			}
			res, err := e.RunPredictiveCycle(ctx, o, "goal", true)
			if err != nil {
				t.Fatalf("FAIL: cycle %d error: %v", i+1, err)
			}
			errs[i] = res.ForecastError
		}
		return errs
	}

	// transfer/control ratio on domain B's early window (<1 = positive transfer).
	ratio := func(withStruct bool) float64 {
		control := NewEngine()
		cB := runAlt(control, obs(bX3, withStruct), obs(bY2, withStruct), 30)
		transfer := NewEngine()
		_ = runAlt(transfer, obs(aX3, withStruct), obs(aY2, withStruct), 25)
		tB := runAlt(transfer, obs(bX3, withStruct), obs(bY2, withStruct), 30)
		return mean(tB[1:9]) / mean(cB[1:9])
	}

	labelRatio := ratio(false)
	structRatio := ratio(true)
	t.Logf("transfer/control ratio -- label-only=%.3fx  structural=%.3fx (lower = better transfer)", labelRatio, structRatio)

	if !(structRatio < labelRatio) {
		t.Fatalf("FAIL: a shared structural token did not improve transfer over surface labels: structural=%.3fx not < label-only=%.3fx", structRatio, labelRatio)
	}
}
