"""Exact audit for dialogue/008: a pre-split exchangeable alternative.

The held-out index J is selected independently after H_one has selected its
exceptional source E.  The observed tables are all sources other than J.
No sampling and no learner implementation are involved.
"""

import itertools
import json
from fractions import Fraction
from pathlib import Path


RULES = tuple(itertools.permutations(range(4)))


def audit_one_exception(source_count):
    """Enumerate uniform (J, E, R, S), conditioning on agreeing observations."""
    total = evidence = exceptional_probe = hits = 0
    for held_out, exceptional, common, alternate in itertools.product(
        range(source_count), range(source_count), RULES, RULES
    ):
        total += 1
        tables = tuple(
            alternate if source == exceptional else common
            for source in range(source_count)
        )
        observed = tuple(tables[source] for source in range(source_count)
                         if source != held_out)
        agrees = len(set(observed)) == 1
        if not agrees:
            continue
        evidence += 1
        exceptional_probe += held_out == exceptional
        # The policy uses the one observed table and asks for output 0.
        chosen_label = observed[0].index(0)
        hits += tables[held_out][chosen_label] == 0

    return {
        "source_count": source_count,
        "outcomes": total,
        "evidence_outcomes": evidence,
        "p_agree": Fraction(evidence, total),
        "p_exceptional_probe_given_agree": Fraction(exceptional_probe, evidence),
        "accuracy_given_agree": Fraction(hits, evidence),
    }


def closed_form(source_count):
    if source_count < 3:
        raise ValueError("agreement is vacuous with fewer than two observed sources")
    n = source_count
    return {
        "p_agree": Fraction(n + 23, 24 * n),
        "p_exceptional_probe_given_agree": Fraction(24, n + 23),
        "accuracy_given_agree": Fraction(n + 5, n + 23),
        "shared_over_one_bayes_factor": Fraction(24 * n, n + 23),
    }


def serialise(row):
    return {key: str(value) if isinstance(value, Fraction) else value
            for key, value in row.items()}


def main():
    rows = []
    for n in (3, 4, 5):
        measured = audit_one_exception(n)
        expected = closed_form(n)
        for key in ("p_agree", "p_exceptional_probe_given_agree",
                    "accuracy_given_agree"):
            assert measured[key] == expected[key], (n, key, measured[key], expected[key])
        measured["shared_over_one_bayes_factor"] = expected["shared_over_one_bayes_factor"]
        rows.append(measured)

    result = {"model": "one pre-split exchangeable exceptional source", "rows": rows}
    result_path = Path(__file__).with_name("astra_008_split_audit_results.json")
    result_path.write_text(json.dumps(
        {"model": result["model"], "rows": [serialise(row) for row in rows]},
        indent=2) + "\n"
    )
    for row in rows:
        print(
            "N={source_count}: P(A)={p_agree}; P(E=J|A)="
            "{p_exceptional_probe_given_agree}; accuracy={accuracy_given_agree}; "
            "BF(shared/one)={shared_over_one_bayes_factor}".format(**row)
        )
    print("exact enumeration agrees with closed forms")


if __name__ == "__main__":
    main()
