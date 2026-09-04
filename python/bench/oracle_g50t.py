"""An agent that KNOWS g50t, so we can read off what knowing it requires.

Not a solver and not a step toward one: a diagnostic. Three destination criteria,
a state-key hypothesis and a body-configuration hypothesis were all guessed and all
refuted (README). So instead of guessing what predicate the general agent is
missing, hand-write an agent that wins and make it declare every fact it consults.
The tally of predicate KINDS is the vocabulary the general mechanism would need.

    python/bench/oracle_g50t.py            # play, print the fact log
"""
from __future__ import annotations
import collections, json, logging, os, sys
from pathlib import Path

KIT = Path(os.environ.get("ARC_KIT",
      "~/ARC-AGI-3-Kaggle-Starter/ARC-AGI-3-Kaggle-Starter")).expanduser()
REPO = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO / "python"))
sys.path.insert(0, str(KIT)); sys.path.insert(0, str(KIT / "vendor" / "ARC-AGI-3-Agents"))
os.chdir(KIT); logging.basicConfig(level=logging.ERROR)

import arc_agi
from arc_agi import OperationMode
from arcengine import GameAction
from agents.agent import Agent
from alphaarc.mdl import _components
from alphaarc.perception import background_color

FACTS: collections.Counter = collections.Counter()
def fact(kind: str, detail: str = "") -> None:
    """Every consultation the oracle makes, by KIND. The kinds are the vocabulary."""
    FACTS[kind] += 1
    if FACTS[kind] == 1:
        print(f"    [fact] {kind}: {detail}")

class Probe(Agent):
    def is_done(self, f, l): return True
    def choose_action(self, f, l): raise NotImplementedError

def grid_of(fr):
    return [r[:] for r in fr.frame[-1]] if (fr.frame and fr.frame[-1]) else None

