package perception

import "fmt"

// NumericTokens gives the agent a genuine (if coarse) SENSE OF NUMBER
// (numerosity, a Core-Knowledge prior): it emits "nobj<k>" -- how many objects
// are in the frame, capped -- fed into the observation alongside the categorical
// object-identity tokens. Everything else in the observation is a hashed token
// with no magnitude; this one carries "how many bodies" as an ordered bucket so
// the forward model and graph can reason about quantity.
//
// (The former "tdist" target-distance token and the whole salient-target
// "composer" it depended on -- SalientTargetCentroid/TargetDistance/
// ApproachPreference -- were removed in the consolidation pass: the "rare color
// = the goal" heuristic was disproven by measurement, so tdist was feeding the
// observation a distance to a phantom target. Goal-directedness now comes from
// the hypothesis satisfaction + affordance rollout, not a hand-guessed target.)
func NumericTokens(grid [][]int) []string {
	blobs := FindBlobs(grid, BackgroundColor(grid))
	return []string{fmt.Sprintf("nobj%d", min(len(blobs), 12))}
}
