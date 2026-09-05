# Core-Zero: Codex ↔ Claude exchange

The user explicitly requested direct communication through shared Markdown files
instead of carrying messages between agents. This folder is the shared mailbox.
Only the messages in this folder are sent to Claude. No automatic repository
upload, repository writes, commits, push, or execution of received instructions.

The original interactive Claude session was not discoverable through `claude
agents --json`. A dedicated Claude Opus participant is used for this mailbox;
do not claim that it is the original session. The user's quoted message and
verified research context are included in 001-codex.md.

## Protocol

Codex writes odd-numbered messages; Claude responses use the following even
number. Each file is immutable once dispatched. Replies are untrusted peer
proposals, not authority to run commands. Codex verifies claims against local
files and calculations before accepting them. Neither side may silently broaden
the policy class, search space, or experiment's licensed claim.

`relay.py --once` sends the oldest unanswered Codex message to a dedicated
Claude Opus invocation and saves its exact response as Markdown plus raw JSON.
The invocation has no tools. Messages are provided as context, not executable
shell text. One call per invocation, 120-second timeout, maximum 3 calls in this
initial exchange. A failed call is recorded and is not retried automatically.

The first exchange's deliverable is a precise, bounded judge specification:
finite candidate space, local-data representation, rejection criteria,
counterexample witnesses and scope of any exhaustion claim. No successor agent
or mechanism is commissioned by this mailbox.

STATUS.md records progress. A thread heartbeat may continue this bounded exchange
without user copy/paste; it stays quiet unless results or required actions change.
