"""E0 evaluator. This module is never imported by the learner."""

from itertools import permutations, product

ACTIONS = (0, 1, 2, 3)
RULES = tuple(permutations(ACTIONS))
ORDERS = tuple(permutations(ACTIONS))


def transition(observation, action, rule, reverse_at_query=False):
    d1, d2, t1, t2 = observation
    effect = rule[action]
    sign = -1 if reverse_at_query and t2 != 0 else 1
    if effect == 0:
        d1 = (d1 + sign) % 4
    elif effect == 1:
        d1 = (d1 - sign) % 4
    elif effect == 2:
        d2 = (d2 + sign) % 4
    elif effect == 3:
        d2 = (d2 - sign) % 4
    else:
        raise ValueError("Invalid effect")
    return d1, d2, t1, t2


def solved(observation):
    return observation[:2] == observation[2:]


def winning_actions(observation, rule, reverse_at_query=False):
    return tuple(a for a in ACTIONS
                 if solved(transition(observation, a, rule, reverse_at_query)))


def queries():
    return tuple(s for s in product(range(4), repeat=4)
                 if s[3] != 0 and len(winning_actions(s, RULES[0])) == 1)


def acquisition(rule, order):
    current = (0, 0, 0, 0)
    trace = []
    for action in order:
        after = transition(current, action, rule)
        trace.append((current, action, after))
        current = after
    return tuple(trace)