def avatar(grid, bg):
    if not grid: return None
    """The controlled body: the 24-cell colour-9 component. KNOWN, not derived."""
    fact("identity_of_controlled_body", "the 24-cell colour-9 component is me")
    best = [cl for col, cl in _components(grid, bg) if col == 9 and len(cl) == 24]
    if not best: return None
    cl = best[0]
    return (sum(c[0] for c in cl)//len(cl), sum(c[1] for c in cl)//len(cl))

GOAL = (52, 46)          # inside the bracket at rows 49-55, cols 43-49
def at_goal(pos):
    fact("goal_region_membership", f"am I inside rows 49-55 x cols 43-49 -> {GOAL}")
    return 49 <= pos[0] <= 55 and 43 <= pos[1] <= 49

WALKABLE = {5, 9, 8}       # corridor, my own colour, and colour 8 -- is it a barrier?
def walkable(grid, pos):
    """Can my 5x5 body stand centred here? Read off the board, not learned by
    bumping into it -- an oracle is allowed to look."""
    fact("walkability_read_from_the_board", "my footprint must land on corridor")
    r, c = pos
    if not (2 <= r < 62 and 2 <= c < 62): return False
    for y in range(r-2, r+3):
        for x in range(c-2, c+3):
            if grid[y][x] not in WALKABLE: return False
    return True

def route(pos, offsets, blocked, grid=None):
    """Shortest key sequence over the lattice the avatar's own offsets define."""
    fact("reachability_over_learned_offsets", "BFS using key->(dr,dc) and known walls")
    seen = {pos}; q = collections.deque([(pos, None)])
    while q:
        cur, first = q.popleft()
        for k, (dr, dc) in sorted(offsets.items()):
            nxt = (cur[0]+dr, cur[1]+dc)
            if nxt in seen or not (0 <= nxt[0] < 64 and 0 <= nxt[1] < 64): continue
            if grid is not None and not walkable(grid, nxt): continue
            step = first if first is not None else k
            if 49 <= nxt[0] <= 55 and 43 <= nxt[1] <= 49: return step
            seen.add(nxt); q.append((nxt, step))
    return None

def main():
    arc = arc_agi.Arcade(operation_mode=OperationMode.NORMAL)
    ag = Probe(card_id="oracle", game_id="g50t", agent_name="oracle.g50t",
               ROOT_URL="http://localhost", record=False,
               arc_env=arc.make("g50t"), tags=["oracle"])
    fr = ag.take_action(GameAction.RESET)
    grid = grid_of(fr); bg = background_color(grid)
    simple = [a for a in GameAction
              if a is not GameAction.RESET and not a.is_complex()
              and a.value in (fr.available_actions or [])]
    print(f"g50t: background={bg}, {len(simple)} simple actions, baseline L1 = 78 actions\n")

    pos = avatar(grid, bg)
    print(f"  avatar at {pos}, goal {GOAL}")
    print("  -- learning controls WHILE navigating: a key that does nothing HERE")
    print("     is retried elsewhere, which is exactly what the agent refuses to do --\n")

    trail = []
    offsets: dict = {}                      # key -> (dr, dc), learned
    unknown = [a for a in simple]           # keys whose effect is not yet known
    blocked, levels_done, actions, tries = set(), 0, 0, 0
    order = 0
    for step in range(360):
        if pos is None: break
        if at_goal(pos):
            fired = False
            for a in simple:
                fr = ag.take_action(a); actions += 1
                fact("act_once_arrived", f"press {a.name} at the destination")
                grid = grid_of(fr)
                done = int(getattr(fr, "levels_completed", 0) or 0)
                if done > levels_done:
                    print(f"    LEVEL {done} CLEARED at {actions} actions "
                          f"(baseline 78) by {a.name} at {pos}")
                    levels_done = done; blocked.clear(); offsets.clear()
                    unknown = [x for x in simple]; fired = True
                    pos = avatar(grid, bg); break
            if levels_done >= 2 or not fired: break
            continue

        k = route(pos, offsets, blocked, grid) if len(offsets) >= 4 else None
        if k is not None and (pos, k) in blocked:
            blocked.discard((pos, k))      # the board says walkable; trust the board
        if k is None:
            # stuck with what we know: try a key whose effect here is unknown.
            cand = [a for a in simple
                    if (pos, a.value) not in blocked and a.value not in offsets]
            if not cand:
                cand = [a for a in simple if (pos, a.value) not in blocked]
            if not cand:
                blocked = {b for b in blocked if b[0] != pos}
                cand = list(simple)
            order = (order + 1) % len(cand)
            act = cand[order]
            fact("retry_a_control_that_was_dead_elsewhere", f"{act.name} at {pos}")
        else:
            act = next(a for a in simple if a.value == k)
        before = pos
        fr = ag.take_action(act); actions += 1
        grid = grid_of(fr)
        if grid: bg = background_color(grid)
        now = avatar(grid, bg)
        done = int(getattr(fr, "levels_completed", 0) or 0)
        if done > levels_done:
            levels_done = done; blocked.clear(); offsets.clear()
            print(f"    LEVEL {done} CLEARED at {actions} actions while moving")
            pos = avatar(grid, bg); continue
        if now is None: break
        if now != before:
            d = (now[0]-before[0], now[1]-before[1])
            if act.value not in offsets:
                offsets[act.value] = d
                fact("effect_of_key_on_my_position", f"{act.name} -> {d}")
                print(f"    learned {act.name} -> {d} (at {before})")
        else:
            fact("wall_learned_from_a_move_that_did_nothing", f"{before} via {act.name}")
            blocked.add((before, act.value))
        pos = now
        trail.append(pos)
    import collections as _c
    print(f"\n  trail: {len(set(trail))} distinct positions, last {trail[-1] if trail else None}")
    print(f"  most visited: {_c.Counter(trail).most_common(6)}")
    rows=sorted({p[0] for p in trail}); cols=sorted({p[1] for p in trail})
    print(f"  rows reached {rows}")
    print(f"  cols reached {cols}")
    print(f"\n  result: {levels_done} level(s), {actions} actions (baseline 78)")
    print("\n=== the vocabulary this required ===")
    for k, n in FACTS.most_common():
        print(f"  {n:5}  {k}")

main()
