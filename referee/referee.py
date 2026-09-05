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


def vote_tables(design):
    """votes[R][K][key] -> label-indexed vote vector, over reachable non-goal states."""
    m = design["mod"]
    states = [s for s in reachable(design) if dist(s, m) != 0]
    subsets = [K for r in range(5) for K in itertools.combinations(range(4), r)]
    tables = {}
    for R in ALL_R:
        per_K = {K: defaultdict(lambda: [0, 0, 0, 0]) for K in subsets}
        for s in states:
            wins = progress_labels(s, R, m)
            for K in subsets:
                v = per_K[K][project(s, K)]
                for l in wins:
                    v[l] += 1
        tables[R] = per_K
    return tables, subsets


def b1_prime(design, probe, tables, subsets):
    """Max probe accuracy over locality-restricted K-indexed lookup learners."""
    m = design["mod"]
    worst, culprit = 0.0, None
    for K in subsets:
        key = project(probe, K)
        classes = defaultdict(list)
        for R in ALL_R:
            entry = tables[R][K]
            classes[tuple(entry[key]) if key in entry else None].append(R)
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


def judge(design, verbose=False):
    tables, subsets = vote_tables(design)
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
    print("Referee v0 -- two-dial family, B1' criterion\n" + "=" * 72)

    print("\nCalibration against the known-rejected Q1 design")
    q1 = {"mod": 4, "frozen": {3: 0}}
    tables, subsets = vote_tables(q1)
    acc, culprit = b1_prime(q1, (0, 0, 0, 1), tables, subsets)
    ok = abs(acc - 0.5) < 1e-9
    print("    {} probe=(0,0,0,1) -> B1'={:.4f} via {}   {}".format(
        describe(q1), acc,
        "+".join(NAMES[i] for i in culprit) if culprit else "-",
        "MATCHES the frozen 0.50" if ok else "CALIBRATION FAILURE"))
    if not ok:
        return 1

    print("\nExhaustive sweep -- best probe each design admits")
    print("    {:<26} {:>10} {:>14}   {}".format(
        "design", "best B1'", "probe", "identified by"))
    passing = []
    for design in enumerate_family():
        best, _all = judge(design)
        if best is None:
            print("    {:<26} {:>10}".format(describe(design), "no probe"))
            continue
        acc, probe, culprit = best
        name = "+".join(NAMES[i] for i in culprit) if culprit else "-"
        if acc <= 0.25 + 1e-9:
            passing.append((design, probe))
        print("    {:<26} {:>10.4f} {:>14}   {}".format(
            describe(design), acc, str(probe), name))

    print("\n" + "=" * 72)
    print("designs whose best probe reaches chance (B1' = 0.25): {}".format(len(passing)))
    for design, probe in passing:
        print("    {}  probe={}".format(describe(design), probe))
    if not passing:
        print("    NONE. No two-dial design in this family admits an unidentifiable probe.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
