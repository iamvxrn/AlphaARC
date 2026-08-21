package macro

import (
	"fmt"
	"sort"
	"strings"
)

// Primitive #3: Number / Numerosity.
//
// A grid holding N identical objects can be described as one shape + N positions
// instead of N full shapes. Each extra copy of a shape of s cells is explained
// for free once the shape is stated => savings = sum over identical-shape groups
// of (count-1) * s.
//
// This is a NON-geometric Core-Knowledge domain (Spelke: number), deliberately
// contrasting Reflect/Translate (both geometry). It answers the skeptic's "you
// only wrote a geometry scorer": the same bits-saved currency now also prices
// object multiplicity, and BestPrimitive picks per grid across two different
// domains.
//
// Degenerate cheat (per the rule that every primitive must be checked): scatter
// many identical SINGLE-cell dots. Guard: a counted shape must be non-trivial
// (>= 2 cells) — sharing a 1-cell shape saves nothing worth an opcode, and
// "scatter pixels" is not "count objects".

// NumerositySavings sums the description saved by repeated identical objects.
// Objects are 4-connected same-colour components; two objects are identical iff
// they have the same colour and the same normalised cell-offset set.
func NumerositySavings(grid [][]int, bg int) int {
	h := len(grid)
	if h == 0 {
		return 0
	}
	visited := make([][]bool, h)
	for r := range grid {
		visited[r] = make([]bool, len(grid[r]))
	}

	groupCount := map[string]int{}
	groupCells := map[string]int{}

	for r := 0; r < h; r++ {
		for c := 0; c < len(grid[r]); c++ {
			if grid[r][c] == bg || visited[r][c] {
				continue
			}
			color := grid[r][c]

			// Flood-fill the 4-connected same-colour component.
			var cells [][2]int
			stack := [][2]int{{r, c}}
			visited[r][c] = true
			for len(stack) > 0 {
				cur := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				cells = append(cells, cur)
				for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nr, nc := cur[0]+d[0], cur[1]+d[1]
					if nr < 0 || nr >= h || nc < 0 || nc >= len(grid[nr]) {
						continue
					}
					if visited[nr][nc] || grid[nr][nc] != color {
						continue
					}
					visited[nr][nc] = true
					stack = append(stack, [2]int{nr, nc})
				}
			}

			// Normalise to a position-independent shape signature.
			minr, minc := cells[0][0], cells[0][1]
			for _, cl := range cells {
				if cl[0] < minr {
					minr = cl[0]
				}
				if cl[1] < minc {
					minc = cl[1]
				}
			}
			offs := make([][2]int, len(cells))
			for i, cl := range cells {
				offs[i] = [2]int{cl[0] - minr, cl[1] - minc}
			}
			sort.Slice(offs, func(i, j int) bool {
				if offs[i][0] != offs[j][0] {
					return offs[i][0] < offs[j][0]
				}
				return offs[i][1] < offs[j][1]
			})
			var sb strings.Builder
			fmt.Fprintf(&sb, "c%d:", color)
			for _, o := range offs {
				fmt.Fprintf(&sb, "%d,%d;", o[0], o[1])
			}
			key := sb.String()
			groupCount[key]++
			groupCells[key] = len(cells)
		}
	}

	savings := 0
	for key, cnt := range groupCount {
		s := groupCells[key]
		if s < 2 { // single-cell shapes don't count: scattering pixels isn't counting
			continue
		}
		if cnt >= 2 {
			savings += (cnt - 1) * s
		}
	}
	return savings
}

// NumerosityPreference is the compression saving under the Count primitive.
// 0 on a blank grid, on a lone object, and on scattered single-cell dots.
func NumerosityPreference(grid [][]int, bg int) int {
	return NumerositySavings(grid, bg)
}
