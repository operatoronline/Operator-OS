# Operator OS — Production Readiness Status

**Last Updated:** 2026-03-06
**Current Phase:** 1 — Foundation
**Overall Progress:** 0%

---

## Phase Overview

| # | Phase | Status | Target | Progress |
|---|---|---|---|---|
| 1 | Foundation (SQLite, logging, metrics, encryption) | 🟡 In Progress | Weeks 1–4 | 88% |
| 2 | User Management (accounts, tenancy, auth) | ⬜ Not Started | Weeks 5–8 | 0% |
| 3 | Billing & Plans (Stripe, metering) | ⬜ Not Started | Weeks 9–12 | 0% |
| 4 | Service Integrations (OAuth, vault, marketplace) | ⬜ Not Started | Weeks 13–16 | 0% |
| 5 | Scaling & Reliability (PostgreSQL, NATS, K8s) | ⬜ Not Started | Weeks 17–20 | 0% |
| 6 | Hardening & Launch (security, compliance, testing) | ⬜ Not Started | Weeks 21–24 | 0% |

---

## Phase 1: Foundation

### Tasks

| ID | Task | Priority | Status | Assignee | Notes |
|---|---|---|---|---|---|
| F1 | Replace JSON sessions with SQLite | P0 | ✅ Done | Cosmo | Implemented `SessionStore` interface + `SQLiteStore` backend. `SessionManager` delegates to store when present via `NewSessionManagerWithStore()`. 15 tests pass. WAL mode, write-through. |
| F2 | Replace JSON state manager with SQLite | P0 | ✅ Done | Cosmo | Implemented `StateStore` interface + `SQLiteStateStore` backend. `Manager` delegates to store via `NewManagerWithStore()`. 9 new tests pass. WAL mode, write-through. Existing JSON tests unaffected. |
| F3 | Replace auth store with encrypted SQLite | P0 | ✅ Done | Cosmo | Implemented `CredentialStore` interface + `SQLiteCredentialStore` backend. AES-256-GCM encryption with Argon2id key derivation from `OPERATOR_ENCRYPTION_KEY`. Base64 fallback when no key set (with warning). Package-level functions delegate via `SetGlobalCredentialStore()`. 22 new tests pass. |
| F4 | Add structured logging (zerolog) | P0 | ✅ Done | Cosmo | Replaced `pkg/logger` internals with `rs/zerolog`. All 20 existing API functions preserved (Debug/Info/Warn/Error/Fatal × plain/C/F/CF). Added 12 context-aware functions (`*Ctx`) with correlation ID propagation via `WithCorrelationID(ctx, id)`. JSON output via `OPERATOR_LOG_FORMAT=json`, console (default). Level via `OPERATOR_LOG_LEVEL` env var. File logging via multi-writer. 10 new test cases (correlation ID, structured JSON, context functions, env config, file logging). |
| F5 | Add OpenTelemetry metrics | P1 | ✅ Done | Cosmo | Prometheus endpoint at `/metrics` via `prometheus/client_golang`. New `pkg/metrics` package with 11 collectors: LLM request duration/tokens/errors, sessions active/messages, bus messages/queue depth, tool execution duration/count, uptime, info. Convenience helpers (`RecordLLMRequest`, `RecordToolExecution`, `RecordBusMessage`). Instrumented `tools.ToolRegistry.ExecuteWithContext` and `bus.MessageBus.Publish*`. Registered on health server mux. `metrics.Init()` called at gateway startup. 11 tests pass. |
| F6 | Add session TTL and eviction | P1 | ✅ Done | Cosmo | `EvictableStore` interface extends `SessionStore` with `SessionCount`, `DeleteSession`, `EvictExpired`, `EvictLRU`. `SQLiteStore` implements all four. `Evictor` runs periodic background sweeps (TTL then LRU). `DefaultEvictorConfig()`: 24h TTL, 10K max sessions, 5min interval. 14 new tests pass. |
| F7 | Add automated SQLite backup | P1 | ✅ Done | Cosmo | New `pkg/backup` package. `VacuumInto()` for atomic snapshots. `Scheduler` with configurable interval, retention (MaxBackups), and auto-pruning. `ListBackups()` utility. 14 tests pass. |
| F8 | Database migration framework | P1 | ⬜ TODO | — | Embedded SQL migrations with version tracking. Auto-run on startup. |

