"""Milestone 3: is prediction failure the same thing as representation failure?

    NOT YET EXECUTED.  Written while the Bash tool was down; every expectation below
    is HAND-TRACED against the fixture, not measured. Nothing here may be quoted, and
    nothing here may be committed as a result, until this file has run end to end.
    Expectations are printed next to outcomes precisely so a hand-trace that was wrong
    shows up as a mismatch instead of being silently absorbed.

The claim under test is `identity(A.shape, B.shape)`, and the question is whether
"the hypothesis was wrong" can be told from "the subject ceased to exist" without
consulting ground truth about which object is B.

    MODEL_ERROR          -> revise the hypothesis
    REPRESENTATION_ERROR -> revise the representation

Milestone 2 section C showed why this is delicate: allowed to nominate a new B' after
seeing the frame, the engine picks two identical fragments and scores "correct". So
the referent is frozen on the BEFORE frame and must afterwards re-resolve or die.

THE FIRST VERSION OF THIS EXPERIMENT WAS CONFOUNDED, and the rewrite is the point.
Its decisive pair was `shape-swapped` (colour, size and place preserved) against
`dissolved` (size destroyed), while the resolver keys on `(colour, size, centroid)`.
One intervention preserved the descriptor and the other broke it, so the verdicts
were settled by construction: a perfect score would have demonstrated that a size
detector detects size. `check_orthogonality` did not catch this because it only
checks that the string "shape" is absent from the descriptor list, and `size` is
geometry as much as `shape` is.

Two structural repairs, so that the run is informative whichever way it goes:

PART 1 removes the resolver from the argument entirely. Two DIFFERENT provenances --
B's cells rearranging, versus B being destroyed and an impostor built in its place --
are constructed to yield an IDENTICAL after-frame. Their correct verdicts are
opposite. Any function of one frame therefore gets at least one wrong, whatever
descriptors it uses. This is an impossibility argument, not a measurement.

PART 2 crosses "identity survived" against "descriptors survived" and scores only the
OFF-DIAGONAL rows, where the two disagree. `grown` keeps identity while breaking size;
`impostor` breaks identity while satisfying every descriptor. The diagonal rows are
reported but carry no information, because a size detector passes them.

PART 3 asks the milestone's one remaining question, and only that one:

    Does transition history recover information the static representation provably
    lacks?

by comparing R0(before, after) against R1(before, history, after) on the same final
frames. It adds exactly one source -- intermediate states -- and no new architectural
class. A positive answer requires the off-diagonal to rise with the controls intact,
and it is qualified by a stride sweep, since a gain that survives only at stride 1 is
dense observation rather than history.

Ground truth about identity appears in the expectations column ONLY, to score the
diagnostic. It is never available to `resolve`.
"""
from __future__ import annotations

import copy
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from referent import (Abstention, Referent, Resolution, check_orthogonality,
                      components, diagnose, diagnose_through, resolve,
                      resolve_through)

SHAPE = [(0, 0), (0, 1), (1, 0), (2, 0), (2, 1)]      # the 5-cell "C"
OTHER = [(0, 0), (0, 1), (0, 2), (1, 1), (2, 1)]      # a 5-cell "T": same size, new shape
COLOUR = 7
A_AT, B_AT = (2, 2), (12, 13)


def _blank():
    return [[0] * 20 for _ in range(20)]


def _place(g, at, shape, colour=COLOUR):
    for r, c in shape:
        g[at[0] + r][at[1] + c] = colour


def before():
    g = _blank()
    _place(g, A_AT, SHAPE)
    _place(g, B_AT, SHAPE)
    return g


# ------------------------------------------------------------------ the variants
def v_intact():
    return before()


# --- PART 1: two PROVENANCES, one identical after-frame ------------------------
# `v_morph` and `v_replace` return grids that are equal cell for cell. They differ
# only in what HAPPENED, which no single frame records.

