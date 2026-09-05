from fractions import Fraction


class TransitionLearner:
    """Learns, per action, the modular displacement it applies to the dials.

    An observation is a tuple ``(d1, d2, t1, t2)`` of integers in ``0..3``:
    the two dial readings followed by the two (fixed) target readings.
    """

    _MOD = 4
    _DIALS = 2

    def __init__(self, actions=(0, 1, 2, 3)):
        labels = tuple(actions)
        if len(labels) != 4:
            raise ValueError("exactly four action labels are required")
        try:
            distinct = set(labels)
        except TypeError:
            raise TypeError("action labels must be hashable") from None
        if len(distinct) != 4:
            raise ValueError("action labels must be distinct")
        self.actions = labels
        self._labels = frozenset(distinct)
        self.memory = {}

    @classmethod
    def _validate_observation(cls, observation, name):
        if not isinstance(observation, tuple):
            raise TypeError("%s must be a tuple" % name)
        if len(observation) != 2 * cls._DIALS:
            raise ValueError("%s must have %d entries" % (name, 2 * cls._DIALS))
        for value in observation:
            if isinstance(value, bool) or not isinstance(value, int):
                raise TypeError("%s entries must be integers" % name)
            if not 0 <= value < cls._MOD:
                raise ValueError(
                    "%s entries must lie in 0..%d" % (name, cls._MOD - 1)
                )
        return observation

    def _validate_action(self, action):
        try:
            known = action in self._labels
        except TypeError:
            raise TypeError("action must be hashable") from None
        if not known:
            raise ValueError("unknown action label: %r" % (action,))
        return action

    @classmethod
    def _apply(cls, dials, displacement):
        return tuple(
            (d + s) % cls._MOD for d, s in zip(dials, displacement)
        )

    def observe(self, before, action, after):
        self._validate_observation(before, "before")
        self._validate_observation(after, "after")
        self._validate_action(action)
        if before[self._DIALS:] != after[self._DIALS:]:
            raise ValueError("targets must be identical in before and after")
        self.memory[action] = tuple(
            (a - b) % self._MOD
            for b, a in zip(before[:self._DIALS], after[:self._DIALS])
        )

    def probabilities(self, observation):
        self._validate_observation(observation, "observation")
        dials = observation[:self._DIALS]
        targets = observation[self._DIALS:]

        candidates = [
            action
            for action in self.actions
            if action in self.memory
            and self._apply(dials, self.memory[action]) == targets
        ]
        if not candidates:
            candidates = [
                action for action in self.actions if action not in self.memory
            ]
        if not candidates:
            candidates = list(self.actions)

        share = Fraction(1, len(candidates))
        distribution = {action: Fraction(0) for action in self.actions}
        for action in candidates:
            distribution[action] = share
        return distribution

    def clear(self):
        self.memory.clear()
