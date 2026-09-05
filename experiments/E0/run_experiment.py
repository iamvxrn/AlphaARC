"""Run E0 with exact probabilities. No dependencies outside Python's library."""

import argparse
import copy
import hashlib
import json
import platform
import sys
import time
from datetime import datetime, timezone
from fractions import Fraction
from pathlib import Path

from environment import ACTIONS, RULES, ORDERS, acquisition, queries, winning_actions
from learner import TransitionLearner

ROOT = Path(__file__).resolve().parent


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def exact_record(total, count, expected):
    value = Fraction(total, count)
    return {"accuracy_exact": str(value), "accuracy": float(value),
            "expected_probability_mass": str(total), "query_cases": count,
            "expected_exact": str(expected), "passed": value == expected}


def run():
    started = time.perf_counter()
    frozen = json.loads((ROOT / "FREEZE.json").read_text())
    if sha256(ROOT / "SPEC.md") != frozen["spec_sha256"]:
        raise RuntimeError("Frozen specification digest changed; refusing execution")
    qs = queries()
    assert len(qs) == 48
    # These tables belong exclusively to the evaluator. Never pass them to learner.
    win = {(r, s): winning_actions(s, r)[0] for r in RULES for s in qs}
    reverse_win = {(r, s): winning_actions(s, r, True)[0] for r in RULES for s in qs}
    matched = [Fraction(0) for _ in range(5)]
    independent = [Fraction(0) for _ in range(5)]
    totals = {name: Fraction(0) for name in
              ("erased", "exact_observation_lookup", "axis_only", "oracle",
               "excluded_identical", "stationarity_violation", "violation_oracle")}
    pair_gains = []
    # Cache probability mass across evaluator rules, not model state.
    # Looping over every rule remains explicit for transparent case counts.
    for old_rule in RULES:
        for order in ORDERS:
            trace = acquisition(old_rule, order)
            learner = TransitionLearner()
            for k in range(5):
                memory_before = copy.deepcopy(learner.__dict__)
                for s in qs:
                    p = learner.probabilities(s)
                    assert set(p) == set(ACTIONS) and sum(p.values()) == 1
                    hit = p[win[old_rule, s]]
                    matched[k] += hit
                    independent_mass = sum(p[win[new_rule, s]] for new_rule in RULES)
                    independent[k] += independent_mass
                    if k == 4:
                        pair_gains.append(hit - independent_mass / 24)
                        totals["excluded_identical"] += sum(
                            p[win[new_rule, s]] for new_rule in RULES if new_rule != old_rule)
                        totals["stationarity_violation"] += p[reverse_win[old_rule, s]]
                        totals["oracle"] += 1
                        totals["violation_oracle"] += 1
                assert learner.__dict__ == memory_before, "Query mutated acquired state"
                if k < 4:
                    learner.observe(*trace[k])

            # Build controls from actual acquisition transitions.
            exact_table = {}
            axes = {}
            for before, action, after in trace:
                exact_table.setdefault(before, {})[action] = after
                changed = [i for i in (0, 1) if before[i] != after[i]]
                assert len(changed) == 1
                axes[action] = changed[0]
            learner.clear()
            for s in qs:
                correct = win[old_rule, s]
                totals["erased"] += learner.probabilities(s)[correct]
                assert s not in exact_table, "Exact lookup encountered a known query"
                lookup_p = {a: Fraction(1, 4) for a in ACTIONS}
                totals["exact_observation_lookup"] += lookup_p[correct]
                required_axis = next(i for i in (0, 1) if s[i] != s[i + 2])
                candidates = [a for a in ACTIONS if axes[a] == required_axis]
                assert len(candidates) == 2
                totals["axis_only"] += Fraction(int(correct in candidates), len(candidates))

    count = len(RULES) * len(ORDERS) * len(qs)
    expected_curve = (Fraction(1, 4), Fraction(1, 2), Fraction(3, 4), Fraction(1), Fraction(1))
    curve = [{"transitions": k,
              "matched": exact_record(matched[k], count, expected_curve[k]),
              "independent": exact_record(independent[k], count * 24, Fraction(1, 4))}
             for k in range(5)]
    expectations = {"erased": Fraction(1, 4), "exact_observation_lookup": Fraction(1, 4),
                    "axis_only": Fraction(1, 2), "oracle": Fraction(1),
                    "excluded_identical": Fraction(5, 23), "stationarity_violation": Fraction(0),
                    "violation_oracle": Fraction(1)}
    controls = {name: exact_record(total, count * (23 if name == "excluded_identical" else 1),
                                  expectations[name]) for name, total in totals.items()}
    checks = [r[arm]["passed"] for r in curve for arm in ("matched", "independent")]
    checks += [r["passed"] for r in controls.values()]
    provenance = {name: sha256(ROOT / name) for name in
                  ("SPEC.md", "FREEZE.json", "learner.py", "environment.py",
                   "run_experiment.py", "test_reference.py")}
    return {
        "experiment": "E0 transition-acquisition reference",
        "status": "PASS" if all(checks) else "FAILURE_FROZEN",
        "completed_utc": datetime.now(timezone.utc).isoformat(),
        "python": platform.python_version(), "elapsed_seconds": time.perf_counter() - started,
        "enumeration": {"rules": 24, "orders": 24, "queries": 48,
                        "matched_cases_per_budget": count,
                        "independent_cases_per_budget": count * 24,
                        "unique_learner_predictions_across_budgets": count * 5,
                        "independent_control_uses_same_prediction_for_24_rules": True,
                        "seeds": 0, "sampling_noise": 0,
                        "qualification": "Finite-population exact expectations, not independent sampled trials"},
        "learning_curve": curve, "controls": controls,
        "paired_full_acquisition_gain": {"mean_exact": str(sum(pair_gains) / len(pair_gains)),
                                        "min_exact": str(min(pair_gains)),
                                        "max_exact": str(max(pair_gains)),
                                        "matched_cases": count,
                                        "qualification": "Each matched case paired with its exact mean over 24 independent query rules"},
        "provenance_sha256": provenance,
        "claim_limit": "Engineering reference under known stationary modular-displacement prior; not minimal K0 or AGI",
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path, default=ROOT / "results.json")
    args = parser.parse_args()
    if args.output.exists():
        parser.error("Output already exists; choose another --output to preserve previous runs")
    result = run()
    args.output.write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps(result, indent=2))
    return 0 if result["status"] == "PASS" else 1


if __name__ == "__main__":
    sys.exit(main())