def v_morph():
    """B's own cells rearrange into a new shape. B never stopped being one connected
    object, so the referent survives and the claim `identity(A.shape, B.shape)` is
    genuinely REFUTED. Correct verdict: MODEL_ERROR -- revise the hypothesis."""
    g = _blank()
    _place(g, A_AT, SHAPE)
    _place(g, B_AT, OTHER)
    return g


def v_replace():
    """B is destroyed and an unrelated object is created where it stood. The subject
    of the claim no longer exists, so the claim has nothing to be about.
    Correct verdict: REFERENT_LOST -- revise the representation."""
    g = _blank()
    _place(g, A_AT, SHAPE)
    _place(g, B_AT, OTHER)          # deliberately identical to v_morph
    return g


# --- PART 2: dissociate "identity survived" from "descriptors survived" --------
# The old case set varied both together, which is what made it prove nothing.

def v_dissolved():
    """identity NO, descriptors NO. One cell deleted; the rest falls into two
    disconnected 2-cell fragments. Both factors broken -- the easy diagonal."""
    g = before()
    g[13][13] = 0
    return g


def v_grown():
    """identity YES, descriptors NO. B accretes three adjacent cells, 5 -> 8, never
    stopping being one connected object in one place. A person says "that is still B,
    it got bigger". The size filter says it is dead. OFF-DIAGONAL."""
    g = before()
    for r, c in ((13, 14), (15, 13), (15, 14)):
        g[r][c] = COLOUR
    return g


def v_impostor():
    """identity NO, descriptors YES. B crumbles into fragments AND an unrelated
    5-cell object of the same colour appears in the same place. Every descriptor the
    resolver owns is satisfied by something that is not B. OFF-DIAGONAL."""
    g = _blank()
    _place(g, A_AT, SHAPE)
    g[12][13] = COLOUR; g[12][14] = COLOUR          # B's surviving fragment
    _place(g, (16, 15), OTHER)                      # the impostor, near B's centroid
    return g


def v_moved():
    """identity YES, descriptors YES-ish. Control: a diagnostic that cries
    REFERENT_LOST for an intact object that shifted two cells is useless."""
    g = _blank()
    _place(g, A_AT, SHAPE)
    _place(g, (B_AT[0] + 2, B_AT[1] + 2), SHAPE)
    return g


def v_erased():
    g = _blank()
    _place(g, A_AT, SHAPE)
    return g


# (label, builder, verdict CORRECT BY PROVENANCE, expected abstention, factors)
# The "correct" column is ground truth about identity. It is used ONLY to score the
# diagnostic, never inside `resolve` -- the experimenter knows the intervention, the
# mechanism does not.
CASES = [
    ("intact",    v_intact,    "OK",                   None,                     "id+ desc+"),
    ("morph",     v_morph,     "MODEL_ERROR",          None,                     "id+ desc+"),
    ("replace",   v_replace,   "REPRESENTATION_ERROR", Abstention.REFERENT_LOST, "id- desc+"),
    ("dissolved", v_dissolved, "REPRESENTATION_ERROR", Abstention.REFERENT_LOST, "id- desc-"),
    ("grown",     v_grown,     "MODEL_ERROR",          None,                     "id+ desc-"),
    ("impostor",  v_impostor,  "REPRESENTATION_ERROR", Abstention.REFERENT_LOST, "id- desc+"),
    ("moved",     v_moved,     "OK",                   None,                     "id+ desc+"),
    ("erased",    v_erased,    "REPRESENTATION_ERROR", Abstention.REFERENT_LOST, "id- desc-"),
]


