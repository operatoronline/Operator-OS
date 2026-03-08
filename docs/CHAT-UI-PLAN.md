# Operator OS — Chat UI Workstream

**Created:** 2026-03-08
**Status:** Planning
**Branch:** `feat/chat-ui` (based on `operatoros-production-readiness`)
**Location:** `/var/www/prototypes/os-go` → `os-go.operator.onl`
**Target:** Production-ready chat interface for Operator OS platform

---

## Overview

Evolve the existing chat interface into a full production platform client. The current `web/index.html` (1568 lines) already provides a functional foundation — dark/light OKLCH theming, chat bubbles with markdown rendering, WebSocket transport, and a monitor panel. This workstream migrates it from the legacy Pico protocol to the production API, adds authentication, and surfaces all platform features (billing, integrations, admin).

### What Already Exists (`web/index.html`)
- ✅ Chat message UI (user/agent/system bubbles, animations)
- ✅ Markdown rendering (marked.js + DOMPurify)
- ✅ Code blocks with syntax styling
- ✅ Dark/light theme with full OKLCH token system
- ✅ WebSocket transport (currently `/pico/ws` with hardcoded token)
- ✅ Input composer with send button
- ✅ Monitor panel (connection status, health, browser iframe)
- ✅ Responsive layout with pill navigation
- ✅ DM Sans + JetBrains Mono typography
- ✅ Phosphor Icons

### What Needs to Change
- ❌ Hardcoded Pico token → JWT auth (login/register flows)
- ❌ `/pico/ws` protocol → production `/api/v1/ws` with JWT handshake
- ❌ Single-file monolith → modular structure (can stay vanilla JS or migrate to React)
- ❌ No session management → multi-session with history
- ❌ No agent selection → agent CRUD and switching
- ❌ No platform features → billing, integrations, admin panels
- ❌ No error handling → proper error states, reconnect UI, empty states

**Stack:** React 19 + TypeScript + Vite. The existing vanilla JS is a reference for styling/UX, not a codebase to extend — platform features (browser, integrations, billing, admin) demand component architecture and shared state from day one.
**Styling:** Tailwind CSS v4 with the existing OKLCH token system ported as CSS custom properties.
**State:** Zustand (lightweight, no boilerplate) for auth, sessions, agents, WebSocket.
**Real-time:** WebSocket migrated from Pico protocol to production `/api/v1/ws` with JWT.
**Auth:** JWT (login/register/verify flows already built in backend).
**Deployment:** Vite build → `web/dist/` → Caddy at `os-go.operator.onl`

---

## Architecture

```
┌─────────────────────────────────────────────────┐
│                   Chat UI (SPA)                 │
│                                                 │
│  ┌──────────┐ ┌──────────┐ ┌────────────────┐  │
│  │  Auth     │ │  Chat    │ │  Dashboard     │  │
│  │  Module   │ │  Module  │ │  Module        │  │
│  └──────────┘ └──────────┘ └────────────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌────────────────┐  │
│  │  Agents  │ │  Billing │ │  Integrations  │  │
│  │  Module  │ │  Module  │ │  Module        │  │
│  └──────────┘ └──────────┘ └────────────────┘  │
│  ┌──────────┐ ┌──────────┐                      │
│  │  Admin   │ │  Settings│                      │
│  │  Module  │ │  Module  │                      │
│  └──────────┘ └──────────┘                      │
└───────────────────┬─────────────────────────────┘
                    │ HTTPS + WSS
┌───────────────────▼─────────────────────────────┐
│              Operator OS Gateway                │
│         (Go API — already built)                │
│                                                 │
│  60+ REST endpoints across 15 API groups        │
│  JWT auth · Stripe billing · OAuth integrations │
└─────────────────────────────────────────────────┘
```

---

## Phase Overview

