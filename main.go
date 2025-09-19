package main

import (
    "bufio"
    "errors"
    "flag"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"
)

type Global struct {
    Dry     bool
    Verbose bool
    Session string
}

var g Global

func main() {
    if err := run(os.Args[1:]); err != nil {
        fmt.Fprintln(os.Stderr, "error:", err)
        os.Exit(1)
    }
}

func run(args []string) error {
    // Global flag set parsed first; split args at first known command token
    commands := map[string]bool{"session": true, "agent": true, "handoff": true, "radio": true, "help": true, "-h": true, "--help": true}
    split := len(args)
    for i, a := range args {
        if commands[a] {
            split = i
            break
        }
    }
    gfs := flag.NewFlagSet("goose", flag.ContinueOnError)
    gfs.SetOutput(newDiscard())
    gfs.BoolVar(&g.Dry, "dry", false, "dry-run; print actions without executing")
    gfs.BoolVar(&g.Verbose, "verbose", false, "verbose logging")
    gfs.StringVar(&g.Session, "session", "", "tmux session name (default: repo basename or 'flight')")
    if err := gfs.Parse(args[:split]); err != nil {
        usage()
        return err
    }
    rest := args[split:]
    if len(rest) == 0 {
        usage()
        return errors.New("no command provided")
    }

    switch rest[0] {
    case "session":
        return cmdSession(rest[1:])
    case "agent":
        return cmdAgent(rest[1:])
    case "handoff":
        return cmdHandoff(rest[1:])
    case "radio":
        return cmdRadio(rest[1:])
    case "help", "-h", "--help":
        usage()
        return nil
    default:
        usage()
        return fmt.Errorf("unknown command: %s", rest[0])
    }
}

func usage() {
    fmt.Println("Goose – multi-agent dev runner (tmux + git worktrees + AI panes)")
    fmt.Println()
    fmt.Println("Usage: goose [--dry] [--verbose] [--session name] <command> [<args>]")
    fmt.Println()
    fmt.Println("Commands:")
    fmt.Println("  session start        Start a tmux session with windows/panes")
    fmt.Println("  agent add            Add/update an agent worktree and branch")
    fmt.Println("  handoff <op>         Write handoff inbox/outbox entries (open|ack|progress|done)")
    fmt.Println("  radio send|all       Send commands to tmux panes")
}

// ---------- session start ----------

func cmdSession(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("missing subcommand for session (try 'session start')")
    }
    switch args[0] {
    case "start":
        return sessionStart(args[1:])
    default:
        return fmt.Errorf("unknown session subcommand: %s", args[0])
    }
}

