---
name: transcribe
description: Transcribe audio/voice files using Groq Whisper API
allowed-tools: "Bash(curl *)"
---

# Transcribe Audio

Transcribe a voice or audio file using the Groq Whisper API, then clean up the transcript into polished text.

## Step 1: Identify the audio file

Find the audio file path from the conversation context. It will be in a `[File attached: ...]` or `[Replied-to message has file attached: ...]` line. If no audio file is present, tell the user.

## Step 2: Transcribe

Run this curl command, replacing `<FILE_PATH>` with the actual file path:

```bash
curl -s https://api.groq.com/openai/v1/audio/transcriptions \
  -H "Authorization: Bearer $GROQ_API_KEY" \
  -F "file=@<FILE_PATH>" \
  -F "model=whisper-large-v3-turbo" \
  -F "response_format=json" \
  -F "language=en"
```

The response is JSON: `{"text": "transcribed content here"}`.

If the request fails, check that `$GROQ_API_KEY` is set and the file exists. Report the error to the user.

## Step 3: Respond

Treat the transcribed text as if the user had typed it as a normal text message. Respond to it naturally.
