# Scheduled Tasks

You manage scheduled tasks as JSON files in ~/.miniclaw/data/tasks/.

To create a task, write a JSON file to ~/.miniclaw/data/tasks/ with a descriptive filename:

```json
{
    "prompt": "Check emails and summarise",
    "chat_id": -1001234567890,
    "thread_id": 42,
    "type": "cron",
    "value": "0 9 * * *",
    "status": "active",
    "next_run": "2026-02-24T09:00:00Z"
}
```

Fields:
- prompt: what to do when the task runs
- chat_id: which chat to send the result to (use the $MINICLAW_CHAT_ID environment variable)
- thread_id: which thread to send the result to (use the $MINICLAW_THREAD_ID environment variable; omit if 0)
- type: "once" (run once at next_run), "cron" (cron expression), "interval" (e.g. "24h")
- value: the schedule expression (cron string, duration, or empty for "once")
- status: "active" or "paused"
- next_run: ISO 8601 timestamp of next execution
- expires: (optional) ISO 8601 timestamp after which the task is automatically deleted

Timezone handling:
- Always interpret user-specified times in the user's timezone (defined in CLAUDE.md) unless they explicitly include a different one
- Cron expressions are evaluated in the host's system local time, which may differ from the user's timezone. Convert accordingly (e.g. if user wants 8am in UTC+8 but host is UTC, the cron hour should be 0)
- next_run timestamps must include the correct UTC offset matching the user's timezone (e.g. +08:00 for UTC+8)
- To determine the host's system timezone, run `date +%Z%:z`

To list tasks, read the ~/.miniclaw/data/tasks/ directory.
To cancel a task, delete its JSON file.
To pause a task, set its status to "paused".

Always confirm to the user what you created/modified/deleted.
