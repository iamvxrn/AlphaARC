"""Milestone 2: abstention vs falsifiability.

Milestone 1.5 ended on a wall: two engines of three answered a corrupted board with
SILENCE rather than error, so an evaluator accumulating prediction error would mostly
receive nothing. The open question was whether engines should emit approximate or
ranked hypotheses so that they can be wrong.

The answer measured here is that the question had the wrong shape. Abstention is not
one phenomenon and "be more approximate" is not one fix.

  1. The INSTRUMENT was weak. A held-out prediction over randomly masked cells
     detects a local violation only at the mask coverage rate, and cannot be fixed by
     masking harder (section A).
  2. MDL's abstention WAS a gate, and the fix is boldness, not approximation: assert
     the universal, be refutable by one cell (section B).
  3. Relational abstention was NOT a gate. Loosening it is measured to be worse than
     silence in two independent ways (section C).

Sections D and E are the honesty checks: the constant is swept rather than asserted,
and a liar engine reports the noise floor next to every real number.
"""
from __future__ import annotations
import random
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))

from protocol import HIDDEN, Kind, Truth, verify
from engines import LiarEngine, MdlEngine, fresh_engines
from cases import _mask, _mirror_grid, _noise_grid, _shuffled_mirror, build


def _by_label(cases):
    return {c[0]: c for c in cases}


def _sites():
    base = _mirror_grid()
    return [(r, c) for r in range(16) for c in range(8, 16) if base[r][c] != 0]


def _run(engine, grid, nmask, seed):
    """Show `engine` a masked board; return (spoke, held_out_error, universal_error)."""
    m, hid = _mask(grid, nmask, seed=seed)
    from protocol import Evidence
    ev = Evidence("probe", "grid", {"grid": m}, hid)
    truth = Truth(grid=grid)
    ho = uni = None
    hs = engine.hypotheses(ev)
    for h in hs:
        for p in h.claim.propose(ev):
            e = verify(p, truth)
            if p.kind is Kind.HELD_OUT_CELLS and e is not None:
                ho = e
            elif p.kind is Kind.UNIVERSAL and e is not None:
                uni = e
    return bool(hs), ho, uni


# ---------------------------------------------------------------- A: the instrument
def section_a():
    print("=" * 78)
    print("A. Why a held-out cell prediction is a WEAK instrument")
    print("=" * 78)
    print("   One violated cell, all 40 corruption sites x 15 masks.")
    print("   'caught' = error > 0, i.e. the damage actually reached the test.\n")
    print(f"   {'hidden':>8} {'coverage':>9} {'caught held_out':>18} {'expect ~1-(1-p)^2':>20}"
          f" {'caught universal':>19}")
    eng = MdlEngine()
    for nmask in (8, 24, 64, 128, 200):
        cho = cuni = n = 0
        for site in _sites():
            for seed in range(1, 16):
                g = _mirror_grid()
                g[site[0]][site[1]] = (g[site[0]][site[1]] + 1) % 5
                spoke, ho, uni = _run(eng, g, nmask, seed)
                if not spoke:
                    continue
                n += 1
                cho += (ho is not None and ho > 0)
                cuni += (uni is not None and uni > 0)
        cov = nmask / 256
        print(f"   {nmask:>8} {cov:>8.1%} {cho/n:>17.1%} {1-(1-cov)**2:>19.1%}"
              f" {cuni/n:>18.1%}")
    print("\n   Coverage explains held_out almost exactly -- and at 200 cells the rate FALLS:")
    print("   masking hard enough to hit the damage also hides the source cell the")
    print("   prediction would have been derived from. There is NO mask size at which")
    print("   this instrument becomes reliable.")
    print("\n   About `universal`, EXACTLY one thing is claimed: refutation of an emitted")
    print("   universal is not limited to held-out locations. That is NOT 'universals are")
    print("   mask-independent' -- the axis and period are still fitted on the MASKED")
    print("   board, so the mask governs WHAT is asserted, not what can refute it.")


