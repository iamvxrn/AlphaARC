"""Exact, seed-free prior-sensitivity audit for dialogue/012."""

from fractions import Fraction


P_AGREE_ONE = Fraction(13, 36)
ACCURACY_ONE = Fraction(4, 13)


def posterior_predictive(shared_prior):
    one_prior = 1 - shared_prior
    numerator = shared_prior + one_prior * P_AGREE_ONE * ACCURACY_ONE
    denominator = shared_prior + one_prior * P_AGREE_ONE
    return numerator / denominator


def main():
    rows = [(weight, posterior_predictive(weight))
            for weight in (Fraction(0), Fraction(1, 2), Fraction(1))]
    for weight, score in rows:
        print("P(H_shared)={}: predictive={}".format(weight, score))

    assert rows == [
        (Fraction(0), Fraction(4, 13)),
        (Fraction(1, 2), Fraction(40, 49)),
        (Fraction(1), Fraction(1)),
    ]
    print("all values are exact; no seed count occurs in this calculation")


if __name__ == "__main__":
    main()
