"""The census classifier decides which world-model work we do next, so its labels
have to be right. Run directly (no pytest)."""

import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(HERE, "..", "bench"))
sys.path.insert(0, os.path.join(HERE, ".."))

from census import (  # noqa: E402
    changed_cells, classify, probe_points, rigid_translation, summarize,
)

BG = 0


def blank(h=10, w=10):
    return [[BG] * w for _ in range(h)]


def test_noop_and_change_detection():
    g = blank()
    assert changed_cells(g, g) == []
    h = blank()
    h[2][3] = 5
    assert changed_cells(g, h) == [(2, 3)]
    assert classify(g, g, BG) == "noop"


def test_rigid_translation_is_recognised():
    a, b = blank(), blank()
    for r, c in ((1, 1), (1, 2), (2, 1)):
        a[r][c] = 4
        b[r + 3][c + 1] = 4
    assert rigid_translation(a, b, BG) == (4, 3, 1)
    assert classify(a, b, BG) == "translate(3,1)"


def test_two_bodies_moving_is_not_a_rigid_translation():
    """A world model that assumes one avatar must NOT be told this is one."""
    a, b = blank(), blank()
    a[1][1], a[5][5] = 4, 7
    b[2][1], b[5][6] = 4, 7
    assert rigid_translation(a, b, BG) is None


def test_click_locality_separates_local_edits_from_buttons():
    a = blank()
    near = [row[:] for row in a]
    near[4][4] = 3
    assert classify(a, near, BG, click=(4, 4)) == "local_edit"      # click is (col,row)
    far = [row[:] for row in a]
    far[9][9] = 3
    assert classify(a, far, BG, click=(0, 0)) == "remote_edit"


def test_global_repaint_beats_the_other_labels():
    a = blank(10, 10)
    b = [[7] * 10 for _ in range(10)]
    assert classify(a, b, BG) == "global"


def test_probe_points_put_the_smallest_object_first():
    """Interactive controls are usually small; the sweep must not bury them."""
    g = blank(16, 16)
    for r in range(1, 6):
        for c in range(1, 6):
            g[r][c] = 2          # a big blob
    g[12][12] = 3                # a one-cell button
    pts = probe_points(g, BG, 20)
    assert pts[0] == (12, 12), pts[:3]


def test_summarize_flags_a_stateful_action():
    """Same action, different effect on repeat -> a mode the model must carry."""
    game = {"simple_actions": {"ACTION1": ["translate(0,1)", "global", "translate(0,1)"]},
            "click_labels": {}, "toggle": None}
    s = summarize(game)
    assert s["varies"] is True
    assert "stateful-mode" in s["classes"]


def test_summarize_maps_labels_to_model_classes():
    game = {"simple_actions": {"ACTION1": ["translate(1,0)", "translate(1,0)"]},
            "click_labels": {"remote_edit": 3, "noop": 20}, "toggle": "toggle"}
    s = summarize(game)
    assert set(s["classes"]) == {"object-translation", "click-remote-effect", "cell-toggle"}


if __name__ == "__main__":
    failures = 0
    for name, fn in sorted(globals().items()):
        if name.startswith("test_") and callable(fn):
            try:
                fn()
                print(f"ok   {name}")
            except AssertionError as e:
                failures += 1
                print(f"FAIL {name}: {e}")
    print("PASS" if not failures else f"{failures} FAILURE(S)")
    sys.exit(1 if failures else 0)
