# Goose CLI

Global flags:
- --dry to avoid side effects
- --verbose for extra logs
- --session <name> (default: repo basename or "flight")

Commands:

1) goose session start
- Start a tmux session with predefined windows and panes.
- Flags:
  - --repo <path> (default: current directory)
  - --ai-lead "<cmd>" to start Codex in lead right pane (optional)
  - --ai-goose "<cmd>" to start Codex in goose right pane (optional)
  - --editor <cmd> (default: nvim)
  - --ops to create an "ops" window that watches inbox and outbox
  - --rebuild to kill an existing session with the same name before creating a new one
- Layout:
  - lead: pane 0 editor, pane 1 shell or AI
  - goose: pane 0 editor, pane 1 shell or AI
  - ops (optional): watch inbox and outbox

2) goose agent add
- Create or update an agent worktree and base branch.
- Flags:
  - --name <agent> (example: goose, phoenix)
  - --base <branch> base branch for new branches (default: dev)
  - --worktree <dir> default: personas/<agent>
  - --branch <branch> default: agent/<agent>
- Effects:
  - Ensure "git worktree add <dir> <branch>", creating branch from --base if missing.

3) goose feature start
- Create a per-feature branch in the agent’s worktree.
- Flags:
  - --agent <name>
  - --name <feature-name> creates feature/<feature-name>
  - --from <branch> base for the new branch (default: dev)

4) goose handoff open | ack | progress | done
- Append structured Markdown lines to shared handoff files in the dev branch.
- Files:
  - handoffs/inbox.md shared
  - handoffs/outbox/AGENT.md per agent
- Flags:
  - --task <ID> like TASK-001
  - --agent <name> who is acting
  - --branch <branch> used for progress and done
  - --note "free text"
- Examples:
  - goose handoff open --task TASK-001 --agent maverick --note "Implement login API"
  - goose handoff ack --task TASK-001 --agent goose
  - goose handoff progress --task TASK-001 --agent goose --branch feature/login-api --note "tests green"
  - goose handoff done --task TASK-001 --agent goose --branch feature/login-api --note "commit abc123"

5) goose radio send
- Send a shell command into a tmux target pane.
- Flags:
  - --target <window.pane> like goose.1 (right shell)
  - -- command here
- Example:
  - goose radio send --target goose.1 -- git pull --rebase

6) goose radio all
- Broadcast to a set of agent panes.
- Flags:
  - --agents "goose phoenix" default: all known
  - --pane 1 pane index default: 1
  - -- command here

7) goose session info
- Print pane addresses and working directories; detect worktrees and map them to windows.

8) goose check
- Verify prerequisites (git and tmux present), repo state, and handoff files.

Conventions:
- Lead window name: lead, Goose window name: goose.
- Pane 0 is editor, pane 1 is shell or AI.
- Handoffs are committed to dev only; features live in feature/* branches.
