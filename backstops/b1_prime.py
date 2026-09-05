"""B1' -- the maximum over K-indexed lookup learners, per B1_PRIME_INSTRUMENT.md.

Measurement tooling, exempt under GROWTH_PROTOCOL's standing exceptions. It
introduces no agent and no U, and it does not amend PHASE1_Q1_SPEC.md: Q1
remains rejected at a4877fb on the 0.50 frozen in B1_FAILURE_FREEZE.md.

q1_backstops.py is deliberately not modified, so its frozen numbers stay
reproducible. The environment definitions below are imported from it rather
than restated, so the two instruments cannot drift apart.

Run: python3 backstops/b1_prime.py
"""

import itertools
from collections import defaultdict

from q1_backstops import (
    ALL_R, FEATURES, PROBE,
    acquisition_states, correct_label, dist, optimal_labels, project,
)


def local_data(K, R):
    """The fitting data visible at the probe key, under locality.

    Returns the label-indexed vote vector recorded at key pi_K(S*), or None if
    that key is never reached during acquisition. Votes are accumulated exactly
    as the frozen implementation accumulates them; only the fitting rule that
    consumes them is being replaced.
    """
    k_star = project(PROBE, K)
    votes = [0] * 4
    seen = False
    for s in acquisition_states():
        if dist(s) == 0:
            continue
        if project(s, K) != k_star:
            continue
        seen = True
        for lab in optimal_labels(s, R):
            votes[lab] += 1
    return tuple(votes) if seen else None


def b1_prime_for_subset(K):
    """Max probe accuracy over every fitting rule obeying locality.

    F is an arbitrary function of the local data, so it can separate two rule
    instances exactly when their local data differ. Partition the 24 instances
    by local data; on each class F emits one label, best case the plurality
    target label. An unreachable key gives one class of all 24 instances, whose
    plurality is 6/24 = 0.25.
    """
    classes = defaultdict(list)
    for R in ALL_R:
        classes[local_data(K, R)].append(R)

    hits = 0
    for _, members in classes.items():
        tally = defaultdict(int)
        for R in members:
            tally[correct_label(R)] += 1
        hits += max(tally.values())
    return hits / len(ALL_R), classes


def run():
    print("B1' -- max over K-indexed lookup learners (locality-restricted)")
    print("     24 rule instances x 16 feature subsets, exact, no sampling\n")

    rows = []
    for r in range(5):
        for K in itertools.combinations(range(4), r):
            acc, classes = b1_prime_for_subset(K)
            name = "+".join(FEATURES[i] for i in K) if K else "(constant)"
            reachable = list(classes.keys()) != [None]
            rows.append((acc, name, K, reachable, len(classes)))

    rows.sort(key=lambda t: (-t[0], t[1]))
    print("     {:<16} {:>10} {:>12} {:>9}   {}".format(
        "key", "B1' acc", "probe key", "classes", "verdict"))
    worst = 0.0
    for acc, name, _K, reachable, ncls in rows:
        worst = max(worst, acc)
        print("     {:<16} {:>10.4f} {:>12} {:>9}   {}".format(
            name, acc, "seen" if reachable else "unreachable", ncls,
            "OK" if acc <= 0.25 + 1e-9 else "EXCEEDS 0.25"))

    print("\n     max over all subsets: {:.4f}".format(worst))
    return worst, rows


def check_predictions(worst, rows):
    by_name = {name: acc for acc, name, _K, _r, _c in rows}
    print("\nPredeclared checks (B1_PRIME_INSTRUMENT.md)")

    p1 = worst >= 0.5 - 1e-9
    print("     1. B1' >= 0.5000 by dominance over majority vote : "
          "{:.4f}  {}".format(worst, "OK" if p1 else "INSTRUMENT BUG"))

    p3 = abs(by_name.get("d1+t1", -1) - 0.5) < 1e-9
    print("     3. d1+t1 == 0.5000, the known minimal case      : "
          "{:.4f}  {}".format(by_name.get("d1+t1", float("nan")),
                              "OK" if p3 else "INSTRUMENT BUG"))

    t2_subsets = {n: a for n, a in by_name.items() if "t2" in n.split("+")}
    p4 = all(abs(a - 0.25) < 1e-9 for a in t2_subsets.values())
    print("     4. every subset containing t2 == 0.2500          : "
          "{}/{} {}".format(
              sum(1 for a in t2_subsets.values() if abs(a - 0.25) < 1e-9),
              len(t2_subsets), "OK" if p4 else "INSTRUMENT BUG"))

    print("\n     2. open question -- is B1' > 0.5000 on some subset?")
    if worst > 0.5 + 1e-9:
        print("        YES: {:.4f}. Q1 is worse than its failure freeze "
              "recorded.".format(worst))
    else:
        print("        NO: majority vote attained the maximum of the class.")

    return p1 and p3 and p4


if __name__ == "__main__":
    print("B1' instrument, Q1 environment frozen at a4877fb\n" + "=" * 70)
    worst, rows = run()
    ok = check_predictions(worst, rows)
    print("\n" + "=" * 70)
    print("SUMMARY")
    print("     B1  (majority vote, frozen)  : 0.5000")
    print("     B1' (max over the class)     : {:.4f}".format(worst))
    print("     instrument self-checks       : {}".format(
        "PASS" if ok else "FAIL -- freeze this before concluding anything"))
