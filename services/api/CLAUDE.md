# Service: Operator OS Gateway API

## Purpose
The Operator OS Gateway is the primary backend API. It handles agent orchestration, session management, authentication, billing, integrations, and multi-channel messaging. Written in Go as a single binary — this service layer provides an optional Docker-containerized deployment path.

## Tech Stack
- **Language**: Go 1.25+
- **Framework**: Standard library `net/http` with custom routing (`pkg/routing/`)
- **Database**: SQLite (default) or PostgreSQL (optional)
- **Cache**: Redis (optional, for rate limiting and session state)
- **Auth**: JWT tokens with refresh flow (`pkg/auth/`, `pkg/users/`)
- **WebSocket**: gorilla/websocket for real-time chat

## Architecture
The Go binary serves both the REST API and the WebSocket endpoint from a single process. The main entry point is `cmd/operator/main.go`.

```
cmd/operator/          # CLI entry points (gateway, agent, onboard)
pkg/
├── admin/             # Admin panel API + user management
├── agent/             # Agent loop, context, memory, registry
├── agents/            # Agent CRUD API + store
├── audit/             # Audit logging
├── auth/              # Credentials, encryption, OAuth, PKCE, tokens
├── backup/            # Backup and restore
├── beta/              # Beta feature flags + readiness checks
├── billing/           # Stripe integration, plans, subscriptions, usage
├── bus/               # Event bus (local + NATS)
├── channels/          # Messaging channels (Slack, Discord, Telegram…)
├── config/            # Configuration loading and defaults
├── cron/              # Scheduled task execution
├── gdpr/              # GDPR data export and deletion
├── health/            # Health check endpoints
├── identity/          # Agent identity management
├── integrations/      # Third-party integration registry + OAuth
├── mcp/               # Model Context Protocol server management
├── metrics/           # Prometheus metrics
├── oauth/             # OAuth token vault, refresh, state management
├── providers/         # LLM provider adapters (OpenAI, Anthropic, Gemini…)
├── routing/           # HTTP routing, session keys, agent ID resolution
├── sandbox/           # Workspace confinement + command filtering
├── session/           # Session management + eviction
├── tools/             # Agent tools (shell, filesystem, web, browser, MCP…)
├── users/             # User authentication, registration, JWT, middleware
└── worker/            # Background worker pool
```

## API Endpoints (Primary)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| POST | `/api/v1/auth/login` | User login (JWT) |
| POST | `/api/v1/auth/register` | User registration |
| POST | `/api/v1/auth/refresh` | Token refresh |
| POST | `/api/v1/auth/verify` | Email verification |
| GET | `/api/v1/agents` | List agents |
| POST | `/api/v1/agents` | Create agent |
| PUT | `/api/v1/agents/:id` | Update agent |
| DELETE | `/api/v1/agents/:id` | Delete agent |
| GET | `/api/v1/sessions` | List sessions |
| POST | `/api/v1/sessions` | Create session |
| GET | `/api/v1/sessions/:id/messages` | Get session messages |
| WS | `/ws` | WebSocket for real-time chat |
| GET | `/api/v1/billing/plans` | List billing plans |
| POST | `/api/v1/billing/checkout` | Create Stripe checkout |
| GET | `/api/v1/billing/usage` | Get usage stats |
| GET | `/api/v1/admin/users` | List users (admin) |
| GET | `/api/v1/admin/stats` | Platform stats (admin) |
| GET | `/api/v1/integrations` | List integrations |
| POST | `/api/v1/integrations/:id/connect` | Connect integration |

## Environment Variables

| Name | Description | Default |
|------|-------------|---------|
| `PORT` | Server port | `18790` |
| `CONFIG_PATH` | Path to config.json | `~/.operator/config.json` |
| `DATABASE_URL` | PostgreSQL connection string | (SQLite if unset) |
| `REDIS_URL` | Redis connection string | (optional) |
| `JWT_SECRET` | JWT signing secret | (auto-generated) |
| `STRIPE_SECRET_KEY` | Stripe API key | (optional) |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing | (optional) |

## Local Development
```bash
# Build and run directly
make build
./build/operator gateway

# Or via Docker
docker compose -f docker/docker-compose.yml --profile gateway up

# Run tests
make test

# Run linter
make lint
```

## Testing
```bash
# All tests
make test

# Specific package
CGO_ENABLED=0 go test ./pkg/auth/...
CGO_ENABLED=0 go test ./pkg/agent/...

# With verbose output
CGO_ENABLED=0 go test -v ./pkg/billing/...
```
