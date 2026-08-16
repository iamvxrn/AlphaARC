package perception

import "testing"

// TestObjectSignatureIsPositionInvariant is the core objectness property: a
// blob and the SAME blob shifted keep one identity (an object that moves is
// still that object), while a different shape or color is a different object.
func TestObjectSignatureIsPositionInvariant(t *testing.T) {
	// An L-shape at one place and the same L-shape shifted by (+3,+5).
	base := Blob{Color: 3, Cells: []Point{{2, 2}, {2, 3}, {3, 3}}}
	moved := Blob{Color: 3, Cells: []Point{{5, 7}, {5, 8}, {6, 8}}}
	if ObjectSignature(base) != ObjectSignature(moved) {
		t.Fatalf("FAIL: a moved object lost its identity: %q vs %q", ObjectSignature(base), ObjectSignature(moved))
	}

	// Different shape -> different object.
	otherShape := Blob{Color: 3, Cells: []Point{{2, 2}, {3, 2}, {4, 2}}} // a line, not an L
	if ObjectSignature(base) == ObjectSignature(otherShape) {
		t.Fatalf("FAIL: different shapes collapsed to one identity")
	}
	// Same shape, different color -> different object.
	otherColor := Blob{Color: 9, Cells: []Point{{2, 2}, {2, 3}, {3, 3}}}
	if ObjectSignature(base) == ObjectSignature(otherColor) {
		t.Fatalf("FAIL: different colors collapsed to one identity")
	}
}

// TestDescribeGridStructuralCarriesPersistentObjectIdentity: an object that
// moves keeps the SAME identity token in the observation across frames, so the
// graph node persists instead of fragmenting.
func TestDescribeGridStructuralCarriesPersistentObjectIdentity(t *testing.T) {
	// A 2-cell color-3 object at row 5, then the same object moved up to row 3.
	frameA := make([][]int, 8)
	frameB := make([][]int, 8)
	for i := range frameA {
		frameA[i] = make([]int, 8)
		frameB[i] = make([]int, 8)
	}
	frameA[5][2], frameA[5][3] = 3, 3 // horizontal pair at row 5
	frameB[3][2], frameB[3][3] = 3, 3 // same pair, moved up to row 3

	sig := ObjectSignature(Blob{Color: 3, Cells: []Point{{2, 5}, {3, 5}}})
	obsA := DescribeGridStructural(frameA, 10, 8, 8)
	obsB := DescribeGridStructural(frameB, 10, 8, 8)
	if !contains(obsA, sig) || !contains(obsB, sig) {
		t.Fatalf("FAIL: object identity %q not present in both frames' observations", sig)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
