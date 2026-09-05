# E0 boundary failure: invisible query-only direction reversal

## Frozen observation

The diagnostic was specified in SPEC.md before execution. The stationary
reference scores 0/27,648 under query-only direction reversal, while the
environment oracle scores 27,648/27,648. No architecture change followed.

Reproduction: `python3 -B run_experiment.py --output results-boundary-recheck.json`.
Exact source/spec digests are recorded in the original `results.json`.
Spec SHA-256: `09e481dda9140a895fedf9b8c630d3e47705ba8912fd77c9af77c1ac67114966`.
No sampled seeds; exact enumeration, zero sampling noise.

## Minimal case

Take the identity label assignment: action 0 increases dial 1, action 1
decreases it, action 2 increases dial 2, action 3 decreases it. Start at
(0,0,0,0), execute labels 0,1,2,3 and record their displacements. Acquisition
ends at (0,0,0,0) in both stationary and query-reversed worlds.

At query (0,0,0,1), the learned reference chooses action 2. In the stationary
world it reaches (0,1,0,1), solving the query. In the reversed world it reaches
(0,3,0,1), failing it; action 3 is now correct.

## Competing explanations considered in this report

1. A learning/scoring implementation defect prevents applying acquired evidence.
2. An explicit stationarity assumption is violated only outside the acquisition
   support, so acquisition cannot identify which query behavior will occur.

The pre-run spec predicted success in the stationary condition and zero in the
reversal condition. These explanations are spelled out here after the run;
this is not a claim that this prose was independently preregistered.

## Discriminator and conclusion

For every rule and action order, acquisition observations/actions/transitions
are identical in the two worlds: t2 remains zero and the reversal predicate is
false. The query observation is also identical. Therefore any learner receiving
only that history and observation has the same action distribution in the two
worlds. The correct actions differ. No such learner can be guaranteed correct
in both worlds on that evidence.

The stationary control succeeds on all cases; the independent transition-based
oracle succeeds in both conditions, and mutation tests detect corrupted learned
directions. This supports an evidence-identifiability limit at this intervention,
not a diagnosis that the learner needs a larger architecture. No claim is made
that this is the only possible failure mechanism in broader tasks.

Status: EXPECTED TRANSFER FAILURE OBSERVED; EXPLICIT PRIOR VIOLATED;
ACQUISITION EVIDENCE DOES NOT IDENTIFY THE INTERVENTION.

This diagnostic does not license installing BDH, adding a world model, or making
a minimal-K0 claim. Further study would first need to state what extra evidence
or what justified family restriction makes the query behavior identifiable.