### Definition of Done — Phase 1
- [ ] All session data persists in SQLite (not JSON files)
- [ ] All state data persists in SQLite
- [x] Credentials encrypted at rest
- [x] Structured JSON logging with correlation IDs
- [x] Prometheus metrics endpoint functional
- [x] Session eviction prevents unbounded memory growth
- [x] Automated backup runs on schedule
- [ ] All existing tests pass
- [ ] New tests cover SQLite stores (≥80% coverage for new code)
- [ ] `make test` passes clean

---

## Phase 2: User Management

### Tasks

| ID | Task | Priority | Status | Notes |
|---|---|---|---|---|
| U1 | Users table + registration API | P0 | ⬜ TODO | `POST /api/v1/auth/register`, email + password (bcrypt) |
| U2 | Login + JWT issuance | P0 | ⬜ TODO | `POST /api/v1/auth/login`, access + refresh tokens |
| U3 | Email verification flow | P1 | ⬜ TODO | Verification token, confirmation endpoint |
| U4 | Tenant-scoped sessions | P0 | ⬜ TODO | Add `tenant_id` to session store, propagate through request lifecycle |
| U5 | Per-user agent configuration | P0 | ⬜ TODO | Users CRUD their own agents with persona, model, tools |
| U6 | Per-user rate limiting | P1 | ⬜ TODO | Token bucket per user, configurable by plan tier |
| U7 | Audit logging | P1 | ⬜ TODO | Structured audit events table: auth, tool exec, config changes |
| U8 | Admin API | P1 | ⬜ TODO | User management, platform config, usage queries |

---

## Phase 3: Billing & Plans

### Tasks

| ID | Task | Priority | Status | Notes |
|---|---|---|---|---|
| B1 | Plan definitions (config-driven) | P0 | ⬜ TODO | Free / Starter / Pro / Enterprise |
| B2 | Stripe integration | P0 | ⬜ TODO | Subscriptions, webhooks, checkout |
| B3 | Token usage metering | P0 | ⬜ TODO | Per-user per-model tracking in `usage_events` table |
| B4 | Usage dashboard API | P1 | ⬜ TODO | Current period usage, historical trends |
| B5 | Overage handling | P1 | ⬜ TODO | Soft caps (warnings) → hard caps (throttle, not cut off) |
| B6 | Plan upgrade/downgrade | P1 | ⬜ TODO | Prorated billing, immediate access changes |

---

## Phase 4: Service Integrations

### Tasks

| ID | Task | Priority | Status | Notes |
|---|---|---|---|---|
| S1 | OAuth 2.0 framework (PKCE) | P0 | ⬜ TODO | Generic OAuth flow for any service |
| S2 | Encrypted credential vault | P0 | ⬜ TODO | Per-user per-integration encrypted token storage |
| S3 | Integration registry (declarative manifests) | P0 | ⬜ TODO | JSON manifest → tools auto-registered |
| S4 | Token refresh manager | P0 | ⬜ TODO | Automatic refresh, concurrent refresh prevention |
| S5 | First integrations: Google (Gmail, Drive, Calendar) | P1 | ⬜ TODO | OAuth + tool definitions |
| S6 | Shopify integration | P1 | ⬜ TODO | OAuth + Admin API tools |
| S7 | Integration management API | P1 | ⬜ TODO | Connect, disconnect, list, status |
| S8 | Per-agent scope narrowing | P1 | ⬜ TODO | Restrict integration access per agent |

---

## Phase 5: Scaling & Reliability

### Tasks

| ID | Task | Priority | Status | Notes |
|---|---|---|---|---|
| R1 | PostgreSQL session/state store (SaaS mode) | P0 | ⬜ TODO | Interface-based swap, connection pooling |
| R2 | NATS JetStream message bus | P0 | ⬜ TODO | Replace in-memory channels, at-least-once delivery |
| R3 | Stateless worker architecture | P0 | ⬜ TODO | Agent loop pulls from queue, reads/writes to DB |
| R4 | Kubernetes Helm chart | P1 | ⬜ TODO | HPA, PDB, resource quotas, ConfigMap/Secrets |
| R5 | Redis session cache | P1 | ⬜ TODO | Hot session caching for latency reduction |
| R6 | Health check improvements | P1 | ⬜ TODO | Component-level checks, dependency health |