# --------------------------------------------------------------------- B: the fix
def section_b():
    print("\n" + "=" * 78)
    print("B. The bold claim: 'this rule holds EVERYWHERE, with no exception'")
    print("=" * 78)
    eng = MdlEngine()
    for name, corrupt in (("intact", False), ("BROKEN", True)):
        spoke = n = fal = 0
        errs = []
        for site in (_sites() if corrupt else [None]):
            for seed in range(1, 16):
                g = _mirror_grid()
                if site:
                    g[site[0]][site[1]] = (g[site[0]][site[1]] + 1) % 5
                s, ho, uni = _run(eng, g, 24, seed)
                n += 1
                spoke += s
                if uni is not None:
                    errs.append(uni); fal += (uni > 0)
        print(f"   {name:<10} engine spoke {spoke}/{n}, "
              f"refuted {fal}/{len(errs)} = {fal/max(len(errs),1):>6.1%}, "
              f"mean error {sum(errs)/max(len(errs),1):.4f}")
    print("\n   On the broken board the gated engine was silent 600/600 times. It now")
    print("   speaks 600/600 and is refuted 100%. Silence became signal.")


# ------------------------------------------------------------ C: what did NOT work
def section_c():
    print("\n" + "=" * 78)
    print("C. Relational silence is NOT a gate (negative result)")
    print("=" * 78)
    print("""   Deleting one cell of a copy does not break the shape match, it DISSOLVES the
   object: five cells fall into two disconnected 2-cell fragments, and the claim
   loses its subject. Loosening the threshold was measured and rejected twice:

     - the claim DODGES refutation. On the broken board the engine re-aims at the
       two identical FRAGMENTS (similarity 1.000) and the verifier scores it 0.0,
       "correct". An engine that picks its subject after seeing the evidence cannot
       be refuted by that evidence.
     - noise becomes error. On scattered same-colour blobs that are copies of
       nothing: it spoke on 37 boards of 40 and was refuted on 23.

   This is a representational limit, not a threshold: the family needs an object
   identity that survives its own cells changing. The same wall as Correspondence in
   the Go stack, from the other side. DO NOT fix by loosening.""")


# --------------------------------------------- D: the constant, swept not asserted
def section_d():
    print("\n" + "=" * 78)
    print("D. The COMMIT_AGREE threshold -- swept, not asserted")
    print("=" * 78)
    print("   The denominator in the 'broken' column is the number of boards the engine")
    print("   spoke on AT ALL (out of 200). It falls as the threshold rises: that fall")
    print("   is the abstention being measured.\n")
    print(f"   {'thresh':>7} {'intact: speaks':>15} {'error':>8} "
          f"{'broken: refuted':>25} {'noise: speaks':>14}")
    for thr in (0.50, 0.70, 0.80, 0.90, 0.95, 0.99, 1.00):
        eng = MdlEngine(commit_agree=thr)
        tp = te = 0
        for seed in range(1, 16):
            s, ho, uni = _run(eng, _mirror_grid(), 24, seed)
            tp += s
            te += (uni or 0)
        bf = bn = 0
        for site in _sites():
            for seed in range(1, 6):
                g = _mirror_grid()
                g[site[0]][site[1]] = (g[site[0]][site[1]] + 1) % 5
                s, ho, uni = _run(eng, g, 24, seed)
                if uni is not None:
                    bn += 1; bf += (uni > 0)
        ns = 0
        for seed in range(1, 31):
            for gen in (_noise_grid, _shuffled_mirror):
                s, _, _ = _run(eng, gen(seed), 24, seed)
                ns += s
        print(f"   {thr:>7.2f} {tp:>10}/15 {te/15:>11.4f} "
              f"{bf:>16}/{bn:<8} {ns:>10}/60")
    print("\n   The plateau is 0.50..0.95: 15/15 honest, 200/200 refutations, 0/60 on noise.")
    print("   0.90 sits in the middle of it, so the number is not fitted -- but note")
    print("   WHAT actually holds noise back: not this threshold but min_agree=20 (see E).")
    print("   At 0.50 noise is still 0/60, so COMMIT_AGREE is not the defence against")
    print("   noise; it only decides whether to be bold. Keep the two roles apart.")
    print("\n   0.99 and 1.00 are the old gate: the denominator falls 200 -> 30, i.e. the")
    print("   engine goes silent on 170 boards of 200. The surviving 30 are refuted in")
    print("   full, which is its own finding: there the mask HID the counterexample from")
    print("   the engine, it saw a perfect symmetry, and it was refuted by a cell it was")
    print("   never shown. Even a strict gate does not prevent error, only make it rare.")


