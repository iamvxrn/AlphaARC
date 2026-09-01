# core — milestone 1: does one protocol fit three existing mechanisms?

This branch does **not** build a new system. `alphaarc-v0` is frozen on its own
branch; `python/bench/` and its 1600 lines of measured negative results are
untouched and stay the reference.

## The bet

Not "three engines share an internal language" — they do not, and forcing them to
emit the same `State'` would kill the idea with an API before it was tested. The
bet is narrower:

    claim  ->  testable prediction  ->  error scored AT THE LEVEL of the claim

`protocol.py` (124 lines) is the whole contract: `Evidence`, `Hypothesis`, `Claim`,
`PredictionSet` over five levels (cell / relation / transition / action-effect /
constraint), and `score()`, which returns **None when a prediction cannot be tested
by this observation**. Silence and error are different answers, and conflating them
is how an ensemble gets mistaken for reasoning.

## What is wrapped

Nothing new was written. Three mechanisms that already existed:

| engine | what it already was | what it now claims at |
|---|---|---|
| `mdl` | `alphaarc.mdl.PRIMITIVES` — Reflect/Translate/Count/Correspondence | CELL |
| `relational` | same-colour components grouped by shape | RELATION |
| `graph` | `(state, action) -> state'`, understands nothing | TRANSITION |

## Milestone 1.5: semantic validity of the predictions

Milestone 1 only proved the interface fits. It did not prove the predictions meant
anything, and two of them did not:

* `MdlClaim.predict` returned every fourth cell of the board it had just been shown
  and asserted they would stay -- so its 0.004 on vc33 measured the stability of a
  sample, not the error of an MDL model;
* the relational verifier checked a relation on the frame AFTER an action while the
  claim had been made about the frame BEFORE, so its 1.000 meant "the relation
  stopped holding", which the claim had never asserted.

Both are fixed by construction, not by tuning.

**Evidence is now masked.** Engines receive a grid with holes and are told where the
holes are, never what is in them. A `HELD_OUT_CELLS` proposition is therefore about
something the engine could not see. `Reflect` proposes `cell(r,c) = cell(r, axis-c)`
and `Translate` proposes `cell(r,c) = cell(r, c±period)`; both fill the actual holes.
Count and Correspondence generate a saving but no per-cell consequence, so they are
**not offered at all** rather than faked into a 0.000.

**Claims are split by tense.** `RELATION_NOW` is a statement about the current frame
and is checked there; `RELATION_PERSISTS` asserts survival across an action and is
checked on the successor. Conflating them was the whole of the earlier 1.000.

**The verifier is generic.** It dispatches on the proposition's kind and never sees
the engine, the source or the task. Digests are SHA-256 over the board rather than
Python's salted `hash()`, so a fixture means the same thing in two processes.

### Verifier probed directly, independent of any engine

    held_out, every value right      0.000   expected 0.000   ok
    held_out, every value wrong      1.000   expected 1.000   ok
    relation_now, no such objects    None    expected None    ok
    transition, correct successor    0.000   expected 0.000   ok
    transition, wrong successor      1.000   expected 1.000   ok
    nothing to check against         None    expected None    ok
                                                              6/6

The third probe was a genuine bug when written: both named points fell inside the
same background component and the verifier scored "X is identical to X" as a correct
claim. It now refuses when two names resolve to one object -- untestable, not right.

### Negative controls, and the result nobody ordered

    static-mirror            mdl predicts 18 held-out cells   error 0.000
    static-mirror-BROKEN     mdl SILENT
    relational-copy          relational                       error 0.000
    relational-copy-BROKEN   relational SILENT
    vc33-transition          graph                            error 0.000
    vc33-transition-BROKEN   graph                            error 1.000

The expectation was `corrupted -> high error`. What happens is that **two engines of
three answer corruption with silence rather than error**: MDL requires a perfect
symmetry, so one violated cell removes the hypothesis; the relational engine pairs
only components with identical shapes, so damaging one removes the pair. Neither
ever emits a wrong prediction, which is honest, and which means their error path is
exercised only by the direct probes above.

