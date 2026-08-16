package perception

import "testing"

// TestBlobSalienceRanksRareHighest: a lone-colored blob among many same-colored
// ones must score highest -- the distinctive thing stands out.
func TestBlobSalienceRanksRareHighest(t *testing.T) {
	// Three color-5 blobs and one lone color-9 blob.
	labeled := []LabeledBlob{
		{Blob: Blob{Color: 5}, Label: "a"},
		{Blob: Blob{Color: 5}, Label: "b"},
		{Blob: Blob{Color: 5}, Label: "c"},
		{Blob: Blob{Color: 9}, Label: "rare"},
	}
	s := BlobSalience(labeled)
	if !(s[3] > s[0] && s[3] == 1.0 && s[0] < 0.5) {
		t.Fatalf("FAIL: rare blob not most salient: %v", s)
	}
}
