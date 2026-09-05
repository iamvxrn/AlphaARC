# Astra 003: bounded audit and existence witness

Written before running the new witness. Tooling only, no change to frozen Q1,
no chosen production learner, no commits or push. Budget: one CPU process,
120 seconds maximum, exhaustive finite enumeration with exact fractions.

The v0 audit is retrospective: known expected results are 74 designs and
per-design minima 1/2. Pin referee source to c039d2a; record its SHA-256.
Record the distribution over ALL probes, not just each design's minimum.
Also check the concrete Q1 probe (0,1,0,2) independently of referee code.

## New mathematical existence witness, outside the dial family

Observation (x,t). Current symbols x in {0,1,2,3,4,5}; targets t in
{0,1,2,3,4}, with target 4 meaning any symbol in {0,1,2,3}.
There are four opaque actions and 24 rules R, each a bijection to {0,1,2,3}.
Every action overwrites x with R[action], preserving t. This state-independent
overwrite family and the objective semantics are explicit supplied priors.

Acquisition resets to (4,4). Every action goes to a goal (R[action],4).
Progress means strictly reducing potential 1-goal; non-goal acquisition support
is exactly {(4,4)}. Enumerate each action once in the transition dataset.
The single probe is (5,0): unseen x and target, unique correct action R^-1(0).

For every subset K of the two observation features, define the same per-key
progress counts and projected progress triples as in referee v1. Prediction:
all subsets have optimum 1/4 under either statistic. The empty subset has
uniform counts or identical projected triples; every nonempty subset has an
unseen probe key. Full acquisition triples recover all 24 assignments, so the
information optimum from full history is 1, and a map lookup from observed
outputs attains it. Erasing that history collapses the information optimum to
1/4. Full triples at the empty key likewise reach 1, demonstrating inclusion.

This tests existential feasibility within an explicit finite family, not a new
benchmark result or learned invariance. It changes dynamics/progress/interface
relative to Q1. If unobserved states may arbitrarily change action semantics,
the transfer claim does not follow; that alternative is outside this family.

Falsifier: any projected-statistic optimum above 1/4, a full-history optimum
below 1, incorrect explicit action witness, or disagreement with direct
enumeration of all four-action decision functions when there is a single
indistinguishable data class. Preserve failing output before any repair.
