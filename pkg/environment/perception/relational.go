package perception

import (
	"fmt"
	"sort"
	"strings"
)

// DescribeGridStructural is the observation for the live predictive cycle:
// the per-blob surface description (DescribeGridCells) PLUS the surface-
// agnostic structural tokens (RelationalTokens). The structural tokens are
// not blob-shaped (no "-cell"), so they never become click targets --
// winningBlobLabel ignores them -- they only enrich the graph and the
// forward model with structure that can transfer across surface-different
// domains, without changing what gets clicked. This is the wiring of the
// representation fix the Stage-6 thermometer motivated: what the agent
// predicts and abstracts over now carries structure, not just surface labels.
func DescribeGridStructural(grid [][]int, maxBlobs, cols, rows int) string {
	base := DescribeGridCells(grid, maxBlobs, cols, rows)
	rel := RelationalTokens(grid)
	if len(rel) == 0 {
		return base
	}
	// Emit the structural tokens structuralWeight times: their SHARE of the
	// observation is the transfer lever (TestExploreStructureWeightOnTransfer /
	// TestStructureWeightImprovesTransfer measured transfer/control dropping
	// from 0.97x at weight 1 to 0.69x at weight 3, then reversing past ~6 as
	// structure drowns the surface detail needed to predict a frame at all).
	// Repetition accumulates on the token's ObservationVector dimension, so the
	// forward model weights structure more heavily and leans on the part that
	// transfers across surface-different domains.
	joined := strings.Join(rel, " ")
	weighted := make([]string, 0, structuralWeight+1)
	if base != "" {
		weighted = append(weighted, base)
	}
	for i := 0; i < structuralWeight; i++ {
		weighted = append(weighted, joined)
	}
	// NOTE: object identity used to be appended here as ObjectSignature tokens,
	// but the exact-pixel-shape hash proved too brittle live (a one-pixel
	// deformation = a whole new identity = fragmentation). Stable object
	// identity now comes from ObjectTracker (continuity-based), fed in by the
	// caller via ChooseClickAction's extraObs, not from a single-frame hash.
	return strings.Join(weighted, " ")
}

// structuralWeight is how many times the structural tokens are repeated in a
// structural observation -- their weight relative to surface tokens. 3 is the
// measured sweet spot (see DescribeGridStructural); tunable, and a live game
// may want re-measuring, but 3 strongly beats 1 without yet drowning surface.
const structuralWeight = 3

// RelationalTokens extracts surface-token-agnostic STRUCTURAL features from a
// grid: the color-multiplicity signature -- how many blobs share each color,
// as sorted group sizes, WITHOUT which colors they are. Two grids that are
// structurally identical but painted in different colors emit the SAME
// relational tokens.
//
// This is the first concrete piece of the representation fix the Stage-6
// transfer thermometer motivated (pkg/pipeline TestStage6TransferThermometer):
// abstraction there did not transfer because every concept node is bound to a
// specific surface token (color5-cell7-0), so a structurally identical domain
// in different colors shares no nodes and no abstraction can carry over.
// A structural token like "cmult3" (three blobs share some color) is identical
// across such domains, giving abstraction something surface-independent to
// bind to. Deliberately NOT yet wired into DescribeGridCells' observation
// string (that ripples through every exact-observation test); this is the
// extractor + its surface-invariance proof, to be wired in and re-measured
// against the thermometer as the next step.
//
// Example: a grid with colors {5,5,5, 8,8} (a 3-blob group and a 2-blob group)
// and a grid with {1,1,1, 2,2} both return ["cmult3", "cmult2"].
func RelationalTokens(grid [][]int) []string {
	background := BackgroundColor(grid)
	blobs := FindBlobs(grid, background)

	perColor := make(map[int]int)
	for _, b := range blobs {
		perColor[b.Color]++
	}

	sizes := make([]int, 0, len(perColor))
	for _, n := range perColor {
		sizes = append(sizes, n)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sizes)))

	tokens := make([]string, len(sizes))
	for i, n := range sizes {
		tokens[i] = fmt.Sprintf("cmult%d", n)
	}
	return tokens
}
