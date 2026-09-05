"""Boundary, invariance and independent scoring checks for E0."""

import ast
import copy
import unittest
from collections import Counter
from fractions import Fraction
from pathlib import Path

from environment import ACTIONS, RULES, ORDERS, acquisition, queries, transition, winning_actions
from learner import TransitionLearner


def distance_oracle(s, rule, reverse=False):
    # Independent formula, not a call to the transition implementation.
    vectors = ((1, 0), (3, 0), (0, 1), (0, 3))
    out = []
    for action, effect in enumerate(rule):
        v = vectors[effect]
        sign = -1 if reverse and s[3] else 1
        gaps = [((s[i] + sign * v[i]) - s[i + 2]) % 4 for i in (0, 1)]
        if sum(min(g, 4 - g) for g in gaps) == 0:
            out.append(action)
    return tuple(out)


class ReferenceTests(unittest.TestCase):
    def test_query_design_and_independent_oracle(self):
        qs = queries()
        self.assertEqual(len(qs), 48)
        self.assertTrue(all(s[3] != 0 for s in qs))
        for rule in RULES:
            counts = Counter()
            for s in qs:
                for reverse in (False, True):
                    winners = winning_actions(s, rule, reverse)
                    self.assertEqual(winners, distance_oracle(s, rule, reverse))
                    self.assertEqual(len(winners), 1)
                counts[rule[winning_actions(s, rule)[0]]] += 1
            self.assertEqual(counts, {e: 12 for e in ACTIONS})

    def test_all_acquisition_orders_and_novelty(self):
        qs = set(queries())
        for rule in RULES:
            for order in ORDERS:
                trace = acquisition(rule, order)
                self.assertEqual(len(trace), 4)
                self.assertEqual(set(a for _, a, _ in trace), set(ACTIONS))
                for before, _, after in trace:
                    self.assertEqual(before[2:], (0, 0))
                    self.assertEqual(after[2:], (0, 0))
                    self.assertNotIn(before, qs)
                    self.assertNotIn(after, qs)
                self.assertEqual(trace[-1][2], (0, 0, 0, 0))

    def test_opaque_label_equivariance(self):
        names = ("alpha", "z!", ("opaque", 7), 91)
        for rule in RULES:
            original, renamed = TransitionLearner(), TransitionLearner(names)
            trace = acquisition(rule, ACTIONS)
            for k in range(5):
                for s in queries():
                    p, renamed_p = original.probabilities(s), renamed.probabilities(s)
                    self.assertEqual(p, {i: renamed_p[names[i]] for i in ACTIONS})
                if k < 4:
                    before, action, after = trace[k]
                    original.observe(before, action, after)
                    renamed.observe(before, names[action], after)

    def test_queries_are_read_only_and_erasure_works(self):
        learner = TransitionLearner()
        for event in acquisition(RULES[7], ORDERS[13]):
            learner.observe(*event)
        frozen = copy.deepcopy(learner.__dict__)
        for s in queries():
            p = learner.probabilities(s)
            self.assertEqual(sum(p.values()), 1)
            self.assertTrue(all(isinstance(x, Fraction) and x >= 0 for x in p.values()))
        self.assertEqual(learner.__dict__, frozen)
        learner.clear()
        self.assertFalse(learner.memory)
        for s in queries():
            self.assertEqual(learner.probabilities(s), {a: Fraction(1, 4) for a in ACTIONS})

    def test_errors_at_interface(self):
        learner = TransitionLearner()
        with self.assertRaises(ValueError):
            TransitionLearner((0, 0, 1, 2))
        for s in ((0, 0), (0, 4, 0, 0), (0, 0, -1, 0), (0.0, 0, 0, 0)):
            with self.assertRaises((ValueError, TypeError)):
                learner.probabilities(s)
        with self.assertRaises((ValueError, KeyError)):
            learner.observe((0, 0, 0, 0), 9, (1, 0, 0, 0))
        with self.assertRaises(ValueError):
            learner.observe((0, 0, 0, 0), 0, (1, 0, 0, 1))

    def test_no_environment_import_or_hidden_inputs(self):
        import inspect
        self.assertEqual(tuple(inspect.signature(TransitionLearner.observe).parameters),
                         ("self", "before", "action", "after"))
        self.assertEqual(tuple(inspect.signature(TransitionLearner.probabilities).parameters),
                         ("self", "observation"))
        tree = ast.parse(Path(__file__).with_name("learner.py").read_text())
        for node in ast.walk(tree):
            if isinstance(node, ast.ImportFrom):
                self.assertEqual(node.module, "fractions")
            elif isinstance(node, ast.Import):
                self.assertTrue(all(alias.name == "fractions" for alias in node.names))

    def test_broken_learners_do_not_pass(self):
        # An explicit intervention: a no-op U leaves the prior uniform.
        no_op_score = sum(TransitionLearner().probabilities(s)[winning_actions(s, r)[0]]
                          for r in RULES for s in queries()) / (24 * 48)
        self.assertEqual(no_op_score, Fraction(1, 4))
        # Deliberately corrupt direction in acquired memory; evaluator must reject it.
        score = 0
        for rule in RULES:
            learner = TransitionLearner()
            for event in acquisition(rule, ACTIONS):
                learner.observe(*event)
            learner.memory = {a: tuple((-x) % 4 for x in d)
                              for a, d in learner.memory.items()}
            score += sum(learner.probabilities(s)[winning_actions(s, rule)[0]]
                         for s in queries())
        self.assertEqual(score, 0)


if __name__ == "__main__":
    unittest.main(verbosity=2)
