# Operator OS — Production Readiness Status

**Last Updated:** 2026-03-07
**Current Phase:** 2 — User Management
**Overall Progress:** 21%

---

## Phase Overview

| # | Phase | Status | Target | Progress |
|---|---|---|---|---|
| 1 | Foundation (SQLite, logging, metrics, encryption) | ✅ Done | Weeks 1–4 | 100% |
| 2 | User Management (accounts, tenancy, auth) | 🟡 In Progress | Weeks 5–8 | 75% |
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
| F8 | Database migration framework | P1 | ✅ Done | Cosmo | New `pkg/dbmigrate` package. Embedded SQL migrations with version tracking in `schema_migrations` table. `Migrator` loads `.sql` files from `embed.FS`, runs pending migrations in version-ordered transactions, skips already-applied. `AutoMigrate(db)` convenience for startup. 3 built-in migrations (sessions, state, credentials). `NewFromList()` for programmatic use. 17 tests pass. |

### Definition of Done — Phase 1
- [x] All session data persists in SQLite (not JSON files)
- [x] All state data persists in SQLite
- [x] Credentials encrypted at rest
- [x] Structured JSON logging with correlation IDs
- [x] Prometheus metrics endpoint functional
- [x] Session eviction prevents unbounded memory growth
- [x] Automated backup runs on schedule
- [x] Database migration framework with version tracking
- [x] All existing tests pass
- [x] New tests cover SQLite stores (≥80% coverage for new code)
- [x] `make test` passes clean

---

## Phase 2: User Management

### Tasks

