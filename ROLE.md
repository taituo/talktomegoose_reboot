# Roles and Protocol

Maverick (lead)
- Writes tasks to handoffs/inbox.md in the dev branch:

    ## TASK-001: Login API (OPEN)
    assignee: @GOOSE
    acceptance:
      - POST /api/login returns 200 and a JWT
      - invalid credentials return 401
    created: ${TODAY}

- Reviews feature diffs and merges approved work into dev.
- Optionally uses goose radio to issue quick pulls and tests to agent panes.
- Does not code in feature branches.

Goose (coder)
- Pulls latest dev and reads the inbox.
- Acknowledges a task:

    [${TODAY}] TASK-001 ACK by @GOOSE

- Starts feature branch, commits, pushes, and reports DONE to outbox.
- Example outbox entry:

    [${TODAY}] TASK-001 DONE @GOOSE commit:abc123 branch:feature/login-api

Controller (optional)
- Periodically runs goose radio all with a command like git pull --rebase.
- Does not edit files; orchestration only.

Branching and files
- Features: feature/<name> in agent worktrees.
- Integration: dev (inbox and outbox live here).
- Releases: main (outside Goose scope).
