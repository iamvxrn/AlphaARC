package perception

// (StructureScore -- the Simpson-concentration "order = good" prior -- was
// removed in the consolidation pass. It was a hand-guessed source of meaning,
// superseded by the hypothesis satisfaction / compression preference, and no
// longer called by any live path.)

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
