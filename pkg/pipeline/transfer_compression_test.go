package pipeline

import (
	"context"
	"strings"
	"testing"

	"alphaarc/pkg/environment/perception"
)

// TestCompressionDoesNotDriveTransfer is variant-A step (2)'s finding: making
// abstraction FIRE MORE (lowering CompressionThreshold below the 0.4
// co-activation edge weight) does NOT improve cross-domain transfer.
//
// Reading (2026-08-14): at threshold 0.75 the transfer/control ratio is
// ~0.967x; lowering it to 0.35 (many more abstractions) leaves it ~0.977x --
// unchanged, if anything marginally worse. The reason, and the point of
// keeping this as a permanent instrument: Stage-3 compression collapses a
// whole cluster -- surface blob nodes AND structural cmult nodes together --
// into ONE abstraction node, which is therefore still surface-bound and
// transfers no better. So "lower the compression threshold" is NOT the lever
// for transfer; structure-specific abstraction (separating structure from
// surface) is the real, deeper requirement.
//
// Asserts the mechanism actually works (compression fires at the low
// threshold) and that transfer is not meaningfully improved by it; the exact
// ratios are logged, not hard-asserted.
func TestCompressionDoesNotDriveTransfer(t *testing.T) {
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
	obs := func(grid [][]int) string {
		return perception.DescribeGridCells(grid, 10, 8, 8) + " " + strings.Join(perception.RelationalTokens(grid), " ")
	}
	aX, aY := obs(blobsGrid(5, 3)), obs(blobsGrid(6, 2))
	bX, bY := obs(blobsGrid(1, 3)), obs(blobsGrid(2, 2))

	runAlt := func(e *Engine, x, y string, cycles int) ([]float64, int) {
		errs := make([]float64, cycles)
		comp := 0
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
			comp += res.AbstractionsCreated
		}
		return errs, comp
	}
	measure := func(threshold float64) (ratio float64, comp int) {
		control := NewEngine()
		control.CompressionThreshold = threshold
		cB, _ := runAlt(control, bX, bY, 30)
		transfer := NewEngine()
		transfer.CompressionThreshold = threshold
		_, ca := runAlt(transfer, aX, aY, 25)
		tB, cb := runAlt(transfer, bX, bY, 30)
		return mean(tB[1:9]) / mean(cB[1:9]), ca + cb
	}

	highR, highC := measure(0.75)
	lowR, lowC := measure(0.35)
	t.Logf("threshold=0.75 transfer=%.3fx abstractions=%d | threshold=0.35 transfer=%.3fx abstractions=%d", highR, highC, lowR, lowC)

	if lowC <= highC {
		t.Fatalf("FAIL: lowering the threshold did not fire more compression (%d vs %d) -- mechanism/setup broken", lowC, highC)
	}
	// More compression must not meaningfully IMPROVE transfer -- if it ever
	// does (ratio drops a lot), this finding is stale and worth revisiting.
	if lowR < highR*0.8 {
		t.Fatalf("UNEXPECTED: heavy compression improved transfer (%.3fx vs %.3fx) -- the 'compression doesn't drive transfer' finding no longer holds; revisit", lowR, highR)
	}
}