**So milestone 1 is NOT closed by the stated criterion.** One generic verifier
discriminates positive from corrupted through an engine for exactly one family --
`transition`. For the other two the discrimination exists in the verifier and is
unreachable through the engines as written.

That is worth more than a green tick, because it lands directly on milestone 2: an
evaluator that accumulates *prediction error* will mostly receive **silence** from
engines that abstain under uncertainty. Error is only available as a control signal
from mechanisms willing to commit when they might be wrong. Whether to make MDL and
the relational engine commit -- a tolerance instead of a perfect-fit requirement --
is a design question the metacontroller's usefulness may depend on, and it is now on
the table before the controller exists rather than after.

## Milestone 2 (IN PROGRESS): abstention vs falsifiability — resolved **for MDL only**

Milestone 1.5 asked whether engines should emit **approximate or ranked hypotheses**
so they can be falsified instead of falling silent. Measured, the answer is no:
approximation is not what was missing, and it is not one problem.

**Scope of the claim.** This is not "abstention resolved". It is resolved for one
family, `mdl`, whose abstention turned out to be a gate. `relational` is not resolved
and is not fixable this way (section C); `graph` never abstained. Milestone 2 stays
open.

**Status of the evidence below: A–E all executed.** A–D reproduced identically on a
second run (2026-09-02), and section E — which had never completed — ran on that same
pass. Every figure in this section is from a real run.

One caveat on provenance: that run predates the removal of Russian from the runner's
output strings. The logic was untouched by that edit, but **the translated
`run_milestone2.py` has not itself been executed**, so re-run it once before relying
on it again — a broken format string would not show up any other way.

### A. The instrument was weak, and cannot be fixed by masking harder

A held-out prediction over randomly masked cells catches a one-cell violation only at
the rate the mask happens to cover it. Over all 40 corruption sites × 15 masks:

| hidden cells | coverage | caught, `held_out` | ≈ 1−(1−p)² | caught, `universal` |
|---|---|---|---|---|
| 8 | 3.1% | 5.3% | 6.2% | 100% |
| 24 | 9.4% | 17.3% | 17.9% | 100% |
| 64 | 25.0% | 37.3% | 43.8% | 100% |
| 128 | 50.0% | 53.0% | 75.0% | 100% |
| 200 | 78.1% | **32.5%** | 95.2% | 100% |

Coverage predicts the column almost exactly, and at 200 the rate **falls**: masking
hard enough to hit the damage also hides the source cell the prediction would have
been derived from. **There is no mask size at which this instrument becomes
reliable.** Any future claim resting on held-out cell error inherits this ceiling.

**Read the last column precisely.** It says that *refutation of an emitted universal
is not limited to held-out locations* — **not** that universals are mask-independent,
which is false. The axis and period are still fitted on the **masked** board, so the
mask governs *what gets asserted*; it does not govern *what can refute it*. Section D
shows both halves at once: at a 1.00 threshold the engine emitted 30 universals that
were refuted in full, because the mask hid the counterexample from the engine while
leaving it in plain view of the verifier. Selection was fooled; refutation was not.

### B. MDL's abstention *was* a gate, and the fix is boldness, not approximation

`bad == 0` demanded a perfect symmetry, so damage removed the hypothesis. Replaced by
a tolerant fit plus a `UNIVERSAL` proposition: *"this rule holds with **no exception
anywhere**"*, refuted by a single cell, including cells the engine never saw.

    intact     speaks  15/15    refuted    0/15  =   0.0%   mean error 0.0000
    BROKEN     speaks 600/600   refuted  600/600 = 100.0%   mean error 0.0250

Where the gated engine was silent 600/600, it now commits 600/600 and is refuted
every time. **Silence became signal.** Note *why* the bold form beats the hedge: a
hedged "symmetric except at these two cells" is unfalsifiable, and in MDL's own
currency it is also *longer*. Approximate-with-a-caveat is a way of never being
wrong — the same abstention wearing a different coat.

### C. Relational abstention was **not** a gate — negative result, do not retry

