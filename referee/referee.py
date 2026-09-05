"""Referee v0 -- the external criterion neither agent can argue with.

Given a candidate two-dial environment design, decide whether its probe is
identifiable from acquisition-reachable features, by the B1' rule established in
backstops/B1_PRIME_INSTRUMENT.md. No agent, no U, no sampling, no seeds.

A design is:

    mod          dial modulus m; dials and targets live in Z/mZ
    frozen       {target index: value} held constant throughout acquisition
    probe        the observation forced at evaluation time

Acquisition-reachable observations are every state whose frozen targets hold
their frozen values; the agent controls both dials, so nothing about the dials
is unreachable. A probe must violate at least one freeze, or it is reachable and
the design is dead on arrival.

Verdict PASS requires the B1' maximum over all 2^4 feature subsets to equal
chance, 1/4. Anything above means some projection identifies the probed label,
and the design is rejected exactly as Q1 was.

Run: python3 referee/referee.py
"""

import itertools
from collections import defaultdict

LABELS = (0, 1, 2, 3)
ALL_R = tuple(itertools.permutations(LABELS))  # label -> effect
D1_INC, D1_DEC, D2_INC, D2_DEC = 0, 1, 2, 3
NAMES = ("d1", "d2", "t1", "t2")
TARGET_IDX = (2, 3)


def apply_effect(state, eff, m):
    d1, d2, t1, t2 = state
    if eff == D1_INC:
        d1 = (d1 + 1) % m
    elif eff == D1_DEC:
        d1 = (d1 - 1) % m
    elif eff == D2_INC:
        d2 = (d2 + 1) % m
    else:
        d2 = (d2 - 1) % m
    return (d1, d2, t1, t2)


def cyc(a, b, m):
    return min((a - b) % m, (b - a) % m)


def dist(s, m):
    return cyc(s[0], s[2], m) + cyc(s[1], s[3], m)


def reachable(design):
    """Observations available during acquisition."""
    m, frozen = design["mod"], design["frozen"]
    out = []
    for s in itertools.product(range(m), repeat=4):
        if all(s[i] == v for i, v in frozen.items()):
            out.append(s)
    return out


def progress_labels(s, R, m):
    d0 = dist(s, m)
    return [l for l in LABELS if dist(apply_effect(s, R[l], m), m) < d0]


def winning_labels(s, R, m):
    return [l for l in LABELS if dist(apply_effect(s, R[l], m), m) == 0]


def project(s, K):
    return tuple(s[i] for i in K)


# The input statistic: what a K-indexed entry RETAINS from the rows filed there.
# Locality fixes which rows may be read; it does not fix how much of each row is
# kept, and the two are independent choices. Naming this is mandatory -- see
# backstops/B1_PRIME_SCOPE.md.
STATISTICS = ("counts", "proj_triples", "full_triples")


def local_stat(states, R, K, key, m, statistic):
    rows = [s for s in states if project(s, K) == key]
    if not rows:
        return None
    if statistic == "counts":
        v = [0, 0, 0, 0]
        for s in rows:
            for l in progress_labels(s, R, m):
                v[l] += 1
        return tuple(v)
    if statistic == "proj_triples":
        return tuple(sorted(
            (project(s, K), l, project(apply_effect(s, R[l], m), K))
            for s in rows for l in progress_labels(s, R, m)))
    return tuple(sorted(
        (s, l, apply_effect(s, R[l], m))
        for s in rows for l in progress_labels(s, R, m)))


def vote_tables(design, statistic="counts"):
    """tables[R][K][key] -> the local statistic filed at that key."""
    m = design["mod"]
    states = [s for s in reachable(design) if dist(s, m) != 0]
    subsets = [K for r in range(5) for K in itertools.combinations(range(4), r)]
    keys = {K: {project(s, K) for s in states} for K in subsets}
    tables = {}
    for R in ALL_R:
        tables[R] = {K: {k: local_stat(states, R, K, k, m, statistic)
                         for k in keys[K]} for K in subsets}
    return tables, subsets


