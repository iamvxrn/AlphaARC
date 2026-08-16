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
