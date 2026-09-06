# The router presses keys without regard to whether they work

## How this was found

Fix B was predeclared at 2d03ce6 with a control:

> the effect must be LARGEST on sp80, which writes off the most. If ls20 moves
> more than sp80, the effect is not the one claimed.

The control fired.

| game | condition | key presses | wasted |
|---|---|---|---|
| sp80 | baseline | k1 28, k2 4, k3 1, k4 21 | 28 of 67 (41.8%) |
| sp80 | fix B | **identical, byte for byte** | identical |
| ls20 | baseline | k1 70, k2 65, k3 61, k4 52 | 66 of 248 |
| ls20 | fix B | k1 63, k2 57, k3 **74**, k4 54 | 67 of 248 |

sp80 did not move at all. The diagnosis -- that k1's presses came from the random
fallback re-drawing a written-off control -- is **refuted for sp80**.

## Where the presses actually come from

`RunPlanner.board_replaced` clears `profiles`, `ran`, `_dest`, `crossed_off` and
`blocked`, and does **not** clear `inert` (R9's finding, still true). So k1 is
written off on its first dead press and stays written off.

Every candidate-selection branch excludes `inert`. But selection is not what
presses k1. This runs first, before any of it:

    route = self._route(grid, bg) if (keys and self.moves) else None
    if route is not None:
        self._run_token = route
        return ("key", int(route[1:]))

**The router returns a key directly and consults nothing about that key's
reliability** -- not `inert`, not the profiles, not its own hit rate. sp80 is a
movement game with a populated key->offset map, so the router steers, and it
picks k1 whenever k1's recorded offset points toward the destination.

Census says sp80's ACTION1 is `translate(-4,0)`, then `edit`, then `edit` -- it
delivers its offset roughly a third of the time. ACTION2, 3 and 4 deliver theirs
on all three probes. The router treats all four as equally good.

So k1: 28 presses, 20 of them nothing. k4: 21 presses, 0 wasted. k3: 1 press,
0 wasted.

## What this licenses

A located defect and nothing more. The key->offset map records *what* a key does
and not *how reliably*, and the router consumes it as if the two were the same.
Whether repairing that changes any score is unknown, and fix A's result argues
against expecting much: the keys already had 209 of 251 actions and produced no
level.

No mechanism is licensed. The next intervention, if there is one, has to state
where reliability would be recorded and predeclare a control that can fail --
this record is the second time today a plausible diagnosis of these presses was
wrong.

Status: FIX B REFUTED FOR sp80 BY ITS OWN CONTROL. DEFECT RELOCATED TO `_route`.
