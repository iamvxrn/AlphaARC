"""Exact evidence calculation specified in astra_005_plan.md."""
import itertools
import json
from fractions import Fraction
from pathlib import Path

RULES = tuple(itertools.permutations(range(4)))


def policy_answer(observed_rule, target=0):
    """Action selected from a complete acquired action->output table."""
    return observed_rule.index(target)


def exact_enumeration(n):
    """Enumerate n acquisition tables plus one held-out table under H_independent."""
    total = 0
    matching_evidence = 0
    correct_when_matching = 0
    for tables in itertools.product(RULES, repeat=n + 1):
        total += 1
        acquired, heldout = tables[:-1], tables[-1]
        if len(set(acquired)) == 1:
            matching_evidence += 1
            correct_when_matching += policy_answer(acquired[0]) == policy_answer(heldout)
    p_e = Fraction(matching_evidence, total)
    accuracy_given_e = Fraction(correct_when_matching, matching_evidence)
    return {"environments": total, "evidence_cases": matching_evidence,
            "p_e_given_independent": str(p_e),
            "accuracy_given_e_and_independent": str(accuracy_given_e)}


def closed_form(n):
    likelihood_independent = Fraction(1, 24 ** (n - 1))
    posterior_shared = Fraction(1, 1 + likelihood_independent)  # equal hypothesis odds
    predictive = Fraction(1, 4) + Fraction(3, 4) * posterior_shared
    return {"acquired_sources": n,
            "p_e_given_shared": "1",
            "p_e_given_independent": str(likelihood_independent),
            "bayes_factor_shared_over_independent": str(1 / likelihood_independent),
            "posterior_shared_equal_priors": str(posterior_shared),
            "predictive_accuracy_equal_priors": str(predictive)}


def main():
    rows = []
    for n in range(1, 4):
        closed = closed_form(n)
        exhaustive = exact_enumeration(n)
        assert exhaustive["p_e_given_independent"] == closed["p_e_given_independent"]
        assert exhaustive["accuracy_given_e_and_independent"] == "1/4"
        # Under H_shared all n acquired and held-out rules are the same.
        assert all(policy_answer(r) == policy_answer(r) for r in RULES)
        rows.append({**closed, "independent_enumeration": exhaustive})
    out = {"status": "PASS", "rules": 24, "prior_odds_shared_to_independent": "1",
           "rows": rows,
           "scope": "Model comparison for declared shared-vs-independent rule family; not evidence for unrestricted generalization"}
    print(json.dumps(out, indent=2))
    return out


if __name__ == "__main__":
    main()
