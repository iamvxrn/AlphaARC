package perception

import (
	"fmt"
	"math"
)

// SalientTargetCentroid returns the grid position of the most DISTINCTIVE
// object -- the centroid of the blob whose color is rarest in the frame -- as
// the current best GUESS at "the thing worth reaching" (the key/goal). It's the
// spatial counterpart of BlobSalience: distinctiveness is a prior for relevance
// without assuming what the goal actually is. ok=false for an empty grid.
//
// HONEST placeholder, exactly like StructureScore: "rare = the target" is a
// guess, right for the many games with a lone special block and wrong where the
// goal is something else. The value is the ARCHITECTURE -- a swappable target
// the composer steers toward -- not this specific rule.
func SalientTargetCentroid(grid [][]int) (Point, bool) {
	blobs := FindBlobs(grid, BackgroundColor(grid))
	if len(blobs) == 0 {
		return Point{}, false
	}
	perColor := make(map[int]int)
	for _, b := range blobs {
		perColor[b.Color]++
	}
	best, bestScore := 0, -1.0
	for i, b := range blobs {
		// rarer color scores higher; tie-break toward the smaller body, which is
		// the more point-like "marker" a lone key tends to be.
		s := 1.0/float64(perColor[b.Color]) - 1e-6*float64(len(b.Cells))
		if s > bestScore {
			bestScore, best = s, i
		}
	}
	return blobs[best].Centroid, true
}

// TargetDistance is the composer's scalar: the distance (in cells) from the
// salient target to the NEAREST OTHER object -- "how close is any controllable
// body to the key". It falls toward 0 as a body approaches the distinctive
// target and is the quantity the spatial drive tries to reduce. ok=false when
// there aren't at least two objects (nothing to bring together yet).
//
// Which body is the mover is deliberately NOT assumed (we can't yet tell the
// agent-avatar from scenery): taking the nearest of all other objects makes the
// scalar drop whenever ANYTHING converges on the target, which is the honest
// signal available before object roles are known.
func TargetDistance(grid [][]int) (float64, bool) {
	blobs := FindBlobs(grid, BackgroundColor(grid))
	if len(blobs) < 2 {
		return 0, false
	}
	perColor := make(map[int]int)
	for _, b := range blobs {
		perColor[b.Color]++
	}
	ti, tscore := 0, -1.0
	for i, b := range blobs {
		s := 1.0/float64(perColor[b.Color]) - 1e-6*float64(len(b.Cells))
		if s > tscore {
			tscore, ti = s, i
		}
	}
	target := blobs[ti].Centroid
	best := math.MaxFloat64
	for i, b := range blobs {
		if i == ti {
			continue
		}
		dx, dy := float64(b.Centroid.X-target.X), float64(b.Centroid.Y-target.Y)
		if d := math.Hypot(dx, dy); d < best {
			best = d
		}
	}
	return best, true
}

// spatialApproachWeight is the maximum pragmatic reward for a body sitting right
// on the salient target -- the strength of the composer's gradient. Set
// comparable to the structural prior (0.1) so it meaningfully steers the
// pre-win agent toward the key without overwhelming a genuinely learned goal
// once one exists. A guess, tunable against a live run.
const spatialApproachWeight = 0.15

// ApproachPreference turns TargetDistance into a prior preference in
// [0, spatialApproachWeight]: maximal when a body is on the distinctive target,
// decaying to 0 as the nearest body gets a full grid-diagonal away. This is the
// COMPOSER, expressed as meaning rather than a separate planner: credited per
// action via Engine.AttributePreferenceGain, it makes BestAction prefer the
// action that brings a body toward the key -- goal-directed spatial navigation
// -- BEFORE any level is completed. 0 when there's nothing to bring together.
func ApproachPreference(grid [][]int) float64 {
	d, ok := TargetDistance(grid)
	if !ok {
		return 0
	}
	rows := len(grid)
	cols := 0
	if rows > 0 {
		cols = len(grid[0])
	}
	diag := math.Hypot(float64(rows), float64(cols))
	if diag == 0 {
		return 0
	}
	prox := 1 - d/diag
	if prox < 0 {
		prox = 0
	}
	return spatialApproachWeight * prox
}

// numericRings is how many quantization buckets the continuous magnitudes
// (object count, target distance) are folded into for NumericTokens. Coarse on
// purpose: a hashed-token observation can't carry a raw real, and coarse
// magnitude ("near/mid/far", "few/many") is the sense of number the forward
// model and graph actually need to reason about approach and quantity.
const numericRings = 6

// NumericTokens gives the agent a genuine (if coarse) SENSE OF NUMBER, fed into
// the observation alongside the object-identity tokens. Everything else in the
// observation is a categorical hashed token with no magnitude; these encode
// quantity and distance as ORDERED buckets so ObservationVector carries "how
// many bodies" and "how far the nearest body is from the key" -- the magnitudes
// the composer reasons over -- instead of smashing them into opaque hashes.
//
// Emitted (when defined): "nobj<k>" (object count, capped) and "tdist<k>"
// (target distance quantized into numericRings, 0 = touching, high = far).
func NumericTokens(grid [][]int) []string {
	blobs := FindBlobs(grid, BackgroundColor(grid))
	tokens := []string{fmt.Sprintf("nobj%d", min(len(blobs), 12))}
	if d, ok := TargetDistance(grid); ok {
		rows := len(grid)
		cols := 0
		if rows > 0 {
			cols = len(grid[0])
		}
		diag := math.Hypot(float64(rows), float64(cols))
		ring := 0
		if diag > 0 {
			ring = int(math.Round(d / diag * float64(numericRings-1)))
			if ring > numericRings-1 {
				ring = numericRings - 1
			}
		}
		tokens = append(tokens, fmt.Sprintf("tdist%d", ring))
	}
	return tokens
}
