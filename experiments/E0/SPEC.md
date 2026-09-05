# E0: transition-acquisition reference experiment

This is a new engineering reference experiment, not a revision or rerun of Q1.
The source repository was read at core-zero HEAD 3094ad1. Q1 remains rejected.
The user authorized constructing a working reference after reviewing the
contradictions in Q1. No claim of minimal K0, general intelligence, or discovery
of an architecture is made.

## Freeze and sequence

Write and SHA-256 this document before implementing or running E0. Preserve
the digest in FREEZE.json. This is a local, auditable ordering record, not an
independent timestamp or immutable preregistration. Do not commit or push.
If a check fails, preserve its output and diagnose before changing anything.

## Explicit engineering assumptions (K0)

Two dial coordinates take values in Z/4Z. Observations are (d1,d2,t1,t2).
The coordinate order, target-equality objective, modulo-four arithmetic,
stationarity of action displacements and four available opaque actions are
known. These are substantial hand-supplied priors. The label-to-displacement
assignment is unknown. Action labels have no semantics and receive a fresh
bijection to (+1,0),(-1,0),(0,+1),(0,-1) in each environment.

This experiment measures acquisition of an instance within a supplied family;
it does not measure acquisition of that family or satisfy contract L5 by itself.
No pretrained weights, language model, neural network, or BDH is used as learner.

## Acquisition and learner

Start at (0,0,0,0). Targets remain (0,0); there are no automatic resets.
Execute each action once, following each of the 24 possible action orders.
The learner receives only (observation, chosen action, next observation).
Store the observed modular dial displacement under the chosen action label.
The update rule is this explicit dictionary insertion; it is an engineering
choice, not a mechanism justified as necessary by the rejected Q1.

At query time, predict successor dials using stored displacements. If a known
action reaches the observed target, select it. Otherwise choose uniformly among
unobserved actions; if all actions are known but none reaches the target, choose
uniformly over all actions. Return action probabilities; score their exact
expectation instead of sampling a tie. All queries are read-only and independent:
no query outcome or feedback is returned to the learner, and no query updates it.

## Held-out queries

Enumerate every observation with t2 in {1,2,3} for which exactly one of the four
environment actions reaches the goal in one press. There are 48 such queries,
balanced across the four required effects. All are unreachable in acquisition
because t2 is fixed to zero there. Evaluate all 24 rule assignments and 24
acquisition orders: 27,648 matched query cases at each acquisition budget.

## Conditions and exact predictions fixed before execution

Report budgets k=0,1,2,3,4 observed transitions from the same acquisition trace.

* MATCHED: acquisition and query use the same rule. Expected aggregate accuracy
  at the five budgets is 1/4, 1/2, 3/4, 1, 1. The k=3 prediction uses elimination
  of known unsuccessful actions when only one action remains unseen.
* INDEPENDENT: query rule is an independent uniform draw from all 24 assignments,
  including the acquisition rule. Use every acquisition rule/order/query-rule/
  query combination, 663,552 cases per budget. Expected accuracy is 1/4 at every
  budget. Reusing a prediction across query rules is allowed because predictions
  have no access to the query rule; count this explicitly.
* ERASED: erase the acquired dictionary after k=4 and before each query.
  Expected accuracy 1/4 over 27,648 cases.
* EXACT-OBSERVATION LOOKUP: memorize action/successor observations from acquisition
  under the complete observation key. At unseen keys choose uniformly. Expected
  accuracy 1/4: every query key is unseen.
* AXIS-ONLY: after full acquisition retain only which dial each action changes;
  discard direction. Choose uniformly between actions on the required dial.
  Expected accuracy 1/2. This is a deliberately constructed policy, not an
  inference that all systems scoring 1/2 must have this representation.
* ORACLE: choose by the environment transition evaluator. Expected accuracy 1.

Also report two diagnostics, not new success requirements for a general learner:

* EXCLUDED-IDENTICAL control: after full acquisition, evaluate all query rules
  different from the acquisition rule. Expected accuracy 5/23 (120/552 rule
  pairs for each fixed required effect), demonstrating Q1's control dependence.
* STATIONARITY VIOLATION: reverse both action directions only when t2 != 0.
  Keep acquisition unchanged. The learned stationary-displacement reference is
  expected to score 0, while the oracle remains 1. This intentionally violates
  an explicit prior and marks the boundary of the reference's transfer.

## Validation and failure handling

Use integer/fraction arithmetic for scores. Implement environment action effects
in a separate module from the learner; the learner must not import that module,
receive R, or access query outcomes. Verify transition shapes and label domain.
Check label-renaming equivariance, acquisition target invariance, query novelty,
balanced effects, absence of query-state mutation, and erasure. Verify scoring
against an independently expressed cyclic-distance oracle, and exercise deliberately
wrong/no-op learning and memory erasure so a broken learner cannot pass.
Include provenance digests for spec, learner, environment, runner and tests in
machine-readable results. Record actual results, counts, timings and any failure.

## What passing licenses

Under the explicit stationary modular-displacement prior, a small implemented
system extracts an initially unknown action assignment from its own transitions
and applies it correctly to unseen target configurations. Its advantage depends
on preserving relevant acquisition information. A perfect result is expected
from construction and validates the reference/instrument; it is not a new
scientific discovery or evidence that Core-Zero's broad hypothesis is established.

The next research question may use this reference as a measured starting point;
changing its prior or task family requires a separately stated experiment.
