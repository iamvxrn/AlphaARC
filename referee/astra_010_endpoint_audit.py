"""Independent endpoint audit for dialogue/010.

Exact enumeration of the already-declared exactly-d-exception model, including
the d=N endpoint.  No sampling, agent, or new environment is involved.
"""

import itertools
from fractions import Fraction


RULES = tuple(itertools.permutations(range(4)))


def measure(source_count, exception_count):
    total = agrees = hits = 0
    for held_out in range(source_count):
        for exceptional_sources in itertools.combinations(range(source_count), exception_count):
            for common in RULES:
                for exceptional_rules in itertools.product(RULES, repeat=exception_count):
                    total += 1
                    tables = [common] * source_count
                    for source, rule in zip(exceptional_sources, exceptional_rules):
                        tables[source] = rule
                    observed = [tables[i] for i in range(source_count) if i != held_out]
                    if len(set(observed)) != 1:
                        continue
                    agrees += 1
                    label = observed[0].index(0)
                    hits += tables[held_out][label] == 0
    return Fraction(agrees, total), Fraction(hits, agrees)


def predictive(probability_of_agreement, accuracy):
    # Equal prior weights for shared and the exactly-d alternative.
    return (1 + probability_of_agreement * accuracy) / (1 + probability_of_agreement)


def main():
    rows = []
    for d in (1, 2, 3):
        agreement, accuracy = measure(3, d)
        score = predictive(agreement, accuracy)
        rows.append((d, agreement, accuracy, score))
        print("N=3 d={}: P(A)={}; acc|A={}; predictive={}".format(
            d, agreement, accuracy, score
        ))

    assert rows[1][3] == rows[2][3], rows
    print("planned endpoint reversal rejected: predictive(d=2) = predictive(d=3)")


if __name__ == "__main__":
    main()
