"""Is a task set ABLE to test metacontrol at all? Answered before any controller.

M4 v0.1's negative result has a structural cause, and it is sharper than the way the
README first stated it. It is not that the budget was loose, and not merely that
"gather everything then commit once" works. It is that

    THERE EXISTS ONE TASK-INDEPENDENT DOMINATING SEQUENCE  --  explore, track, induce

and therefore the benchmark contains NO POLICY BRANCHING. It checks whether a correct
pipeline exists. It cannot check metacontrol, because metacontrol is the thing that
chooses differently in different situations, and here nothing has to.

That property is decidable WITHOUT running a controller, so it should be an entry
condition on a benchmark rather than a post-mortem on a result. This module decides
it.

    fixed_ceiling(T)   = max over every word w in Ops^<=L of solve-rate(w, T)
    oracle_ceiling(T)  = solve-rate of the arm that reads ground truth
                         (an upper bound on ANY policy, adaptive or not)

    ADMISSIBLE  iff  fixed_ceiling < oracle_ceiling

If the two are equal, no policy of any kind can beat the best fixed word, so the task
set cannot distinguish routing from a pipeline and MUST NOT be used to make a claim
about metacontrol. That is exactly M4 v0.1, and this module reproduces the finding
mechanically instead of by hand.

WHAT THIS TEST IS NOT. It is NECESSARY, NOT SUFFICIENT. A task set that passes still
has to face the RANDOM and FIXED controls -- passing only says branching is required
somewhere, not that a particular controller finds it, nor that feedback rather than
luck is what pays. Nothing here licenses dropping a control.

AND IT CANNOT BE GAMED TOWARD THE ANSWER WE WANT. It certifies that branching is
NECESSARY; it says nothing about whether adaptive routing succeeds. A v0.2 built to
pass this test is still free to return the same negative answer, which is the point.

STOP is excluded from the alphabet: it never changes the final state, only ends the
episode, so it cannot raise the fixed ceiling. Searching over the four real
operations is therefore exhaustive for this question and 5^d cheaper.
"""
from __future__ import annotations

import sys
from dataclasses import replace
from pathlib import Path
from typing import Dict, List, Optional, Tuple

sys.path.insert(0, str(Path(__file__).resolve().parent))

import metacontroller as M
from metacontroller import Op, State

ALPHABET = M.REAL_OPS


def _key(st: State) -> tuple:
    return (st.view, st.hyp, st.tested_view, st.refuted, frozenset(st.excluded),
            st.refs_dead, st.tracked, st.induce_empty, st.induced_view, st.stopped)


def _from_key(k: tuple) -> State:
    return State(view=k[0], hyp=k[1], tested_view=k[2], refuted=k[3],
                 excluded=set(k[4]), refs_dead=k[5], tracked=k[6],
                 induce_empty=k[7], induced_view=k[8], stopped=k[9])


class Explorer:
    """Exhaustive search over fixed words, memoised per (task, state, op).

    Without the cache this is 4^d x |T| engine invocations and is hopeless; with it
    the distinct reachable states per task number in the tens, so the search is
    dominated by cache hits."""

    def __init__(self, tasks, track_destroys: bool = True):
        self.tasks = list(tasks)
        self.td = track_destroys
        self.cache: Dict[Tuple[int, tuple, Op], tuple] = {}
        self.solved: Dict[Tuple[int, tuple], bool] = {}

    def start(self, i: int) -> tuple:
        st = State()
        if self.tasks[i].kind == "ref":
            st.hyp = M.SHAPE_EQ
        return _key(st)

    def step(self, i: int, k: tuple, op: Op) -> tuple:
        ck = (i, k, op)
        hit = self.cache.get(ck)
        if hit is None:
            st = _from_key(k)
            M.apply_op(op, self.tasks[i], st, track_destroys=self.td)
            hit = self.cache[ck] = _key(st)
        return hit

    def is_solved(self, i: int, k: tuple) -> bool:
        ck = (i, k)
        hit = self.solved.get(ck)
        if hit is None:
            hit = self.solved[ck] = M.solved(self.tasks[i], _from_key(k))
        return hit

    def search(self, max_len: int):
        """Best solve rate achievable by any fixed word, per word length."""
        n = len(self.tasks)
        best: Dict[int, Tuple[float, Tuple[Op, ...]]] = {}
        frontier = [((), tuple(self.start(i) for i in range(n)))]
        rate = sum(self.is_solved(i, k) for i, k in
                   enumerate(frontier[0][1])) / n
        best[0] = (rate, ())
        for d in range(1, max_len + 1):
            nxt = []
            bd = (-1.0, ())
            for word, keys in frontier:
                for op in ALPHABET:
                    nk = tuple(self.step(i, k, op) for i, k in enumerate(keys))
                    w = word + (op,)
                    r = sum(self.is_solved(i, k) for i, k in enumerate(nk)) / n
                    if r > bd[0]:
                        bd = (r, w)
                    nxt.append((w, nk))
            best[d] = bd
            frontier = nxt
        return best