# =============================================================================
# PART 3 -- histories. The trajectory IS the intervention, not a tuning knob.
# =============================================================================
# The risk here is the same one that killed the first case set: choosing histories
# that make R1 win. The defence is that each trajectory is forced by the WORDS of its
# intervention, and edits are applied one cell per step in sorted order -- a stated,
# case-independent rule, not a per-case choice.
#
#   morph     "B's own cells rearranged"        -> additions first, then deletions, so
#                                                  the object stays one component
#   replace   "B destroyed, impostor built"     -> every cell removed, THEN the new
#                                                  object built. B is absent in between
#   grown     "B accreted cells"                -> one cell per step
#   impostor  "B crumbled, a stranger appeared" -> B erodes, then the stranger is built
#   moved     "B shifted"                       -> translated one diagonal step at a time
#
# morph and replace still END on the identical frame of PART 1. Only the middle differs,
# which is precisely the information PART 1 proved the endpoints cannot carry.

def _cells(at, shape):
    return {(at[0] + r, at[1] + c) for r, c in shape}


def _grid_of(*cellsets):
    g = _blank()
    for cells, colour in cellsets:
        for r, c in cells:
            g[r][c] = colour
    return g


A_CELLS = _cells(A_AT, SHAPE)
B0 = _cells(B_AT, SHAPE)
BT = _cells(B_AT, OTHER)
IMP = _cells((16, 15), OTHER)


def _edit_path(start, end):
    """Additions first, then deletions, one cell per step, sorted. Keeps a morph
    connected; makes no promise for any other pair, and needs none."""
    out = [set(start)]
    cur = set(start)
    for c in sorted(end - start):
        cur = cur | {c}; out.append(set(cur))
    for c in sorted(start - end):
        cur = cur - {c}; out.append(set(cur))
    return out


def _erode_then_build(start, end):
    """Every cell of `start` removed before any cell of `end` is placed."""
    out = [set(start)]
    cur = set(start)
    for c in sorted(start):
        cur = cur - {c}; out.append(set(cur))
    for c in sorted(end):
        cur = cur | {c}; out.append(set(cur))
    return out


OCCLUDER = 4

# A history step is (paints, b_truth): what the frame SHOWS, and what the experimenter
# knows B to be. The two come apart only under occlusion, which is the point of that
# control -- B is still there while nothing in the frame says so. `b_truth` is used for
# the per-step change statistics and never by the mechanism.

def _steps(cellsets):
    return [([(s, COLOUR)], set(s)) for s in cellsets]


def h_intact():
    return _steps([B0])


def h_morph():
    return _steps(_edit_path(B0, BT))


def h_replace():
    return _steps(_erode_then_build(B0, BT))


def h_dissolved():
    return _steps([B0, B0 - {(13, 13)}])


def h_grown():
    return _steps(_edit_path(B0, B0 | {(13, 14), (15, 13), (15, 14)}))


def h_impostor():
    frag = {(12, 13), (12, 14)}
    out = [([(set(B0), COLOUR)], set(B0))]
    cur = set(B0)
    for c in sorted(B0 - frag):
        cur = cur - {c}
        out.append(([(set(cur), COLOUR)], set(cur)))
    built = set()
    for c in sorted(IMP):
        built = built | {c}
        out.append(([(set(cur), COLOUR), (set(built), COLOUR)], set(cur)))
    return out


def h_moved():
    out = []
    for k in (0, 1, 2):
        s = {(r + k, c + k) for r, c in B0}
        out.append(([(s, COLOUR)], set(s)))
    return out


def h_erased():
    return _steps(_erode_then_build(B0, set()))


def h_occluded():
    """NEGATIVE CONTROL FOR R1, and the one case R0 is expected to get RIGHT.

    B is covered by another object for one frame and then uncovered, unchanged. Its
    identity plainly persists -- a thing does not stop being itself because something
    was in front of it -- and the endpoints show it perfectly, so R0 answers OK.

    R1's irreversible-death rule must fail here: the referent goes DEAD on the covered
    frame, and R1 does not allow resurrection. That rule is what makes `impostor` work
    (a matching object reappearing is not the original), and the same rule makes this
    wrong. It is not a bug to be patched -- it is the scope boundary of R1, and it is
    in the suite so the boundary is measured rather than discovered later."""
    return [([(set(B0), COLOUR)], set(B0)),
            ([(set(B0), OCCLUDER)], set(B0)),
            ([(set(B0), COLOUR)], set(B0))]


