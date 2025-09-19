# Goose – Multi-Agent Dev Runner (tmux + git worktrees + AI panes)

## Purpose
Produce a single Go binary named goose that:
- boots a tmux session with multiple windows/panes for agents,
- uses git worktree per agent for isolated feature work,
- runs AI assistants (Codex) in selected panes,
- coordinates work by reading and writing Markdown handoffs (inbox and outbox) in the dev branch,
- supports “radio” control via tmux send-keys.

## Non-goals
- No GUI and no long-lived background daemons.
- No complex message buses; file-based messaging only.
- No deploy/release orchestration (lives outside this tool).

## Agents and Roles
- Maverick (lead): assigns work, reviews, merges agent branches into dev, writes tasks to handoffs/inbox.md, reads handoffs/outbox/*; does not code.
- Goose (coder): implements tasks in feature branches (per task) in its own worktree, commits and pushes, reports status to outbox.
- Controller (optional): a simple broadcaster that issues tmux commands (for example pull and test) to agent panes.

## Must-have features
1. Session: "goose session start" creates a tmux session with:
   - window "lead" (Maverick): left editor, right shell (optional Codex);
   - window "goose" (coder): left editor, right shell (Codex on demand);
   - optional window "ops": inbox and outbox watcher.
2. Worktrees: "goose agent add --name goose" creates personas/goose worktree on agent/goose (or per-feature worktrees).
3. Handoffs: "goose handoff open, ack, progress, done" appends structured entries to:
   - handoffs/inbox.md (single shared file in dev),
   - handoffs/outbox/GOOSE.md and handoffs/outbox/MAVERICK.md.
4. Radio: "goose radio send --target window.pane -- command" uses tmux send-keys.
   - Broadcast helper: "goose radio all --agents list --pane N -- command".
5. AI launch: flags to start Codex in a pane, for example --ai-lead "codex --cd . --full-auto -m gpt-5".
6. Safety: detect missing tools, readable errors, and a dry-run mode with --dry.
7. Help: "goose --help" and per-command help.

## Two orchestration modes
- Scripted leadership: Maverick pane runs a simple loop that reads inbox and emits send-keys to agents at intervals.
- Human-driven leadership: a person types prompts directly into Maverick’s Codex pane; Goose still follows the file protocol.

## Branching Model
- Feature work happens on feature/* branches inside each agent’s worktree.
- Integration happens on dev (where inbox and outbox live).
- Lead merges feature branches into dev; later dev to main (outside Goose’s scope).

## Acceptance
- Starting a session creates the panes and optional AI.
- Creating an agent adds a worktree and branch.
- Writing inbox and outbox entries modifies files in dev.
- Radio commands reach the intended pane, and pane addressing is documented.
- README explains quickstart and examples.

## Constraints
- Single static binary for Linux and macOS, no heavy dependencies.
- tmux and vim friendly, standard input and output only.
