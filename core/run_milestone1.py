"""Milestone 1.5: are the predictions semantically valid?

Every proposition is about cells or outcomes the engine was NOT shown, and the
verifier sees only the proposition's kind -- never the engine, never the task.
Each case is run in a positive and a corrupted form; a suite without the corrupted
twin only proves the engines were asked flattering questions.
"""
from __future__ import annotations
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))

from protocol import Kind, verify
from engines import fresh_engines
from cases import build


def _by_label(cases):
    """Look cases up by NAME, not by index. Positional indexing broke silently the
    first time a case was inserted in the middle of the list."""
    return {c[0]: c for c in cases}


def main() -> int:
    engines = [e for e in fresh_engines() if e.name != "liar"]
    graph = next(e for e in engines if e.name == "graph")
    rows = []
    for label, ev, truth, expect in build():
        print(f"\n=== {label}   expected: {expect} ===")
        if label == "vc33-transition-BROKEN":
            graph.observe(ev.payload["before"], ev.payload["action"],
                          _by_label(build())["vc33-transition"][1].payload["after"])
        spoke = 0
        for eng in engines:
            hs = eng.hypotheses(ev)
            if not hs:
                print(f"  {eng.name:<11} silent")
                continue
            spoke += 1
            for h in hs:
                for p in h.claim.propose(ev):
                    err = verify(p, truth)
                    n = len(p.value) if p.kind is Kind.HELD_OUT_CELLS else 1
                    e = "untestable" if err is None else f"{err:.3f}"
                    print(f"  {eng.name:<11} {h.describe()[:52]:<52} "
                          f"[{p.kind.value:<17}] n={n:<3} error {e}")
                    rows.append((label, eng.name, p.kind.value, err))
        if ev.kind == "transition" and label == "vc33-transition":
            graph.observe(ev.payload["before"], ev.payload["action"], ev.payload["after"])
            for h in graph.hypotheses(ev):
                p = h.claim.propose(ev)[0]
                print(f"  {'graph':<11} {h.describe()[:52]:<52} "
                      f"[{p.kind.value:<17}] n=1   error {verify(p, truth):.3f}"
                      f"   <- after observing")
                rows.append((label, "graph", p.kind.value, verify(p, truth)))
        print(f"    spoke: {spoke}/{len(engines)}")

    # --- the verifier under direct test, independent of any engine ---
    print("\n=== the verifier head-on: deliberately FALSE propositions ===")
    from protocol import Proposition, Truth, digest
    cases = _by_label(build())
    g_true = cases["static-mirror"][2].grid
    liar_cells = {(r, c): (g_true[r][c] + 1) % 9 for (r, c) in list(cases["static-mirror"][1].hidden)[:6]}
    honest_cells = {(r, c): g_true[r][c] for (r, c) in list(cases["static-mirror"][1].hidden)[:6]}
    after = cases["vc33-transition"][2].after
    probes = [
        ("held_out, every value right",   Proposition(Kind.HELD_OUT_CELLS, "t", honest_cells), Truth(grid=g_true), 0.0),
        ("held_out, every value wrong",   Proposition(Kind.HELD_OUT_CELLS, "t", liar_cells),   Truth(grid=g_true), 1.0),
        ("relation_now, no such objects", Proposition(Kind.RELATION_NOW, "t", ((0, 0), (5, 5), "identity")), Truth(grid=cases["relational-copy"][2].grid), None),
        ("transition, correct successor", Proposition(Kind.TRANSITION, "t", digest(after)),    Truth(after=after), 0.0),
        ("transition, wrong successor",   Proposition(Kind.TRANSITION, "t", "deadbeefdeadbeef"), Truth(after=after), 1.0),
        ("nothing to check against",      Proposition(Kind.TRANSITION, "t", digest(after)),    Truth(), None),
    ]
    okc = 0
    for name, p, t, want in probes:
        got = verify(p, t)
        ok = (got is None and want is None) or (got is not None and want is not None and abs(got - want) < 1e-9)
        okc += ok
        print(f"  {name:<32} -> {('None' if got is None else f'{got:.3f}'):<6} "
              f"expected {('None' if want is None else f'{want:.3f}'):<6} {'ok' if ok else 'FAIL'}")
    print(f"  verifier: {okc}/{len(probes)} probes correct")

    print("\n" + "=" * 74)
    tested = [r for r in rows if r[3] is not None]
    print(f"propositions {len(rows)}, testable {len(tested)}, "
          f"silent/untestable {len(rows)-len(tested)}")
    print("\ndoes ONE generic verifier separate the outcomes:")
    for fam in sorted({r[2] for r in rows}):
        pos = [r[3] for r in rows if r[2] == fam and "BROKEN" not in r[0] and r[3] is not None]
        neg = [r[3] for r in rows if r[2] == fam and "BROKEN" in r[0] and r[3] is not None]
        f = lambda v: f"{sum(v)/len(v):.3f}" if v else "—"
        print(f"  {fam:<18} positive {f(pos):<7} corrupted {f(neg):<7} "
              f"{'SEPARATES' if pos and neg and sum(neg)/len(neg) > sum(pos)/len(pos) else ''}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
