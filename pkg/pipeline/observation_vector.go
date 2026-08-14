package pipeline

import (
	"hash/fnv"
	"math"
	"strings"
)

// ObservationVectorDim is the fixed length ObservationVector always
// returns -- kept equal to the placeholder stateVector length
// RunPredictiveCycle currently hardcodes (predictive_loop.go), so this can
// later drop in as a replacement without touching Predictor/Actor/
// Associator MLP input-layer sizing.
const ObservationVectorDim = 20

// ObservationVector deterministically embeds an observation string (e.g.
// perception.DescribeGridCells output, "color3-cell0-0 north color5-cell1-1")
// into a fixed ObservationVectorDim-length vector via the hashing trick: each
// whitespace-separated token is hashed to one dimension and a +1/-1 sign,
// and tokens landing on the same dimension accumulate.
//
// This IS RunPredictiveCycle's stateVector (predictive_loop.go Step 0b). It
// replaced a pseudo-random vector derived purely from e.StepCounter's parity
// -- which carried zero information about what was actually observed, so the
// same observation content never produced the same vector twice and
// different observations were indistinguishable to the Predictor MLP by
// construction. ObservationVector fixes exactly that: the same observation
// string always maps to the same vector (a hard requirement for the forward
// model in Step 5 to have any hope of learning "given this content, predict
// the next"), and different token sets map to different vectors with high
// probability.
func ObservationVector(observation string) []float64 {
	vec := make([]float64, ObservationVectorDim)
	for _, token := range strings.Fields(observation) {
		h := fnv.New32a()
		_, _ = h.Write([]byte(token)) // fnv.Write never errors
		sum := h.Sum32()

		idx := int(sum % uint32(ObservationVectorDim))
		sign := 1.0
		if (sum/uint32(ObservationVectorDim))%2 == 0 {
			sign = -1.0
		}
		vec[idx] += sign
	}

	// L2-normalize to unit length so the vector's MAGNITUDE is scale-invariant:
	// a 3-token frame and a 300-token one (or structural tokens weighted 3x)
	// produce vectors of the same length. This is the root fix for a whole
	// class of scale bugs -- most acutely, an unnormalized large observation
	// (the live game's ~27 blobs times a 3x structural weight) drove the
	// forward-model MLP's input magnitude high enough that training diverged
	// and forecast error hit ~1e281. With unit inputs the MLP stays bounded,
	// and forecast error (MSE) lands in a fixed ~[0, 0.2] range regardless of
	// observation size, which is what lets the surprise/settled thresholds be
	// fixed numbers that hold across frames of any size. An all-zero vector
	// (empty observation) is left as-is (nothing to normalize).
	norm := 0.0
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec
}
