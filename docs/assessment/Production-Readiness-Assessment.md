# Operator OS — Production Readiness Assessment

**Date:** 2026-03-06
**Branch:** `operatoros-production-readiness`
**Assessed Version:** Current `main` (Go 1.25, ~63K LoC, 107 test files)
**Target:** Self-hosted deployments + Managed SaaS for non-technical users

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Current System Capabilities](#2-current-system-capabilities)
3. [Current State Summary](#3-current-state-summary)
4. [Gaps Preventing Production Use](#4-gaps-preventing-production-use)
5. [Risks](#5-risks)
6. [Required Improvements](#6-required-improvements)
7. [Recommended Architecture for Production](#7-recommended-architecture-for-production)
8. [Managed SaaS vs Self-Hosted Differences](#8-managed-saas-vs-self-hosted-differences)
9. [Roadmap to Production Readiness](#9-roadmap-to-production-readiness)

---

## 1. Architecture Overview

### 1.1 System Design

Operator OS is a **single-binary Go monolith** designed for ultra-lightweight deployment on constrained hardware (as little as $10 devices with <10MB RAM). The architecture follows a single-tenant, single-process model.

```
┌─────────────────────────────────────────────────────────┐
│                    Operator Binary                       │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐             │
│  │ Channels │  │  Agent   │  │  Tools    │             │
│  │ Manager  │──│  Loop    │──│  Registry │             │
│  └──────────┘  └──────────┘  └───────────┘             │
│       │             │              │                    │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐             │
│  │ Message  │  │ Session  │  │  Provider │             │
│  │   Bus    │  │ Manager  │  │  Factory  │             │
│  └──────────┘  └──────────┘  └───────────┘             │
│       │             │              │                    │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐             │
│  │ Health   │  │  State   │  │   Cron    │             │
│  │ Server   │  │ Manager  │  │  Service  │             │
│  └──────────┘  └──────────┘  └───────────┘             │
│                                                         │
│  Storage: Filesystem (JSON files)                       │
│  Bus: In-memory Go channels (buffer: 64)                │
│  Auth: File-based credential store (~/.operator/auth)   │
└─────────────────────────────────────────────────────────┘
```

### 1.2 Key Components

| Component | Implementation | Storage |
|---|---|---|
| **Message Bus** | In-memory Go channels (64-buffer) | None (volatile) |
| **Session Manager** | Per-workspace JSON files | `workspace/sessions/*.json` |
| **State Manager** | Atomic JSON writes | `workspace/state/state.json` |
| **Auth Store** | Plaintext JSON | `~/.operator/auth.json` |
| **Config** | Single JSON file | `~/.operator/config.json` |
| **Memory** | Markdown files | `workspace/memory/` |
| **Health Server** | HTTP `/health` + `/ready` | None |
| **Agent Registry** | In-memory, config-driven | Config file |
| **Routing** | 7-level priority cascade | Config bindings |
| **Providers** | Multi-provider with fallback chains + round-robin | Config |
| **Skills** | Filesystem + ClawHub registry | `workspace/skills/` |

### 1.3 Binary Characteristics

- **Language:** Go 1.25 with CGO disabled (`CGO_ENABLED=0`)
- **Binary Size:** ~27MB
- **RAM:** <10MB at idle
- **Boot Time:** <1 second
- **Architectures:** x86_64, ARM64, ARMv7, RISC-V, LoongArch64
- **Platforms:** Linux, macOS, Windows
- **Dependencies:** SQLite (modernc, pure Go), no C dependencies

---

## 2. Current System Capabilities

### 2.1 What Works

| Capability | Status | Notes |
|---|---|---|
| Multi-LLM provider support | ✅ Solid | OpenAI, Anthropic, Gemini, Ollama, Groq, DeepSeek, 15+ providers |
| Multi-channel messaging | ✅ Solid | Telegram, Discord, Slack, WhatsApp, LINE, 13 channels |
| Tool execution (shell, FS, web) | ✅ Solid | Sandboxed exec, file ops, web search/fetch |
| MCP server integration | ✅ Solid | stdio + HTTP/SSE transports |
| Persistent memory | ✅ Basic | Long-term (MEMORY.md) + daily notes |
| Session management | ✅ Basic | Per-session JSON with summarization |
| Agent routing | ✅ Advanced | 7-level priority cascade with bindings |
| Multi-agent support | ✅ Basic | Agent list, subagent spawning, workspace isolation |
| Cron scheduling | ✅ Basic | Periodic tasks with exec timeout |
| Heartbeat monitoring | ✅ Basic | Configurable interval |
| Health checks | ✅ Basic | `/health` and `/ready` endpoints |
| Docker deployment | ✅ Basic | Alpine + Full (with Node.js for MCP) |
| CLI interface | ✅ Solid | `operator agent`, `operator gateway`, commands |
| Fallback chains | ✅ Advanced | Cooldown tracking, error classification, round-robin |
| Context compression | ✅ Basic | Token-based summarization + emergency truncation |
| Workspace sandboxing | ✅ Basic | Path restriction + command deny patterns |
| OAuth support | ✅ Basic | Antigravity (Google), GitHub Copilot |
| Skills marketplace | ✅ Basic | ClawHub registry search + install |
| Web UI (Pico) | ✅ Basic | WebSocket-based chat protocol |
| Hardware I/O | ✅ Niche | I2C + SPI tools for embedded Linux |

### 2.2 Test Coverage

- **107 test files** covering ~25K lines
- Unit tests for core packages (config, session, routing, providers, tools, bus)
- No integration test suite
- No end-to-end tests
- No load/performance tests
- No fuzz testing

---

## 3. Current State Summary

### Verdict: **Excellent self-hosted single-user tool. Not production-ready for multi-tenant SaaS.**

Operator OS is a well-engineered, remarkably lightweight agent framework optimized for personal use on constrained hardware. The codebase is clean, well-structured, and thoroughly tested at the unit level. The provider system with fallback chains and the 7-level routing cascade are particularly sophisticated.

However, the architecture makes fundamental assumptions that prevent multi-tenant production deployment:

1. **Single-user, single-process design** — no concept of "users" or "tenants"
2. **Filesystem as database** — JSON files for all persistence
3. **In-memory message bus** — messages lost on crash
4. **Credentials in plaintext** — API keys in `config.json` and `auth.json`
5. **No observability** — custom logger, no metrics, no tracing
6. **No horizontal scaling** — stateful process with no shared-nothing separation

---

## 4. Gaps Preventing Production Use

### 4.1 Critical (Blockers)

#### G1: No User Management
- No concept of user accounts, registration, or authentication
- `allow_from` is a static allowlist of platform-specific IDs
- No sign-up flow, email verification, or password management
- **Impact:** Cannot support "sign up → choose plan → use agents" flow

#### G2: No Multi-Tenancy
- Single config file governs entire instance
- All agents share one workspace or use workspace-per-agent (not workspace-per-user)
- Session keys are channel:chatID scoped, not user-scoped
- No data isolation between users
- **Impact:** One user's data is accessible to other users' agents

#### G3: No Database
- Sessions: JSON files in `workspace/sessions/`
- State: JSON file in `workspace/state/state.json`
- Auth: JSON file in `~/.operator/auth.json`
- Memory: Markdown files
- No indexing, no querying, no transactions, no ACID guarantees
- **Impact:** Cannot scale, cannot query, cannot maintain consistency under concurrent writes

#### G4: No Billing or Subscription System
- No plan management, usage tracking, or payment integration
- No per-user LLM token metering
- No usage limits or quotas
- **Impact:** Cannot monetize or control resource consumption

#### G5: No Encryption at Rest
- API keys stored in plaintext JSON
- OAuth tokens stored in plaintext JSON
- No key management system
- **Impact:** Single breach exposes all credentials

#### G6: In-Memory Message Bus
- Go channels with buffer of 64
- All messages lost on process crash or restart
- No persistence, no acknowledgment, no retry
- **Impact:** Message loss during restarts, crashes, or high load

### 4.2 High Priority

#### G7: No Horizontal Scaling
- Stateful process: sessions, bus, and state all in-memory/local-filesystem
- No shared-nothing architecture
- Cannot run multiple instances behind a load balancer
- **Impact:** Single point of failure; vertical scaling only

#### G8: No Observability Stack
- Custom `logger` package (file + stdout)
- No structured metrics (Prometheus, OpenTelemetry)
- No distributed tracing
- No alerting integration
- Health endpoint is binary (ok/not-ready), no detailed component status
- **Impact:** Cannot monitor production SaaS at scale

#### G9: No Audit Logging
- No record of who did what, when
- No compliance trail
- Tool executions logged but not in auditable format
- **Impact:** Cannot meet compliance requirements, cannot investigate incidents

#### G10: No Rate Limiting
- Provider-level RPM config exists but is per-model, not per-user
- No API rate limiting for the Pico WebSocket endpoint
- No throttling of tool executions per user
- **Impact:** Single user can exhaust LLM quota for entire instance

#### G11: No Backup Strategy
- No automated backup mechanism
- Atomic writes prevent corruption but not data loss
- No point-in-time recovery
- **Impact:** Disk failure = total data loss

### 4.3 Medium Priority

#### G12: No Admin Dashboard
- Configuration via JSON file editing
- No web UI for platform management
- No user management interface
- **Impact:** Cannot manage a SaaS platform without engineering intervention

#### G13: No Service Integration Framework
- No OAuth flow for connecting user services (Shopify, Gmail, etc.)
- No credential vault per user
- No permission/scope management
- **Impact:** Users cannot connect their own services

#### G14: No Version Management / Rolling Updates
- Binary replacement + restart
- No database migrations framework
- No config version tracking
- **Impact:** Updates may require manual intervention

#### G15: Insufficient Error Recovery
- Context window errors trigger emergency compression (drop 50% of messages)
- No circuit breaker patterns beyond provider fallback
- No dead letter queue for failed messages
- **Impact:** Degraded experience under failure conditions

---

## 5. Risks

### 5.1 Security Risks

| Risk | Severity | Description |
|---|---|---|
| **Credential Exposure** | 🔴 Critical | All API keys and tokens in plaintext JSON. A single file read vulnerability exposes everything. |
| **Agent Escape** | 🟡 High | `exec` tool deny patterns are regex-based; sophisticated prompt injection could bypass them. No seccomp/AppArmor/container isolation per agent. |
| **Session Cross-Contamination** | 🟡 High | Without user-scoped isolation, a misconfigured binding could route user A's message to user B's agent. |
| **No CSRF/XSS Protection** | 🟡 Medium | Pico WebSocket endpoint has token auth but no CSRF protection. Web UI embeds user-generated markdown. |
| **DoS via Tool Abuse** | 🟡 Medium | Shell execution has timeout but no concurrent execution limits per user. |

### 5.2 Operational Risks

| Risk | Severity | Description |
|---|---|---|
| **Single Point of Failure** | 🔴 Critical | One process. If it crashes, all users lose service. |
| **Message Loss** | 🔴 Critical | In-memory bus means crashes lose in-flight messages with no recovery. |
| **Disk Full** | 🟡 High | JSON session files grow unbounded. No rotation or size limits. |
| **Memory Leak Potential** | 🟡 Medium | Session map grows indefinitely (no eviction/TTL for inactive sessions). |
| **Provider Outage Cascade** | 🟡 Medium | Fallback chains help, but all users share the same cooldown tracker. |

### 5.3 Business Risks

| Risk | Severity | Description |
|---|---|---|
| **No Usage Metering** | 🔴 Critical | Cannot attribute LLM costs to individual users. |
| **No SLA Enforcement** | 🔴 Critical | No mechanisms for availability guarantees, response time SLOs, or degraded-mode operation. |
| **Data Sovereignty** | 🟡 High | No configurable data residency. User data goes wherever the LLM provider is. |
| **Vendor Lock-in** | 🟢 Low | Multi-provider architecture actually mitigates this well. |

---

## 6. Required Improvements

### 6.1 Infrastructure Layer

#### I1: Database (PostgreSQL)
Replace filesystem storage with PostgreSQL for:
- User accounts and profiles
- Sessions and conversation history
- Agent configurations
- Billing records and usage metrics
- Audit logs
- Credential vault (encrypted)

**Why PostgreSQL:** Mature, battle-tested, excellent JSON support (for flexible schemas during early development), row-level security for multi-tenancy, and strong ecosystem (pgBouncer, pg_stat_statements, logical replication).

#### I2: Message Queue (NATS or Redis Streams)
Replace in-memory Go channels with a persistent message queue:
- Message durability across restarts
- At-least-once delivery guarantees
- Consumer groups for horizontal scaling
- Dead letter queue for failed messages
- Backpressure handling

**Recommendation:** NATS JetStream — lightweight (aligns with Operator's ethos), embedded Go library available, built-in persistence, and consumers.

#### I3: Secrets Management
- Encrypt credentials at rest (AES-256-GCM or similar)
- Per-user encryption keys derived from user secrets
- Integration with external vaults (HashiCorp Vault, AWS Secrets Manager) for managed deployments
- Key rotation support

#### I4: Container Orchestration
- Kubernetes manifests + Helm chart for managed SaaS
- Per-tenant resource limits (CPU, memory, network)
- Health-based auto-scaling
- Rolling updates with zero downtime

#### I5: Observability
- **Metrics:** OpenTelemetry SDK → Prometheus endpoint
  - LLM request latency, token usage, error rates (per provider, per user)
  - Message bus throughput, queue depth
  - Session count, active users
  - Tool execution duration and failure rates
- **Tracing:** OpenTelemetry distributed tracing
  - Message lifecycle: inbound → routing → LLM call → tool execution → response
- **Logging:** Structured JSON logs with correlation IDs
- **Alerting:** Integration with PagerDuty/Opsgenie

### 6.2 Application Layer

#### A1: User Management Service
```
POST /api/v1/auth/register    → email + password sign-up
POST /api/v1/auth/login       → JWT issuance
POST /api/v1/auth/verify      → email verification
POST /api/v1/auth/refresh     → token refresh
GET  /api/v1/users/me         → user profile
PUT  /api/v1/users/me         → update profile
```

#### A2: Multi-Tenant Agent Isolation
- Tenant ID propagated through entire request lifecycle
- Per-tenant workspace directories (or database-scoped)
- Per-tenant agent configurations
- Per-tenant session isolation
- Per-tenant LLM provider credentials

#### A3: Billing and Usage Tracking
- Stripe or Paddle integration for subscriptions
- Real-time token usage tracking per user per agent
- Usage-based billing or tiered plans
- Rate limiting based on plan tier
- Overage notifications and hard/soft caps

#### A4: API Gateway
- REST API with OpenAPI spec
- WebSocket endpoint (evolving from Pico protocol)
- API key authentication for programmatic access
- Rate limiting (per user, per endpoint)
- Request validation and sanitization

#### A5: Admin Control Plane
- User management (view, suspend, delete)
- Platform-wide configuration
- Usage dashboards and analytics
- System health monitoring
- Feature flags and gradual rollouts

### 6.3 Security Layer

#### S1: Authentication
- Email/password with bcrypt hashing
- OAuth 2.0 social login (Google, GitHub, Apple)
- Multi-factor authentication (TOTP)
- Session management with secure cookies + JWT

#### S2: Authorization
- Role-based access control (RBAC): admin, user, viewer
- Per-agent permission model
- Per-tool permission scoping
- API scope management

#### S3: Agent Sandboxing (Enhanced)
- Container-level isolation per agent execution (gVisor or Firecracker for managed)
- Network policy restrictions per agent
- Filesystem isolation (overlayfs or similar)
- Resource quotas (CPU, memory, execution time) per agent per user

#### S4: Data Protection
- Encryption at rest (database + filesystem)
- Encryption in transit (TLS everywhere)
- PII detection and redaction in logs
- GDPR-compliant data export and deletion
- Configurable data retention policies

---

## 7. Recommended Architecture for Production

### 7.1 Self-Hosted (Single User / Small Team)

Closest to current architecture. Enhancements:

```
┌─────────────────────────────────────────┐
│            Operator Binary              │
│                                         │
│  ┌────────────┐  ┌──────────────────┐   │
│  │ Agent Loop │  │ Embedded SQLite  │   │
│  │ + Channels │  │ (sessions, state,│   │
│  │ + Tools    │  │  audit, creds)   │   │
│  └────────────┘  └──────────────────┘   │
│                                         │
│  ┌────────────┐  ┌──────────────────┐   │
│  │  Health    │  │ Encrypted Config │   │
│  │  + Metrics │  │  + Key Derivation│   │
│  └────────────┘  └──────────────────┘   │
└─────────────────────────────────────────┘
```

Key changes:
- Replace JSON files with embedded SQLite (already a dependency via `modernc.org/sqlite`)
- Add credential encryption via user-provided passphrase
- Add Prometheus metrics endpoint
- Add automated backup to local/cloud storage
- Keep single-binary deployment model

### 7.2 Managed SaaS (Multi-Tenant)

```
                    ┌─────────────────┐
                    │   CDN / Edge    │
                    │  (CloudFlare)   │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │   API Gateway   │
                    │   (Kong/Envoy)  │
                    │ Auth + Rate Lim │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
     ┌────────▼───┐  ┌──────▼─────┐  ┌─────▼──────┐
     │   Web App  │  │  Operator  │  │  Operator  │
     │  (Next.js) │  │  Workers   │  │  Workers   │
     │  Dashboard │  │ (Stateless)│  │ (Stateless)│
     └────────────┘  └──────┬─────┘  └─────┬──────┘
                             │              │
              ┌──────────────┼──────────────┐
              │              │              │
     ┌────────▼───┐  ┌──────▼─────┐  ┌─────▼──────┐
     │ PostgreSQL │  │   NATS     │  │   Redis    │
     │ (Primary)  │  │ JetStream  │  │  (Cache +  │
     │ + Replicas │  │ (Messages) │  │  Sessions) │
     └────────────┘  └────────────┘  └────────────┘
              │
     ┌────────▼───┐
     │  Vault /   │
     │  KMS       │
     │ (Secrets)  │
     └────────────┘
```

Key architectural changes:
1. **Stateless Operator Workers:** Agent loop refactored to pull work from NATS, read state from PostgreSQL/Redis
2. **Shared Database:** PostgreSQL for durable state, Redis for session cache
3. **API Gateway:** Authentication, rate limiting, request routing
4. **Web Dashboard:** User-facing admin panel (Next.js or similar)
5. **Secrets Management:** HashiCorp Vault or cloud KMS
6. **Auto-Scaling:** Kubernetes HPA based on queue depth and CPU

### 7.3 Refactoring Strategy

The Go codebase is well-structured enough that this refactoring can be **incremental**:

1. **Interface extraction:** The `SessionManager`, `state.Manager`, and auth store already have clean interfaces. Swap filesystem implementations for database-backed ones.
2. **Bus abstraction:** `MessageBus` uses `PublishInbound`/`ConsumeInbound` — replace channel-backed implementation with NATS-backed one.
3. **Provider factory:** Already decoupled. Add per-tenant credential injection.
4. **Routing:** Already supports bindings. Add tenant-scoped resolution.

The architecture does NOT require a rewrite. The separation of concerns is good enough for incremental migration.

---

## 8. Managed SaaS vs Self-Hosted Differences

| Dimension | Self-Hosted | Managed SaaS |
|---|---|---|
| **Deployment** | Single binary, `operator gateway` | Kubernetes cluster, auto-scaling |
| **Database** | Embedded SQLite | PostgreSQL + Redis |
| **Message Bus** | In-process (enhanced) | NATS JetStream |
| **User Management** | `allow_from` allowlist | Full auth system (email, OAuth, MFA) |
| **Billing** | None | Stripe/Paddle integration |
| **Multi-Tenancy** | Single tenant (owner) | Full tenant isolation |
| **Secrets** | Encrypted local file | Vault / KMS |
| **Scaling** | Vertical only | Horizontal auto-scaling |
| **Updates** | Manual binary replacement | Rolling zero-downtime deploys |
| **Monitoring** | Optional Prometheus endpoint | Full observability stack |
| **Sandboxing** | Process-level | Container-level (gVisor) |
| **Data Residency** | User's machine | Configurable regions |
| **SLA** | Best-effort | 99.9%+ with degraded modes |
| **Support** | Community/docs | Tiered support plans |
| **Agent Limits** | Hardware-bound | Plan-based quotas |

### Shared Codebase Strategy

The Go binary should support **both modes** via build tags or config flags:

```go
// Build tags for deployment mode
// +build saas        → PostgreSQL, NATS, Vault
// +build selfhosted  → SQLite, in-process, local encryption
```

Alternatively, use interface-based dependency injection at startup:

```go
func main() {
    mode := config.DeploymentMode() // "saas" or "selfhosted"
    
    var store SessionStore
    var bus MessageBus
    
    switch mode {
    case "saas":
        store = postgres.NewSessionStore(dbConn)
        bus = nats.NewMessageBus(natsConn)
    default:
        store = sqlite.NewSessionStore(dbPath)
        bus = memory.NewMessageBus()
    }
}
```

---

## 9. Roadmap to Production Readiness

### Phase 1: Foundation (Weeks 1–4)
**Goal:** Stabilize data layer, add observability

| Task | Priority | Effort | Details |
|---|---|---|---|
| Replace JSON sessions with SQLite | 🔴 P0 | 1 week | Use existing `modernc.org/sqlite` dependency. Schema: `sessions`, `messages`, `state` tables. |
| Add structured logging | 🔴 P0 | 3 days | Replace custom logger with `zerolog` (already a transitive dependency). Add correlation IDs. |
| Add OpenTelemetry metrics | 🟡 P1 | 4 days | Prometheus endpoint: LLM latency, token usage, error rates, active sessions. |
| Encrypt auth store | 🟡 P1 | 3 days | AES-256-GCM encryption for `auth.json` using passphrase-derived key. |
| Add session TTL + eviction | 🟡 P1 | 2 days | Prevent unbounded memory growth from inactive sessions. |
| Automated backup (local) | 🟡 P1 | 2 days | SQLite `.backup` command on configurable schedule. |

### Phase 2: User Management (Weeks 5–8)
**Goal:** Multi-user support, basic auth

| Task | Priority | Effort | Details |
|---|---|---|---|
| User accounts table + API | 🔴 P0 | 1 week | Registration, login, email verification, JWT tokens. |
| Tenant-scoped sessions | 🔴 P0 | 1 week | Add `tenant_id` to all data models. Propagate through request lifecycle. |
| Per-user agent configuration | 🔴 P0 | 4 days | Users can configure their own agent persona, model, and tools. |
| Per-user rate limiting | 🟡 P1 | 3 days | Token bucket per user, configurable by plan tier. |
| Audit logging | 🟡 P1 | 3 days | Structured audit events: auth, tool execution, config changes. |
| Admin API | 🟡 P1 | 4 days | User management, platform config, usage analytics. |

### Phase 3: Billing + Plans (Weeks 9–12)
**Goal:** Monetization infrastructure

| Task | Priority | Effort | Details |
|---|---|---|---|
| Plan definitions | 🔴 P0 | 3 days | Free, Starter ($9/mo), Pro ($29/mo), Enterprise (custom). |
| Stripe integration | 🔴 P0 | 1 week | Subscriptions, usage-based billing, webhooks. |
| Token usage metering | 🔴 P0 | 4 days | Per-user, per-model token tracking in database. |
| Usage dashboards | 🟡 P1 | 4 days | User-facing usage analytics. |
| Overage handling | 🟡 P1 | 3 days | Soft caps (warnings) and hard caps (request rejection). |
| Plan upgrade/downgrade flows | 🟡 P1 | 3 days | Self-service via dashboard. |

### Phase 4: Service Integrations (Weeks 13–16)
**Goal:** Third-party service connections

| Task | Priority | Effort | Details |
|---|---|---|---|
| OAuth 2.0 integration framework | 🔴 P0 | 1 week | Generic OAuth flow for connecting user services. |
| Per-user credential vault | 🔴 P0 | 1 week | Encrypted storage for user's service tokens. |
| First integrations: Google, Shopify | 🟡 P1 | 1 week | Gmail, Drive, Calendar, Shopify Admin API. |
| Integration marketplace UI | 🟡 P1 | 1 week | Browse, connect, manage integrations. |
| See: [User-Onboarding-and-Service-Integration.md](./User-Onboarding-and-Service-Integration.md) | | | Full integration architecture. |

### Phase 5: Scaling + Reliability (Weeks 17–20)
**Goal:** Multi-instance, high availability

| Task | Priority | Effort | Details |
|---|---|---|---|
| Migrate to PostgreSQL (SaaS mode) | 🔴 P0 | 1.5 weeks | Database-backed session/state stores with connection pooling. |
| NATS JetStream message bus | 🔴 P0 | 1 week | Persistent, distributed message bus replacing Go channels. |
| Stateless worker architecture | 🔴 P0 | 1 week | Refactor agent loop to pull from queue, read/write to database. |
| Kubernetes deployment | 🟡 P1 | 1 week | Helm chart, HPA, PDB, resource quotas. |
| Redis session cache | 🟡 P1 | 3 days | Hot session caching for latency reduction. |
| Health check improvements | 🟡 P1 | 2 days | Component-level checks, dependency health. |

### Phase 6: Hardening + Launch (Weeks 21–24)
**Goal:** Security hardening, launch preparation

| Task | Priority | Effort | Details |
|---|---|---|---|
| Container-level agent sandboxing | 🟡 P1 | 1 week | gVisor or Firecracker per agent execution. |
| GDPR compliance toolkit | 🟡 P1 | 4 days | Data export, right-to-deletion, retention policies. |
| Load testing + performance tuning | 🟡 P1 | 1 week | Identify bottlenecks at target concurrency (1K, 10K users). |
| Security audit (external) | 🔴 P0 | 2 weeks | Professional penetration testing. |
| Documentation + developer guides | 🟡 P1 | 1 week | API docs, integration guides, self-hosting docs. |
| Beta launch | 🔴 P0 | — | Limited rollout with monitoring. |

### Timeline Summary

```
Weeks 1–4:   Foundation (SQLite, logging, metrics, encryption)
Weeks 5–8:   User Management (accounts, tenancy, auth)
Weeks 9–12:  Billing (Stripe, metering, plans)
Weeks 13–16: Integrations (OAuth, vault, marketplace)
Weeks 17–20: Scaling (PostgreSQL, NATS, K8s)
Weeks 21–24: Hardening (security, compliance, load testing, launch)
```

### Suggested Plan Tiers

| Feature | Free | Starter ($9/mo) | Pro ($29/mo) | Enterprise |
|---|---|---|---|---|
| Agents | 1 | 3 | 10 | Unlimited |
| Messages/mo | 500 | 5,000 | 50,000 | Unlimited |
| Integrations | 1 | 5 | 20 | Unlimited |
| Models | GPT-4o-mini | GPT-4o, Claude Haiku | All models | All + custom |
| Storage | 100MB | 1GB | 10GB | Custom |
| Support | Community | Email | Priority | Dedicated |
| Custom Skills | No | No | Yes | Yes |
| Team Members | 1 | 1 | 5 | Unlimited |
| API Access | No | Basic | Full | Full + SLA |

---

## Appendix A: Dependency Audit

### Direct Dependencies (Production)

| Dependency | Purpose | Risk |
|---|---|---|
| `gorilla/websocket` | Pico WebSocket channel | Low (mature, archived but stable) |
| `mymmrac/telego` | Telegram bot API | Low |
| `bwmarrin/discordgo` | Discord bot API | Low |
| `slack-go/slack` | Slack bot API | Low |
| `go.mau.fi/whatsmeow` | WhatsApp native (optional) | Medium (reverse-engineered protocol) |
| `anthropics/anthropic-sdk-go` | Anthropic provider | Low |
| `openai/openai-go` | OpenAI provider | Low |
| `github/copilot-sdk/go` | GitHub Copilot | Low |
| `modelcontextprotocol/go-sdk` | MCP client | Low |
| `modernc.org/sqlite` | Pure Go SQLite | Low (widely used) |
| `spf13/cobra` | CLI framework | Low |
| `adhocore/gronx` | Cron expression parser | Low |
| `caarlos0/env` | Environment variable parsing | Low |
| `google/uuid` | UUID generation | Low |

### Notable Absences

- No web framework (HTTP handlers are hand-rolled — fine for current scope)
- No ORM (would be needed for PostgreSQL migration)
- No migration tool (would be needed for schema management)
- No OpenTelemetry SDK (needed for observability)

## Appendix B: Codebase Quality Assessment

| Dimension | Rating | Notes |
|---|---|---|
| **Code Organization** | ⭐⭐⭐⭐ | Clean package structure, good separation of concerns |
| **Error Handling** | ⭐⭐⭐⭐ | Consistent error wrapping, good error classification in providers |
| **Concurrency Safety** | ⭐⭐⭐⭐ | Proper mutex usage, atomic operations, no obvious races |
| **Test Coverage** | ⭐⭐⭐ | Good unit tests, missing integration/e2e/load tests |
| **Documentation** | ⭐⭐⭐ | README is solid, inline docs vary, no architecture docs |
| **Security Practices** | ⭐⭐ | Sandbox exists but relies on regex deny patterns |
| **Observability** | ⭐⭐ | Custom logger only, no metrics or tracing |
| **Deployment Maturity** | ⭐⭐ | Docker exists, no K8s, no CI/CD pipeline visible |
| **API Design** | ⭐⭐⭐ | Internal APIs clean, no external REST API spec |
| **Extensibility** | ⭐⭐⭐⭐ | Provider interface, tool registry, skill system, MCP support |

---

*Assessment conducted by deep code review of the full Operator OS repository (63K LoC Go, 107 test files, full dependency tree, Docker configuration, and deployment artifacts).*
