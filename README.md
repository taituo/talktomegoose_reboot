# Goose

A tmux and vim friendly multi-agent dev runner:
- tmux windows and panes per role,
- git worktrees per agent or feature,
- file-based handoffs (inbox and outbox) in the dev branch,
- optional AI (Codex) processes in panes,
- radio commands via tmux send-keys.

See SPEC.md, CLI.md, ROLE.md, and OPERATIONS.md for details.

## Usage

Build:
- go build -o goose

Global flags:
- `--dry` print actions without executing
- `--verbose` extra logs
- `--session <name>` tmux session name (defaults to repo basename or `flight`)

Examples:
- Start a session with AI in lead and ops watcher:
  - `./goose --session flight session start --repo . --editor nvim --ops --ai-lead "codex --cd . -m gpt-5"`
- Add an agent worktree and branch:
  - `./goose agent add --name goose --base dev`
- Open/ack/progress/done handoffs:
  - `./goose handoff open --task TASK-001 --agent maverick --note "Implement Login API"`
  - `./goose handoff ack --task TASK-001 --agent goose`
  - `./goose handoff progress --task TASK-001 --agent goose --branch feature/login-api --note "tests green"`
  - `./goose handoff done --task TASK-001 --agent goose --branch feature/login-api --note "commit abc123"`
- Send a radio command to a pane:
  - `./goose --session flight radio send --target goose.1 -- git pull --rebase`
- Broadcast to multiple agents (pane 1):
  - `./goose --session flight radio all --agents "goose" --pane 1 -- make test`

Notes:
- Session panes: `lead.0` editor, `lead.1` shell/AI; `goose.0` editor, `goose.1` shell/AI.
- Ops window (optional) tails `handoffs/inbox.md` and `handoffs/outbox/*`.
- Handoff files are created under `handoffs/` but commits are left to you.

## Session Info (planned)
- Purpose: print pane addresses and working directories; map worktrees to windows.
- Expected usage: `./goose --session flight session info`
- Status: planned. For now, manually inspect with:
  - `tmux list-windows -t flight` and `tmux list-panes -t flight:lead`
  - `tmux display-message -p -t flight:goose.0 '#{pane_current_path}'`

## Preflight Check (planned)
- Purpose: verify prerequisites and repo state.
- Expected usage: `./goose check`
- Checks: `tmux` present, `git` present, repo detected, `handoffs/` exists.
- Status: planned. For now, manually verify:
  - `tmux -V` and `git --version`
  - `test -d .git && echo ok`
  - `test -d handoffs && ls -la handoffs` or `mkdir -p handoffs/outbox`
