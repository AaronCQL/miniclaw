# Long-Running Processes

When you need to run a process that stays alive indefinitely (dev servers, watchers, etc.), use tmux so the CLI session can exit and miniclaw can respond to the user.

- **Start:** `tmux new-session -d -s mc-<name> '<command>'`
- **List:** `tmux ls | grep ^mc-`
- **Check output:** `tmux capture-pane -t mc-<name> -p`
- **Stop:** `tmux kill-session -t mc-<name>`

All agent-managed sessions MUST use the `mc-` prefix. Never touch tmux sessions without this prefix. They belong to the user or other tools.

## Running Claude CLI

Claude Code blocks direct subprocess calls to `claude`. To run the Claude CLI for one-off introspection (e.g. checking version, config, or running a quick prompt), wrap it in a tmux session:

```sh
tmux new-session -d -s mc-claude 'unset CLAUDECODE && claude'
```

Then send your prompt or slash command, wait 2s for output, capture the pane, and exit. If the output is not ready, wait another 2s and capture again:

```sh
tmux send-keys -t mc-claude '/cost' Enter
sleep 2
tmux capture-pane -t mc-claude -p -S -50
tmux send-keys -t mc-claude Escape
tmux send-keys -t mc-claude '/exit' Enter
```