func sessionStart(args []string) error {
    fs := flag.NewFlagSet("session start", flag.ContinueOnError)
    fs.SetOutput(newDiscard())
    var repo, aiLead, aiGoose, editor string
    var ops, rebuild bool
    fs.StringVar(&repo, "repo", ".", "repository path")
    fs.StringVar(&aiLead, "ai-lead", "", "command to run AI in lead pane")
    fs.StringVar(&aiGoose, "ai-goose", "", "command to run AI in goose pane")
    fs.StringVar(&editor, "editor", "nvim", "editor command for left panes")
    fs.BoolVar(&ops, "ops", false, "create ops window to watch handoffs")
    fs.BoolVar(&rebuild, "rebuild", false, "kill existing session before creating")
    if err := fs.Parse(args); err != nil {
        return err
    }

    if err := requireTool("tmux"); err != nil {
        return err
    }

    absRepo, _ := filepath.Abs(repo)
    sname := g.Session
    if sname == "" {
        base := filepath.Base(absRepo)
        if base == "/" || base == "." || base == "" {
            sname = "flight"
        } else {
            sname = base
        }
    }

    if rebuild {
        _ = runCmd("tmux", "kill-session", "-t", sname)
    }

    // new detached session
    if err := runCmd("tmux", "new-session", "-d", "-s", sname, "-c", absRepo, "-n", "lead"); err != nil {
        return err
    }
    // split lead window into left(editor)/right(shell or AI)
    if err := runCmd("tmux", "split-window", "-h", "-t", sname+":lead"); err != nil {
        return err
    }
    if editor != "" {
        if err := tmuxSend(sname+":lead.0", editor); err != nil {
            return err
        }
    }
    if aiLead != "" {
        if err := tmuxSend(sname+":lead.1", aiLead); err != nil {
            return err
        }
    }

    // goose window
    if err := runCmd("tmux", "new-window", "-t", sname, "-n", "goose", "-c", absRepo); err != nil {
        return err
    }
    if err := runCmd("tmux", "split-window", "-h", "-t", sname+":goose"); err != nil {
        return err
    }
    if editor != "" {
        if err := tmuxSend(sname+":goose.0", editor); err != nil {
            return err
        }
    }
    if aiGoose != "" {
        if err := tmuxSend(sname+":goose.1", aiGoose); err != nil {
            return err
        }
    }

    if ops {
        if err := runCmd("tmux", "new-window", "-t", sname, "-n", "ops", "-c", absRepo); err != nil {
            return err
        }
        watcher := "while true; do clear; date; echo INBOX:; " +
            "[ -f handoffs/inbox.md ] && tail -n 80 handoffs/inbox.md || echo '(missing handoffs/inbox.md)'; " +
            "echo; echo OUTBOX:; for f in handoffs/outbox/*; do echo --- $f; tail -n 40 \"$f\"; done; sleep 5; done"
        if err := tmuxSend(sname+":ops.0", watcher); err != nil {
            return err
        }
    }

    fmt.Printf("session '%s' ready. Attach with: tmux attach -t %s\n", sname, sname)
    return nil
}

// ---------- agent add (git worktree) ----------

func cmdAgent(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("missing subcommand for agent (try 'agent add')")
    }
    switch args[0] {
    case "add":
        return agentAdd(args[1:])
    default:
        return fmt.Errorf("unknown agent subcommand: %s", args[0])
    }
}

func agentAdd(args []string) error {
    if err := requireTool("git"); err != nil {
        return err
    }
    fs := flag.NewFlagSet("agent add", flag.ContinueOnError)
    fs.SetOutput(newDiscard())
    var name, base, worktree, branch string
    fs.StringVar(&name, "name", "", "agent name (e.g. goose)")
    fs.StringVar(&base, "base", "dev", "base branch (default: dev)")
    fs.StringVar(&worktree, "worktree", "", "worktree directory (default personas/<agent>)")
    fs.StringVar(&branch, "branch", "", "branch name (default agent/<agent>)")
    if err := fs.Parse(args); err != nil {
        return err
    }
    if name == "" {
        return errors.New("--name is required")
    }
    if worktree == "" {
        worktree = filepath.Join("personas", name)
    }
    if branch == "" {
        branch = filepath.Join("agent", name)
    }

    // Ensure base exists; if branch missing create from base
    // Detect if branch exists
    if err := gitEnsureBranch(branch, base); err != nil {
        return err
    }
    // Ensure worktree exists
    if _, err := os.Stat(worktree); os.IsNotExist(err) {
        if err := runCmd("git", "worktree", "add", worktree, branch); err != nil {
            return err
        }
    } else if err == nil {
        if g.Verbose {
            fmt.Printf("worktree exists at %s (skipping add)\n", worktree)
        }
    } else {
        return err
    }
    fmt.Printf("agent '%s' ready: worktree=%s branch=%s\n", name, worktree, branch)
    return nil
}

func gitEnsureBranch(branch, base string) error {
    // Check if branch exists
    if err := runCmdSilent("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
        if g.Verbose {
            fmt.Printf("branch '%s' exists\n", branch)
        }
        return nil
    }
    // Create from base
    if err := runCmd("git", "branch", branch, base); err != nil {
        return fmt.Errorf("failed creating branch '%s' from '%s': %w", branch, base, err)
    }
    return nil
}

