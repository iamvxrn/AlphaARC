"""Milestone 1: can three existing mechanisms make testable claims about three
different kinds of experience, through one protocol, with no task-specific code?

Prints, per case: who spoke, what they claimed, and the error scored AT THE LEVEL
each claim was made. Silence is reported as silence, not as error.
"""
from __future__ import annotations
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent))

from protocol import Level, score
from engines import ENGINES, GraphEngine
from cases import CASES


def observation_for(pred, ev):
    """What in this evidence can test a prediction at that level."""
    p = ev.payload
    if pred.level is Level.CELL:
        return p.get("after") or p.get("grid")
    if pred.level is Level.RELATION:
        # Verify the relation against the board rather than assuming it. The two
        # boxes named in the target are re-read and their shapes compared; that is
        # what makes a relational claim falsifiable instead of decorative.
        g = p.get("after") or p.get("grid")
        if g is None:
            return None
        try:
            a_s, b_s = pred.target.split("~")
            a = eval(a_s); b = eval(b_s)
        except Exception:
            return None
        def mask_at(top_left):
            r0, c0 = top_left
            col = g[r0][c0]
            seen, st, cells = {(r0, c0)}, [(r0, c0)], []
            while st:
                r, c = st.pop(); cells.append((r, c))
                for dr in (-1, 0, 1):
                    for dc in (-1, 0, 1):
                        q = (r + dr, c + dc)
                        if (0 <= q[0] < len(g) and 0 <= q[1] < len(g[0])
                                and q not in seen and g[q[0]][q[1]] == col):
                            seen.add(q); st.append(q)
            ys = [x[0] for x in cells]; xs = [x[1] for x in cells]
            y0, x0 = min(ys), min(xs)
            h, w = max(ys) - y0 + 1, max(xs) - x0 + 1
            m = [[0] * w for _ in range(h)]
            for r, c in cells:
                m[r - y0][c - x0] = 1
            return tuple(tuple(r) for r in m)
        return "identity" if mask_at(a) == mask_at(b) else "different"
    if pred.level is Level.TRANSITION:
        after = p.get("after")
        return GraphEngine._h(after) if after else None
    return None


def main() -> int:
    graph = next(e for e in ENGINES if e.name == "graph")
    rows = []
    for ev in CASES:
        print(f"\n=== {ev.id}  ({ev.kind}) ===")
        print(f"    truth: {ev.payload.get('truth')}")
        spoke = 0
        for eng in ENGINES:
            hs = eng.hypotheses(ev)
            if not hs:
                print(f"  {eng.name:<11} silent")
                rows.append((ev.id, eng.name, None, None))
                continue
            spoke += 1
            for h in hs[:2]:
                preds = h.claim.predict(ev)
                for pr in preds:
                    err = score(pr, observation_for(pr, ev))
                    e = "untestable" if err is None else f"error {err:.3f}"
                    print(f"  {eng.name:<11} {h.describe()[:58]:<58} "
                          f"[{pr.level.value}] {e}")
                    rows.append((ev.id, eng.name, pr.level.value, err))
        # after seeing the transition, the graph engine learns the edge
        if ev.kind == "transition":
            graph.observe(ev.payload["before"], ev.payload["action"], ev.payload["after"])
            hs = graph.hypotheses(ev)
            if hs:
                pr = hs[0].claim.predict(ev)[0]
                err = score(pr, observation_for(pr, ev))
                print(f"  {'graph':<11} {hs[0].describe()[:58]:<58} "
                      f"[{pr.level.value}] error {err:.3f}   <- after observing")
        print(f"    spoke: {spoke} of {len(ENGINES)}")
    print("\n" + "="*70)
    tested = [r for r in rows if r[3] is not None]
    print(f"propositions total: {len(rows)}, testable: {len(tested)}")
    lv = sorted({r[2] for r in rows if r[2]})
    print(f"levels spoken at: {lv}")
    for name in [e.name for e in ENGINES]:
        mine = [r for r in rows if r[1] == name]
        t = [r for r in mine if r[3] is not None]
        print(f"  {name:<11} propositions {len(mine):>2}, testable {len(t):>2}, "
              f"mean error {sum(r[3] for r in t)/len(t):.3f}" if t else
              f"  {name:<11} propositions {len(mine):>2}, testable  0")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
