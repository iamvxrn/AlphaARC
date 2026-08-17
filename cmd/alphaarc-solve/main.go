package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"alphaarc/pkg/macro"
)

type ARCTask struct {
	Train []ARCPair `json:"train"`
	Test  []ARCPair `json:"test"`
}

type ARCPair struct {
	Input  [][]int `json:"input"`
	Output [][]int `json:"output"`
}

func inferColorMap(train []ARCPair) *macro.ColorMapProgram {
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
	return &macro.ColorMapProgram{Mapping: mapping}
}

func gridError(a, b [][]int) float64 {
	if len(a) != len(b) || len(a) > 0 && len(a[0]) != len(b[0]) {
		return 1000.0 
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
	quiet := flag.Bool("quiet", false, "Suppress verbose output")
	flag.Parse()

	if *taskPath == "" { log.Fatal("Provide -task") }

	data, err := ioutil.ReadFile(*taskPath)
	if err != nil { log.Fatal(err) }

	var task ARCTask
	if err := json.Unmarshal(data, &task); err != nil { log.Fatal(err) }

	shapeChanges := false
	for _, pair := range task.Train {
		if len(pair.Input) != len(pair.Output) || len(pair.Input[0]) != len(pair.Output[0]) {
			shapeChanges = true
			break
		}
	}

	if !*quiet { fmt.Printf("Loaded task %s: ShapeChanges=%v\n", *taskPath, shapeChanges) }

	var baseSpace []macro.Program
	baseSpace = append(baseSpace, &macro.IdentityProgram{})
	baseSpace = append(baseSpace, &macro.CropBackgroundProgram{})
	baseSpace = append(baseSpace, &macro.FractalExpandProgram{})
	baseSpace = append(baseSpace, &macro.MirrorXProgram{})
	baseSpace = append(baseSpace, &macro.MirrorYProgram{})
	baseSpace = append(baseSpace, &macro.GravityProgram{DX: 0, DY: 1})
	baseSpace = append(baseSpace, &macro.GravityProgram{DX: 0, DY: -1})
	baseSpace = append(baseSpace, &macro.GravityProgram{DX: 1, DY: 0})
	baseSpace = append(baseSpace, &macro.GravityProgram{DX: -1, DY: 0})
	
	for c := 1; c < 10; c++ {
		baseSpace = append(baseSpace, &macro.FillInteriorProgram{NewColor: c})
	}
	for c := 0; c < 10; c++ {
		baseSpace = append(baseSpace, &macro.RecolorProgram{TargetColor: c})
	}
	if cmap := inferColorMap(task.Train); cmap != nil {
		baseSpace = append(baseSpace, cmap)
	}
	for c := 1; c < 10; c++ {
		for nc := 1; nc < 10; nc++ {
			if c == nc { continue }
			baseSpace = append(baseSpace, &macro.RecolorObjectProgram{TargetColor: c, NewColor: nc})
		}
	}
	for c := 1; c < 10; c++ {
		for dx := -3; dx <= 3; dx++ {
			for dy := -3; dy <= 3; dy++ {
				if dx == 0 && dy == 0 { continue }
				baseSpace = append(baseSpace, &macro.TranslateProgram{TargetColor: c, DX: dx, DY: dy})
			}
		}
	}

	type SearchNode struct {
		Prog  macro.Program
		Depth int
	}

	queue := []SearchNode{}
	for _, bp := range baseSpace { queue = append(queue, SearchNode{Prog: bp, Depth: 1}) }

	lambda := 0.1
	bestSurprise := math.MaxFloat64
	var bestProgram macro.Program
	programsEvaluated := 0

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		totalError := 0.0
		for _, pair := range task.Train {
			predicted := node.Prog.Apply(pair.Input)
			totalError += gridError(predicted, pair.Output)
		}
		surprise := totalError + lambda*node.Prog.Complexity()
		programsEvaluated++

		if surprise < bestSurprise {
			if !*quiet { fmt.Printf("[NEW BEST] %-40s: Error=%.1f, Compl=%.1f -> Surprise=%.2f\n", node.Prog.Name(), totalError, node.Prog.Complexity(), surprise) }
			bestSurprise = surprise
			bestProgram = node.Prog
		}

		if totalError == 0 { break }

		if node.Depth < 2 {
			for _, bp := range baseSpace {
				if bp.Name() == "Identity()" { continue }
				if shapeChanges && !node.Prog.ChangesShape() && !bp.ChangesShape() && node.Depth == 1 { continue }
				seq := &macro.SequenceProgram{Steps: []macro.Program{node.Prog, bp}}
				queue = append(queue, SearchNode{Prog: seq, Depth: node.Depth + 1})
			}
		}
	}

	if !*quiet {
		fmt.Printf("\n=== WINNER (Evaluated %d) ===\n", programsEvaluated)
		if bestProgram != nil { fmt.Printf("Program: %s\nSurprise: %.2f\n", bestProgram.Name(), bestSurprise) }
		fmt.Printf("\n--- Applying to Test Set ---\n")
	}

	if bestProgram != nil {
		for i, pair := range task.Test {
			predicted := bestProgram.Apply(pair.Input)
			if !*quiet {
				fmt.Printf("Test Pair %d Output:\n", i)
				for _, row := range predicted { fmt.Printf("  %v\n", row) }
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