// ---------- handoff ops ----------

func cmdHandoff(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("missing handoff op (open|ack|progress|done)")
    }
    op := args[0]
    fs := flag.NewFlagSet("handoff", flag.ContinueOnError)
    fs.SetOutput(newDiscard())
    var task, agent, branch, note string
    fs.StringVar(&task, "task", "", "task ID (e.g. TASK-001)")
    fs.StringVar(&agent, "agent", "", "agent name (e.g. maverick or goose)")
    fs.StringVar(&branch, "branch", "", "branch (for progress/done)")
    fs.StringVar(&note, "note", "", "free text note")
    if err := fs.Parse(args[1:]); err != nil {
        return err
    }
    if task == "" {
        return errors.New("--task is required")
    }
    if agent == "" {
        return errors.New("--agent is required")
    }

    upperAgent := strings.ToUpper(agent)
    now := time.Now().Format("2006-01-02")
    if err := os.MkdirAll("handoffs/outbox", 0o755); err != nil {
        return err
    }

    switch op {
    case "open":
        if err := os.MkdirAll("handoffs", 0o755); err != nil {
            return err
        }
        path := filepath.Join("handoffs", "inbox.md")
        title := fmt.Sprintf("## %s (OPEN)", task)
        b := &strings.Builder{}
        fmt.Fprintf(b, "%s\n", title)
        fmt.Fprintf(b, "by @%s\n", strings.ToUpper(agent))
        if note != "" {
            fmt.Fprintf(b, "note: %s\n", note)
        }
        fmt.Fprintf(b, "created: %s\n\n", now)
        return appendFile(path, b.String())
    case "ack":
        line := fmt.Sprintf("[%s] %s ACK by @%s\n", now, task, upperAgent)
        path := filepath.Join("handoffs", "outbox", upperAgent+".md")
        return appendFile(path, line)
    case "progress":
        if branch == "" {
            return errors.New("--branch is required for progress")
        }
        line := fmt.Sprintf("[%s] %s PROGRESS @%s branch:%s", now, task, upperAgent, branch)
        if note != "" {
            line += " note:" + note
        }
        line += "\n"
        path := filepath.Join("handoffs", "outbox", upperAgent+".md")
        return appendFile(path, line)
    case "done":
        if branch == "" {
            return errors.New("--branch is required for done")
        }
        line := fmt.Sprintf("[%s] %s DONE @%s branch:%s", now, task, upperAgent, branch)
        if note != "" {
            line += " note:" + note
        }
        line += "\n"
        path := filepath.Join("handoffs", "outbox", upperAgent+".md")
        return appendFile(path, line)
    default:
        return fmt.Errorf("unknown handoff op: %s", op)
    }
}

func appendFile(path, content string) error {
    if g.Verbose || g.Dry {
        fmt.Printf("append %s:\n%s", path, content)
    }
    if g.Dry {
        return nil
    }
    f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
    if err != nil {
        return err
    }
    defer f.Close()
    w := bufio.NewWriter(f)
    if _, err := w.WriteString(content); err != nil {
        return err
    }
    return w.Flush()
}

// ---------- radio ----------

func cmdRadio(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("missing radio subcommand (send|all)")
    }
    switch args[0] {
    case "send":
        return radioSend(args[1:])
    case "all":
        return radioAll(args[1:])
    default:
        return fmt.Errorf("unknown radio subcommand: %s", args[0])
    }
}

