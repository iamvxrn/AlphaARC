"""Q1 backstops B1/B2/B3, per PHASE1_Q1_SPEC.md frozen at a4877fb.

Executes the preregistered instrument checks. Contains no agent and no U:
this is measurement tooling, exempt under GROWTH_PROTOCOL's standing
exceptions. It does not modify the specification.

Run: python3 backstops/q1_backstops.py
"""

import itertools
from collections import defaultdict

# ---------------------------------------------------------------- environment

# Effect indices, per spec section 2.
D1_INC, D1_DEC, D2_INC, D2_DEC = 0, 1, 2, 3

# A rule instance R is a bijection label -> effect, represented as a tuple
# where R[label] = effect. All 24 of them.
ALL_R = list(itertools.permutations(range(4)))

PROBE = (0, 0, 0, 1)  # S* = (d1=0, d2=0, t1=0, t2=1), spec section 6


def apply_effect(state, eff):
    d1, d2, t1, t2 = state
    if eff == D1_INC:
        d1 = (d1 + 1) % 4
    elif eff == D1_DEC:
        d1 = (d1 - 1) % 4
    elif eff == D2_INC:
        d2 = (d2 + 1) % 4
    elif eff == D2_DEC:
        d2 = (d2 - 1) % 4
    return (d1, d2, t1, t2)


def cyc(a, b):
    return min((a - b) % 4, (b - a) % 4)


def dist(state):
    d1, d2, t1, t2 = state
    return cyc(d1, t1) + cyc(d2, t2)


def correct_label(R):
    """The label that increments dial 2 under R -- the probe's unique optimum."""
    return R.index(D2_INC)


def acquisition_states():
    """Every observation reachable during acquisition.

    Spec section 4: t2 == 0 always. Episodes start with t1 != d1 and d2 == 0,
    but d1 and d2 both evolve under the agent's own presses, so every
    combination of (d1, d2, t1) is reachable with t2 pinned to 0.
    """
    return [(d1, d2, t1, 0)
            for d1 in range(4) for d2 in range(4) for t1 in range(4)]


def optimal_labels(state, R):
    """Labels that strictly reduce distance to the target under R."""
    d0 = dist(state)
    return [lab for lab in range(4)
            if dist(apply_effect(state, R[lab])) < d0]


# ------------------------------------------------------------------------- B1
# "Over the enumerated transition table, exhaustively compute the best probe
#  accuracy achievable by every lookup table fitted to acquisition-reachable
#  data. Required: 0.25. Anything above rejects the design."  -- spec 11
#
# A lookup policy is (K, f): K a subset of observation features, f a map from
# observed key-values to actions. Fitted from acquisition data by recording,
# at each key, which actions made progress -- something the agent can read off
# its own transitions without being told R. Majority vote per key; ties are
# broken uniformly, and the tie is scored in expectation rather than sampled.

FEATURES = ("d1", "d2", "t1", "t2")


def project(state, K):
    return tuple(state[i] for i in K)


def b1_for_subset(K):
    """Expected probe accuracy of the best acquisition-fitted lookup table."""
    total = 0.0
    per_R = []
    for R in ALL_R:
        # Fit: accumulate progress-making actions per key.
        votes = defaultdict(lambda: [0] * 4)
        for s in acquisition_states():
            if dist(s) == 0:
                continue  # goal already met; no action to memorise
            for lab in optimal_labels(s, R):
                votes[project(s, K)][lab] += 1

        k_star = project(PROBE, K)
        if k_star not in votes:
            acc = 0.25  # key never observed in acquisition -> no entry -> chance
        else:
            v = votes[k_star]
            best = max(v)
            winners = [lab for lab in range(4) if v[lab] == best]
            acc = (1.0 / len(winners)) if correct_label(R) in winners else 0.0
        per_R.append(acc)
        total += acc
    return total / len(ALL_R), per_R


def run_b1():
    print("B1 -- exhaustive lookup-table shortcut audit")
    print("    24 rule instances x 16 feature subsets, uniform over "
          "acquisition-reachable states\n")
    rows = []
    for r in range(5):
        for K in itertools.combinations(range(4), r):
            acc, _ = b1_for_subset(K)
            key_name = "{}".format(
                "+".join(FEATURES[i] for i in K) if K else "(constant)")
            seen = project(PROBE, K) in _keys_seen(K)
            rows.append((acc, key_name, seen))

    rows.sort(reverse=True)
    print("    {:<16} {:>10}  {:>12}   {}".format(
        "key", "probe acc", "probe key", "verdict"))
    worst = 0.0
    for acc, name, seen in rows:
        verdict = "OK" if acc <= 0.25 + 1e-9 else "EXCEEDS 0.25"
        worst = max(worst, acc)
        print("    {:<16} {:>10.4f}  {:>12}   {}".format(
            name, acc, "seen" if seen else "unreachable", verdict))
    print("\n    max over all subsets: {:.4f}   (spec 11 requires 0.25)".format(worst))
    return worst


def _keys_seen(K):
    out = set()
    for s in acquisition_states():
        if dist(s) == 0:
            continue
        out.add(project(s, K))
    return out


# ------------------------------------------------------------------------- B2
# "Uniform-random agent. Required: 0.25 +/- noise."  -- spec 11
# Computed exactly rather than sampled: the answer is analytic.

def run_b2():
    print("\nB2 -- uniform-random agent at the probe")
    total = sum(1.0 / 4 for _ in ALL_R) / len(ALL_R)
    print("    exact expectation over all 24 R: {:.4f}   (required 0.25)".format(total))
    return total


# ------------------------------------------------------------------------- B3
# "Oracle handed R. Required: 1.00."  -- spec 11

def run_b3():
    print("\nB3 -- oracle handed R")
    hits = 0
    for R in ALL_R:
        lab = correct_label(R)
        after = apply_effect(PROBE, R[lab])
        # The probe is solved iff that one press satisfies both dials.
        if after[0] == after[2] and after[1] == after[3]:
            hits += 1
    acc = hits / len(ALL_R)
    print("    {}/{} rule instances solved in one press: {:.4f}   "
          "(required 1.00)".format(hits, len(ALL_R), acc))
    return acc


# ---------------------------------------------------------------- structural

def check_probe_unreachable():
    """Spec section 5: no observation with t2 != 0 is reachable in acquisition."""
    print("\nStructural check -- probe configuration unreachable in acquisition")
    reachable = set(acquisition_states())
    print("    acquisition-reachable observations: {}".format(len(reachable)))
    print("    probe S* = {} in that set: {}".format(
        PROBE, PROBE in reachable))
    return PROBE not in reachable


if __name__ == "__main__":
    print("Q1 backstops, spec frozen at a4877fb\n" + "=" * 68)
    ok_struct = check_probe_unreachable()
    print()
    b1 = run_b1()
    b2 = run_b2()
    b3 = run_b3()

    print("\n" + "=" * 68)
    print("SUMMARY")
    print("    probe unreachable in acquisition : {}".format(
        "PASS" if ok_struct else "FAIL"))
    print("    B1 max lookup-table accuracy     : {:.4f}  {}".format(
        b1, "PASS" if b1 <= 0.25 + 1e-9 else "FAIL -- spec 11 rejects the design"))
    print("    B2 random agent                  : {:.4f}  {}".format(
        b2, "PASS" if abs(b2 - 0.25) < 1e-9 else "FAIL"))
    print("    B3 oracle                        : {:.4f}  {}".format(
        b3, "PASS" if abs(b3 - 1.0) < 1e-9 else "FAIL"))
