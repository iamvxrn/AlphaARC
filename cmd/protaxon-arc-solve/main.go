package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"

	"protaxon/pkg/environment/perception"
)

// ARCTask represents the standard ARC-AGI JSON format.
type ARCTask struct {
	Train []ARCPair `json:"train"`
	Test  []ARCPair `json:"test"`
}

type ARCPair struct {
	Input  [][]int `json:"input"`
	Output [][]int `json:"output"`
}

type Program interface {
	Apply(input [][]int) [][]int
	Complexity() float64
	Name() string
	ChangesShape() bool
}

// --- Base Primitives ---

type IdentityProgram struct{}

func (p *IdentityProgram) Apply(input [][]int) [][]int {
	out := make([][]int, len(input))
	for r := range input {
		out[r] = make([]int, len(input[r]))
		copy(out[r], input[r])
	}
	return out
}
func (p *IdentityProgram) Complexity() float64 { return 0.5 }
func (p *IdentityProgram) Name() string        { return "Identity()" }
func (p *IdentityProgram) ChangesShape() bool  { return false }

type RecolorProgram struct {
	TargetColor int
}

func (p *RecolorProgram) Apply(input [][]int) [][]int {
	out := make([][]int, len(input))
	for r := range input {
		out[r] = make([]int, len(input[r]))
		for c := range input[r] {
			out[r][c] = p.TargetColor
		}
	}
	return out
}
func (p *RecolorProgram) Complexity() float64 { return 1.0 }
func (p *RecolorProgram) Name() string        { return fmt.Sprintf("Recolor(All, %d)", p.TargetColor) }
func (p *RecolorProgram) ChangesShape() bool  { return false }

type ColorMapProgram struct {
	Mapping map[int]int
}

func (p *ColorMapProgram) Apply(input [][]int) [][]int {
	out := make([][]int, len(input))
	for r := range input {
		out[r] = make([]int, len(input[r]))
		for c := range input[r] {
			if target, ok := p.Mapping[input[r][c]]; ok {
				out[r][c] = target
			} else {
				out[r][c] = input[r][c]
			}
		}
	}
	return out
}
func (p *ColorMapProgram) Complexity() float64 { return float64(len(p.Mapping)) }
func (p *ColorMapProgram) Name() string        { return fmt.Sprintf("ColorMap(%d rules)", len(p.Mapping)) }
func (p *ColorMapProgram) ChangesShape() bool  { return false }

func inferColorMap(train []ARCPair) *ColorMapProgram {
	mapping := make(map[int]int)
	for _, pair := range train {
		if len(pair.Input) != len(pair.Output) || len(pair.Input[0]) != len(pair.Output[0]) {
			return nil
		}
		for r := range pair.Input {
			for c := range pair.Input[r] {
				if pair.Input[r][c] != pair.Output[r][c] {
					if existing, ok := mapping[pair.Input[r][c]]; ok && existing != pair.Output[r][c] {
						return nil
					}
					mapping[pair.Input[r][c]] = pair.Output[r][c]
				}
			}
		}
	}
	if len(mapping) == 0 {
		return nil
	}
	return &ColorMapProgram{Mapping: mapping}
}

// --- Geometry ---

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

// --- New: Crop ---
// CropBackgroundProgram removes rows/cols that are purely background color (0).
type CropBackgroundProgram struct{}