HISTORIES = {
    "intact": h_intact, "morph": h_morph, "replace": h_replace,
    "dissolved": h_dissolved, "grown": h_grown, "impostor": h_impostor,
    "moved": h_moved, "erased": h_erased, "occluded": h_occluded,
}


def frames_of(steps):
    return [_grid_of((A_CELLS, COLOUR), *paints) for paints, _ in steps]


def truths_of(steps):
    return [t for _, t in steps]


def step_change(truths):
    """Per-step change magnitude, as the experimenter measures it. Three numbers,
    because the resolver gates on two of them and Hamming is the neutral one:

        cells   |B_i XOR B_{i-1}|         how much of the object was rewritten
        dsize   |size_i - size_{i-1}| / size_{i-1}   what `size_tol` gates on
        dcent   euclidean centroid jump             what `radius` gates on

    Reporting these instead of a stride number is the difference between a continuity
    ENVELOPE and a magic constant: stride is a proxy that only means something on
    these particular trajectories, whereas the envelope transfers."""
    worst = {"cells": 0, "dsize": 0.0, "dcent": 0.0}
    for a, b in zip(truths, truths[1:]):
        if not a or not b:
            worst["cells"] = max(worst["cells"], len(a ^ b))
            continue
        ca = (sum(p[0] for p in a) / len(a), sum(p[1] for p in a) / len(a))
        cb = (sum(p[0] for p in b) / len(b), sum(p[1] for p in b) / len(b))
        worst["cells"] = max(worst["cells"], len(a ^ b))
        worst["dsize"] = max(worst["dsize"], abs(len(b) - len(a)) / len(a))
        worst["dcent"] = max(worst["dcent"],
                             ((ca[0] - cb[0]) ** 2 + (ca[1] - cb[1]) ** 2) ** 0.5)
    return worst


