"""Flatten `python/alphaarc/` into the single self-contained agent file Kaggle needs.

The competition notebook builder splices ONLY `agent/my_agent.py`, so the
submission must be one file with no package imports. Hand-inlining the core on
every change is how the repo copy and the submitted copy silently diverge -- and
the submitted copy is the only one that scores. This makes the package canonical
and the agent file a build product.

    python python/bench/bundle.py                 # -> <kit>/agent/my_agent.py
    python python/bench/bundle.py --out /tmp/a.py --check

Modules are emitted in dependency order; intra-package imports are dropped and
stdlib imports are hoisted and de-duplicated at the top.
"""
from __future__ import annotations

import argparse
import ast
import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
PKG = REPO / "python" / "alphaarc"
DEFAULT_KIT = Path.home() / "ARC-AGI-3-Kaggle-Starter" / "ARC-AGI-3-Kaggle-Starter"

# The adapter to the live engine goes last: it defines MyAgent over everything above.
GLUE = "agent_glue"


def module_deps(path: Path) -> set[str]:
    """Names of sibling package modules this module imports."""
    tree = ast.parse(path.read_text())
    deps: set[str] = set()
    for node in ast.walk(tree):
        if isinstance(node, ast.ImportFrom) and node.level == 1 and node.module:
            deps.add(node.module)
        elif isinstance(node, ast.ImportFrom) and node.level == 1 and node.module is None:
            deps.update(a.name for a in node.names)  # from . import x
    return deps


def order_modules(mods: dict[str, Path]) -> list[str]:
    """Topological order; deterministic (alphabetical) among independent modules."""
    deps = {m: {d for d in module_deps(p) if d in mods} for m, p in mods.items()}
    out: list[str] = []
    while deps:
        ready = sorted(m for m, d in deps.items() if not (d - set(out)))
        if not ready:
            raise SystemExit(f"import cycle among {sorted(deps)}")
        out.extend(ready)
        for m in ready:
            deps.pop(m)
    # glue last regardless of what it happens to import
    if GLUE in out:
        out.append(out.pop(out.index(GLUE)))
    return out


def strip_module(src: str) -> tuple[list[str], str]:
    """Return (hoisted stdlib import lines, body with package imports removed)."""
    tree = ast.parse(src)
    lines = src.splitlines()
    drop: set[int] = set()
    hoist: list[str] = []
    for node in tree.body:
        if isinstance(node, (ast.Import, ast.ImportFrom)):
            seg = "\n".join(lines[node.lineno - 1:node.end_lineno])
            for i in range(node.lineno - 1, node.end_lineno):
                drop.add(i)
            if isinstance(node, ast.ImportFrom) and node.level:
                continue  # intra-package: the code is inlined right here
            if isinstance(node, ast.ImportFrom) and node.module == "__future__":
                continue  # emitted once, at the very top
            hoist.append(seg)
        elif isinstance(node, ast.Expr) and isinstance(node.value, ast.Constant) \
                and isinstance(node.value.value, str) and node is tree.body[0]:
            for i in range(node.lineno - 1, node.end_lineno):
                drop.add(i)  # module docstring -> replaced by a section banner
    body = "\n".join(l for i, l in enumerate(lines) if i not in drop)
    return hoist, body.strip("\n")


def build() -> str:
    mods = {p.stem: p for p in sorted(PKG.glob("*.py")) if p.stem != "__init__"}
    if GLUE not in mods:
        raise SystemExit(f"missing {PKG / (GLUE + '.py')}: the engine adapter that defines MyAgent")
    order = order_modules(mods)

    hoisted: list[str] = []
    sections: list[str] = []
    for m in order:
        hoist, body = strip_module(mods[m].read_text())
        for h in hoist:
            if h not in hoisted:
                hoisted.append(h)
        sections.append(f"# {'=' * 24} {m}.py {'=' * 24}\n\n{body}\n")

    header = (
        '"""AlphaARC submission agent -- GENERATED FILE, DO NOT EDIT.\n\n'
        "Built by `python/bench/bundle.py` from python/alphaarc/{"
        + ", ".join(order) + "}.py.\n"
        "Edit the package modules and re-run the bundler; edits here are lost.\n"
        '"""\n\nfrom __future__ import annotations\n'
    )
    return header + "\n" + "\n".join(sorted(hoisted)) + "\n\n\n" + "\n\n".join(sections)


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--kit", type=Path, default=DEFAULT_KIT)
    ap.add_argument("--out", type=Path, default=None,
                    help="output path (default: <kit>/agent/my_agent.py)")
    ap.add_argument("--check", action="store_true",
                    help="fail if the output is stale instead of writing it")
    args = ap.parse_args()

    out = args.out or (args.kit.expanduser() / "agent" / "my_agent.py")
    text = build()
    ast.parse(text)  # a bundle that doesn't parse never reaches a game

    if args.check:
        current = out.read_text() if out.exists() else ""
        if current != text:
            print(f"STALE: {out} does not match python/alphaarc/ -- run bundle.py", file=sys.stderr)
            raise SystemExit(1)
        print(f"up to date: {out}")
        return

    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(text)
    n_defs = len(re.findall(r"^(?:class|def) ", text, re.M))
    print(f"wrote {out}  ({len(text.splitlines())} lines, {n_defs} top-level defs)")


if __name__ == "__main__":
    main()
