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

func diffBoard(a, b [][]int) []string {
	var d []string
	for r := range a {
		if r == 0 {
			continue // counter bar
		}
		for c := range a[r] {
			if c < len(b[r]) && a[r][c] != b[r][c] {
				d = append(d, fmt.Sprintf("(%d,%d):%d->%d", c, r, a[r][c], b[r][c]))
			}
		}
	}
	return d
}

// growToLevel clicks the L1 grow button until levels_completed reaches `want`
// (or gives up). Returns the frame at that level.
func growToLevel(sess *remote.Session, want int) (environment.Frame, int) {
	f, _ := sess.Reset()
	clicks := 0
	for i := 0; i < 12 && f.LevelsCompleted < want; i++ {
		f, _ = sess.Step(environment.Action{ID: environment.Action6, X: 60, Y: 32})
		clicks++
	}
	return f, clicks
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

	// Reach Level 2 by solving L1 with the known grow button.
	f, clicks := growToLevel(sess, 1)
	fmt.Printf("=== reached level=%d after %d grow clicks, state=%s ===\n", f.LevelsCompleted, clicks, f.State)
	bg := perception.BackgroundColor(f.Grid)
	fmt.Printf("=== LEVEL 2 grid %dx%d ===\n", len(f.Grid[0]), len(f.Grid))
	render(f.Grid)
	fmt.Printf("colors: %v  bg=%d\n", colorCounts(f.Grid), bg)
	bp, sav := macro.BestPrimitive(f.Grid, bg)
	fmt.Printf("L2 drive: best=%s savings=%d score=%.3f residualClusters=%d\n",
		bp.Name, sav, macro.DriveScore(f.Grid, bg), len(macro.ResidualTargets(f.Grid, bg, 12)))

	if os.Getenv("RENDER_ONLY") != "" {
		return
	}
	// Targeted sweep of L2: probe a coarse grid, re-solving L1 before each click so
	// every probe is tested from the SAME L2 state. Report only board-changing clicks.
	step := 4
	fmt.Printf("\n=== L2 CLICK SWEEP (step %d), board-changing clicks only ===\n", step)
	h := len(f.Grid)
	w := len(f.Grid[0])
	changers := 0
	for y := 0; y < h; y += step {
		for x := 0; x < w; x += step {
			base, _ := growToLevel(sess, 1) // back to fresh L2
			if base.LevelsCompleted < 1 {
				continue
			}
			f1, err := sess.Step(environment.Action{ID: environment.Action6, X: x, Y: y})
			if err != nil {
				fmt.Println("step:", err)
				return
			}
			d := diffBoard(base.Grid, f1.Grid)
			lvUp := f1.LevelsCompleted > base.LevelsCompleted
			if len(d) > 0 || lvUp || f1.State != environment.StateNotFinished {
				changers++
				show := d
				if len(show) > 5 {
					show = show[:5]
				}
				_, s0 := macro.BestPrimitive(base.Grid, bg)
				_, s1 := macro.BestPrimitive(f1.Grid, bg)
				fmt.Printf("click (%d,%d) -> %d cells, levels=%d state=%s dSavings=%+d %v\n",
					x, y, len(d), f1.LevelsCompleted, f1.State, s1-s0, show)
			}
		}
	}
	fmt.Printf("=== L2 sweep done: %d board-changing clicks ===\n", changers)
}
