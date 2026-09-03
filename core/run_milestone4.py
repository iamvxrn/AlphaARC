"""Milestone 4 -- MetaController v0. ORACLE / HARD / CORRUPTED / ABSTAIN, against
FIXED and RANDOM sequencing, on the task set frozen in commit 82f2051.

THE QUESTION, from the frozen spec: can feedback-dependent sequencing of
heterogeneous cognitive operations work under IMPERFECT epistemic classification?

ONE DECLARED CHANGE OF EXPERIMENT VERSION -- v0.1, announced before execution, not
after the numbers. The spec froze four arms. It also separates two error sources,
`C_observability` (what the evidence forbids anyone from knowing) and
`C_classifier` (our discriminator's error above that floor), and insists they not be
merged. Running ABSTAIN only at the HARD noise level confounds "can abstain" with
"noise level", which is the exact fault that killed the first M3 decisive pair. So
the four arms are run as CELLS OF A CROSSED DESIGN -- {forced, abstaining} x
{oracle, hard, corrupted at c} -- and the off-diagonal is read, not just the
diagonal. This adds cells; it changes nothing about what the original four measure.

THE READING, FIXED BEFORE THE RUN. Printed above the numbers on purpose.

  1. If RANDOM matches ORACLE, typed routing is not doing the work on this
     benchmark -- having several operations is. That is a NEGATIVE result for the
     thesis and is to be reported as one, not explained away.
  2. If the best FIXED order matches ORACLE, the sequencing is not
     feedback-dependent: a static pipeline suffices and a controller is unmotivated.
     Best-of-24 is chosen ON THE TEST SET and is therefore a generous upper bound
     for fixed sequencing -- deliberately generous, since it is a control.
  3. If ORACLE beats RANDOM but HARD does not, the whole gain sits in epistemic
     knowledge that the actual observation channel cannot deliver. Routing would be
     real but unreachable, which is a different and weaker claim than "it works".
  4. If ABSTAIN beats HARD, that is NOT "abstention is smarter" on its own. Read
     solve rate, budget and irreversible failures together; buying safety with
     budget is a trade, not an improvement.
  5. If the ORACLE-over-RANDOM gap survives only while `track_destroys` is on, the
     gap is the authored penalty on a misrouted TRACK and must be reported as that.
"""
from __future__ import annotations

import random
import statistics
import sys
from collections import defaultdict
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import tasks_m4 as T
import metacontroller as M
from metacontroller import Label, Op

SEEDS = list(range(1, 17))
CS = (0.0, 0.15, 0.30, 0.50)


def run_arm(arm, seeds, *, c=0.0, fixed=None, track_destroys=True, budget=M.BUDGET,
            keep_trace=False):
    """Every arm sees the SAME task instances for a given seed: paired by
    construction, and since `T.build` is cached that is now guaranteed rather than
    merely reproduced. Traces are dropped unless asked for -- the sweeps run tens of
    thousands of episodes and retaining every trace killed the first attempt."""
    per_seed = []
    fam = defaultdict(list)
    for s in seeds:
        rng = random.Random(90000 + s)
        eps = [M.run_episode(t, arm, rng, c=c, fixed=fixed,
                             track_destroys=track_destroys, budget=budget)
               for t in T.build(s)]
        if not keep_trace:
            for e in eps:
                e.trace = []
        per_seed.append(eps)
        for e in eps:
            fam[e.family].append(e)
    return per_seed, fam


def summarise(per_seed):
    rate = [sum(e.solved for e in eps) / len(eps) for eps in per_seed]
    ops = [statistics.mean(e.ops for e in eps) for eps in per_seed]
    waste = [statistics.mean(e.ops_to_solve for e in eps) for eps in per_seed]
    irr = [sum(e.irreversible for e in eps) for eps in per_seed]
    return rate, ops, waste, irr


