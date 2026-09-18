# Sound Stage — Backend

Scalable, real-time **live audio rooms** platform (Clubhouse-style) built in Go.
Audio is routed through a custom **SFU (Selective Forwarding Unit)** on [Pion WebRTC](https://github.com/pion/webrtc), with WebSocket signaling, presence via Redis pub/sub, and a clean layered architecture.

## Features

- 🎙️ **SFU audio routing** — one PeerConnection per client, Opus @ 48 kHz, server-side track fan-out
- 🚪 **Rooms** — public & private (rotating join codes), categories, tags, filters
- 👥 **Roles & moderation** — speakers/listeners, hand raises, mute, kick, block
- 💬 **Realtime chat** — pinned messages, per-room toggle, delivered over WS
- 🟢 **Presence** — live room state in Redis, fanned out via pub/sub
- 🔐 **Passwordless auth** — email OTP (Mailgun) + JWT access/refresh tokens
- ⚙️ **Background workers** — asynq task pool for email & async jobs

## Architecture

```
                 ┌────────────────────────────────────────────┐
   REST / WS ──▶ │  Gin Router  (CORS · rate-limit · JWT auth) │
                 └──────────────┬─────────────────────────────┘
                                │
              ┌─────────────────┼──────────────────┐
              ▼                 ▼                  ▼
        HTTP Handlers     WS Hub (gorilla)    Media Router (SFU)
        per domain        signaling +         Pion PeerConnections
              │           room events         RTP fan-out, Opus
              ▼                 ▲                  │
        Services  ──▶  Room State Service ─────────┘
        (business        Redis repo + pub/sub
         logic,              │
         authz)              ▼
              │         Subscriber ──▶ pushes events to WS Hub
              ▼
        Repositories (GORM)
              │
      ┌───────┴────────┐
      ▼                ▼
  PostgreSQL        Redis        +  asynq workers · Mailgun · Cloudinary
  (goose migrations)
```

Each domain under `internal/` follows the same layered pattern:

```
handler → service → authz → repo        (HTTP)
WSHandler → service → media router      (realtime)
```

| Layer                   | Responsibility                                      |
| ----------------------- | --------------------------------------------------- |
| `handler` / `WSHandler` | Transport: HTTP binding or WS event decoding        |
| `service`               | Business logic, transactions, orchestration         |
| `authz`                 | Room-scoped authorization (role, block, membership) |
| `repo`                  | Persistence (GORM / Redis)                          |

## Tech Stack

**Go 1.25** · Gin · Pion WebRTC (SFU) · gorilla/websocket · GORM + PostgreSQL · goose · Redis (state + pub/sub) · asynq · Mailgun · Cloudinary · JWT

## Getting Started

**Prerequisites:** Go 1.25+, PostgreSQL, Redis, [Task](https://taskfile.dev) _(optional)_

```bash
# 1. Configure environment
cp .env.example .env        # fill in DB, Redis, Mailgun, JWT, TURN/STUN values

# 2. Run migrations (uses GOOSE_* vars from .env)
goose up

# 3. Start the server
task run                    # or: go run ./cmd/main.go
task dev                    # hot reload (requires air)
```

Server listens on `:8000` — REST at `/`, WebSocket at `/ws/rooms/:roomId`.

```bash
task test                   # go test -race -cover ./...
task lint                   # golangci-lint run
```

## Docker

```bash
docker compose up -d        # runs goose migrations, then the app
```

Publishes `:8000` (HTTP/WS) and the ICE UDP port range for media.

## Project Structure

```
cmd/                  entrypoint
internal/
  ├─ router/          route registration + middleware chain
  ├─ server/          dependency wiring, lifecycle, graceful shutdown
  ├─ ws/              WebSocket hub, clients, event protocol
  ├─ web_rtc/         Pion peer connection, ICE, RTP forwarding
  ├─ media_router/    SFU: subscribe/fan-out/revoke of audio tracks
  ├─ room_state/      presence + room state (Redis repo, pub/sub)
  ├─ room/ room_user/ chat_message/ auth/ user/ ...   domain modules
  ├─ infra/           database, redis, mailer, worker
  ├─ middleware/      auth, rate limiter, logger, recovery
  └─ pkg/             shared utils (httpx, env, listopts, testutil)
```