| # | Phase | Description | Tasks | Target |
|---|---|---|---|---|
| 1 | Foundation | Project scaffold, auth, routing, API client | C1–C5 | Week 1–2 |
| 2 | Chat Core | Real-time messaging, markdown, streaming | C6–C10 | Week 3–4 |
| 3 | Agent & Session Management | Multi-agent, sessions, history | C11–C14 | Week 5–6 |
| 4 | Platform Features | Billing, integrations, usage dashboard | C15–C19 | Week 7–8 |
| 5 | Admin & Settings | Admin panel, user management, security audit | C20–C23 | Week 9–10 |
| 6 | Polish & Launch | Mobile responsive, a11y, performance, deploy | C24–C28 | Week 11–12 |

---

## Phase 1: Foundation

| ID | Task | Priority | Status | Description |
|---|---|---|---|---|
| C1 | React scaffold | P0 | ✅ DONE | `web/` → Vite + React 19 + TypeScript. Tailwind v4 with OKLCH tokens ported from existing `index.html`. Directory: `src/{components,pages,hooks,services,stores,types}`. DM Sans + JetBrains Mono + Phosphor Icons carried over. Archive `index.html` → `web/legacy/index.html` as reference. **Done 2026-03-08:** Full scaffold with Vite 6, React 19, TS, Tailwind v4 `@theme` mapping all OKLCH tokens. AppShell with floating pill nav (desktop) + bottom tabs (mobile). Zustand uiStore for theme toggle. Stub pages for all 6 routes. Login page shell. Build passes clean (273 KB JS gzipped to 85 KB). |
| C2 | API client + types | P0 | ✅ DONE | `src/services/api.ts` — typed fetch client for all 60+ backend endpoints. Auto-attach JWT. Refresh token interceptor. Error normalization with typed error responses. Generate request/response types from OpenAPI spec (`/api/v1/docs/openapi.json`). **Done 2026-03-08:** Full TypeScript types in `src/types/api.ts` (36 interfaces, 8 union types covering all OpenAPI schemas). API client in `src/services/api.ts` with: namespaced `api.*` object (auth, agents, billing, usage, integrations, userIntegrations, oauth, admin, audit, gdpr, rateLimit, health), JWT auto-attach via `tokenStore`, deduped refresh-token interceptor on 401, `ApiRequestError` class with status/code/body, query param builder, `os:auth:expired` custom event for store-level redirect. Build clean. |
| C3 | Auth flows | P0 | ✅ DONE | `src/pages/{Login,Register,Verify}.tsx`. Zustand auth store (user, tokens, isAuthenticated). `<ProtectedRoute>` wrapper. Redirect to `/login` on 401. Calls: `POST /auth/register`, `POST /auth/login`, `POST /auth/verify-email`, `POST /auth/resend-verification`, `POST /auth/refresh`. **Done 2026-03-08:** Full auth store in `src/stores/authStore.ts` with login, register, logout, verifyEmail, resendVerification, initialize (restores session from stored JWT, auto-refresh on expiry). JWT payload parsing for session restore without API call. `ProtectedRoute` component with loading spinner and return-to redirect. Login page with error banners, loading states, register link. Register page with password confirmation, min-length validation, redirects to verify. Verify page handles both post-registration (check your email + resend) and token verification (?token=xxx with auto-verify). AppShell updated with SignOut button + user email tooltip. `os:auth:expired` event listener auto-logs out. Build clean (294 KB JS → 90 KB gzipped). |
| C4 | App shell & routing | P0 | ✅ DONE | `<AppShell>` with collapsible sidebar, top bar (user menu, theme toggle, logout). React Router v7. Routes: `/chat`, `/agents`, `/integrations`, `/billing`, `/settings`, `/admin`. Mobile: bottom tab nav mirroring the existing pill pattern. **Done 2026-03-08:** Refactored AppShell into composed layout: `Sidebar` (desktop, collapsible with localStorage persistence, tooltip labels when collapsed, branded OS logo), `TopBar` (page title from route, theme toggle, user dropdown menu with avatar + email + sign out), `BottomTabs` (mobile, 5 primary routes with filled/regular icon states), `MobileSidebar` (slide-over panel with backdrop blur, body scroll lock, Escape to close, all 6 routes). Sidebar collapse state persisted to localStorage. Build clean (310 KB JS → 93 KB gzipped). |
| C5 | Theme system | P1 | ✅ DONE | Port existing OKLCH tokens to Tailwind config + CSS custom properties. Dark/light toggle with system preference detection. Persist to localStorage. Same visual identity as current UI — just in React/Tailwind. **Done 2026-03-08:** `useTheme` hook with live `prefers-color-scheme` listener + `followSystem()` reset. Smooth theme transition via `theme-transitioning` class (0.3s on all token-driven properties). Glass morphism utilities (`.glass`, `.glass-subtle`). Form input base styles inheriting OKLCH tokens. Focus ring utility (`.focus-ring`). 7 animation keyframes (fadeSlide, fadeSlideDown, fadeIn, scaleIn, slideInLeft, slideInRight, pulse-glow) with utility classes. `color-scheme` set on root for native controls. `prefers-reduced-motion` media query. `::selection` styled with accent color. Shared primitives: `Button` (4 variants × 3 sizes, loading state), `Badge` (5 semantic variants, optional dot), `Input` (label, error, icon slot, ARIA). Build clean (311 KB JS → 94 KB gzipped). |