Deleting one cell of a 5-cell copy does not break the shape match, it **dissolves the
object**: the remainder falls into two disconnected 2-cell fragments, and the claim
loses its subject. Lowering the size floor and pairing by nearest shape was tried and
is worse than silence, on two independent counts:

* **the claim dodges refutation** — on the broken board the engine re-aims at the two
  identical *fragments* (similarity 1.000) and the verifier scores it **correct**. An
  engine free to choose its subject after seeing the evidence cannot be falsified by
  that evidence;
* **noise becomes error** — on scattered same-colour blobs that are copies of
  nothing, it committed on **37 of 40** boards and was refuted on **23**.

This is a representational limit, not a threshold: the family needs an object
identity that survives its own cells changing. Same wall as `Correspondence` in the
Go stack, reached from a new direction.

### D. The one new constant is swept, not asserted

`COMMIT_AGREE` sits on a wide plateau — every value in **0.50–0.95** gives 15/15 on
intact boards, 200/200 refutations on broken ones, and 0/60 on noise; 0.90 is in the
middle of it. But the sweep also shows **`COMMIT_AGREE` is not what holds noise back**
— `min_agree = 20` is, since noise stays at 0/60 even at a 0.50 threshold. The two
roles are separate and should be kept separate.

At 0.99–1.00 (the old gate) the denominator collapses 200 → 30: the engine goes
silent on 170 boards of 200. The surviving 30 are refuted in full, which is its own
finding — there the mask *hid the counterexample from the engine*, it saw a perfect
symmetry, and it was refuted by a cell it was never shown. **Even a strict gate does
not prevent error; it only makes error rare.**

### E. The noise floor — executed 2026-09-02

`LiarEngine` is not a mechanism — it always commits to the best-looking axis, however
bad the fit. It reports what an engine that knows nothing scores on the same boards.

| board | mdl speaks | mdl error | liar speaks | liar error |
|---|---|---|---|---|
| mirror (intact) | 30/30 | 0.0000 | 30/30 | **0.0000** |
| noise, no structure | **0/30** | — | 30/30 | 0.7239 |
| shuffled mirror | **0/30** | — | 30/30 | 0.7154 |

The real engine is silent on all 60 structureless boards; the liar commits on all 60
and pays ≈0.72. That much was the expectation.

**The first row is the one that teaches something, and it was not anticipated.** On
the board that genuinely *is* mirror-symmetric, the liar scores **0.0000 — identical
to the real engine**. It is not being lucky: the structure is really there, so the
best-looking axis is the true axis. An engine that knows nothing is indistinguishable
from one that knows something, *by error alone*, on any board where the regularity
holds.

So the noise floor is not a single number. The result stated at its actual scope:

> **Prediction error conditional on commitment is insufficient to distinguish
> selective from indiscriminate hypothesis generation; commit/abstain behaviour
> carries additional information.**

Not "prediction error does not separate knowledge from noise" — that is too strong and
was the first way this was written up. The liar's errors on structureless boards are
real and large, so a *sequence* of episodes does expose it. What fails is the
**conditional** quantity: restricted to boards where both engines commit, the two are
observationally identical.

Two quantities are worth logging **separately, and not collapsed into one number**:

    coverage = P(commit)          risk = E[error | commit]

    structured   mdl coverage high, risk 0   |  liar coverage high, risk 0
    noise        mdl coverage 0,    risk —   |  liar coverage 1,    risk ≈0.72

This is richer than a single `confidence`, and the temptation to fuse them into one
scalar should be resisted for the reasons catalogued under the MetaController spec.

**What E did not measure.** It measured *whether* an engine abstains — never whether
it correctly reports *why*. The reason taxonomy (`NO_MODEL`,
`INSUFFICIENT_EVIDENCE`, `REFERENT_LOST`, `UNTESTABLE`) is entirely untested here;
that is M3's territory and an open epistemic-classification problem. An earlier draft
of this section claimed E validated `abstention reason` as an input. It does not.

### Where this leaves milestone 2

