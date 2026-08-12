package perception

import "testing"

// grid indices are grid[y][x] throughout, matching environment.Frame.Grid.

func TestBackgroundColorDominantWins(t *testing.T) {
	grid := [][]int{
		{0, 0, 0, 0, 0},
		{0, 3, 3, 0, 0},
		{0, 3, 0, 0, 0},
		{0, 0, 0, 5, 5},
		{0, 0, 0, 0, 0},
	}
	if bg := BackgroundColor(grid); bg != 0 {
		t.Fatalf("FAIL: expected background 0 (20 of 25 cells), got %d", bg)
	}
}

func TestBackgroundColorAllOneValue(t *testing.T) {
	grid := [][]int{{5, 5}, {5, 5}}
	if bg := BackgroundColor(grid); bg != 5 {
		t.Fatalf("FAIL: expected background 5, got %d", bg)
	}
}

func TestBackgroundColorTieBreaksOnLowestValue(t *testing.T) {
	// color 1 and color 9 each appear twice -- deterministic tie-break
	// must pick the lower value (1), not depend on map iteration order.
	grid := [][]int{{1, 9}, {9, 1}}
	if bg := BackgroundColor(grid); bg != 1 {
		t.Fatalf("FAIL: expected tie-break to pick lowest color 1, got %d", bg)
	}
}

func TestFindBlobsTwoDisjointRegions(t *testing.T) {
	grid := [][]int{
		{0, 0, 0, 0, 0},
		{0, 3, 3, 0, 0},
		{0, 3, 0, 0, 0},
		{0, 0, 0, 5, 5},
		{0, 0, 0, 0, 0},
	}
	blobs := FindBlobs(grid, 0)
	if len(blobs) != 2 {
		t.Fatalf("FAIL: expected 2 blobs, got %d: %+v", len(blobs), blobs)
	}

	// Discovery order is row-major, so the color-3 blob (rows 1-2) is
	// found before the color-5 blob (row 3).
	b3, b5 := blobs[0], blobs[1]
	if b3.Color != 3 || len(b3.Cells) != 3 {
		t.Fatalf("FAIL: expected first blob color=3 size=3, got color=%d size=%d", b3.Color, len(b3.Cells))
	}
	if b3.Centroid != (Point{X: 1, Y: 1}) {
		t.Fatalf("FAIL: expected color-3 centroid (1,1) (sum (4,4)/3 truncated), got %+v", b3.Centroid)
	}

	if b5.Color != 5 || len(b5.Cells) != 2 {
		t.Fatalf("FAIL: expected second blob color=5 size=2, got color=%d size=%d", b5.Color, len(b5.Cells))
	}
	if b5.Centroid != (Point{X: 3, Y: 3}) {
		t.Fatalf("FAIL: expected color-5 centroid (3,3), got %+v", b5.Centroid)
	}
}

func TestFindBlobsCentroidTruncatesTowardZero(t *testing.T) {
	// L-shape: (0,0),(1,0),(0,1) -- sum=(1,1), n=3, true centroid (0.33,0.33),
	// integer division truncates to (0,0).
	grid := [][]int{
		{4, 4, 0},
		{4, 0, 0},
		{0, 0, 0},
	}
	blobs := FindBlobs(grid, 0)
	if len(blobs) != 1 {
		t.Fatalf("FAIL: expected 1 blob, got %d", len(blobs))
	}
	if blobs[0].Centroid != (Point{X: 0, Y: 0}) {
		t.Fatalf("FAIL: expected centroid (0,0) via truncation, got %+v", blobs[0].Centroid)
	}
}

func TestFindBlobsOnUniformGridIsEmpty(t *testing.T) {
	grid := [][]int{{0, 0}, {0, 0}}
	blobs := FindBlobs(grid, 0)
	if len(blobs) != 0 {
		t.Fatalf("FAIL: expected 0 blobs on an all-background grid, got %d", len(blobs))
	}
}

func TestFindBlobsRespectsFourConnectivityNotDiagonal(t *testing.T) {
	// Two same-color cells touching only at a corner (diagonal) must be
	// two separate blobs under 4-connectivity.
	grid := [][]int{
		{2, 0},
		{0, 2},
	}
	blobs := FindBlobs(grid, 0)
	if len(blobs) != 2 {
		t.Fatalf("FAIL: expected 2 diagonal-only blobs under 4-connectivity, got %d", len(blobs))
	}
}

func TestDescribeGridRanksBySizeThenDirection(t *testing.T) {
	grid := [][]int{
		{0, 0, 0, 0, 0},
		{0, 3, 3, 0, 0},
		{0, 3, 0, 0, 0},
		{0, 0, 0, 5, 5},
		{0, 0, 0, 0, 0},
	}
	// grid center (cx,cy) = (2,2). color3 centroid (1,1) -> north west.
	// color5 centroid (3,3) -> south east. color3 (size 3) outranks
	// color5 (size 2).
	got := DescribeGrid(grid, 2)
	want := "color3 north west color5 south east"
	if got != want {
		t.Fatalf("FAIL: expected %q, got %q", want, got)
	}
}

func TestDescribeGridTieBreaksBySizeEqualOnColorNotDiscoveryOrder(t *testing.T) {
	// color7 is discovered first (top-left, scanned first) but color2
	// (bottom-right) must sort ahead of it once sizes tie, since the
	// tie-break is by color ascending, not discovery order.
	grid := [][]int{
		{7, 0, 0},
		{0, 0, 0},
		{0, 0, 2},
	}
	got := DescribeGrid(grid, 2)
	want := "color2 south east color7 north west"
	if got != want {
		t.Fatalf("FAIL: expected %q (sorted by color, not scan order), got %q", want, got)
	}
}

func TestDescribeGridRespectsMaxBlobs(t *testing.T) {
	grid := [][]int{
		{0, 0, 0, 0, 0},
		{0, 3, 3, 0, 0},
		{0, 3, 0, 0, 0},
		{0, 0, 0, 5, 5},
		{0, 0, 0, 0, 0},
	}
	got := DescribeGrid(grid, 1)
	want := "color3 north west"
	if got != want {
		t.Fatalf("FAIL: expected only the top-1 blob %q, got %q", want, got)
	}
}

func TestDescribeGridEmptyOnUniformGrid(t *testing.T) {
	grid := [][]int{{5, 5}, {5, 5}}
	if got := DescribeGrid(grid, 3); got != "empty" {
		t.Fatalf("FAIL: expected \"empty\" for an all-background grid, got %q", got)
	}
}

func TestDescribeGridEmptyOnZeroSizedGrid(t *testing.T) {
	if got := DescribeGrid(nil, 3); got != "" {
		t.Fatalf("FAIL: expected \"\" for a nil grid, got %q", got)
	}
	if got := DescribeGrid([][]int{}, 3); got != "" {
		t.Fatalf("FAIL: expected \"\" for an empty grid, got %q", got)
	}
}

func TestDescribeGridCenteredBlobEmitsCenterWord(t *testing.T) {
	// 3x3 grid, center cell (1,1) is the only non-background cell --
	// centroid == grid center exactly, so direction() must return "center".
	grid := [][]int{
		{0, 0, 0},
		{0, 9, 0},
		{0, 0, 0},
	}
	got := DescribeGrid(grid, 1)
	want := "color9 center"
	if got != want {
		t.Fatalf("FAIL: expected %q, got %q", want, got)
	}
}
