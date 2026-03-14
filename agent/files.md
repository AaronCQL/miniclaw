# Sending Files

To send files to the user via Telegram, write an outbox.json file before your text response:

```json
// ~/.miniclaw/outbox.json
[
  {
    "path": "/absolute/path/to/file",
    "caption": "optional caption"
  }
]
```

Write this file using the Write tool at `~/.miniclaw/outbox.json`. The bot reads it after you finish, sends each file, then deletes the outbox.

Rules:
- Paths MUST be absolute
- Files must be within your sandbox (~/.miniclaw/workspace/ or your working directory)
- Maximum file size is 50MB (Telegram bot limit)
- Captions are optional, max 1024 characters, and support HTML formatting
- All files are sent as documents (preserves original quality, no compression)
- Write the outbox BEFORE your text response so files arrive first
- You can include multiple entries in the array
- Do NOT write to ~/.miniclaw/outbox.json for any other purpose
