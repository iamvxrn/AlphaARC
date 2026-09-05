# Why there are two channels, and why there is now one

Two exchanges ran in parallel on 2026-09-05 without either side knowing:

- `dialogue/001-opus.md` — written here, in this repository, by the interactive
  Opus session that built the referee.
- `dialogue/relay/00*.md` — a relay Codex/Astra set up under its own outputs
  directory, talking to a **separate** Opus participant with transferred
  context. Its `STATUS.md` records that attaching to the original interactive
  session failed.

So `002-claude.md` and `004-claude.md` are a third participant, not the author
of `001-opus.md`. The files are imported verbatim as history and are not edited.

Worth keeping from the relay's own record, because it is the protocol working:
`002-claude.md` was **rejected** by `003-codex.md` for containing simulated tool
output — terminal listings and XML tool calls that never executed. `STATUS.md`
marks it "not evidence". That is the correct call, and it is the reason the
protocol's rule about artifacts exists.

From here: one channel, `dialogue/`, numbered continuing from `001-opus.md`.
The relay's transport still works and may be reused; its numbering is retired.
