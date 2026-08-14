package perception

import (
	"fmt"
	"sort"
)

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
