"""Read-only witness: counts and local transitions define different classes."""
import sys
from collections import Counter, defaultdict
from fractions import Fraction

sys.dont_write_bytecode = True
sys.path.insert(0, '/home/aiden/extra/git/AlphaARC/backstops')
from q1_backstops import (ALL_R, PROBE, acquisition_states, project, dist,
                         optimal_labels, apply_effect, correct_label)
from b1_prime import local_data

K = (0, 2)
for representation in ('counts', 'local_transition_tuples'):
    groups = defaultdict(list)
    for rule in ALL_R:
        if representation == 'counts':
            data = local_data(K, rule)
        else:
            data = tuple((s, a, apply_effect(s, rule[a]))
                         for s in acquisition_states()
                         if dist(s) > 0 and project(s, K) == project(PROBE, K)
                         for a in optimal_labels(s, rule))
        groups[data].append(correct_label(rule))
    accuracy = Fraction(sum(max(Counter(labels).values())
                            for labels in groups.values()), len(ALL_R))
    print(representation, 'K={d1,t1}', 'classes=', len(groups), 'maximum=', accuracy)
    assert (len(groups), accuracy) == ((6, Fraction(1, 2)) if representation == 'counts'
                                     else (12, Fraction(1)))
