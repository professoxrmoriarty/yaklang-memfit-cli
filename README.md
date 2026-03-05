# yaklang-memfit-cli

[English](./README.md) | [简体中文](./README.zh-CN.md)

`yaklang-memfit-cli` is a terminal AI agent client for Yaklang.
It connects to `yaklang ai-http-gateway` and provides an interactive TUI-based workflow for creating sessions, streaming agent output, and sending follow-up inputs from the command line.

## What it does

- Connects to a Yaklang AI gateway endpoint (default: `http://127.0.0.1:8089/agent`)
- Creates and manages agent runs/sessions
- Streams real-time run events via SSE
- Sends user inputs/events during a live session
- Supports runtime AI setting updates (provider/model/review policy/focus mode)
- Supports run cancellation from the TUI

## Requirements

- Go `1.24.1+`
- A running Yaklang `ai-http-gateway` service
- Accessible API prefix (default: `/agent`)

## Quick start

```bash
# 1) Enter project directory
cd /Users/z3/Code/yaklang-aitui

# 2) Run directly
go run ./cmd
```

By default, the client connects to:

- Host: `127.0.0.1`
- Port: `8089`
- Prefix: `/agent`

So the final base URL is:

```text
http://127.0.0.1:8089/agent
```

## CLI flags

```bash
go run ./cmd \
  -host 127.0.0.1 \
  -port 8089 \
  -prefix /agent \
  -token <JWT_TOKEN>
```

- `-host`: gateway host
- `-port`: gateway port
- `-prefix`: API route prefix
- `-token`: optional JWT bearer token

## Build binary

```bash
go build -o yaklang-memfit-cli ./cmd
./yaklang-memfit-cli -host 127.0.0.1 -port 8089 -prefix /agent
```

## In-app slash commands

- `/help` show command help
- `/setting` show current AI settings
- `/provider` list/select provider
- `/model` list/select model
- `/focus_mode` list/select focus mode
- `/set <key> <value>` update setting fields
- `/review_policy <auto|ai|manual|ai-auto>` update review policy
- `/new` start a new session
- `/cancel` cancel current running session
- `/clear` clear screen content
- `/exit` or `/quit` quit the app

## High-level flow

1. Create a session via `POST /session`
2. Open SSE stream via `GET /run/{run_id}/events`
3. Trigger run execution via `POST /run/{run_id}`
4. Push follow-up user input via `POST /run/{run_id}/events/push`
5. Cancel run (if needed) via `POST /run/{run_id}/cancel`

## Project structure

- `cmd/main.go`: CLI entrypoint and flags
- `client.go`: HTTP/SSE client for gateway APIs
- `tui.go`: Bubble Tea TUI implementation
- `models.go`: request/response models
- `event_content.go`: event content parsing helpers

## Notes

- This repository is focused on a terminal-first AI agent experience.
- The UX is designed for iterative, stream-driven agent workflows.
- If your gateway requires authentication, pass `-token`.