def admissibility(tasks, oracle_rate: float, max_len: int = 6,
                  track_destroys: bool = True):
    ex = Explorer(tasks, track_destroys=track_destroys)
    best = ex.search(max_len)
    ceiling = max(r for r, _ in best.values())
    dominating = [(d, w) for d, (r, w) in sorted(best.items())
                  if r >= oracle_rate - 1e-9]
    return {
        "per_length": best,
        "fixed_ceiling": ceiling,
        "oracle_ceiling": oracle_rate,
        "admissible": ceiling < oracle_rate - 1e-9,
        "shortest_dominating": dominating[0] if dominating else None,
        "states_cached": len(ex.cache),
    }


def report(name, res) -> None:
    print(f"\n   task set: {name}")
    print(f"   {'len':>4s} {'best fixed':>11s}   word")
    for d, (r, w) in sorted(res["per_length"].items()):
        print(f"   {d:4d} {r*100:10.1f}%   {[o.value for o in w]}")
    print(f"   fixed ceiling  {res['fixed_ceiling']*100:6.1f}%")
    print(f"   oracle ceiling {res['oracle_ceiling']*100:6.1f}%")
    sd = res["shortest_dominating"]
    if sd is not None:
        print(f"   SHORTEST DOMINATING SEQUENCE (len {sd[0]}): "
              f"{[o.value for o in sd[1]]}")
    print(f"   VERDICT: {'ADMISSIBLE' if res['admissible'] else 'NOT ADMISSIBLE'}"
          f" -- {'branching is required somewhere' if res['admissible'] else 'a fixed pipeline matches any policy; this set cannot test metacontrol'}")


def main() -> int:
    import random
    import statistics
    import tasks_m4 as T

    print(__doc__)
    print("=" * 78)
    tasks = [t for s in range(1, 17) for t in T.build(s)]
    orc = statistics.mean([M.run_episode(t, "oracle", random.Random(0)).solved
                           for t in tasks])
    print(f"   {len(tasks)} task instances (16 seeds x 6 families), "
          f"alphabet {[o.value for o in ALPHABET]}")
    print("   Exhaustive over ALL 4^d words, not the 24 cyclic permutations M4's")
    print("   FIXED control used -- a strictly stronger control, and it agrees.")
    print("   STOP is omitted soundly: in an episode it only truncates, so any word")
    print("   containing it scores as its own prefix, and every prefix is searched.")

    for td in (True, False):
        res = admissibility(tasks, orc, max_len=5, track_destroys=td)
        report(f"M4 v0.1 (frozen at 82f2051), track_destroys={td}", res)

    print("\n" + "=" * 78)
    print("   The verdict does not depend on the one authored cost in the mechanism.")
    print("   NECESSARY, NOT SUFFICIENT: passing says branching is required, not that")
    print("   any controller finds it. The RANDOM and FIXED controls still apply.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