---

## Phase 6: Hardening & Launch

### Tasks

| ID | Task | Priority | Status | Notes |
|---|---|---|---|---|
| H1 | Container-level agent sandboxing | P1 | ⬜ TODO | gVisor or Firecracker per agent execution |
| H2 | GDPR compliance toolkit | P1 | ⬜ TODO | Data export, right-to-deletion, retention policies |
| H3 | Load testing | P1 | ⬜ TODO | Target: 1K concurrent users, 10K total |
| H4 | Security audit (external) | P0 | ⬜ TODO | Professional pen testing |
| H5 | API documentation (OpenAPI) | P1 | ⬜ TODO | Full API spec + developer guide |
| H6 | Beta launch | P0 | ⬜ TODO | Limited rollout with monitoring |

---

## Change Log

| Date | Change |
|---|---|
| 2026-03-06 | F7 complete: Automated SQLite backup. New `pkg/backup` package with `VacuumInto()` for atomic snapshots using SQLite's VACUUM INTO. `Scheduler` struct runs periodic backups with configurable interval (default 6h), retention limit (default 7), and automatic pruning of oldest backups. `ListBackups()` lists existing backups sorted chronologically. `Config` struct with `DefaultConfig()`. 14 new tests covering: VacuumInto success/failure, scheduler validation, directory creation, RunOnce, Start/Stop lifecycle, prune logic (over/under limit, non-DB file filtering), list sorting, multiple backups with pruning, backup content verification. |
| 2026-03-06 | F6 complete: Session TTL and eviction. New `EvictableStore` interface with `SessionCount`, `DeleteSession`, `EvictExpired(ttl)`, `EvictLRU(maxSessions)`. SQLiteStore implements all methods (CASCADE deletes for messages). `Evictor` struct runs background goroutine with configurable interval; `RunOnce()` for manual sweeps. `DefaultEvictorConfig()`: 24h TTL, 10K max sessions, 5min sweep. 14 new tests covering: count, delete, TTL eviction, LRU eviction, combined TTL+LRU, no-op cases, start/stop lifecycle, default config. |
| 2026-03-06 | F5 complete: Prometheus metrics endpoint. New `pkg/metrics` package with `prometheus/client_golang`. 11 collectors: LLM (request_duration_seconds histogram, tokens_total counter, errors_total counter), Sessions (active gauge, messages_total counter), Bus (messages_total counter, queue_depth gauge), Tools (execution_duration_seconds histogram, executions_total counter), System (uptime_seconds gauge, info gauge). Instrumented ToolRegistry.ExecuteWithContext and MessageBus.Publish*. Registered `/metrics` on health server. 11 new tests. |
| 2026-03-06 | F4 complete: Structured logging with zerolog. Replaced pkg/logger internals with rs/zerolog while preserving all 20 existing API functions. Added 12 context-aware Ctx functions with correlation ID propagation. JSON/console output modes via OPERATOR_LOG_FORMAT env. Log level via OPERATOR_LOG_LEVEL env. Multi-writer file logging. 10 new tests. |
| 2026-03-06 | F3 complete: Encrypted SQLite credential store with CredentialStore interface, SQLiteCredentialStore implementation, AES-256-GCM + Argon2id encryption. 22 new tests (7 encrypt + 15 store). Package-level functions delegate via SetGlobalCredentialStore(). |
| 2026-03-06 | F2 complete: SQLite state store with StateStore interface, SQLiteStateStore implementation, 9 new tests. Manager delegates to store via NewManagerWithStore(). |
| 2026-03-06 | F1 complete: SQLite session store with SessionStore interface, SQLiteStore implementation, 15 new tests. Fixed pre-existing auth/oauth.go compile error. |
| 2026-03-06 | Initial assessment completed. Branch `operatoros-production-readiness` created. STATUS.md and CLAUDE.md written. |
