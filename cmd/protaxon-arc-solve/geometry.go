package main

import (
	"fmt"
	"protaxon/pkg/environment/perception"
)

// --- FloodFill Program ---
type FloodFillProgram struct {
	TargetColor int
	NewColor    int
}

func (p *FloodFillProgram) Apply(input [][]int) [][]int {
	if len(input) == 0 { return input }
	out := make([][]int, len(input))
	for r := range input {
		out[r] = make([]int, len(input[r]))
		copy(out[r], input[r])
	}
	
	R, C := len(out), len(out[0])
	visited := make([][]bool, R)
	for r := 0; r < R; r++ { visited[r] = make([]bool, C) }

	var fill func(r, c int)
	fill = func(r, c int) {
		if r < 0 || r >= R || c < 0 || c >= C || visited[r][c] || out[r][c] != p.TargetColor { return }
		visited[r][c] = true
		out[r][c] = p.NewColor
		fill(r+1, c)
		fill(r-1, c)
		fill(r, c+1)
		fill(r, c-1)
	}

	for r := 0; r < R; r++ {
		for c := 0; c < C; c++ {
			if out[r][c] == p.TargetColor && !visited[r][c] {
				fill(r, c)
			}
		}
	}
	return out
}
func (p *FloodFillProgram) Complexity() float64 { return 2.0 }
func (p *FloodFillProgram) Name() string        { return "FloodFill()" }
func (p *FloodFillProgram) ChangesShape() bool  { return false }


// --- Symmetry Programs ---
type MirrorXProgram struct{}

func (p *MirrorXProgram) Apply(input [][]int) [][]int {
	if len(input) == 0 { return input }
	R, C := len(input), len(input[0])
	out := make([][]int, R)
	for r := 0; r < R; r++ {
		out[r] = make([]int, C)
		for c := 0; c < C; c++ {
			out[r][c] = input[R-1-r][c]
		}
	}
	return out
}
func (p *MirrorXProgram) Complexity() float64 { return 1.5 }
func (p *MirrorXProgram) Name() string        { return "MirrorX()" }
func (p *MirrorXProgram) ChangesShape() bool  { return false }


type MirrorYProgram struct{}

func (p *MirrorYProgram) Apply(input [][]int) [][]int {
	if len(input) == 0 { return input }
	R, C := len(input), len(input[0])
	out := make([][]int, R)
	for r := 0; r < R; r++ {
		out[r] = make([]int, C)
		for c := 0; c < C; c++ {
			out[r][c] = input[r][C-1-c]
		}
	}
	return out
}
func (p *MirrorYProgram) Complexity() float64 { return 1.5 }
func (p *MirrorYProgram) Name() string        { return "MirrorY()" }
func (p *MirrorYProgram) ChangesShape() bool  { return false }

// --- Fill Interior ---
// Fills cavities (0-pixels that cannot reach the grid border) with NewColor
type FillInteriorProgram struct {
	NewColor int
}

func (p *FillInteriorProgram) Apply(input [][]int) [][]int {
	if len(input) == 0 { return input }
	R, C := len(input), len(input[0])
	out := make([][]int, R)
	for r := 0; r < R; r++ {
		out[r] = make([]int, C)
		copy(out[r], input[r])
	}

	reachesBorder := make([][]bool, R)
	for r := 0; r < R; r++ { reachesBorder[r] = make([]bool, C) }

	var dfs func(r, c int)
	dfs = func(r, c int) {
		if r < 0 || r >= R || c < 0 || c >= C || reachesBorder[r][c] || out[r][c] != 0 { return }
		reachesBorder[r][c] = true
		dfs(r+1, c)
		dfs(r-1, c)
		dfs(r, c+1)
		dfs(r, c-1)
	}

	for r := 0; r < R; r++ {
		dfs(r, 0)
		dfs(r, C-1)
	}
	for c := 0; c < C; c++ {
		dfs(0, c)
		dfs(R-1, c)
	}

	for r := 0; r < R; r++ {
		for c := 0; c < C; c++ {
			if out[r][c] == 0 && !reachesBorder[r][c] {
				out[r][c] = p.NewColor
			}
		}
	}
	return out
}
func (p *FillInteriorProgram) Complexity() float64 { return 2.0 }
func (p *FillInteriorProgram) Name() string        { return "FillInterior()" }
func (p *FillInteriorProgram) ChangesShape() bool  { return false }

// --- Gravity Program ---
// Drops all rigid objects in a specified direction until they hit the bounds or another object.
type GravityProgram struct {
	DX, DY int
}

func (p *GravityProgram) Apply(input [][]int) [][]int {
	if len(input) == 0 { return input }
	out := make([][]int, len(input))
	for r := range input {
		out[r] = make([]int, len(input[r]))
		copy(out[r], input[r])
	}

	blobs := perception.FindBlobs(out, 0)

	movedAny := true
	for movedAny {
		movedAny = false
		for i, b := range blobs {
			canMove := true
			for _, cell := range b.Cells {
				ny, nx := cell.Y+p.DY, cell.X+p.DX
				if ny < 0 || ny >= len(out) || nx < 0 || nx >= len(out[0]) {
					canMove = false
					break
				}
				if out[ny][nx] != 0 {
					isSameBlob := false
					for _, oc := range b.Cells {
						if oc.X == nx && oc.Y == ny {
							isSameBlob = true
							break
						}
					}
					if !isSameBlob {
						canMove = false
						break
					}
				}
			}

			if canMove {
				// Erase current
				for _, cell := range b.Cells {
					out[cell.Y][cell.X] = 0
				}
				// Update cells & draw
				for j, cell := range b.Cells {
					b.Cells[j].Y = cell.Y + p.DY
					b.Cells[j].X = cell.X + p.DX
					out[b.Cells[j].Y][b.Cells[j].X] = b.Color
				}
				blobs[i] = b // Update the blob in slice
				movedAny = true
			}
		}
	}
	return out
}
func (p *GravityProgram) Complexity() float64 { return 2.0 }
func (p *GravityProgram) Name() string {
	dir := "Unknown"
	if p.DY == 1 { dir = "Down" }
	if p.DY == -1 { dir = "Up" }
	if p.DX == 1 { dir = "Right" }
	if p.DX == -1 { dir = "Left" }
	return fmt.Sprintf("Gravity(%s)", dir)
}
func (p *GravityProgram) ChangesShape() bool { return false }