`mdl` is now falsifiable, `graph` always was, `relational` is not — and section C
says the reason is representational rather than a threshold. Two families of three
yield error; that is a partial result, not a closed milestone.

The open question it hands forward is whether **prediction failure and representation
failure are the same thing**. They are currently indistinguishable: both arrive as
silence. See `core/run_milestone3.py` — **designed, deliberately not yet run.**

Its first design was confounded and is worth recording as a near miss. The decisive
pair varied `size` — preserved in one arm, destroyed in the other — while the resolver
keyed on `(colour, size, centroid)`, so a perfect score would only have shown that a
size detector detects size. `check_orthogonality()` missed it: it checks that the
string `"shape"` is absent from the descriptor list, which is *syntactic* orthogonality,
and `size` is geometry just as `shape` is. The rewrite fixes this structurally rather
than by tuning the resolver — one part removes the resolver from the argument (two
provenances, one identical after-frame, opposite correct verdicts), the other crosses
identity against descriptors and scores only the rows where they disagree.

The impossibility result in part 1 should be stated exactly, since a looser reading is
tempting: **identity is not derivable from the appearance of a frame pair, when
different causal histories produce identical observations.** It does *not* establish
"identity requires time" — that is a general claim about identity, where this shows
only that the information is absent from this pair. Part 3 asks the empirical
follow-up (`R0(before, after)` against `R1(before, history, after)`) instead of
assuming history is the answer, and a change-mask does not qualify as history: the
diff of the endpoints is a function of the endpoints, so part 1 defeats it too.

Until part 3 has run, the resemblance between this and the `stateful-mode` blocker in
`CLAUDE.md` is **a plausible link between two symptoms, not a measured one**, and
should not be written up as the same gap.

### The M3 close-out procedure — agreed before the run, so the result cannot drift

The design is **frozen**. Nothing below authorises extending it.

Run order, once the execution environment is back:

1. ~~`python3 core/run_milestone2.py` — end to end, for section **E**.~~ **DONE
   2026-09-02**; E's numbers are in the milestone 2 section above. Re-run once after
   the output strings were translated, to confirm no format string broke.
2. `python3 core/run_milestone3.py` — part 2 as a diagnostic.
3. the same run's part 3, including the occlusion control.
4. record the **actual** result, with **no tuning of `size_tol` or `radius`**. Tuning
   a row is not permitted even if a row is one step from passing.

If the hand-traced prediction holds, the claim is exactly this, and is not to be
paraphrased upward:

> Sequential local correspondence can recover identity distinctions unavailable from
> endpoint appearance under continuous observation and bounded per-step change; it is
> not persistence through occlusion.

Whatever the `dsize` column reports as the boundary — the hand-trace expects ≈20% —
it is **the observed boundary of this resolver on this fixture**: one `size_tol`
against one set of hand-built trajectories. It is not a general threshold for how fast
a thing may change and remain itself, and it must not be quoted as one or carried into
other code as a constant.

**M3 then closes on whatever the run says**, positive or negative. No new identity
mechanism and no new tracker. The next code written after the commit belongs to
`MetaController`.

## Milestone 3 result — executed 2026-09-02, closed

Every pre-registered expectation held; nothing was tuned.

**PART 1 — impossibility.** `v_morph()` and `v_replace()` produce an identical grid,
and the diagnostic returns `MODEL_ERROR` for both where the correct answers are
`MODEL_ERROR` and `REPRESENTATION_ERROR`. One is necessarily wrong, for any function of
that frame. Stated at scope: **identity is not derivable from the appearance of a frame
pair when different causal histories produce identical observations.** This is an
identifiability constraint, i.e. an error *floor* — on a balanced pair the best
achievable is 0.5, not 0.

**PART 2 — R0 closed.** Diagonal 5/5, **off-diagonal 0/3**. `replace` and `impostor`
are false survivals, `grown` a false referent loss. The descriptor resolver is a
same-colour-same-size-same-place detector; `radius` and `size_tol` were not touched,
per the rule that tuning only moves which row fails.

**PART 3 — R1 recovers all three.**

