package pipeline

import (
	"hash/fnv"
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
// This exists to replace RunPredictiveCycle's current stateVector, which is
// a pseudo-random vector derived purely from e.StepCounter's parity --
// carrying zero information about what was actually observed, so the same
// observation content never produces the same vector twice and different
// observations are indistinguishable to the Predictor MLP by construction.
// ObservationVector fixes exactly that: the same observation string always
// maps to the same vector (needed for a real forward model to have any
// hope of learning "given this content, predict the next"), and different
// token sets map to different vectors with high probability.
//
// Deliberately NOT yet wired into RunPredictiveCycle -- this is the first,
// isolated, independently testable piece of a larger predictive-coding
// change (caching a cycle's prediction and comparing it against the next
// cycle's actual embedding as a real prediction-error signal, replacing the
// same-cycle synthetic target RunPredictiveCycle's Predictor is trained
// against today). That wiring is a substantially bigger, riskier change --
// see predictive_loop.go's own comment on why the current mlpLoss-derived
// signal was already found to be unlearnable noise and deliberately
// excluded from driving Hebbian plasticity -- and deserves its own review
// before touching the shared engine loop every existing test exercises.
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
	return vec
}
