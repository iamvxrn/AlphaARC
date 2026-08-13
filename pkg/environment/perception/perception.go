// Package perception turns a real ARC-AGI-3 grid ([][]int, 16 colors) into
// graph-seedable observation words, grounded in actual pixel data instead
// of hand-fed coordinates.
//
// This is coarse, hand-designed symbolic feature extraction -- background-
// color subtraction, flood-filled same-color connected regions ("blobs"),
// and blob-centroid-vs-grid-center direction -- not learned vision. It also
// does not by itself solve action selection for a real game: what a given
// blob layout should mean for which of environment.Action{1..7} to press is
// game-specific (unlike pkg/environment/bridge's DescribeFrame, which is
// fed synthetic agent/target positions for one fixed local practice game
// and can therefore hard-map direction words to actions). Turning
// DescribeGrid's words into a real-game policy remains open.
package perception

import (
	"fmt"
	"sort"
	"strings"
)

// Point is one grid cell coordinate, (0,0) at the top-left.
type Point struct {
	X, Y int
}

// Blob is one flood-filled, single-color connected region of non-background
// cells (4-connectivity: up/down/left/right, not diagonals).
type Blob struct {
	Color    int
	Cells    []Point
	Centroid Point // integer centroid (sum / count, truncated)
}

// BackgroundColor returns the grid's most frequent color -- the simplest
// possible background guess. This is a heuristic, not a guarantee: it
// assumes the dominant color is background fill, which holds for typical
// ARC-AGI-3 grids but isn't verified against every real game.
func BackgroundColor(grid [][]int) int {
	counts := map[int]int{}
	for _, row := range grid {
		for _, c := range row {
			counts[c]++
		}
	}
	best, bestCount := 0, -1
	// Deterministic tie-break: lowest color value wins, so ties never
	// depend on Go's randomized map iteration order.
	for c, n := range counts {
		if n > bestCount || (n == bestCount && c < best) {
			best, bestCount = c, n
		}
	}
	return best
}

// FindBlobs flood-fills every non-background cell into same-color connected
// components. Blob discovery order (and therefore the returned slice's
// order) follows a row-major scan, but callers that need a stable ranking
// should sort explicitly -- see DescribeGrid.
func FindBlobs(grid [][]int, background int) []Blob {
	if len(grid) == 0 || len(grid[0]) == 0 {
		return nil
	}
	h, w := len(grid), len(grid[0])
	visited := make([][]bool, h)
	for i := range visited {
		visited[i] = make([]bool, w)
	}

	var blobs []Blob
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if visited[y][x] || grid[y][x] == background {
				continue
			}
			color := grid[y][x]
			var cells []Point
			queue := []Point{{X: x, Y: y}}
			visited[y][x] = true
			for len(queue) > 0 {
				p := queue[0]
				queue = queue[1:]
				cells = append(cells, p)
				for _, d := range [4][2]int{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
					nx, ny := p.X+d[0], p.Y+d[1]
					if nx < 0 || ny < 0 || ny >= h || nx >= w {
						continue
					}
					if visited[ny][nx] || grid[ny][nx] != color {
						continue
					}
					visited[ny][nx] = true
					queue = append(queue, Point{X: nx, Y: ny})
				}
			}
			blobs = append(blobs, Blob{Color: color, Cells: cells, Centroid: centroid(cells)})
		}
	}
	return blobs
}

func centroid(cells []Point) Point {
	var sx, sy int
	for _, p := range cells {
		sx += p.X
		sy += p.Y
	}
	n := len(cells)
	return Point{X: sx / n, Y: sy / n}
}

// direction gives a coarse compass description of (x,y) relative to
// (cx,cy), using the same north/south/east/west (+ diagonal combos)
// vocabulary already used elsewhere in the engine. Returns "center" when
// (x,y) == (cx,cy).
func direction(x, y, cx, cy int) string {
	var vertical, horizontal string
	if y < cy {
		vertical = "north"
	} else if y > cy {
		vertical = "south"
	}
	if x < cx {
		horizontal = "west"
	} else if x > cx {
		horizontal = "east"
	}
	switch {
	case vertical != "" && horizontal != "":
		return vertical + " " + horizontal
	case vertical != "":
		return vertical
	case horizontal != "":
		return horizontal
	default:
		return "center"
	}
}

// rankedBlobs finds grid's blobs and sorts them deterministically: cell
// count descending, ties broken by color ascending then row-major centroid
// -- shared by every Describe* function so they all rank blobs the same
// way regardless of Go's unordered map iteration during flood fill.
func rankedBlobs(grid [][]int) []Blob {
	background := BackgroundColor(grid)
	blobs := FindBlobs(grid, background)
	sort.Slice(blobs, func(i, j int) bool {
		if len(blobs[i].Cells) != len(blobs[j].Cells) {
			return len(blobs[i].Cells) > len(blobs[j].Cells)
		}
		if blobs[i].Color != blobs[j].Color {
			return blobs[i].Color < blobs[j].Color
		}
		if blobs[i].Centroid.Y != blobs[j].Centroid.Y {
			return blobs[i].Centroid.Y < blobs[j].Centroid.Y
		}
		return blobs[i].Centroid.X < blobs[j].Centroid.X
	})
	return blobs
}

