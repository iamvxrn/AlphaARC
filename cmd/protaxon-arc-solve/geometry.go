package main

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

	// In ARC, usually we want to fill the "inside" of closed contours.
	// But as a primitive, a standard FloodFill on specific connected components is a start.
	// We will apply this to all blobs of TargetColor.
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

	// 1. Find all 0-pixels that CAN reach the border
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

	// 2. Any 0-pixel that CANNOT reach the border is interior
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
