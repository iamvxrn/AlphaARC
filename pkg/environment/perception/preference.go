package perception

// StructureScore is a first "source of meaning" -- a prior preference over
// states (Friston's C: which observations are DESIRED), the piece the whole
// ecosystem was missing. Every other mechanism optimizes a means (predict
// better, wire tighter); none knew whether a state was GOOD. This says one
// thing: a more STRUCTURED / consolidated grid is preferred.
//
// The score is the Simpson concentration of blob colors -- sum(group_i^2) /
// total^2, in (0, 1]. It's 1.0 when every blob shares one color (maximally
// consolidated) and ~1/N when all N blobs are distinct colors (maximally
// scattered). Higher = more order.
//
// HONEST placeholder: "order = the goal" is a GUESS, right for the many ARC
// games about grouping/matching/tidying and wrong for those that want the
// opposite (Chollet built them to resist any single prior). The VALUE here is
// the architecture -- a swappable preference the pragmatic loop steers toward
// -- not this specific rule. Returns 0 for an empty grid (no blobs, nothing
// to prefer).
func StructureScore(grid [][]int) float64 {
	blobs := FindBlobs(grid, BackgroundColor(grid))
	if len(blobs) == 0 {
		return 0
	}
	perColor := make(map[int]int)
	for _, b := range blobs {
		perColor[b.Color]++
	}
	total := len(blobs)
	sumSq := 0
	for _, n := range perColor {
		sumSq += n * n
	}
	return float64(sumSq) / float64(total*total)
}

// BlobSalience scores each labeled blob by how much it STANDS OUT -- the
// inverse frequency of its color in the frame. A blob whose color appears once
// scores 1.0; one of ten same-colored blobs scores 0.1. This is the "curiosity
// toward the unusual" signal: the agent is drawn to what's rare/distinctive
// (a lone different block), not to a "key" it couldn't recognize as a key --
// distinctiveness is a prior for relevance without assuming the goal. Computed
// over the WHOLE set passed (see the map: nothing distinctive is missed as
// long as the caller doesn't cap the blob set). Returns one score per input
// blob, in order.
func BlobSalience(labeled []LabeledBlob) []float64 {
	colorCount := make(map[int]int)
	for _, lb := range labeled {
		colorCount[lb.Blob.Color]++
	}
	out := make([]float64, len(labeled))
	for i, lb := range labeled {
		out[i] = 1.0 / float64(colorCount[lb.Blob.Color])
	}
	return out
}