// DescribeGrid converts grid into observation words: for up to maxBlobs of
// the grid's largest color blobs, emits "color<N>" followed by that blob's
// direction from the grid's center, as SEPARATE words (so "color2" and
// "north" are independent, reusable graph concepts, not one unit). Callers
// that need each blob to compete as a single bound candidate -- e.g. to
// pick one blob's centroid as a click target -- want DescribeGridCells
// instead, which never splits a blob's label across multiple words.
func DescribeGrid(grid [][]int, maxBlobs int) string {
	if len(grid) == 0 || len(grid[0]) == 0 {
		return ""
	}
	h, w := len(grid), len(grid[0])
	cx, cy := w/2, h/2

	var words []string
	for i, b := range rankedBlobs(grid) {
		if i >= maxBlobs {
			break
		}
		words = append(words, fmt.Sprintf("color%d", b.Color))
		words = append(words, strings.Fields(direction(b.Centroid.X, b.Centroid.Y, cx, cy))...)
	}
	if len(words) == 0 {
		return "empty"
	}
	return strings.Join(words, " ")
}

// CellToken returns a stable categorical label for point (x,y) in a
// gridW x gridH space, quantized into a cols x rows lattice of buckets --
// e.g. "cell3-2" for column 3, row 2 (0-indexed). Finer lattices (higher
// cols/rows) shrink each bucket's footprint, which reduces how many
// physically different targets can share one category label (the
// "referent instability" problem a coarse compass-direction bucket has),
// at the cost of a larger vocabulary -- more distinct graph nodes for the
// same amount of experience to spread across. Points on or past the far
// edge (x == gridW or y == gridH) clamp into the last column/row rather
// than going out of bounds.
func CellToken(x, y, gridW, gridH, cols, rows int) string {
	if gridW <= 0 || gridH <= 0 || cols <= 0 || rows <= 0 {
		return "cell0-0"
	}
	col := x * cols / gridW
	row := y * rows / gridH
	if col >= cols {
		col = cols - 1
	}
	if row >= rows {
		row = rows - 1
	}
	if col < 0 {
		col = 0
	}
	if row < 0 {
		row = 0
	}
	return fmt.Sprintf("cell%d-%d", col, row)
}

// LabeledBlob pairs a blob with the composite category label
// DescribeGridCells would assign it at a given lattice resolution.
type LabeledBlob struct {
	Blob  Blob
	Label string
}

// RankedLabeledBlobs returns grid's top maxBlobs blobs (the same
// deterministic ranking DescribeGridCells uses: cell count descending,
// ties by color ascending then row-major centroid), each paired with its
// "color<N>-cell<C>-<R>" composite label at the given cols x rows
// resolution. This is the shared source of truth behind DescribeGridCells
// and is also what a caller needs for the "bind" step of action selection:
// after the graph/router picks a winning category label (stable across
// frames), re-deriving which of THIS frame's actual blobs currently
// carries that label to recover a concrete (X,Y).
func RankedLabeledBlobs(grid [][]int, maxBlobs, cols, rows int) []LabeledBlob {
	if len(grid) == 0 || len(grid[0]) == 0 {
		return nil
	}
	h, w := len(grid), len(grid[0])

	var out []LabeledBlob
	for i, b := range rankedBlobs(grid) {
		if i >= maxBlobs {
			break
		}
		label := fmt.Sprintf("color%d-%s", b.Color, CellToken(b.Centroid.X, b.Centroid.Y, w, h, cols, rows))
		out = append(out, LabeledBlob{Blob: b, Label: label})
	}
	return out
}

// DescribeGridCells converts grid into observation words: each blob
// becomes ONE composite token -- "color<N>-cell<C>-<R>" -- instead of
// separate "color<N>" and direction words. A composite token is a single
// unit to the graph's tokenizer (EnsureConceptNodes splits on whitespace,
// not hyphens), so it gets one stable node ID reused every time the same
// (color, cell) combination recurs, exactly like "north" already does --
// and unlike a fresh per-frame blob object/ID, which would never
// accumulate Hebbian weight or eligibility trace across frames at all.
func DescribeGridCells(grid [][]int, maxBlobs, cols, rows int) string {
	labeled := RankedLabeledBlobs(grid, maxBlobs, cols, rows)
	if len(labeled) == 0 {
		if len(grid) == 0 || len(grid[0]) == 0 {
			return ""
		}
		return "empty"
	}
	words := make([]string, len(labeled))
	for i, lb := range labeled {
		words[i] = lb.Label
	}
	return strings.Join(words, " ")
}