| | off-diagonal | controls |
|---|---|---|
| R0 (endpoints) | 0/3 | 5/5 |
| R1 (through history) | **3/3** | **5/5** |

**The envelope transfers, and the evidence is non-monotonicity.** Per `id+` case,
across strides 1–4, all 16 rows agree: every step with `dsize > 0.25` fails, every step
with `dsize ≤ 0.25` passes. Decisively, `morph` **fails at stride 2 and passes at
strides 3 and 4** — impossible if sampling rate were the driver, since coarser
sampling happens to skip its peak step. So the binding quantity is per-step change, not
history density.

The observed boundary is only **bracketed in (0.20, 0.40]** — the fixture contains no
step between those — and it coincides with the `size_tol = 0.25` gate by construction.
The "≈20%" predicted earlier was the largest *passing* value, not the boundary. Either
way it is **the observed boundary of this resolver on this fixture**, not a general
limit on how fast a thing may change and stay itself.

**Occlusion — R1 loses where R0 wins.** B covered for one frame, then uncovered
unchanged. R0 `OK` (correct); R1 `REPRESENTATION_ERROR` (wrong), dying at step 1. The
irreversible-death rule is exactly what makes `impostor` work, and the same rule cannot
also let B return. **More information made the answer worse** — R1 is a different
trade, not a strict improvement.

The sanctioned claim, unchanged from before the run:

> Sequential local correspondence can recover identity distinctions unavailable from
> endpoint appearance under continuous observation and bounded per-step change; it is
> not persistence through occlusion.

M3 is **closed**. No new identity mechanism, no new tracker.

## MetaController v0 — specified before execution, not yet built

Agreed 2026-09-02. **No code for any of this until M2 section E and M3 have run.**
Recorded here for the same reason as the M3 close-out above: a specification written
after the numbers arrive is a specification that fits the numbers.

The thesis it tests: *do not combine the answers of different solvers — combine the
ways a mind changes its knowledge state* (`induce`, `predict`, `falsify`, `explore`,
`refine`, `re-represent`, `plan`). Existing work takes roles rather than becoming
engines: program synthesis is INDUCE, MDL is SELECT, TTT is REFINE, Just-Explore is
SEEK EVIDENCE, graph memory is PREDICT DYNAMICS.

The two finished milestones each hand it one constraint, and both were forced by
negative results rather than chosen:

* **M2 → selection is itself evidence.** Two hypothesis generators can post identical
  prediction error on the tasks where both commit and differ radically in where they
  are willing to commit at all. So the controller cannot be fed one error scalar.
* **M3 → attribution is not free.** When a prediction fails, *what broke* is a further
  inference, and PART 1 shows it is not always recoverable from the evidence at hand.

Together: the controller receives **typed, partially uncertain evidence**, not a
number. That is the shape the experiments force, not a diagram drawn in advance.

**Inputs come in two tiers, and conflating them would manufacture an oracle.**

| tier | signals | trust |
|---|---|---|
| directly observable | commit / abstain; realized prediction error where testable; hypothesis complexity; disagreement between hypotheses; unexplained residual | high — these are facts about what happened |
| self-reported or inferred epistemic type | `MODEL_ERROR`, `REPRESENTATION_ERROR`, `NO_MODEL`, `REFERENT_LOST`, `UNTESTABLE` | **fallible — each is itself a hypothesis** |

The second tier is what the ORACLE / HARD / CORRUPTED / ABSTAIN arms are testing. It
must never be taken at face value: otherwise an engine may tell the controller "I am
silent because REPRESENTATION_ERROR" and we will have quietly granted ourselves the
perfect epistemic oracle whose absence M3 PART 1 proves. Milestone 2 section E is the
concrete warning — it establishes the *observable* half (commit/abstain carries
information) and establishes nothing whatever about the *reported* half.

Log `coverage = P(commit)` and `risk = E[error | commit]` **as two numbers**. Do not
fuse them, for the same reason there is no scalar objective below.

**First experiment: ORACLE / HARD / CORRUPTED / ABSTAIN.** Not iid fault injection —
that tests one controller, not the class, and answers only "alive or dead".

