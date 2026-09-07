# Every memory in the agent is one value per key, and the key path has no state in its key

## The finding

The agent has memory, and a fair amount of it. What it does not have is a place
where two conflicting observations about the same control can coexist.

Read from `python/alphaarc/planner.py` and `policy.py`:

    self.moves: Dict[str, Tuple[int, int]]       moves[token] = (dr, dc)   overwrite
    self.inert: set[str]                         a control is dead or not, binary
    self.crossed_off: set                        tried or not
    self.blocked: set                            wall or not
    self.tries: Dict[str, int]                   a count
    self.dead: Dict[str, float]                  a decaying count
    self.succ: Dict[Tuple[str, key], List[float]]   succ[...] = list(level)  overwrite

Every store is a **summary**, never a **record**. The second observation replaces
the first, so "this worked once and then failed thirteen times" cannot be held:
by the thirteenth press the first is gone.

## The asymmetry that matters

`succ` is the exception in shape: its key is `(control, state)`. Two world-states
give two keys, so in principle it can hold "k2 in state A" apart from "k2 in
state B". That is the right structure for conditionality.

But `succ` belongs to `Policy`, the **click** path. The key path -- `moves` and
`inert` in `RunPlanner` -- keys on the control name alone. No state, at all.

The door on g50t is a key mechanic. So the half of the agent facing it is the
half whose memory has no state in its key.

And even where the shape is right, it only helps if `_key` captures whatever
changed. If the door is not in those features, both observations land on the same
key and overwrite as before. Rejected #6 is the log of that question being asked
from the other direction.

## Why this is the ceiling and not a bug

`moves[k2] = (6, 0)` is not wrong. k2 does move down six, most of the time. There
is simply nowhere in the type to write "except while the door is shut", so the
thirteen failures at `(34, 16)` are unrepresentable rather than mislearned. They
are recorded as noise because noise is the only category available.

This is a statement about what the agent can express, not about how well it
learns. No amount of experience, budget, or tuning changes it: a `Dict[str,
Tuple[int,int]]` holds one tuple.

## What it does not license

A mechanism. "Add state to the key" is the obvious move and it is exactly what
Rejected #6 already probed once, from the state-free side, and what the position
and body-configuration refutations closed twice more: a finer key partitions the
same data into more cells and loses repeats without buying truth (g50t went from
43 repeats to 13 that way).

So the open question is not *whether* to condition, it is **on what** -- a
variable that separates the door's two states without shattering everything else.
That has now been asked and failed three times with hand-picked variables. The
next attempt should say in advance how it avoids the reach/truth trade, or it
will be the fourth.

Status: EXPRESSIVE CEILING LOCATED. THE OBVIOUS REPAIR HAS ALREADY FAILED THREE
TIMES. NO MECHANISM LICENSED.