# ------------------------------------------------------------- E: the noise floor
def section_e():
    print("\n" + "=" * 78)
    print("E. The noise floor: a liar that always speaks")
    print("=" * 78)
    liar, eng = LiarEngine(), MdlEngine()
    print(f"   {'board':<24} {'mdl speaks':>12} {'mdl error':>11} "
          f"{'liar speaks':>13} {'liar error':>13}")
    boards = [("mirror (intact)", lambda s: _mirror_grid()),
              ("noise, no structure", _noise_grid),
              ("shuffled mirror", _shuffled_mirror)]
    for name, gen in boards:
        ms = me = ls = le = n = 0
        mes, les = [], []
        for seed in range(1, 31):
            g = gen(seed)
            s, _, uni = _run(eng, g, 24, seed)
            ms += s
            if uni is not None:
                mes.append(uni)
            s2, _, uni2 = _run(liar, g, 24, seed)
            ls += s2
            if uni2 is not None:
                les.append(uni2)
            n += 1
        f = lambda v: f"{sum(v)/len(v):.4f}" if v else "--"
        print(f"   {name:<24} {ms:>9}/{n:<3} {f(mes):>11} {ls:>10}/{n:<3} {f(les):>13}")
    print("\n   The real engine is silent where there is no structure; the liar commits")
    print("   everywhere and pays ~0.72 for it.")
    print("\n   But read the FIRST row, which is the one that teaches something. On a board")
    print("   that really IS mirror-symmetric the liar scores exactly what mdl scores. It")
    print("   is not lucky -- the structure is there, so the best-looking axis is the true")
    print("   one. So the claim is NARROW: prediction error CONDITIONAL ON COMMITMENT")
    print("   cannot separate a selective generator from an indiscriminate one. It is not")
    print("   that error is uninformative -- over a sequence of boards the liar's 0.72 on")
    print("   noise exposes it. What carries the extra information is WHERE each engine is")
    print("   willing to speak at all.")
    print("\n   Log the two separately and do not fuse them into one number:")
    print("       coverage = P(commit)          risk = E[error | commit]")
    print("\n   Note what this does NOT show: whether an engine reports the REASON for its")
    print("   silence correctly. That is untested here and belongs to milestone 3.")


def main() -> int:
    section_a(); section_b(); section_c(); section_d(); section_e()
    print("\n" + "=" * 78)
    print("SUMMARY")
    print("=" * 78)
    print("""   The question "should engines emit approximate or ranked hypotheses so that
   they can be refuted" is answered thus: approximation is NOT what was missing.

   What was missing was BOLDNESS and a reliable instrument.
     - boldness: assert the universal, refutable by a single cell, instead of
       hedging. A hedged "symmetric except at these two cells" is unfalsifiable and,
       in MDL's own currency, longer. An approximate hypothesis with a caveat is a
       way of never being wrong -- the same abstention by another route.
     - instrument: a prediction over a random mask catches a violation at the
       coverage rate, and is not fixed by enlarging the mask.

   Honestly still open: mdl is refutable, graph always was, relational is not, and
   that is a representational limit rather than a setting. Two families of three
   yield error. Error is now a control signal, but it is still not available along
   ONE axis.

   MetaController is still NOT built.""")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