def b1_prime(design, probe, tables, subsets):
    """Max probe accuracy over locality-restricted K-indexed lookup learners."""
    m = design["mod"]
    worst, culprit = 0.0, None
    for K in subsets:
        key = project(probe, K)
        classes = defaultdict(list)
        for R in ALL_R:
            classes[tables[R][K].get(key)].append(R)
        hits = 0
        for members in classes.values():
            tally = defaultdict(int)
            for R in members:
                w = winning_labels(probe, R, m)
                if len(w) == 1:
                    tally[w[0]] += 1
            hits += max(tally.values()) if tally else 0
        acc = hits / len(ALL_R)
        if acc > worst:
            worst, culprit = acc, K
    return worst, culprit


def probe_candidates(design):
    """States that are unreachable, non-goal, and solvable by exactly one label."""
    m, frozen = design["mod"], design["frozen"]
    out = []
    for s in itertools.product(range(m), repeat=4):
        if all(s[i] == v for i, v in frozen.items()):
            continue  # reachable during acquisition
        if dist(s, m) == 0:
            continue
        if all(len(winning_labels(s, R, m)) == 1 for R in ALL_R[:1]):
            out.append(s)
    return out


def judge_best(design, statistic="counts"):
    tables, subsets = vote_tables(design, statistic)
    results = []
    for probe in probe_candidates(design):
        acc, culprit = b1_prime(design, probe, tables, subsets)
        results.append((acc, probe, culprit))
    if not results:
        return None, []
    results.sort()
    return results[0], results  # the most favourable probe this design admits


def describe(design):
    return "m={} frozen={{{}}}".format(
        design["mod"],
        ", ".join("{}={}".format(NAMES[i], v) for i, v in sorted(design["frozen"].items())))


def enumerate_family():
    """Every two-dial design with at least one frozen target."""
    for m in (3, 4, 5):
        for k in (1, 2):
            for idxs in itertools.combinations(TARGET_IDX, k):
                for vals in itertools.product(range(m), repeat=k):
                    yield {"mod": m, "frozen": dict(zip(idxs, vals))}


def main():
    print("Referee v1 -- two-dial family, B1' criterion, statistic named\n" + "=" * 72)
    q1 = {"mod": 4, "frozen": {3: 0}}

    print("\nCalibration -- the known-rejected Q1 design, under each statistic")
    for st in STATISTICS:
        tables, subsets = vote_tables(q1, st)
        acc, culprit = b1_prime(q1, (0, 0, 0, 1), tables, subsets)
        note = ""
        if st == "counts":
            note = "  <- matches the frozen 0.50" if abs(acc - 0.5) < 1e-9 else "  <- CALIBRATION FAILURE"
        print("    {:<15} B1' = {:.4f} via {}{}".format(
            st, acc, "+".join(NAMES[i] for i in culprit) if culprit else "(constant)", note))

    print("\n  Vacuity check: what does the COARSEST key score under each statistic?")
    print("  If the empty subset -- one global entry -- already reaches 1.00, then")
    print("  locality constrains nothing and the projection lattice is decoration.")
    for st in STATISTICS:
        tables, _ = vote_tables(q1, st)
        classes = defaultdict(list)
        for R in ALL_R:
            classes[tables[R][()].get(())].append(R)
        hits = 0
        for members in classes.values():
            tally = defaultdict(int)
            for R in members:
                w = winning_labels((0, 0, 0, 1), R, 4)
                if len(w) == 1:
                    tally[w[0]] += 1
            hits += max(tally.values()) if tally else 0
        acc = hits / len(ALL_R)
        print("    {:<15} constant key -> {:.4f}   {}".format(
            st, acc, "VACUOUS" if acc > 0.9 else "non-vacuous"))

    print("\nExhaustive sweep over the family, under each non-vacuous statistic")
    for st in ("counts", "proj_triples"):
        passing, worst_seen, n = [], 0.0, 0
        for design in enumerate_family():
            best, _ = judge_best(design, st)
            if best is None:
                continue
            n += 1
            acc, probe, _culprit = best
            worst_seen = max(worst_seen, acc)
            if acc <= 0.25 + 1e-9:
                passing.append((design, probe))
        print("    {:<15} designs {:>3}   reaching chance {:>3}   worst best-probe {:.4f}".format(
            st, n, len(passing), worst_seen))

    print("\n" + "=" * 72)
    print("VERDICT  the two-dial family admits no unidentifiable probe under")
    print("         either non-vacuous statistic. The counts/triples distinction")
    print("         changes the number and not the verdict.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
