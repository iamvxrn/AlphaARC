package main

import (
	"context"
	"fmt"
	"os"

	"alphaarc/pkg/environment"
	"alphaarc/pkg/environment/perception"
	"alphaarc/pkg/environment/remote"
	"alphaarc/pkg/macro"
)

func render(grid [][]int) {
	glyph := func(v int) byte {
		switch {
		case v == 0:
			return '.'
		case v < 10:
			return byte('0' + v)
		default:
			return byte('a' + (v - 10))
		}
	}
	for _, row := range grid {
		b := make([]byte, len(row))
		for c, v := range row {
			b[c] = glyph(v)
		}
		fmt.Println(string(b))
	}
}

func colorCounts(grid [][]int) map[int]int {
	m := map[int]int{}
	for _, row := range grid {
		for _, v := range row {
			m[v]++
		}
	}
	return m
}

func diff(a, b [][]int) []string {
	var d []string
	for r := range a {
		if r == 0 {
			continue // row 0 is the move-counter bar (every click flips (63,0)); ignore it
		}
		for c := range a[r] {
			if c < len(b[r]) && a[r][c] != b[r][c] {
				d = append(d, fmt.Sprintf("(%d,%d):%d->%d", c, r, a[r][c], b[r][c]))
			}
		}
	}
	return d
}

func main() {
	ctx := context.Background()
	client, err := remote.NewClientFromEnv()
	if err != nil {
		fmt.Println("client:", err)
		os.Exit(1)
	}
	card, err := client.OpenScorecard(ctx, []string{"click"})
	if err != nil {
		fmt.Println("scorecard:", err)
		os.Exit(1)
	}
	defer client.CloseScorecard(ctx, card)

	sess := remote.NewSession(client, "vc33-5430563c", card)
	frame, err := sess.Reset()
	if err != nil {
		fmt.Println("reset:", err)
		os.Exit(1)
	}
	h := len(frame.Grid)
	w := 0
	if h > 0 {
		w = len(frame.Grid[0])
	}
	fmt.Printf("=== INITIAL vc33 grid %dx%d state=%s actions=%v levels=%d ===\n", w, h, frame.State, frame.AvailableActions, frame.LevelsCompleted)
	render(frame.Grid)
	fmt.Printf("colors: %v\n", colorCounts(frame.Grid))

	// DOES THE COMPRESSION DRIVE GIVE A GRADIENT ON THE REAL MECHANIC?
	// vc33 = two buttons: (60,32) grows the colour-3 region, (60,24) shrinks it.
	// Click GROW repeatedly and watch DriveScore/BestPrimitive + the region's right
	// edge. A monotone move = the drive can steer this game.
	probe := func(label string, bx, by, n int) {
		f, _ := sess.Reset()
		bg := perception.BackgroundColor(f.Grid)
		fmt.Printf("\n=== %s button (%d,%d) x%d ===\n", label, bx, by, n)
		report := func(step int, fr environment.Frame) {
			bp, sav := macro.BestPrimitive(fr.Grid, bg)
			// right edge of the colour-3 block on row 1
			edge := -1
			for c := 0; c < len(fr.Grid[1]); c++ {
				if fr.Grid[1][c] == 3 {
					edge = c
				}
			}
			fmt.Printf("step %2d: levels=%d state=%s | best=%s savings=%d score=%.3f | 3-edge(row1)=%d residualClusters=%d\n",
				step, fr.LevelsCompleted, fr.State, bp.Name, sav, macro.DriveScore(fr.Grid, bg),
				edge, len(macro.ResidualTargets(fr.Grid, bg, 8)))
		}
		report(0, f)
		for i := 1; i <= n; i++ {
			f, err = sess.Step(environment.Action{ID: environment.Action6, X: bx, Y: by})
			if err != nil {
				fmt.Println("step:", err)
				return
			}
			report(i, f)
			if f.State != environment.StateNotFinished {
				fmt.Printf("  -> terminal %s at step %d\n", f.State, i)
				break
			}
		}
	}
	probe("GROW", 60, 32, 20)
	probe("SHRINK", 60, 24, 20)
}