func (p *CropBackgroundProgram) Apply(input [][]int) [][]int {
	if len(input) == 0 {
		return input
	}
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

	if maxR < minR { // Completely empty grid
		return [][]int{{0}}
	}

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


// --- New: Fractal Expand ---
// Replaces every non-zero pixel with a full copy of the original input grid.
type FractalExpandProgram struct{}

func (p *FractalExpandProgram) Apply(input [][]int) [][]int {
	if len(input) == 0 {
		return input
	}
	R := len(input)
	C := len(input[0])
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

// --- Program Composition ---

type SequenceProgram struct {
	Steps []Program
}

func (p *SequenceProgram) Apply(input [][]int) [][]int {
	curr := input
	for _, step := range p.Steps {
		curr = step.Apply(curr)
	}
	return curr
}

func (p *SequenceProgram) Complexity() float64 {
	c := 0.0
	for _, step := range p.Steps {
		c += step.Complexity()
	}
	return c + 0.5
}

func (p *SequenceProgram) Name() string {
	names := ""
	for i, step := range p.Steps {
		if i > 0 {
			names += " -> "
		}
		names += step.Name()
	}
	return fmt.Sprintf("Seq(%s)", names)
}
func (p *SequenceProgram) ChangesShape() bool {
	for _, step := range p.Steps {
		if step.ChangesShape() {
			return true
		}
	}
	return false
}

// gridError calculates pixel-wise mismatch
func gridError(a, b [][]int) float64 {
	if len(a) != len(b) || len(a) > 0 && len(a[0]) != len(b[0]) {
		return 1000.0 // Massive penalty for wrong shape
	}
	err := 0.0
	for r := range a {
		for c := range a[r] {
			if a[r][c] != b[r][c] {
				err += 1.0
			}
		}
	}
	return err
}

func main() {
	taskPath := flag.String("task", "", "Path to ARC-AGI JSON task file")
	quiet := flag.Bool("quiet", false, "Suppress verbose search output")
	flag.Parse()

	if *taskPath == "" {
		log.Fatal("Please provide a -task path")
	}

	data, err := ioutil.ReadFile(*taskPath)
	if err != nil {
		log.Fatalf("Failed to read task: %v", err)
	}

	var task ARCTask
	if err := json.Unmarshal(data, &task); err != nil {
		log.Fatalf("Failed to parse task JSON: %v", err)
	}

	// 1. Analyze Task Properties (Heuristics)
	shapeChanges := false
	for _, pair := range task.Train {
		if len(pair.Input) != len(pair.Output) || len(pair.Input[0]) != len(pair.Output[0]) {
			shapeChanges = true
			break
		}
	}

	if !*quiet {
		fmt.Printf("Loaded task %s: ShapeChanges=%v\n", *taskPath, shapeChanges)
	}

	// 2. Generate Base Primitives
	var baseSpace []Program
	baseSpace = append(baseSpace, &IdentityProgram{})
	baseSpace = append(baseSpace, &CropBackgroundProgram{})
	baseSpace = append(baseSpace, &FractalExpandProgram{})
	baseSpace = append(baseSpace, &MirrorXProgram{})
	baseSpace = append(baseSpace, &MirrorYProgram{})
	baseSpace = append(baseSpace, &GravityProgram{DX: 0, DY: 1})
	baseSpace = append(baseSpace, &GravityProgram{DX: 0, DY: -1})
	baseSpace = append(baseSpace, &GravityProgram{DX: 1, DY: 0})
	baseSpace = append(baseSpace, &GravityProgram{DX: -1, DY: 0})
	for c := 1; c < 10; c++ {
		baseSpace = append(baseSpace, &FillInteriorProgram{NewColor: c})
	}
	
	for c := 0; c < 10; c++ {
		baseSpace = append(baseSpace, &RecolorProgram{TargetColor: c})
	}
	if cmap := inferColorMap(task.Train); cmap != nil {
		baseSpace = append(baseSpace, cmap)
	}
	for c := 1; c < 10; c++ {
		for nc := 1; nc < 10; nc++ {
			if c == nc { continue }
			baseSpace = append(baseSpace, &RecolorObjectProgram{TargetColor: c, NewColor: nc})
		}
	}
	for c := 1; c < 10; c++ {
		for dx := -3; dx <= 3; dx++ {
			for dy := -3; dy <= 3; dy++ {
				if dx == 0 && dy == 0 { continue }
				baseSpace = append(baseSpace, &TranslateProgram{TargetColor: c, DX: dx, DY: dy})
			}
		}
	}

	// 3. BFS Search Queue
	// We will search up to Depth 2 for now, but via an expandable queue.
	type SearchNode struct {
		Prog  Program
		Depth int
	}

	queue := []SearchNode{}
	for _, bp := range baseSpace {
		queue = append(queue, SearchNode{Prog: bp, Depth: 1})
	}

	lambda := 0.1
	bestSurprise := math.MaxFloat64
	var bestProgram Program
	programsEvaluated := 0

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		// Heuristic Pruning:
		// If task requires shape change, but program doesn't change shape, 
		// evaluating it alone is pointless (error will be 1000).
		// We still let it pass if depth < max_depth so it can be chained with a shape-changer later,
		// but if it's the final node, we could prune. 
		
		totalError := 0.0
		for _, pair := range task.Train {
			predicted := node.Prog.Apply(pair.Input)
			totalError += gridError(predicted, pair.Output)
		}
		surprise := totalError + lambda*node.Prog.Complexity()
		programsEvaluated++

		if surprise < bestSurprise {
			if !*quiet {
				fmt.Printf("[NEW BEST] %-40s: Error=%.1f, Compl=%.1f -> Surprise=%.2f\n", node.Prog.Name(), totalError, node.Prog.Complexity(), surprise)
			}
			bestSurprise = surprise
			bestProgram = node.Prog
		}

		// Stop early if perfect!
		if totalError == 0 {
			break
		}

		// Generate children (up to Depth 2 for now to avoid hang, but now scalable)
		if node.Depth < 2 {
			for _, bp := range baseSpace {
				if bp.Name() == "Identity()" { continue }
				// Prune: If task requires shape change, don't build deep chains of non-shape-changers
				if shapeChanges && !node.Prog.ChangesShape() && !bp.ChangesShape() && node.Depth == 1 {
					// We can skip this branch. It's a chain of 2 non-shape-changers, 
					// and we are at max depth 2. It will never solve a shape-changing task.
					continue
				}

				seq := &SequenceProgram{Steps: []Program{node.Prog, bp}}
				queue = append(queue, SearchNode{Prog: seq, Depth: node.Depth + 1})
			}
		}
	}

	if !*quiet {
		fmt.Printf("\n=== WINNER (Evaluated %d) ===\n", programsEvaluated)
		if bestProgram != nil {
			fmt.Printf("Program: %s\nSurprise: %.2f\n", bestProgram.Name(), bestSurprise)
		}
		fmt.Printf("\n--- Applying to Test Set ---\n")
	}

	if bestProgram != nil {
		for i, pair := range task.Test {
			predicted := bestProgram.Apply(pair.Input)
			if !*quiet {
				fmt.Printf("Test Pair %d Output:\n", i)
				for _, row := range predicted {
					fmt.Printf("  %v\n", row)
				}
			}
			
			if len(pair.Output) > 0 {
				err := gridError(predicted, pair.Output)
				if err == 0 {
					if !*quiet { fmt.Printf("  -> SUCCESS! Matches ground truth.\n") } else { fmt.Printf("SUCCESS\n") }
				} else {
					if !*quiet { fmt.Printf("  -> FAILED! Mismatch error = %.1f\n", err) } else { fmt.Printf("FAILED\n") }
				}
			}
		}
	}
}