| arm | outcome labels it gets |
|---|---|
| ORACLE | the true epistemic state; the upper bound |
| HARD | the best available classifier under the **actual** observation channel |
| CORRUPTED | HARD plus controlled **extra** confusion |
| ABSTAIN | the same evidence as HARD, but allowed to answer UNKNOWN |

`HARD` and `CORRUPTED` are deliberately **separate conditions**. Merging them mixes
evidence that is fundamentally insufficient with noise we injected ourselves, and those
have different remedies.

**What M3 supplies, stated correctly.** PART 1 does *not* establish that
`MODEL_ERROR ↔ REPRESENTATION_ERROR` confusion has probability 1. It establishes
**non-identifiability**: the two causal histories satisfy

    O(h_morph) = O(h_replace)   while   Y(h_morph) ≠ Y(h_replace)

so any classifier receiving only that observation must emit the **same** label for
both and therefore cannot be correct on both. On a balanced pair the minimum
achievable error is **0.5**, not 1.0 — a classifier that always answers MODEL_ERROR is
right on `morph` and wrong on `replace`. The number 1.0 would describe a different
quantity ("the observation carries no distinguishing information"), which is not a
cell of a confusion matrix.

So M3 gives an **identifiability constraint / error floor per observational regime**,
not measured confusion cells:

| regime | floor on that adversarial pair |
|---|---|
| endpoint appearance only | 0.5 (balanced pair) |
| dense history | possibly 0, *if* PART 3 confirms the separation — unrun |
| temporary occlusion | a different ambiguity regime, with its own floor |

The controller therefore faces a **composition** of two distinct sources, and the
experiment must keep them apart:

* `C_observability` — what the available evidence forbids anyone from knowing; an
  information-theoretic floor, not a defect;
* `C_classifier` — the real error of our epistemic discriminator *above* that floor.

Three questions, one per gap:

1. **Oracle gap** — what does lacking perfect epistemic knowledge cost at all?
2. **Observability gap** — what is lost specifically because some outcomes are
   *in principle* indistinguishable from the available evidence?
3. **Classifier robustness** — how much error above the floor does the controller
   still tolerate?

Measure **solve rate, wasted budget before recovery, and irreversible failures** —
the third separately, because the two misroutings are not symmetric: RE-REPRESENT in
place of REFINE discards a sound hypothesis and rebuilds the ontology, while REFINE in
place of RE-REPRESENT grinds forever on a claim with no subject.

Reading fixed in advance: if HARD is brittle where ABSTAIN degrades gracefully, the
conclusion is **"epistemic outcomes cannot be treated as error-free discrete labels"**
— not that the architecture is dead.

**Three constraints on v0:**

* **No scalar `E[progress] / cost`.** It sums incommensurable units (unbounded error,
  0–1 confidence, bits, a *categorical* abstention reason) — the same shape as the
  failure catalogued in `CLAUDE.md` item 5, where unit-matching measured −0.1653 ±
  0.2080 over 16 paired seeds and lost on both games it was built for (Rejected #7).
  v0 uses a **typed decision policy**; a weighted objective must later beat it in a
  paired measurement to earn its place.
* **`RE-REPRESENT` stays an outcome/need, not a callable operation**, until a separate
  experiment supplies a real mechanism. M3 says when it is needed and nothing about
  how to do it; naming it as an operation would turn a measured gap into a stub.
* **Synergy benchmark frozen before the controller policy exists.** Compare adaptive
  sequencing against **fixed and random** sequencing with identical primitives and
  identical budget. Include tasks solvable by a single operation, so over-sequencing
  costs something. Random sequencing is the noise floor, exactly as `LiarEngine` is in
  milestone 2.

**Intervention log:** decision-level counterfactual replay and **regret**, not a
causal narrative. Replace the controller's choice with its second-best and re-run; a
choice that changes no outcome was not load-bearing. A log of successes only is a
highlight reel, not evidence.

`MetaController` is still **not** built.

## Milestone 1 result (interface only)

