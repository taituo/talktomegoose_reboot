# Operations

Quickstart
1) Start a session:
    goose session start --repo . --session flight --editor nvim --ops --ai-lead "codex --cd . -m gpt-5"

2) Add an agent worktree:
    goose agent add --name goose --base dev

3) Open a task:
    goose handoff open --task TASK-001 --agent maverick --note "Implement Login API"

4) Goose acknowledges and starts the feature:
    goose handoff ack --task TASK-001 --agent goose
    goose feature start --agent goose --name login-api --from dev

5) Broadcast a pull or test:
    goose radio all --agents "goose" --pane 1 -- git pull --rebase

Pane map
- lead.0 editor, lead.1 shell or AI
- goose.0 editor, goose.1 shell or AI

Handoffs live in dev
- Commit messages for handoffs: chore(handoffs): TASK-001 ack
- Commit messages for code: [Goose][TASK-001] <change>

Safety
- Run goose check before sessions.
- If panes drift, run goose session info.