func radioSend(args []string) error {
    if err := requireTool("tmux"); err != nil {
        return err
    }
    // Parse flags up to --, then capture command tail
    fs := flag.NewFlagSet("radio send", flag.ContinueOnError)
    fs.SetOutput(newDiscard())
    var target string
    fs.StringVar(&target, "target", "", "tmux target like window.pane (e.g. goose.1)")
    // locate the -- separator
    sep := indexOf(args, "--")
    var flagsPart, cmdPart []string
    if sep >= 0 {
        flagsPart = args[:sep]
        if sep+1 < len(args) {
            cmdPart = args[sep+1:]
        }
    } else {
        flagsPart = args
    }
    if err := fs.Parse(flagsPart); err != nil {
        return err
    }
    if target == "" {
        return errors.New("--target is required")
    }
    if len(cmdPart) == 0 {
        return errors.New("missing command after --")
    }
    t := qualifyTarget(target)
    return tmuxSend(t, strings.Join(cmdPart, " "))
}

func radioAll(args []string) error {
    if err := requireTool("tmux"); err != nil {
        return err
    }
    fs := flag.NewFlagSet("radio all", flag.ContinueOnError)
    fs.SetOutput(newDiscard())
    var agents string
    var pane int
    fs.StringVar(&agents, "agents", "", "space-separated agent names (default: goose)")
    fs.IntVar(&pane, "pane", 1, "pane index (default: 1)")
    // parse before --
    sep := indexOf(args, "--")
    var flagsPart, cmdPart []string
    if sep >= 0 {
        flagsPart = args[:sep]
        if sep+1 < len(args) {
            cmdPart = args[sep+1:]
        }
    } else {
        flagsPart = args
    }
    if err := fs.Parse(flagsPart); err != nil {
        return err
    }
    if len(cmdPart) == 0 {
        return errors.New("missing command after --")
    }
    list := fieldsOrDefault(agents, []string{"goose"})
    cmd := strings.Join(cmdPart, " ")
    for _, a := range list {
        t := qualifyTarget(fmt.Sprintf("%s.%d", a, pane))
        if err := tmuxSend(t, cmd); err != nil {
            return err
        }
    }
    return nil
}

// ---------- helpers ----------

func runCmd(name string, args ...string) error {
    if g.Verbose || g.Dry {
        fmt.Printf("$ %s %s\n", name, strings.Join(args, " "))
    }
    if g.Dry {
        return nil
    }
    cmd := exec.Command(name, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func runCmdSilent(name string, args ...string) error {
    if g.Verbose {
        fmt.Printf("$ %s %s\n", name, strings.Join(args, " "))
    }
    if g.Dry {
        return nil
    }
    cmd := exec.Command(name, args...)
    return cmd.Run()
}

func tmuxSend(target, cmd string) error {
    // send the command and press Enter
    return runCmd("tmux", "send-keys", "-t", target, cmd, "C-m")
}

func qualifyTarget(wp string) string {
    // If session provided, prefix it; otherwise raw
    if g.Session == "" {
        return wp
    }
    // Already has session?
    if strings.Contains(wp, ":") {
        return wp
    }
    return g.Session + ":" + wp
}

func requireTool(name string) error {
    if g.Dry {
        return nil
    }
    if _, err := exec.LookPath(name); err != nil {
        return fmt.Errorf("required tool not found in PATH: %s", name)
    }
    return nil
}

func indexOf(ss []string, needle string) int {
    for i, s := range ss {
        if s == needle {
            return i
        }
    }
    return -1
}

func parseLeadingFlags(fs *flag.FlagSet, args []string) ([]string, error) {
    for i, a := range args {
        if !strings.HasPrefix(a, "-") {
            // Parse up to i
            if err := fs.Parse(args[:i]); err != nil {
                return nil, err
            }
            return args[i:], nil
        }
    }
    if err := fs.Parse(args); err != nil {
        return nil, err
    }
    return []string{}, nil
}

type discard struct{}

func newDiscard() *discard { return &discard{} }

func (*discard) Write(p []byte) (int, error) { return len(p), nil }

func fieldsOrDefault(s string, def []string) []string {
    f := strings.Fields(s)
    if len(f) == 0 {
        return def
    }
    return f
}