| ID | Task | Priority | Status | Notes |
|---|---|---|---|---|
| U1 | Users table + registration API | P0 | ✅ Done | New `pkg/users` package. `UserStore` interface + `SQLiteUserStore` backend. `POST /api/v1/auth/register` with email validation, bcrypt hashing (cost 12), case-insensitive email. Migration `004_create_users.sql`. 28 new tests (18 store + 10 API). |
| U2 | Login + JWT issuance | P0 | ✅ Done | `POST /api/v1/auth/login` + `POST /api/v1/auth/refresh`. `TokenService` with HMAC-SHA256 (golang-jwt/jwt/v5). Access tokens (15min) + refresh tokens (7d). `AuthMiddleware` for protected routes. Context helpers (`UserIDFromContext`, `EmailFromContext`, `ClaimsFromContext`). Account status checks (suspended/deleted). Anti-enumeration error messages. 28 new tests (JWT: 18, Login API: 10). |
| U3 | Email verification flow | P1 | ✅ Done | New `VerificationStore` interface + `SQLiteVerificationStore` backend. Crypto-random hex tokens (32 bytes) with configurable TTL (default 24h). `VerifyEmail()` validates token existence/expiry/usage, marks user verified, transitions status pending→active. `ResendVerification()` with cooldown enforcement (default 60s). API: `POST /api/v1/auth/verify-email` (token validation with distinct error codes: not_found/expired/used/already_verified), `POST /api/v1/auth/resend-verification` (anti-enumeration: returns 200 for nonexistent emails). `DeleteExpired()` for token cleanup. Migration `007_create_verification_tokens.sql`. `NewAPIFull()` constructor. 32 new tests (store: 10, function: 6, API: 14, persistence: 1, flow: 1). |
| U4 | Tenant-scoped sessions | P0 | ✅ Done | New `TenantStore` implements `SessionStore` with full tenant isolation. Scoped key format `tenant:<id>:<key>`. `TenantStoreFactory` creates stores per-tenant from shared DB. Context helpers `WithTenantID`/`TenantIDFromContext`. Migration `005_add_tenant_id.sql` adds `tenant_id` column + indexes. `EvictableStore` methods (SessionCount, Delete, EvictExpired, EvictLRU) all tenant-scoped. `ListSessions()` returns unscoped keys. 25 new tests. |
| U5 | Per-user agent configuration | P0 | ✅ Done | New `pkg/agents` package. `UserAgentStore` interface + `SQLiteUserAgentStore` backend. Full CRUD API: `GET/POST /api/v1/agents`, `GET/PUT/DELETE /api/v1/agents/{id}`, `POST /api/v1/agents/{id}/default`. Per-user isolation enforced at API layer. Agents have: name, description, system_prompt (persona), model, model_fallbacks, tools, skills, max_tokens, temperature, max_iterations, is_default, status. Unique name per user. Max 50 agents/user. Migration `006_create_user_agents.sql`. 40 new tests (22 store + 18 API). |
| U6 | Per-user rate limiting | P1 | ✅ Done | New `pkg/ratelimit` package. Token bucket algorithm with configurable plan tiers (free/starter/pro/enterprise). `Limiter` with in-memory fast path + `SQLiteRateLimitStore` persistence. HTTP `Middleware` adds rate limit headers (X-RateLimit-Limit/Remaining/Daily-*) and returns 429 with Retry-After. `StatusHandler` for `GET /api/v1/rate-limit/status`. `PersistMiddleware` for periodic state saving. Per-minute burst + daily rolling limits. Plan upgrade resets bucket. Migration `008_create_rate_limits.sql`. 34 new tests. |
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
| 2026-03-07 | U6 complete: Per-user rate limiting. New `pkg/ratelimit` package with token bucket algorithm. `Limiter` struct provides in-memory per-user rate limiting with configurable plan tiers: free (10 RPM, burst 15, 500/day), starter (30 RPM, burst 50, 5K/day), pro (60 RPM, burst 100, 50K/day), enterprise (120 RPM, burst 200, unlimited). `SQLiteRateLimitStore` backend for state persistence across restarts via `SaveBucket`/`LoadBucket`/`DeleteBucket`. HTTP `Middleware` integrates with `AuthMiddleware` context, adds `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Daily-Limit`, `X-RateLimit-Daily-Remaining`, `X-RateLimit-Reset` headers, returns 429 with `Retry-After` on limit. `StatusHandler` at `GET /api/v1/rate-limit/status` returns full status JSON. `PersistMiddleware` runs periodic background save with graceful shutdown. `AllowAt()` for time-controlled testing. `SetUserPlan` resets bucket on plan change (upgrade/downgrade). `RestoreUser` for loading from store on-demand. `RemoveUser` cleans up memory and store. Migration `008_create_rate_limits.sql`. 34 new tests covering: token bucket exhaustion/refill, daily limits/reset, burst, plan upgrade, multi-user isolation, SQLite CRUD/upsert/delete, persist+restore integration, middleware allow/block/no-auth/no-plan, status handler auth/no-auth/no-plan, persist middleware lifecycle, enterprise unlimited, format helpers. |
| 2026-03-07 | U3 complete: Email verification flow. New `VerificationStore` interface + `SQLiteVerificationStore` backend in `pkg/users/verification.go`. Crypto-random 32-byte hex tokens with configurable TTL (default 24h). `VerifyEmail()` function validates token existence, expiry, and usage status; marks user's email as verified and transitions status from `pending_verification` to `active`. `ResendVerification()` with configurable cooldown enforcement (default 60s) prevents spam. API endpoints: `POST /api/v1/auth/verify-email` with distinct HTTP status codes per error (404 not found, 410 expired, 409 used/already verified), `POST /api/v1/auth/resend-verification` with anti-enumeration (returns 200 OK for nonexistent emails, 409 for already verified, 429 for cooldown). `DeleteExpired()` for periodic token cleanup. `NewAPIFull()` constructor for API with all services. Migration `007_create_verification_tokens.sql` with indexes on user_id and token. 32 new tests covering: store CRUD (create, get, not found, mark used, last token time, delete expired), verification logic (success, not found, expired, used, already verified), resend logic (success, already verified, cooldown, user not found), API handlers (verify success/missing/invalid JSON/not found/expired/used/no store, resend success/missing email/nonexistent/already verified/invalid JSON/no store), persistence across DB reopen, full registration→verification flow. |
| 2026-03-07 | U5 complete: Per-user agent configuration. New `pkg/agents` package with `UserAgentStore` interface + `SQLiteUserAgentStore` backend. Full REST API: `GET/POST /api/v1/agents`, `GET/PUT/DELETE /api/v1/agents/{id}`, `POST /api/v1/agents/{id}/default`. Each agent stores: name (unique per user), description, system_prompt (persona, up to 50K chars), model, model_fallbacks, tools, skills, max_tokens, temperature, max_iterations, is_default, status (active/archived). Per-user isolation enforced at API layer. Max 50 agents per user. Partial update support via PUT (only specified fields change). `SetDefault` uses transaction to atomically clear old + set new default. Migration `006_create_user_agents.sql` with composite unique index on (user_id, name). Exported `ContextKeyUserID()` from `pkg/users` for cross-package context injection. 40 new tests: store (22) — create, duplicate name, same-name-different-users, get by ID, not found, update, update not found, update duplicate name, delete, delete not found, list by user, count by user, get default, get default not found, set default, set default not found, set default wrong user, nil temperature, empty slices, custom ID, persistence, marshal/unmarshal helpers; API (18) — create, missing name, name too long, duplicate, invalid JSON, get, get not found, list, list empty, update, update not found, invalid status, delete, delete not found, set default, set default not found, temperature, prompt too long. |
| 2026-03-07 | U4 complete: Tenant-scoped sessions. New `TenantStore` in `pkg/session/tenant.go` implements `SessionStore` with full tenant-level isolation. All operations scoped via `tenant_id` column and composite key format `tenant:<tenantID>:<originalKey>`. `TenantStoreFactory` creates per-tenant stores from a shared `*sql.DB`. Context propagation helpers `WithTenantID(ctx, id)` / `TenantIDFromContext(ctx)`. Migration `005_add_tenant_id.sql` adds `tenant_id TEXT NOT NULL DEFAULT ''` column + `idx_sessions_tenant` and `idx_sessions_tenant_updated` indexes. Implements `EvictableStore` methods (SessionCount, DeleteSession, EvictExpired, EvictLRU) all tenant-scoped. `ListSessions()` returns unscoped original keys. Updated `initSchema` in `SQLiteStore` to include `tenant_id` column for new databases. 25 new tests covering: context helpers, tenant isolation (cross-tenant read/write/delete blocked), CRUD operations, summary, set/truncate history, session count, eviction (TTL + LRU with cross-tenant safety), list sessions, factory, tool calls round-trip, save/close no-ops, validation. |
| 2026-03-07 | U2 complete: Login + JWT issuance. New `jwt.go` with `TokenService` (HMAC-SHA256 via `golang-jwt/jwt/v5`), `TokenClaims` extending `jwt.RegisteredClaims` with `uid`/`email`/`type` claims, `IssueTokenPair()` for access (15min) + refresh (7d) tokens, `ValidateAccessToken()`/`ValidateRefreshToken()` type-checked validation. New `middleware.go` with `AuthMiddleware` (extracts Bearer token, validates, injects claims into context), `UserIDFromContext()`/`EmailFromContext()`/`ClaimsFromContext()` helpers. Updated `api.go`: `NewAPIWithAuth()` constructor, `POST /api/v1/auth/login` (email lookup, status checks for suspended/deleted, bcrypt verify, token issuance, anti-enumeration errors), `POST /api/v1/auth/refresh` (validates refresh token type, re-checks user existence/status, issues new pair). 28 new tests in `jwt_test.go` (service creation, empty/nil key, options, issuance, access/refresh validation, wrong key, expired, type rejection, unique JTI, middleware valid/missing/invalid/refresh-rejected/malformed, context helpers) and `login_test.go` (success, wrong password, nonexistent user, missing email/password, invalid JSON, case-insensitive, suspended, deleted, no token service, refresh success/invalid/access-rejected/missing/suspended/deleted/invalid-JSON/no-service, full auth flow). |
| 2026-03-06 | U1 complete: Users table + registration API. New `pkg/users` package with `UserStore` interface, `SQLiteUserStore` implementation (CRUD, case-insensitive email, UUID generation), `API` HTTP handler for `POST /api/v1/auth/register` (email validation via net/mail, bcrypt cost-12 hashing, password strength check ≥8 chars). Migration `004_create_users.sql` adds users table with status, email_verified, indexes. 28 new tests (store: create, duplicate, case-insensitive, get by ID/email, update, delete, list, count, persistence, custom ID, shared DB; API: success, duplicate, missing email, invalid email, weak password, invalid JSON, case normalization, empty password, whitespace trimming; password: hash, match, mismatch, validation table-driven). |
| 2026-03-06 | F8 complete: Database migration framework. New `pkg/dbmigrate` package with embedded SQL migrations, `schema_migrations` version tracking table, transactional per-migration execution, idempotent `Up()`, `AutoMigrate()` convenience. 3 built-in migrations consolidating existing schemas (sessions, state, credentials). `Migrator` supports both `embed.FS` and programmatic `NewFromList()`. 17 new tests covering: nil DB, bad dir, duplicates, full apply, idempotency, incremental, applied/pending/version queries, failed migration rollback, non-SQL file filtering, embedded migrations, FK-dependent ordering. **Phase 1 complete.** |
| 2026-03-06 | F7 complete: Automated SQLite backup. New `pkg/backup` package with `VacuumInto()` for atomic snapshots using SQLite's VACUUM INTO. `Scheduler` struct runs periodic backups with configurable interval (default 6h), retention limit (default 7), and automatic pruning of oldest backups. `ListBackups()` lists existing backups sorted chronologically. `Config` struct with `DefaultConfig()`. 14 new tests covering: VacuumInto success/failure, scheduler validation, directory creation, RunOnce, Start/Stop lifecycle, prune logic (over/under limit, non-DB file filtering), list sorting, multiple backups with pruning, backup content verification. |
| 2026-03-06 | F6 complete: Session TTL and eviction. New `EvictableStore` interface with `SessionCount`, `DeleteSession`, `EvictExpired(ttl)`, `EvictLRU(maxSessions)`. SQLiteStore implements all methods (CASCADE deletes for messages). `Evictor` struct runs background goroutine with configurable interval; `RunOnce()` for manual sweeps. `DefaultEvictorConfig()`: 24h TTL, 10K max sessions, 5min sweep. 14 new tests covering: count, delete, TTL eviction, LRU eviction, combined TTL+LRU, no-op cases, start/stop lifecycle, default config. |
| 2026-03-06 | F5 complete: Prometheus metrics endpoint. New `pkg/metrics` package with `prometheus/client_golang`. 11 collectors: LLM (request_duration_seconds histogram, tokens_total counter, errors_total counter), Sessions (active gauge, messages_total counter), Bus (messages_total counter, queue_depth gauge), Tools (execution_duration_seconds histogram, executions_total counter), System (uptime_seconds gauge, info gauge). Instrumented ToolRegistry.ExecuteWithContext and MessageBus.Publish*. Registered `/metrics` on health server. 11 new tests. |
| 2026-03-06 | F4 complete: Structured logging with zerolog. Replaced pkg/logger internals with rs/zerolog while preserving all 20 existing API functions. Added 12 context-aware Ctx functions with correlation ID propagation. JSON/console output modes via OPERATOR_LOG_FORMAT env. Log level via OPERATOR_LOG_LEVEL env. Multi-writer file logging. 10 new tests. |
| 2026-03-06 | F3 complete: Encrypted SQLite credential store with CredentialStore interface, SQLiteCredentialStore implementation, AES-256-GCM + Argon2id encryption. 22 new tests (7 encrypt + 15 store). Package-level functions delegate via SetGlobalCredentialStore(). |
| 2026-03-06 | F2 complete: SQLite state store with StateStore interface, SQLiteStateStore implementation, 9 new tests. Manager delegates to store via NewManagerWithStore(). |
| 2026-03-06 | F1 complete: SQLite session store with SessionStore interface, SQLiteStore implementation, 15 new tests. Fixed pre-existing auth/oauth.go compile error. |
| 2026-03-06 | Initial assessment completed. Branch `operatoros-production-readiness` created. STATUS.md and CLAUDE.md written. |
