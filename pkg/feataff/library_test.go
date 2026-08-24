package feataff

import "testing"

// A framed exemplar + a partial copy: correspondence should read > 0, and the
// residual should yield real click controls at the mismatch. Offline (no engine)
// -- locks the live-readout wiring against the macro layer.
func TestLibrary_FeaturesAndControlsOnRealStructure(t *testing.T) {
	bg := 9
	box := [][]int{
		{2, 2, 2, 2, 2},
		{2, 4, 3, 4, 2},
		{2, 3, 4, 3, 2},
		{2, 4, 3, 4, 2},
		{2, 2, 2, 2, 2},
	}
	g := make([][]int, 7)
	for r := range g {
		g[r] = make([]int, 14)
		for c := range g[r] {
			g[r][c] = bg
		}
	}
	place := func(top, left int) {
		for r := range box {
			for c := range box[r] {
				g[top+r][left+c] = box[r][c]
			}
		}
	}
	place(1, 1)
	place(1, 8)

	feats := DefaultFeatures()
	byName := map[string]float64{}
	for _, f := range feats {
		byName[f.Name] = f.Eval(g)
	}
	if len(feats) != 5 {
		t.Fatalf("expected 5 default features, got %d", len(feats))
	}
	if byName["correspondence"] <= 0 {
		t.Fatalf("correspondence should read >0 on a matched framed pair, got %v", byName["correspondence"])
	}

	// Knock out one interior cell -> residual should now surface a control there.
	g[3][10] = bg
	ctrls := ResidualControls(g, 12)
	if len(ctrls) == 0 {
		t.Fatal("expected residual to yield real candidate controls")
	}
	for _, c := range ctrls {
		if c.Kind != "click" {
			t.Fatalf("controls should be clicks, got %+v", c)
		}
	}
}
