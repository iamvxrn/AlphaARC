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
	// Object-level observation (Core Knowledge #3): the raw per-cell position
	// labels (DescribeGridCells) are DROPPED. They created a graph node per
	// (color, cell) and exploded the graph (~2900 edges, 77 clusters live),
	// and baked identity into position so a moving body fragmented. The graph
	// now sees only structural tokens here plus the stable object-id + motion
	// tokens the caller adds via extraObs -- vertices are objects, not cells.
	// (maxBlobs/cols/rows kept for signature stability; RelationalTokens and
	// the object tokens don't need a lattice.)
	_ = maxBlobs
	_ = cols
	_ = rows
	rel := RelationalTokens(grid)
	if len(rel) == 0 {
		return ""
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
	weighted := make([]string, 0, structuralWeight)
	for i := 0; i < structuralWeight; i++ {
		weighted = append(weighted, joined)
	}
	// Stable object identity + motion are added by the caller via extraObs
	// (ObjectTracker), not here -- see ChooseClickAction.
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

// SpatialRelationTokens is the missing RELATIONAL half of Core-Knowledge
// perception: beyond "which objects exist" it reports how they RELATE --
// touching, containment (inside/outside), and alignment. These feed the
// observation (like the object-id and numeric tokens), giving the graph and
// forward model surface-agnostic relational structure to bind concepts to
// ("click the object that's inside another") and to transfer across levels.
// Emitted as bounded structural COUNTS (color/id-agnostic, capped at 9) so they
// stay transfer-friendly and don't explode the token space:
//   - "touch<k>":  pairs of different objects that are 4-adjacent.
//   - "inside<k>": objects walled off from the border by other foreground
//                  (enclosed -> "inside"); the topological inside/outside prior.
//   - "aligned<k>": pairs of objects sharing a centroid row or column.
func SpatialRelationTokens(grid [][]int) []string {
	bg := BackgroundColor(grid)
	blobs := FindBlobs(grid, bg)
	var tokens []string
	if t := countTouchingPairs(grid, blobs, bg); t > 0 {
		tokens = append(tokens, fmt.Sprintf("touch%d", min(t, 9)))
	}
	if c := countContained(grid, blobs, bg); c > 0 {
		tokens = append(tokens, fmt.Sprintf("inside%d", min(c, 9)))
	}
	if a := countAlignedPairs(blobs); a > 0 {
		tokens = append(tokens, fmt.Sprintf("aligned%d", min(a, 9)))
	}
	return tokens
}

// countTouchingPairs counts distinct pairs of DIFFERENT objects that are
// 4-adjacent anywhere (objects in contact -- a Core-Knowledge relation).
func countTouchingPairs(grid [][]int, blobs []Blob, bg int) int {
	h := len(grid)
	if h == 0 {
		return 0
	}
	w := len(grid[0])
	idx := make([][]int, h) // cell -> blob index+1 (0 = background)
	for r := range idx {
		idx[r] = make([]int, w)
	}
	for i, b := range blobs {
		for _, p := range b.Cells {
			if p.Y >= 0 && p.Y < h && p.X >= 0 && p.X < w {
				idx[p.Y][p.X] = i + 1
			}
		}
	}
	seen := map[[2]int]bool{}
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			a := idx[r][c]
			if a == 0 {
				continue
			}
			for _, d := range neighbors4 {
				nr, nc := r+d[0], c+d[1]
				if nr < 0 || nr >= h || nc < 0 || nc >= w {
					continue
				}
				b := idx[nr][nc]
				if b == 0 || b == a {
					continue
				}
				lo, hi := a, b
				if lo > hi {
					lo, hi = hi, lo
				}
				seen[[2]int{lo, hi}] = true
			}
		}
	}
	return len(seen)
}

// countContained counts objects that are ENCLOSED -- none of their cells touches
// background reachable from the grid border, i.e. they're walled off by other
// foreground ("inside" something).
func countContained(grid [][]int, blobs []Blob, bg int) int {
	h := len(grid)
	if h == 0 {
		return 0
	}
	w := len(grid[0])
	reach := make([][]bool, h) // background reachable from the border
	for i := range reach {
		reach[i] = make([]bool, w)
	}
	var q []Point
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			if (r == 0 || r == h-1 || c == 0 || c == w-1) && grid[r][c] == bg && !reach[r][c] {
				reach[r][c] = true
				q = append(q, Point{X: c, Y: r})
			}
		}
	}
	for len(q) > 0 {
		p := q[len(q)-1]
		q = q[:len(q)-1]
		for _, d := range neighbors4 {
			nr, nc := p.Y+d[0], p.X+d[1]
			if nr >= 0 && nr < h && nc >= 0 && nc < w && !reach[nr][nc] && grid[nr][nc] == bg {
				reach[nr][nc] = true
				q = append(q, Point{X: nc, Y: nr})
			}
		}
	}
	contained := 0
	for _, b := range blobs {
		open := false
		for _, p := range b.Cells {
			for _, d := range neighbors4 {
				nr, nc := p.Y+d[0], p.X+d[1]
				if nr < 0 || nr >= h || nc < 0 || nc >= w { // touches the grid edge -> not enclosed
					open = true
					break
				}
				if grid[nr][nc] == bg && reach[nr][nc] {
					open = true
					break
				}
			}
			if open {
				break
			}
		}
		if !open {
			contained++
		}
	}
	return contained
}

// countAlignedPairs counts pairs of objects whose centroids share a row or a
// column exactly -- the geometric alignment prior.
func countAlignedPairs(blobs []Blob) int {
	n := 0
	for i := 0; i < len(blobs); i++ {
		for j := i + 1; j < len(blobs); j++ {
			if blobs[i].Centroid.X == blobs[j].Centroid.X || blobs[i].Centroid.Y == blobs[j].Centroid.Y {
				n++
			}
		}
	}
	return n
}