def main() -> int:
    print("=" * 78)
    print("Milestone 3: prediction failure != representation failure?")
    print("=" * 78)

    print("\nPRECONDITION -- descriptors must not include the property under claim")
    print(f"   resolve by : {Referent.RESOLVE_BY}")
    print(f"   claim about: {Referent.CLAIMS_ABOUT!r}")
    ok = check_orthogonality()
    print(f"   orthogonal : {ok}  {'ok' if ok else 'FAIL -- the diagnostic is circular'}")
    if not ok:
        return 1

    # --- referents are frozen HERE, on the before-frame, once ---
    g0 = before()
    comps0 = components(g0)
    assert len(comps0) == 2, f"fixture broken: {len(comps0)} components before corruption"
    comps0.sort(key=lambda c: c["top_left"])
    ref_a = Referent.from_component("A", comps0[0])
    ref_b = Referent.from_component("B", comps0[1])
    print(f"\nreferents frozen on the BEFORE frame, and never renegotiated:")
    print(f"   A colour={ref_a.colour} size={ref_a.size} centroid={ref_a.centroid}")
    print(f"   B colour={ref_b.colour} size={ref_b.size} centroid={ref_b.centroid}")

    # ---------------------------------------------------------------- PART 1
    print("\n" + "=" * 78)
    print("PART 1 -- the same after-frame, two provenances, opposite correct answers")
    print("=" * 78)
    gm, gr = v_morph(), v_replace()
    same = gm == gr
    print(f"   v_morph() and v_replace() produce an identical grid: {same}")
    if not same:
        print("   FIXTURE BROKEN -- part 1 only means anything if the frames are equal")
        return 1
    vm, vr = diagnose(ref_a, ref_b, gm), diagnose(ref_a, ref_b, gr)
    print(f"   correct for morph  : MODEL_ERROR          (B's own cells rearranged)")
    print(f"   correct for replace: REPRESENTATION_ERROR (B destroyed, impostor built)")
    print(f"   diagnostic says    : {vm.label} / {vr.label}")
    print(f"""
   This does not depend on which descriptors the resolver happens to use. The two
   frames are equal, so ANY function of the after-frame alone returns one answer for
   both, while the correct answers differ. At least one is necessarily wrong.

   State the result exactly, because a looser version is tempting and is not proved:

     PROVED     identity is not derivable from the APPEARANCE of a frame pair, when
                different causal histories produce identical observations.
     NOT PROVED "identity requires time". That is a claim about what identity needs
                in general; this construction shows only that THIS information is
                absent from THIS pair. Whether some other source recovers it is an
                empirical question -- PART 3 asks it, rather than assuming the
                answer is history.

   The practical form: over a single before/after pair, "B changed" and "B was
   replaced" are the same observation and demand opposite responses.""")

    # ---------------------------------------------------------------- PART 2
    print("\n" + "=" * 78)
    print("PART 2 -- does the resolver track identity, or just size?")
    print("=" * 78)
    print("   'id' = identity survived by provenance; 'desc' = resolver's descriptors")
    print("   survived. The old case set varied the two TOGETHER, so it could not tell")
    print("   an identity detector from a size detector. The off-diagonal rows do.\n")
    print(f"{'case':<11} {'factors':<10} {'A':<9} {'B':<9} {'verdict':<22} {'err':<4} correct")
    print("-" * 78)
    agree = 0
    diag_hit = diag_n = off_hit = off_n = 0
    for name, gen, want_label, want_abst, factors in CASES:
        g = gen()
        ra, rb = resolve(ref_a, g), resolve(ref_b, g)
        v = diagnose(ref_a, ref_b, g)
        hit = (v.label == want_label
               and (want_abst is None or v.abstention is want_abst))
        agree += hit
        off = factors in ("id+ desc-", "id- desc+")
        if off:
            off_n += 1; off_hit += hit
        else:
            diag_n += 1; diag_hit += hit
        err = "--" if v.error is None else f"{v.error:.1f}"
        print(f"{name:<11} {factors:<10} {ra.status.value:<9} {rb.status.value:<9} "
              f"{v.label:<22} {err:<4} {want_label:<22} {'ok' if hit else 'WRONG'}")
        print(f"{'':<11} {v.detail}")
    print("-" * 78)
    print(f"   overall      {agree}/{len(CASES)}")
    print(f"   diagonal     {diag_hit}/{diag_n}   (descriptors and identity agree -- easy)")
    print(f"   OFF-DIAGONAL {off_hit}/{off_n}   <- the only rows that carry information")
    print(f"""
   Read ONLY the off-diagonal. A high overall score means nothing here: the diagonal
   rows are satisfiable by a detector that reports "size changed" and knows no more,
   which is exactly the confound this rewrite exists to remove.

     grown    identity survived, size did not -> a FALSE referent loss. The object is
              plainly still there and still B; the size filter buries it.
     impostor identity did not survive, descriptors did -> a FALSE survival, and the
              worst outcome available: the diagnostic reports on the shape of an
              object that is not B, and may call the claim upheld.

   If both off-diagonal rows are wrong, the finding is clean and negative: `resolve`
   is a same-colour-same-size-same-place detector, and "identity" in this
   representation means nothing more than that. In that case the descriptor resolver
   is CLOSED as a general identity mechanism -- do not tune `radius` or `size_tol`,
   which only moves which rows fail.""")

    # ---------------------------------------------------------------- PART 3
    print("\n" + "=" * 78)
    print("PART 3 -- does transition history recover what the static pair provably lacks?")
    print("=" * 78)
    print("""   TWO SEPARATE TESTS. R0 being closed as a general identity mechanism does NOT
   close R1: R1 is given information R0 provably does not have, so it is answering a
   different question and is scored on its own.

       R0 asks: is endpoint APPEARANCE sufficient for identity?
       R1 asks: does intermediate transition evidence RECOVER identity in the cases
                where endpoints are insufficient?

   A change-mask would NOT count as history: the diff of the endpoints is a function
   of the endpoints, so PART 1 defeats it as surely as it defeats R0. Only genuine
   intermediate states add information.

   Both arms are scored on the SAME final frame, so any difference is attributable to
   what was visible on the way there and to nothing else.

   CLAIM LIMITS, fixed before the run so the result cannot be talked upward:
     - R1 lifts the off-diagonal with controls intact
                 -> intermediate transition evidence recovers identity here.
     - the gain survives only at the densest sampling
                 -> a DENSE CONTINUITY result. Not "history solved representation",
                    and not support for the stateful-mode link, where observation is
                    sparse. Say dense continuity and stop.\n""")
    print(f"{'case':<11} {'factors':<10} {'steps':<6} {'R0':<22} {'R1':<22} correct")
    print("-" * 78)
    r0_off = r1_off = n_off = 0
    r0_ctl = r1_ctl = n_ctl = 0
    for name, gen, want_label, want_abst, factors in CASES:
        frames = frames_of(HISTORIES[name]())
        if frames[-1] != gen():
            print(f"{name:<11} FIXTURE MISMATCH: history does not end on the PART 2 frame")
            return 1
        v0 = diagnose(ref_a, ref_b, frames[-1])
        v1 = diagnose_through(ref_a, ref_b, frames)
        h0 = (v0.label == want_label and (want_abst is None or v0.abstention is want_abst))
        h1 = (v1.label == want_label and (want_abst is None or v1.abstention is want_abst))
        off = factors in ("id+ desc-", "id- desc+")
        if off:
            n_off += 1; r0_off += h0; r1_off += h1
        else:
            n_ctl += 1; r0_ctl += h0; r1_ctl += h1
        mark = lambda h: "ok" if h else "WRONG"
        print(f"{name:<11} {factors:<10} {len(frames)-1:<6} "
              f"{v0.label + ' ' + mark(h0):<22} {v1.label + ' ' + mark(h1):<22} {want_label}")
    print("-" * 78)
    print(f"   OFF-DIAGONAL  R0 {r0_off}/{n_off}   ->   R1 {r1_off}/{n_off}")
    print(f"   controls      R0 {r0_ctl}/{n_ctl}   ->   R1 {r1_ctl}/{n_ctl}")

    # --- the continuity envelope: what R1 survives, in change units not strides ---
    print("\n   How much continuity does R1 actually need? Subsampling the history is")
    print("   only the WAY to vary that; the quantity is per-step change, so it is the")
    print("   per-step change that is reported. `size_tol` and `radius` stay fixed.")
    print("   The three change columns are measured over the `id+` cases ONLY -- the")
    print("   ones where the referent is SUPPOSED to survive. That is where a per-step")
    print("   jump is a threat rather than the correct cause of death.")
    print(f"\n   {'stride':>6} {'off-diag':>9} {'controls':>9} "
          f"{'worst cells':>12} {'worst dsize':>12} {'worst dcent':>12}")
    for stride in (1, 2, 3, 4, 99):
        o = c = 0
        w = {"cells": 0, "dsize": 0.0, "dcent": 0.0}
        for name, gen, want_label, want_abst, factors in CASES:
            steps = HISTORIES[name]()
            sub = steps[::stride]
            if sub[-1] is not steps[-1]:
                sub = sub + [steps[-1]]
            v = diagnose_through(ref_a, ref_b, frames_of(sub))
            hit = (v.label == want_label and (want_abst is None or v.abstention is want_abst))
            if factors in ("id+ desc-", "id- desc+"):
                o += hit
            else:
                c += hit
            # The envelope is only meaningful on cases where identity SURVIVES. On an
            # `id-` case a large per-step jump is the correct cause of death, so
            # folding those in would mix "R1 broke" with "R1 worked" in one column.
            if factors.startswith("id+"):
                ch = step_change(truths_of(sub))
                for k in w:
                    w[k] = max(w[k], ch[k])
        print(f"   {stride:>6} {o:>7}/{n_off} {c:>7}/{n_ctl} "
              f"{w['cells']:>12} {w['dsize']:>12.2f} {w['dcent']:>12.2f}")
    print(f"\n   The gate R1 applies per step is size_tol={0.25} and radius={6.0}, so the")
    print("   envelope should be read off the `dsize` column: R1 is expected to hold")
    print("   while per-step dsize stays under the tolerance and to fail above it,")
    print("   whatever stride produced that change. If the failures line up with dsize")
    print("   rather than with stride, the envelope transfers to other trajectories; if")
    print("   they line up with stride instead, this is a fact about these fixtures only.")

    # --- the case R1 is expected to LOSE, and R0 to win ---
    print("\n" + "-" * 78)
    print("   NEGATIVE CONTROL: temporary occlusion  (R0 right, R1 expected WRONG)")
    print("-" * 78)
    steps = HISTORIES["occluded"]()
    frames = frames_of(steps)
    v0 = diagnose(ref_a, ref_b, frames[-1])
    v1 = diagnose_through(ref_a, ref_b, frames)
    print(f"   B is covered for one frame, then uncovered unchanged. Correct: OK.")
    print(f"   R0 (endpoints only) : {v0.label:<22} {'ok' if v0.label == 'OK' else 'WRONG'}")
    print(f"   R1 (through history): {v1.label:<22} {'ok' if v1.label == 'OK' else 'WRONG'}")
    print(f"                         {v1.detail}")
    print("""
   R1 is expected to fail this, and the failure is not a defect to patch. The rule
   that a dead referent stays dead is exactly what makes `impostor` work -- a matching
   object appearing later is not the original returning. The same rule cannot also
   allow B back after an occlusion. R1 therefore cannot tell "briefly hidden" from
   "destroyed, and something similar built later", and that is its scope boundary.

   Note the direction: here MORE information makes the answer WORSE. R1 is not a
   strict improvement on R0, it is a different trade, and any summary that says
   "history helps" without this row is overselling.""")

    print("""
   READING THIS, and then M3 closes on whatever it says.

   Positive only if R1 lifts the off-diagonal WITHOUT costing controls. Buying the
   hard rows by breaking the easy ones is not progress.

   THE SANCTIONED WORDING, fixed before the run. If the off-diagonal lifts with the
   controls intact and occlusion fails as expected, the claim is exactly this and is
   not to be paraphrased upward:

       Sequential local correspondence can recover identity distinctions unavailable
       from endpoint appearance under continuous observation and bounded per-step
       change; it is not persistence through occlusion.

   Every clause is load-bearing. "Sequential local correspondence" -- not a tracker
   and not a theory of identity. "Unavailable from endpoint appearance" -- the thing
   PART 1 proved absent, no more. "Bounded per-step change" -- the envelope, which is
   a measurement, not a capability. "Not persistence through occlusion" -- the row
   above, kept in the sentence so it cannot be dropped in summary.

   If the off-diagonal does not lift, say that the added source does not recover it,
   and stop there.

   THE ENVELOPE IS NOT A THRESHOLD. Whatever number the `dsize` column yields, it is
   the observed boundary OF THIS RESOLVER ON THIS FIXTURE -- a property of one
   `size_tol` against one set of hand-built trajectories. It is not a general limit on
   how fast a thing may change and stay itself, and it must never be quoted as one or
   carried into other code as a constant.

   In no outcome does the `stateful-mode` link become a finding: that needs the same
   measurement on sparse, real transitions, which this fixture does not contain. The
   occlusion row stands as the reminder that more information can also make the answer
   worse.

   M3 CLOSES ON WHATEVER THIS PRINTS. No new identity mechanism, no new tracker, no
   tuning of `size_tol` or `radius` to improve a row. The next code written after the
   commit belongs to MetaController.""")
    print("\nMetaController is still NOT built.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