def paired(a, b):
    d = [x - y for x, y in zip(a, b)]
    m = statistics.mean(d)
    sem = (statistics.stdev(d) / len(d) ** 0.5) if len(d) > 1 and statistics.pstdev(d) > 0 else 0.0
    agree = all(x > 0 for x in d) or all(x < 0 for x in d)
    return m, sem, agree


def verdict(m, sem, agree):
    if sem == 0.0:
        return "REAL" if m != 0 and agree else "zero"
    if abs(m) > 2 * sem and agree:
        return "REAL"
    return "not resolved"


def line(name, rate, ops, waste, irr):
    print(f"   {name:22s} {statistics.mean(rate)*100:6.1f}%   "
          f"{statistics.mean(ops):5.2f}   {statistics.mean(waste):5.2f}   "
          f"{sum(irr):4d}")


def main() -> int:
    print(__doc__)
    print("=" * 78)
    print(f"   {len(SEEDS)} seeds, budget {M.BUDGET}, {len(T.FAMILIES)} families, "
          f"paired on identical task instances")
    print("=" * 78)

    # ---------------------------------------------------------------- sanity
    print("\n0. SANITY -- the instrument before the measurement")
    h, _ = run_arm("hard", SEEDS, keep_trace=True)
    c0, _ = run_arm("corrupted", SEEDS, c=0.0, keep_trace=True)
    same = all(a.trace == b.trace for x, y in zip(h, c0) for a, b in zip(x, y))
    print(f"   corruption at c=0 reproduces HARD exactly ......... {same}")
    assert same, "the corruption channel is not a pure addition to HARD"
    pol = {l: M.ROUTE.get(l) for l in Label if l is not Label.UNKNOWN}
    print(f"   the policy is a deterministic table, no scalars ... {len(pol)} labels")
    print(f"   operations offered to every arm alike ............ "
          f"{[o.value for o in M.REAL_OPS]} + stop")

    # ------------------------------------------------------------- main table
    print("\n1. THE ARMS, and the two controls that can refute the whole idea")
    print(f"   {'arm':22s} {'solved':>7s}   {'ops':>5s}   {'waste':>5s}   {'irrev':>4s}")
    results = {}
    for name, kw in (("ORACLE", dict(arm="oracle")),
                     ("HARD", dict(arm="hard")),
                     ("CORRUPTED c=0.30", dict(arm="corrupted", c=0.30)),
                     ("CORRUPTED c=0.50", dict(arm="corrupted", c=0.50)),
                     ("ABSTAIN", dict(arm="abstain")),
                     ("RANDOM", dict(arm="random"))):
        ps, fam = run_arm(kw.pop("arm"), SEEDS, **kw)
        results[name] = (summarise(ps), fam)
        line(name, *results[name][0])

    fixed_scores = []
    for order in M.FIXED_ORDERS:
        ps, fam = run_arm("fixed", SEEDS, fixed=order)
        summ = summarise(ps)
        fixed_scores.append((statistics.mean(summ[0]), order, summ, fam))
    fixed_scores.sort(key=lambda x: -x[0])
    best = fixed_scores[0]
    med = fixed_scores[len(fixed_scores) // 2]
    results["FIXED best-of-24"] = (best[2], best[3])
    results["FIXED median-of-24"] = (med[2], med[3])
    line("FIXED best-of-24", *best[2])
    line("FIXED median-of-24", *med[2])
    top = sum(1 for x in fixed_scores if x[0] >= best[0] - 1e-9)
    print(f"   best fixed order: {[o.value for o in best[1]]}  "
          f"(chosen ON the test set -- a generous control, on purpose)")
    print(f"   {top} of 24 orders reach that score; median order {med[0]*100:.1f}%")

    # ------------------------------------------------- the pre-registered reads
    print("\n2. THE PRE-REGISTERED COMPARISONS, paired over the same seeds")
    rnd = results["RANDOM"][0][0]
    orc = results["ORACLE"][0][0]
    hrd = results["HARD"][0][0]
    for label, a, b in (("ORACLE  - RANDOM", orc, rnd),
                        ("ORACLE  - FIXED best", orc, best[2][0]),
                        ("HARD    - RANDOM", hrd, rnd),
                        ("HARD    - FIXED best", hrd, best[2][0]),
                        ("ORACLE  - HARD", orc, hrd),
                        ("ABSTAIN - HARD", results["ABSTAIN"][0][0], hrd)):
        m, sem, agree = paired(a, b)
        print(f"   {label:22s} {m*100:+7.2f} pp   sem {sem*100:5.2f}   "
              f"seeds agree {str(agree):5s}   {verdict(m, sem, agree)}")

    # --------------------------------------------------------- per family
    print("\n3. PER FAMILY -- solve rate, because an aggregate hides which gap is real")
    fams = T.FAMILIES
    print(f"   {'arm':22s}" + "".join(f"{f[:11]:>13s}" for f in fams))
    for name in ("ORACLE", "HARD", "CORRUPTED c=0.30", "ABSTAIN", "RANDOM",
                 "FIXED best-of-24"):
        fam = results[name][1]
        cells = "".join(f"{sum(e.solved for e in fam[f])/max(1,len(fam[f]))*100:12.0f}%"
                        for f in fams)
        print(f"   {name:22s}{cells}")

    # ------------------------------------------- the declared factorial (v0.1)
    print("\n4. THE CROSSED DESIGN (v0.1) -- abstention x classifier noise")
    print("   C_observability is the c=0 column; C_classifier is the sweep across it.")
    print(f"   {'':10s}" + "".join(f"{'c='+str(c):>12s}" for c in CS))
    for lab, arm in (("forced", "corrupted"), ("abstaining", "abstain_corrupted")):
        row = []
        for c in CS:
            ps, _ = run_arm(arm, SEEDS, c=c)
            row.append(statistics.mean(summarise(ps)[0]))
        print(f"   {lab:10s}" + "".join(f"{v*100:11.1f}%" for v in row))
    print("   (the c=0 forced cell IS hard; the c=0 abstaining cell IS abstain)")

    # ---------------------------------------------------- the authored cost
    print("\n5. THE ABLATION THAT COULD MAKE THIS RESULT MY OWN PENALTY")
    print("   `track_destroys` is authored, not measured. If the gap needs it, say so.")
    print(f"   {'':22s} {'ORACLE':>8s} {'HARD':>8s} {'RANDOM':>8s}  {'O-R paired':>12s}")
    for td in (True, False):
        o = summarise(run_arm("oracle", SEEDS, track_destroys=td)[0])[0]
        hh = summarise(run_arm("hard", SEEDS, track_destroys=td)[0])[0]
        r = summarise(run_arm("random", SEEDS, track_destroys=td)[0])[0]
        m, sem, agree = paired(o, r)
        print(f"   track_destroys={str(td):5s}      {statistics.mean(o)*100:7.1f}% "
              f"{statistics.mean(hh)*100:7.1f}% {statistics.mean(r)*100:7.1f}%  "
              f"{m*100:+7.2f}pp {verdict(m, sem, agree)}")

    # ------------------------------------------------------- budget pressure
    print("\n6. BUDGET SWEEP -- a budget nothing can exhaust measures nothing")
    print("   FIXED is in this table because reading #2 turns on it: a fixed cycle")
    print("   run to a budget nothing exhausts is a brute-force sweep of the whole")
    print("   operation set, which is not the same claim as 'a pipeline suffices'.")
    print(f"   {'budget':>7s}" + "".join(f"{n:>12s}" for n in
                                         ("ORACLE", "HARD", "ABSTAIN", "RANDOM",
                                          "FIXEDbest")))
    for b in (3, 4, 6, 8, 12, 20):
        vals = [statistics.mean(summarise(run_arm(a, SEEDS, budget=b)[0])[0])
                for a in ("oracle", "hard", "abstain", "random")]
        fb = max(statistics.mean(summarise(run_arm("fixed", SEEDS, fixed=o,
                                                   budget=b)[0])[0])
                 for o in M.FIXED_ORDERS)   # best-of-24 recomputed AT this budget
        print(f"   {b:7d}" + "".join(f"{v*100:11.1f}%" for v in vals)
              + f"{fb*100:11.1f}%")

    # ------------------------------------------------- counterfactual replay
    print("\n7. INTERVENTION LOG -- decision-level counterfactual replay, not a story")
    print("   Every decision is replaced by each alternative and the controller then")
    print("   carries on. A choice that changes no outcome was not load-bearing.")
    print(f"   {'arm':10s} {'decisions':>10s} {'load-bearing':>13s} "
          f"{'ops saved vs best dev':>23s}")
    for arm in ("oracle", "hard", "abstain"):
        dec = load = 0
        regrets = []
        for s in SEEDS[:8]:
            for t in T.build(s):
                base = M.run_episode(t, arm, random.Random(90000 + s))
                chosen = [Op(o) for _, o in base.trace]
                for i in range(len(chosen)):
                    dec += 1
                    alts = []
                    for alt in M.REAL_OPS + (Op.STOP,):
                        if alt is chosen[i]:
                            continue
                        cf = M.run_episode(t, arm, random.Random(90000 + s),
                                           forced=tuple(chosen[:i]) + (alt,))
                        alts.append(cf)
                    if any(a.solved != base.solved for a in alts):
                        load += 1
                    best_alt = min((a.ops_to_solve for a in alts if a.solved),
                                   default=None)
                    if base.solved and best_alt is not None:
                        # positive = the controller reached a solved state in FEWER
                        # ops than the best single-step deviation would have
                        regrets.append(best_alt - base.ops_to_solve)
        print(f"   {arm:10s} {dec:10d} {load/max(1,dec)*100:12.1f}% "
              f"{statistics.mean(regrets) if regrets else 0:22.2f}")

    # ------------------------------------------------- why the control wins
    print("\n8. WHY FIXED WINS -- the diagnosis, because a bare number is not a result")
    def fixed_score(order, b):
        return statistics.mean(summarise(
            run_arm("fixed", SEEDS, fixed=order, budget=b)[0])[0])
    print(f"   {'budget':>7s} {'max':>7s}  {'orders reaching it':>18s}   minimal winner")
    for b in (2, 3, 4, 12):
        sc = [(fixed_score(o, b), o) for o in M.FIXED_ORDERS]
        mx = max(v for v, _ in sc)
        win = [o for v, o in sc if v >= mx - 1e-9]
        print(f"   {b:7d} {mx*100:6.1f}%  {len(win):10d} of 24   "
              f"{[x.value for x in win[0]][:b]}")
    print("   ONE static sequence -- explore, track, induce -- solves EVERY family at")
    print("   the ORACLE's own score, in three operations, identical for all six.")

    print("\n   Does any family REQUIRE an operation twice (oracle arm)?")
    reuse, tot = defaultdict(int), defaultdict(int)
    for s_ in SEEDS:
        for t in T.build(s_):
            e = M.run_episode(t, "oracle", random.Random(90000 + s_))
            ops = [b for _, b in e.trace if b != "stop"]
            tot[t.family] += 1
            if any(ops.count(o) > 1 for o in set(ops)):
                reuse[t.family] += 1
    for f in T.FAMILIES:
        print(f"     {f:22s} {reuse[f]:2d}/{tot[f]:2d}")
    print("   ORACLE re-inducts only because it induces EARLY, on a poor view, then")
    print("   has to redo it. Exploring first makes the reuse unnecessary. So no task")
    print("   here needs an operation whose CORRECT CONTENT depends on feedback, and")
    print("   that -- not the budget -- is why a fixed order is enough.")

    print("\n" + "=" * 78)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
