package macro

import (
	"fmt"
	"alphaarc/pkg/environment/perception"
)

// CropBackgroundProgram removes rows/cols that are purely background color (0).
type CropBackgroundProgram struct{}

func (p *CropBackgroundProgram) Apply(input [][]int) [][]int {
	if len(input) == 0 { return input }
	minR, maxR := len(input), -1
	minC, maxC := len(input[0]), -1

	for r := range input {
		for c := range input[r] {
			if input[r][c] != 0 {
				if r < minR { minR = r }
				if r > maxR { maxR = r }
				if c < minC { minC = c }
				if c > maxC { maxC = c }
			}
		}
	}

	if maxR < minR { return [][]int{{0}} }

	out := make([][]int, maxR-minR+1)
	for r := minR; r <= maxR; r++ {
		out[r-minR] = make([]int, maxC-minC+1)
		copy(out[r-minR], input[r][minC:maxC+1])
	}
	return out
}
func (p *CropBackgroundProgram) Complexity() float64 { return 1.5 }
func (p *CropBackgroundProgram) Name() string        { return "CropBackground()" }
func (p *CropBackgroundProgram) ChangesShape() bool  { return true }


// FractalExpandProgram replaces every non-zero pixel with a full copy of the original input grid.
type FractalExpandProgram struct{}

func (p *FractalExpandProgram) Apply(input [][]int) [][]int {
	if len(input) == 0 { return input }
	R, C := len(input), len(input[0])
	out := make([][]int, R*R)
	for i := range out {
		out[i] = make([]int, C*C)
	}

	for r := 0; r < R; r++ {
		for c := 0; c < C; c++ {
			if input[r][c] != 0 {
				for ir := 0; ir < R; ir++ {
					for ic := 0; ic < C; ic++ {
						out[r*R+ir][c*C+ic] = input[ir][ic]
					}
				}
			}
		}
	}
	return out
}
func (p *FractalExpandProgram) Complexity() float64 { return 2.5 }
func (p *FractalExpandProgram) Name() string        { return "FractalExpand()" }
func (p *FractalExpandProgram) ChangesShape() bool  { return true }


// TranslateProgram shifts objects of a specific color by DX, DY.
type TranslateProgram struct {
	TargetColor int
	DX, DY      int
}

func (p *TranslateProgram) Apply(input [][]int) [][]int {
	out := make([][]int, len(input))
	for r := range input {
		out[r] = make([]int, len(input[r]))
		copy(out[r], input[r])
	}

	blobs := perception.FindBlobs(input, 0)
	for _, b := range blobs {
		if b.Color == p.TargetColor {
			for _, cell := range b.Cells {
				out[cell.Y][cell.X] = 0
			}
			for _, cell := range b.Cells {
				ny, nx := cell.Y+p.DY, cell.X+p.DX
				if ny >= 0 && ny < len(out) && nx >= 0 && nx < len(out[0]) {
					out[ny][nx] = p.TargetColor
				}
			}
		}
	}
	return out
}
func (p *TranslateProgram) Complexity() float64 { return 2.0 }
func (p *TranslateProgram) Name() string        { return fmt.Sprintf("Translate(Color:%d, dX:%d, dY:%d)", p.TargetColor, p.DX, p.DY) }
func (p *TranslateProgram) ChangesShape() bool  { return false }


// RecolorObjectProgram changes the color of connected components of a specific color.
type RecolorObjectProgram struct {
	TargetColor int
	NewColor    int
}

func (p *RecolorObjectProgram) Apply(input [][]int) [][]int {
	out := make([][]int, len(input))
	for r := range input {
		out[r] = make([]int, len(input[r]))
		copy(out[r], input[r])
	}
	blobs := perception.FindBlobs(input, 0)
	for _, b := range blobs {
		if b.Color == p.TargetColor {
			for _, cell := range b.Cells {
				out[cell.Y][cell.X] = p.NewColor
			}
		}
	}
	return out
}
func (p *RecolorObjectProgram) Complexity() float64 { return 2.0 }
func (p *RecolorObjectProgram) Name() string        { return fmt.Sprintf("RecolorObj(%d->%d)", p.TargetColor, p.NewColor) }
func (p *RecolorObjectProgram) ChangesShape() bool  { return false }


// FloodFillProgram finds connected components of TargetColor and changes them to NewColor.
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


// MirrorXProgram reflects the grid vertically.
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


// MirrorYProgram reflects the grid horizontally.
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


// FillInteriorProgram fills cavities (0-pixels that cannot reach the grid border) with NewColor.
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


// GravityProgram drops all rigid objects in a specified direction.
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
				for _, cell := range b.Cells {
					out[cell.Y][cell.X] = 0
				}
				for j, cell := range b.Cells {
					b.Cells[j].Y = cell.Y + p.DY
					b.Cells[j].X = cell.X + p.DX
					out[b.Cells[j].Y][b.Cells[j].X] = b.Color
				}
				blobs[i] = b
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