---

## Phase 2: Chat Core

| ID | Task | Priority | Status | Description |
|---|---|---|---|---|
| C6 | WebSocket transport | P0 | ✅ DONE | Backend: add `/api/v1/ws` endpoint with JWT auth on handshake. Frontend: `src/services/ws.ts` + `useWebSocket` hook. Auto-reconnect with exponential backoff. Connection state in Zustand store. Message send/receive typed. Reference existing `/pico/ws` protocol for UX patterns. **Done 2026-03-08:** `WebSocketManager` class in `src/services/ws.ts` — typed protocol (`src/types/ws.ts` with 7 inbound + 3 outbound message types), JWT via query param, exponential backoff + jitter (1s→30s), 25s ping keepalive, auth failure detection (4001/4003 close codes). `chatStore.ts` bridges WS events to Zustand (messages, typing, connection state, optimistic sends, streaming updates, cancel). `useWebSocket` hook auto-connects on auth. `ConnectionStatus` component (4 states: dot/icon/label modes). Chat page updated with connection banner, empty state, stub composer. Build clean (334 KB → 100 KB gzipped). |
| C7 | Message thread UI | P0 | ✅ DONE | `<MessageList>`, `<MessageBubble>` (user/agent/system variants), `<ScrollToBottom>`. Port existing bubble styles + animations from legacy CSS. Add timestamps, loading skeleton for history. Virtualized rendering (`react-window`) for long threads. **Done 2026-03-08:** Five components in `src/components/chat/`: `MessageBubble` (memo'd, user/agent/system variants with legacy OKLCH styling — user gets bordered pill with `user-bg`/`user-border`, agent gets clean minimal text, system centered dim), `MessageList` (auto-scroll with 80px threshold, unread counter, timestamp grouping at 2min gaps or role change), `ScrollToBottom` (glass-morphism floating button with unread badge), `TypingIndicator` (three-dot bounce), `MessageSkeleton` (conversation-shaped loading shimmer). Blinking cursor animation for streaming messages. Chat page refactored to use MessageList with proper flex layout (no overflow issues). Virtualization deferred to C26 (Performance). Build clean (339 KB JS → 101 KB gzipped). |
| C8 | Markdown & code rendering | P0 | ✅ DONE | `<MarkdownRenderer>` using `react-markdown` + `rehype-highlight` + `DOMPurify`. Code blocks with copy button + language label. Tables, blockquotes, inline code — matching existing visual treatment. **Done 2026-03-08:** `MarkdownRenderer` component with react-markdown v10, remark-gfm, rehype-highlight, DOMPurify sanitization (allowlisted tags). `CodeBlock` component with language header bar + copy-to-clipboard (Check icon on success). Custom OKLCH-based hljs theme (`src/styles/hljs.css`) adapting to dark/light. All block elements styled: headings (3 levels), blockquotes (accent left border), tables (surface-2 header, border), lists, inline code (surface-2 bg, accent-text, mono), links (accent-text, new tab), hr. Streaming mode skips rehype-highlight for perf. MessageBubble updated to render agent messages through MarkdownRenderer. Build: 706 KB JS → 214 KB gzipped (code-split in C26). |
| C9 | Streaming responses | P0 | ✅ DONE | WebSocket streaming handler for token-by-token display. `<TypingIndicator>` animation. Cancel generation button. Incremental markdown rendering during stream. **Done 2026-03-08:** Added `streamingMessageId` tracking to chatStore for active stream state management. MessageList auto-scrolls during streaming content growth (instant scroll to avoid jitter). "Stop generating" pill button with Stop icon, animated in, cancels WS stream and finalizes message locally with `cancelled` flag. MessageBubble shows "stopped" label on cancelled messages. Composer disables during streaming with "Waiting for response…" placeholder. Cancel sends `message.cancel` with the streaming message ID. Streaming ↔ typing indicator properly coordinated (typing dots show before first token, hide once streaming starts). Build clean (709 KB JS → 215 KB gzipped). |
| C10 | Input composer | P1 | ✅ DONE | `<Composer>` — auto-resizing textarea, file/image upload (drag-and-drop + clipboard paste), preview thumbnails. Send on Enter, newline on Shift+Enter. Model selector dropdown. Agent selector. **Done 2026-03-08:** Full Composer component in `src/components/chat/Composer.tsx` replacing ComposerStub. Auto-resizing textarea (max 160px), Enter to send / Shift+Enter for newline, IME composition awareness. File attachments via click (Paperclip button), drag-and-drop (ring highlight + drop overlay), and clipboard paste (images). Preview thumbnails (64×64 images) and file cards with name+size. Max 5 attachments / 10MB each. Model selector dropdown (glass popup, bottom-anchored). Send button with PaperPlaneRight icon matching legacy accent treatment. Glass morphism container (`.glass` class) matching legacy floating input. Hint text shows file count or keyboard shortcuts. Agent selector deferred to C11 (Agent CRUD). Build clean (723 KB JS → 218 KB gzipped). |

---

## Phase 3: Agent & Session Management

| ID | Task | Priority | Status | Description |
|---|---|---|---|---|
| C11 | Agent CRUD | P0 | ✅ DONE | List agents, create/edit/delete. Agent card showing name, model, system prompt preview, status, integration scopes. Set default agent. Calls: `GET/POST /api/v1/agents`, `GET/PUT/DELETE /api/v1/agents/{id}`, `POST /api/v1/agents/{id}/default`. **Done 2026-03-08:** Zustand `agentStore` with full CRUD (fetch, create, update, delete, setDefault, selectAgent). `AgentCard` component (memo'd, 3-dot action menu with edit/delete/set-default, model label, prompt preview, tool/skill/integration counts, default badge, archived state). `AgentEditor` modal (create/edit form with name, description, system prompt textarea, model dropdown, temperature/max_tokens/max_iterations, tools/skills as CSV, client validation). `AgentList` with responsive grid (1/2/3 cols), loading skeleton, error state with retry, empty state with CTA. `AgentsPage` wires it all: fetch on mount, filter pills (All/Active/Archived), create button, error banner, `ConfirmDialog` for delete. New shared components: `Modal` (backdrop blur, ESC close, scroll lock, focus), `ConfirmDialog` (danger/primary variants). Build clean (755 KB JS → 225 KB gzipped). |
| C12 | Multi-session UI | P0 | ✅ DONE | Session sidebar: list active sessions, create new, rename, delete. Session = conversation thread tied to an agent. Switch between sessions without losing state. Session metadata (created, message count, last active). **Done 2026-03-08:** Session types (`Session`, `CreateSessionRequest`, `UpdateSessionRequest`, `SessionMessage`) + `api.sessions` namespace (list/get/create/update/delete/messages). `sessionStore` (Zustand) with CRUD, sorting (pinned→recency), rename tracking. `SessionItem` (memo'd) with inline rename, 3-dot context menu (pin/rename/delete), relative timestamps, active/pinned indicators. `SessionPanel` — desktop: inline 256px sidebar; mobile: slide-over overlay with backdrop blur — search filter, pinned/recent sections, create button, empty state with CTA, delete confirmation dialog. `chatStore` updated with `loadSessionHistory` (paginated fetch, race-condition guard), `loadingHistory`/`historyError` state. Chat page refactored: session panel as left panel, chat header bar (session name + mobile hamburger + connection dot), loading/error/empty states for history, composer hidden when no session. `.gitignore` fix for `/sessions/` root-only pattern. Build clean (786 KB → 232 KB gzipped). |
| C13 | Conversation history | P1 | ✅ DONE | Search across sessions. Filter by agent, date range. Export conversation as markdown/JSON. Pin important conversations. Archive old sessions. **Done 2026-03-08:** SessionPanel extended with filter bar (agent dropdown, date-from/date-to pickers), archive toggle (view archived sessions), and active filter indicator. SessionItem context menu expanded with Archive/Unarchive, Export submenu (Markdown + JSON). Export utility (`src/utils/exportSession.ts`) fetches all paginated messages and downloads as `.md` or `.json` with proper formatting. `useMemo`-based filtering pipeline (archived → text search → agent → date range). Pin was already implemented in C12. Build clean (810 KB JS → 236 KB gzipped). |
| C14 | Agent integration scopes | P1 | ✅ DONE | Per-agent integration permission editor. Visual scope selector showing available integrations, tools, and OAuth scopes. Calls: `AllowedIntegrations` field on agent create/update. **Done 2026-03-08:** `ScopeSelector` component (`src/components/agents/ScopeSelector.tsx`) — visual toggle for each available integration with checkbox enable/disable, expandable detail panel showing per-tool toggle chips (select all/deselect all), search filter, category grouping, tool count indicator. Integrated into `AgentEditor` modal: fetches available integrations on open, maps `allowed_integrations` through form state, includes in create/update payload. `AgentCard` footer updated to show integration + tool count with tooltip. OKLCH styling consistent with existing components — accent border/bg for enabled integrations, glass-style chips for tools. Build clean (833 KB JS → 241 KB gzipped). Phase 3 complete. |

---

## Phase 4: Platform Features

| ID | Task | Priority | Status | Description |
|---|---|---|---|---|
| C15 | Billing & plans | P0 | ✅ DONE | Plan comparison page (Free/Starter/Pro/Enterprise). Current plan badge. Upgrade/downgrade with proration preview. Stripe Checkout redirect. Billing portal link. Calls: `GET /api/v1/billing/plans`, `POST /billing/checkout`, `POST /billing/portal`, `GET /billing/subscription`, `POST /billing/change-plan`, `POST /billing/preview-change`. **Done 2026-03-08:** Zustand `billingStore` with full plan/subscription lifecycle (fetch plans, fetch subscription, Stripe checkout redirect, portal redirect, preview proration, confirm change). `PlanCard` component (memo'd, 4 plan variants with icons/colors, feature list from PlanLimits, current/recommended badges, yearly savings %, contextual button states). `CurrentSubscription` banner (plan name, status badge, billing interval, renewal date, cancel/past-due warnings, Manage button → Stripe portal). `PlanChangePreview` modal (from→to direction, proration amount, timing info, downgrade warning). `IntervalToggle` (monthly/yearly pill toggle with savings indicator). `BillingPage` with Stripe redirect banners (success/canceled), error handling, loading skeletons, responsive 4-col grid. Also fixed pre-existing TS errors in ScopeSelector and SessionPanel. Build clean (875 KB JS → 251 KB gzipped). |
| C16 | Usage dashboard | P0 | ✅ DONE | Token usage charts (daily, by model). Current period summary. Usage vs plan limits with progress bars. Overage warnings. Calls: `GET /billing/usage`, `GET /billing/usage/daily`, `GET /billing/usage/models`, `GET /billing/usage/limits`, `GET /billing/overage`. **Done 2026-03-08:** Zustand `usageStore` with 5 parallel fetchers (summary, daily, byModel, limits, overage) + `fetchAll`. Five components in `src/components/usage/`: `SummaryCards` (4 stat cards — total tokens, input, output, requests — with OKLCH-themed icons + breakdown subs), `DailyChart` (pure-CSS stacked bar chart — last 14 days, input/output colored, hover tooltips, y-axis labels, legend, responsive), `ModelBreakdown` (sorted model list with proportional progress bars, OKLCH hue rotation per model, in/out/req/cost detail), `LimitsBar` (tokens + messages vs plan limits with progress bars, 80%/95% warning thresholds, unlimited support), `OverageWarning` (severity-colored banner for 5 overage levels — none/warning/soft_cap/hard_cap/blocked — with per-resource detail + upgrade CTA). Billing page refactored with Plans/Usage tab switcher (pill toggle). Usage tab lazy-fetches on activation. Build clean (917 KB JS → 260 KB gzipped). |
| C17 | Integration marketplace | P1 | ⬜ TODO | Browse available integrations by category. Connect/disconnect OAuth integrations (Google, Shopify). API key integrations. Status indicators (active/failed/revoked). Token health display. Calls: `GET /integrations`, `GET /integrations/categories`, `POST /manage/integrations/connect`, `POST /manage/integrations/disconnect`, `GET /manage/integrations/status`. |
| C18 | OAuth connect flow | P1 | ⬜ TODO | In-app OAuth popup/redirect for Google, Shopify. Callback handling. Scope consent display. Reconnect for expired/revoked tokens. Calls: `POST /oauth/authorize`, `GET /oauth/callback`. |
| C19 | Rate limit display | P2 | ⬜ TODO | Show current rate limit status from `X-RateLimit-*` response headers. Visual indicator when approaching limits. Calls: `GET /api/v1/rate-limit/status`. |

---

## Phase 5: Admin & Settings

| ID | Task | Priority | Status | Description |
|---|---|---|---|---|
| C20 | Admin panel | P1 | ⬜ TODO | User list with search/filter. Suspend/activate/delete users. Role management (user/admin). Platform stats dashboard. Requires admin role. Calls: `GET/PUT/DELETE /admin/users/*`, `POST /admin/users/{id}/suspend`, `POST /admin/users/{id}/activate`, `POST /admin/users/{id}/role`, `GET /admin/stats`. |
| C21 | Audit log viewer | P1 | ⬜ TODO | Filterable event log (by user, action, time range). Action categories (auth, agent, billing, admin). Expandable detail rows. CSV export. Calls: `GET /admin/audit`, `GET /admin/audit/count`. |
| C22 | Security audit dashboard | P2 | ⬜ TODO | Run security audit from UI. Risk score visualization (gauge). Check results grouped by category with pass/fail/warning. Remediation guidance. CWE/OWASP references. Calls: `GET /admin/security-audit`. |
| C23 | User settings | P1 | ⬜ TODO | Profile (email, password change). Theme preference. Notification settings. GDPR: data export request, account deletion request. API key management. Calls: `POST /gdpr/export`, `POST /gdpr/erase`, `GET /gdpr/requests`. |

---

## Phase 6: Polish & Launch

| ID | Task | Priority | Status | Description |
|---|---|---|---|---|
| C24 | Mobile responsive | P0 | ⬜ TODO | Bottom tab navigation (porting existing pill nav pattern). Slide-over panels for settings/agents. Touch-friendly composer. iOS Safari safe area (`env(safe-area-inset-*)`). Responsive breakpoints: 640/768/1024/1280. Test on iOS Safari + Android Chrome. |
| C25 | Accessibility | P1 | ⬜ TODO | WCAG 2.1 AA compliance. Keyboard navigation throughout. Screen reader landmarks and ARIA labels. Focus management on route changes. Reduced motion support. Color contrast validation against OKLCH palette. |
| C26 | Performance | P1 | ⬜ TODO | Code splitting per route. Lazy load heavy components (markdown renderer, charts). Service worker for offline shell. Bundle analysis < 200KB initial JS. Lighthouse score > 90. Virtual scrolling for long message lists. |
| C27 | Error handling & empty states | P1 | ⬜ TODO | Global error boundary with recovery. Toast notifications for API errors. Offline detection banner. Empty states for all list views (no agents, no sessions, no integrations). Loading skeletons. |
| C28 | Production deployment | P0 | ⬜ TODO | Vite build → `web/dist/`. Caddy serves `web/dist/` (update root path). Gzip/Brotli compression. Hashed asset filenames with cache-forever headers. CSP headers. GitHub Actions CI: lint + type-check + build on push to `feat/chat-ui`. |

---

## Backend Requirements (New Endpoints Needed)

The chat UI requires a few backend additions not yet in the platform:

| ID | Endpoint | Purpose |
|---|---|---|
| B-WS | `GET /api/v1/ws` | WebSocket upgrade for real-time chat. JWT auth on handshake. Message send/receive + streaming tokens. |
| B-SESSIONS | `GET/POST/DELETE /api/v1/sessions` | Session CRUD — list user's chat sessions, create new, delete. |
| B-MESSAGES | `GET /api/v1/sessions/{id}/messages` | Paginated message history for a session. |
| B-SEND | `POST /api/v1/sessions/{id}/messages` | Send a message (triggers agent processing). |
| B-STREAM | `GET /api/v1/sessions/{id}/stream` | SSE fallback for streaming responses if WebSocket isn't available. |
| B-PROFILE | `GET/PUT /api/v1/user/profile` | Get/update current user profile. |
| B-PASSWORD | `POST /api/v1/user/password` | Change password (requires current password). |

---

## Key Components (React)

```
src/
├── components/
│   ├── chat/          MessageBubble, MessageList, Composer, TypingIndicator, ScrollToBottom
│   ├── browser/       BrowserPanel, BrowserFrame (iframe wrapper for go-browser/neko)
│   ├── agents/        AgentCard, AgentEditor, AgentList, ScopeSelector
│   ├── billing/       PlanCard, PlanComparison, UsageChart, OverageBar
│   ├── integrations/  IntegrationCard, OAuthFlow, StatusBadge, MarketplaceGrid
│   ├── admin/         UserTable, AuditLog, SecurityDashboard, StatsCards
│   ├── settings/      ProfileForm, ThemeToggle, GDPRPanel
│   ├── layout/        AppShell, Sidebar, TopBar, BottomTabs, Panel
│   └── shared/        Button, Input, Modal, Toast, Badge, Skeleton, EmptyState
├── pages/             Login, Register, Verify, Chat, Agents, Billing, Integrations, Settings, Admin
├── hooks/             useWebSocket, useAuth, useAgent, useSession, useTheme, useBilling
├── services/          api.ts, ws.ts, auth.ts
├── stores/            authStore, chatStore, agentStore, sessionStore, uiStore
└── types/             Generated from OpenAPI spec
```

---

## Design Principles

1. **API-first** — Every UI feature maps to an existing backend endpoint. No frontend hacks.
2. **Progressive disclosure** — Chat is front and center. Platform features (billing, integrations, admin) are one click away but never in the way.
3. **Real-time by default** — WebSocket for chat, polling fallback for dashboards. No manual refresh.
4. **Mobile-native feel** — Not a desktop app squeezed onto a phone. Touch targets, gestures, native-like transitions.
5. **Type-safe end-to-end** — OpenAPI spec → generated TypeScript types → zero runtime type mismatches.

---

## Deployment

| Environment | Domain | Branch | Path |
|---|---|---|---|
| Dev | `os-go.operator.onl` | `feat/chat-ui` | `/var/www/prototypes/os-go/web` |
| Production | `os-go.operator.onl` | `main` (after merge) | Same |

---

## Changelog

| Date | Change |
|---|---|
| 2026-03-08 | Initial plan created. 28 tasks across 6 phases + 7 backend requirements. |
| 2026-03-08 | Updated: plan now builds on existing `web/index.html` (1568-line chat UI). Branch: `feat/chat-ui`. Deployment: `os-go.operator.onl`. |
| 2026-03-08 | Migrated to React 19 + TypeScript + Vite + Tailwind v4 + Zustand. Existing `index.html` becomes reference (archived to `web/legacy/`). All tasks updated for component architecture. Added component inventory. |
| 2026-03-08 | C1 complete: React scaffold. Vite 6 + React 19 + TS + Tailwind v4. Full OKLCH token system ported via CSS vars + `@theme`. AppShell, routing, theme store, stub pages, login shell. Build clean. |
| 2026-03-08 | C2 complete: API client + types. 36 interfaces + 8 union types from OpenAPI spec. Namespaced `api.*` client with JWT auto-attach, deduped token refresh, error normalization. |
| 2026-03-08 | C3 complete: Auth flows. Zustand auth store, Login/Register/Verify pages, ProtectedRoute, session restore from JWT, auto-refresh, logout with SignOut in nav. |
| 2026-03-08 | C4 complete: App shell & routing. Sidebar (collapsible, persistent), TopBar (user menu dropdown, theme toggle, page title), BottomTabs (mobile), MobileSidebar (slide-over overlay with backdrop). |
| 2026-03-08 | C5 complete: Theme system. useTheme hook with OS preference listener, smooth theme transitions, glass morphism utilities, form input styles, animation library, reduced-motion support, shared Button/Badge/Input components. |
| 2026-03-08 | C6 complete: WebSocket transport. WebSocketManager with typed protocol, exponential backoff reconnect, ping keepalive, auth failure detection. Zustand chatStore bridging WS events to reactive state. useWebSocket hook, ConnectionStatus component, Chat page with connection UX + stub composer. |
| 2026-03-08 | C7 complete: Message thread UI. MessageBubble (3 variants, memo'd), MessageList (auto-scroll, unread tracking, timestamp grouping), ScrollToBottom (glass button + badge), TypingIndicator, MessageSkeleton. Streaming cursor blink animation. |
| 2026-03-08 | C8 complete: Markdown & code rendering. react-markdown + remark-gfm + rehype-highlight + DOMPurify. CodeBlock with language label + copy button. Custom OKLCH hljs theme. All legacy block/inline element styles ported. Streaming skips highlighting. |
| 2026-03-08 | C9 complete: Streaming responses. streamingMessageId tracking, auto-scroll during streaming, Stop generating button, cancelled message indicator, composer disabled during stream. |
| 2026-03-08 | C10 complete: Input composer. Full Composer component with auto-resize textarea, file/image upload (drag-drop + paste + click), preview thumbnails, model selector dropdown, glass morphism container. Phase 2 complete. |
| 2026-03-08 | C11 complete: Agent CRUD. Phase 3 started. agentStore (Zustand), AgentCard, AgentEditor modal, AgentList grid, AgentsPage with filters. New shared: Modal, ConfirmDialog. |
| 2026-03-08 | C12 complete: Multi-session UI. sessionStore, SessionItem (inline rename, context menu), SessionPanel (desktop sidebar + mobile slide-over, search, pinned/recent sections). chatStore updated with loadSessionHistory. Chat page refactored with session panel, header bar, history states. |
| 2026-03-08 | C13 complete: Conversation history. Filter bar (agent dropdown, date range pickers), archive toggle, export as Markdown/JSON. SessionItem context menu expanded with archive + export submenu. Export utility with paginated fetch + download. |
| 2026-03-08 | C14 complete: Agent integration scopes. ScopeSelector component with visual toggle per integration, expandable tool selector chips, search filter, category grouping. Integrated into AgentEditor form. AgentCard shows integration + tool counts. Phase 3 complete. |
| 2026-03-08 | C15 complete: Billing & plans. Phase 4 started. billingStore (Zustand), PlanCard (4 variants, feature lists, savings %), CurrentSubscription banner, PlanChangePreview modal (proration), IntervalToggle, BillingPage with Stripe redirect handling + loading skeletons. Fixed pre-existing TS errors in ScopeSelector + SessionPanel. |
| 2026-03-08 | C16 complete: Usage dashboard. usageStore (Zustand, 5 parallel fetchers). SummaryCards (4 stat cards), DailyChart (pure-CSS stacked bars, 14-day view, tooltips), ModelBreakdown (sorted model list, OKLCH hue bars), LimitsBar (tokens + messages vs plan, warning thresholds), OverageWarning (5-level severity banner). Billing page refactored with Plans/Usage tab switcher. |
