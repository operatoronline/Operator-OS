# Operator OS — User Onboarding & Service Integration

**Date:** 2026-03-06
**Branch:** `operatoros-production-readiness`
**Companion Document:** [Production-Readiness-Assessment.md](./Production-Readiness-Assessment.md)

---

## Table of Contents

1. [Vision: The 60-Second Experience](#1-vision-the-60-second-experience)
2. [Account Creation & Authentication](#2-account-creation--authentication)
3. [Plan Selection & Billing](#3-plan-selection--billing)
4. [Agent Creation & Onboarding](#4-agent-creation--onboarding)
5. [Service Integration Architecture](#5-service-integration-architecture)
6. [Credential Management & Security](#6-credential-management--security)
7. [OAuth vs API Key Strategies](#7-oauth-vs-api-key-strategies)
8. [Permission & Scope Management](#8-permission--scope-management)
9. [Agent Identity & Access Boundaries](#9-agent-identity--access-boundaries)
10. [User Dashboard & Integration Management](#10-user-dashboard--integration-management)
11. [Integration Marketplace](#11-integration-marketplace)
12. [Agent Templates & Preconfigured Workflows](#12-agent-templates--preconfigured-workflows)
13. [Developer Experience for New Integrations](#13-developer-experience-for-new-integrations)
14. [Reducing Cognitive Overhead](#14-reducing-cognitive-overhead)

---

## 1. Vision: The 60-Second Experience

The target experience for a new user:

```
1. Land on operator.onl → "Get Started" button                     (5 seconds)
2. Sign up with email or Google/Apple                               (15 seconds)
3. Choose a plan (Free tier highlighted)                            (10 seconds)
4. Pick an agent template ("Shopify Store Manager")                 (10 seconds)
5. Connect Shopify with one click (OAuth)                           (15 seconds)
6. Agent says: "I can see your store. You have 3 unfulfilled 
   orders. Want me to check inventory levels?"                      (5 seconds)
```

**Total: ~60 seconds from landing page to value.**

Every design decision below serves this goal. The system should feel like hiring a competent assistant — not configuring a developer tool.

---

## 2. Account Creation & Authentication

### 2.1 Registration Flow

**Primary:** Email + password
**Social:** Google OAuth, Apple Sign-In, GitHub OAuth
**Enterprise:** SAML/SSO (Phase 2)

```
┌─────────────────────────────────────────┐
│         Create Your Account             │
│                                         │
│  ┌─────────────────────────────────┐    │
│  │  Continue with Google      [G]  │    │
│  └─────────────────────────────────┘    │
│  ┌─────────────────────────────────┐    │
│  │  Continue with Apple       [🍎] │    │
│  └─────────────────────────────────┘    │
│                                         │
│  ──────── or use email ────────         │
│                                         │
│  Email:    [________________________]   │
│  Password: [________________________]   │
│                                         │
│  [        Create Account         ]      │
│                                         │
│  By signing up, you agree to our        │
│  Terms of Service and Privacy Policy    │
└─────────────────────────────────────────┘
```

### 2.2 Design Principles

- **No username required** — email is the identifier
- **Passwordless option** — magic link via email for low-friction entry
- **Progressive profile** — collect name/company/use-case *after* first agent interaction, not during sign-up
- **No phone number** — reduces friction and privacy concerns
- **Email verification** — required before connecting paid services, not before first agent use

### 2.3 Authentication Architecture

```
┌────────────┐     ┌──────────────┐     ┌─────────────┐
│   Client   │────▶│  Auth Service │────▶│  PostgreSQL  │
│ (Web/Mobile)│     │  (Go / JWT)  │     │  (users,     │
└────────────┘     └──────────────┘     │   sessions)  │
                         │               └─────────────┘
                         │
                    ┌────▼─────┐
                    │  Redis   │
                    │ (session │
                    │  tokens) │
                    └──────────┘
```

**Token strategy:**
- **Access token:** JWT, 15-minute expiry, contains `user_id`, `tenant_id`, `plan_tier`
- **Refresh token:** Opaque, 30-day expiry, stored in Redis with device fingerprint
- **Session cookie:** HTTP-only, Secure, SameSite=Strict for web dashboard

### 2.4 Data Model

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT UNIQUE NOT NULL,
    password_hash   TEXT,                  -- NULL for social-only accounts
    name            TEXT,
    avatar_url      TEXT,
    plan_id         TEXT NOT NULL DEFAULT 'free',
    email_verified  BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    last_login_at   TIMESTAMPTZ,
    metadata        JSONB DEFAULT '{}'     -- progressive profile data
);

CREATE TABLE auth_providers (
    user_id         UUID REFERENCES users(id),
    provider        TEXT NOT NULL,          -- 'google', 'apple', 'github'
    provider_id     TEXT NOT NULL,          -- external provider user ID
    access_token    TEXT,                   -- encrypted
    refresh_token   TEXT,                   -- encrypted
    PRIMARY KEY (user_id, provider)
);
```

---

## 3. Plan Selection & Billing

### 3.1 Plan Architecture

Plans are defined as configuration, not code:

```json
{
  "plans": {
    "free": {
      "name": "Free",
      "price_monthly": 0,
      "agents": 1,
      "messages_per_month": 500,
      "integrations": 1,
      "models": ["gpt-4.1-mini"],
      "storage_mb": 100,
      "features": ["basic_tools", "web_search"]
    },
    "starter": {
      "name": "Starter",
      "price_monthly": 900,
      "agents": 3,
      "messages_per_month": 5000,
      "integrations": 5,
      "models": ["gpt-4.1", "claude-haiku"],
      "storage_mb": 1024,
      "features": ["basic_tools", "web_search", "cron", "skills"]
    },
    "pro": {
      "name": "Pro",
      "price_monthly": 2900,
      "agents": 10,
      "messages_per_month": 50000,
      "integrations": 20,
      "models": ["*"],
      "storage_mb": 10240,
      "features": ["*"]
    }
  }
}
```

### 3.2 Billing UX

```
┌─────────────────────────────────────────────────────┐
│                 Choose Your Plan                      │
│                                                       │
│  ┌──────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │   Free   │  │   Starter    │  │     Pro      │    │
│  │          │  │              │  │              │    │
│  │  $0/mo   │  │   $9/mo      │  │   $29/mo     │    │
│  │          │  │              │  │              │    │
│  │ 1 agent  │  │  3 agents    │  │  10 agents   │    │
│  │ 500 msgs │  │  5K msgs     │  │  50K msgs    │    │
│  │ 1 integ  │  │  5 integs    │  │  20 integs   │    │
│  │          │  │              │  │              │    │
│  │ [Start]  │  │ [Start Free  │  │ [Start Free  │    │
│  │          │  │  Trial - 14d]│  │  Trial - 14d]│    │
│  └──────────┘  └──────────────┘  └──────────────┘    │
│                                                       │
│  All plans include: Web search, persistent memory,    │
│  multi-channel messaging, and community support.      │
└─────────────────────────────────────────────────────┘
```

### 3.3 Billing Principles

- **Free tier always available** — no credit card required
- **14-day free trial** for paid plans — no commitment
- **Monthly billing by default**, annual with 2 months free
- **Transparent usage tracking** — users always know their remaining quota
- **Graceful degradation** — exceeding limits throttles, never cuts off mid-conversation
- **Prorated upgrades** — switch plans anytime, pay the difference

### 3.4 Usage Metering

```sql
CREATE TABLE usage_events (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id),
    agent_id    TEXT NOT NULL,
    event_type  TEXT NOT NULL,           -- 'llm_request', 'tool_exec', 'message'
    model       TEXT,
    tokens_in   INT DEFAULT 0,
    tokens_out  INT DEFAULT 0,
    cost_usd    NUMERIC(10,6) DEFAULT 0,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_usage_user_month ON usage_events (user_id, created_at);
```

---

## 4. Agent Creation & Onboarding

### 4.1 One-Click Agent Creation

After signing up and selecting a plan, the user creates their first agent:

```
┌─────────────────────────────────────────────────┐
│         What would you like help with?            │
│                                                   │
│  ┌─────────────┐  ┌─────────────┐                 │
│  │ 🛍️ Shopify   │  │ 📱 Social    │                │
│  │ Store        │  │ Media       │                │
│  │ Management   │  │ Manager     │                │
│  └─────────────┘  └─────────────┘                 │
│  ┌─────────────┐  ┌─────────────┐                 │
│  │ 📚 Research  │  │ 💼 Business  │                │
│  │ & Learning   │  │ Operations  │                │
│  │ Assistant    │  │ Assistant   │                │
│  └─────────────┘  └─────────────┘                 │
│  ┌─────────────┐  ┌─────────────┐                 │
│  │ 🏋️ Coaching  │  │ ✨ Custom    │                │
│  │ & Wellness   │  │ (Start from │                │
│  │ Tracker      │  │ scratch)    │                │
│  └─────────────┘  └─────────────┘                 │
│                                                   │
│  Each template comes with suggested integrations  │
│  and pre-configured skills. You can customize     │
│  everything later.                                │
└─────────────────────────────────────────────────┘
```

### 4.2 Template-Driven Setup

When a user picks a template, the system:

1. Creates an agent with the template's default persona (SOUL.md)
2. Pre-selects relevant skills
3. Suggests integrations with a "connect now" or "skip" option
4. Opens a chat where the agent introduces itself and asks setup questions

```
┌─────────────────────────────────────────────────┐
│  🛍️ Shopify Store Manager                        │
│                                                   │
│  Agent: Hi! I'm your Shopify store assistant.     │
│  I can help you with:                             │
│  • Order management and fulfillment               │
│  • Inventory monitoring                            │
│  • Customer inquiries                              │
│  • Sales analytics and reports                     │
│                                                   │
│  Let's get started. Would you like to connect      │
│  your Shopify store now?                           │
│                                                   │
│  [Connect Shopify] [Set up later]                  │
│                                                   │
│  You: ___________________________________          │
└─────────────────────────────────────────────────┘
```

### 4.3 Progressive Complexity

The onboarding follows a progressive disclosure pattern:

| Stage | User Action | System Behavior |
|---|---|---|
| **1. Immediate** | Pick template, start chatting | Agent works with built-in capabilities |
| **2. First integration** | Connect one service | Agent unlocks service-specific features |
| **3. Customize** | Adjust persona, add skills | Agent becomes more specialized |
| **4. Automate** | Set up cron tasks | Agent works proactively |
| **5. Scale** | Add more agents, team members | Multi-agent workflows |

Never force the user through all stages. Many users will stay at Stage 1-2 and that's fine.

---

## 5. Service Integration Architecture

### 5.1 Integration Registry

Every integration is defined as a declarative manifest:

```json
{
  "id": "shopify",
  "name": "Shopify",
  "icon": "shopify.svg",
  "category": "ecommerce",
  "description": "Manage your Shopify store: orders, inventory, customers, analytics.",
  "auth_type": "oauth2",
  "oauth": {
    "authorization_url": "https://{shop}.myshopify.com/admin/oauth/authorize",
    "token_url": "https://{shop}.myshopify.com/admin/oauth/access_token",
    "scopes": ["read_orders", "write_orders", "read_products", "read_customers"],
    "dynamic_params": {
      "shop": {
        "label": "Your Shopify store URL",
        "placeholder": "mystore.myshopify.com",
        "required": true
      }
    }
  },
  "tools": [
    {
      "name": "shopify_get_orders",
      "description": "List recent orders with filtering",
      "parameters": { ... }
    },
    {
      "name": "shopify_get_products",
      "description": "List and search products",
      "parameters": { ... }
    }
  ],
  "suggested_templates": ["shopify-store-manager"],
  "required_plan": "starter"
}
```

### 5.2 Integration Lifecycle

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│ Discover │────▶│ Connect  │────▶│ Configure│────▶│  Active  │
│          │     │ (OAuth)  │     │ (Scopes) │     │          │
└──────────┘     └──────────┘     └──────────┘     └──────────┘
                      │                                  │
                      │           ┌──────────┐           │
                      └──────────▶│  Failed  │◀──────────┘
                                  │ (Retry)  │
                                  └──────────┘
```

**State machine per integration:**

```sql
CREATE TABLE user_integrations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    integration_id  TEXT NOT NULL,               -- 'shopify', 'gmail', etc.
    status          TEXT NOT NULL DEFAULT 'pending',  -- pending, active, failed, revoked
    config          JSONB DEFAULT '{}',          -- integration-specific config (e.g., shop URL)
    scopes          TEXT[] DEFAULT '{}',          -- granted scopes
    last_used_at    TIMESTAMPTZ,
    error_message   TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(user_id, integration_id)
);
```

### 5.3 Token Storage Architecture

```
┌─────────────────────────────────────────────────┐
│              Credential Vault                    │
│                                                  │
│  ┌──────────────────────────────────────┐        │
│  │  Encryption Layer (AES-256-GCM)     │        │
│  │                                      │        │
│  │  Master Key: Derived from            │        │
│  │  • User password (self-hosted)       │        │
│  │  • KMS DEK (managed SaaS)           │        │
│  │                                      │        │
│  │  Per-credential encryption:          │        │
│  │  • Each token encrypted with         │        │
│  │    unique DEK wrapped by master key  │        │
│  └──────────────────────────────────────┘        │
│                                                  │
│  ┌──────────────────────────────────────┐        │
│  │  Storage Layer                       │        │
│  │                                      │        │
│  │  Self-hosted: SQLite (encrypted col) │        │
│  │  SaaS: PostgreSQL + Vault transit    │        │
│  └──────────────────────────────────────┘        │
└─────────────────────────────────────────────────┘
```

```sql
CREATE TABLE credential_vault (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    integration_id  TEXT NOT NULL,
    credential_type TEXT NOT NULL,          -- 'oauth_access', 'oauth_refresh', 'api_key'
    encrypted_value BYTEA NOT NULL,         -- AES-256-GCM encrypted
    iv              BYTEA NOT NULL,         -- Initialization vector
    key_version     INT NOT NULL DEFAULT 1, -- For key rotation
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(user_id, integration_id, credential_type)
);
```

### 5.4 Shared Credentials Across Agents

A single service connection is shared across all of a user's agents:

```
User connects Shopify once
    │
    ├── Agent: "Store Manager" → has access to Shopify tools
    ├── Agent: "Analytics Bot" → has access to Shopify tools (read-only)
    └── Agent: "Custom Agent"  → no Shopify access (not assigned)
```

Per-agent scoping is managed through agent-level integration assignments:

```sql
CREATE TABLE agent_integrations (
    agent_id        TEXT NOT NULL,
    user_id         UUID NOT NULL REFERENCES users(id),
    integration_id  TEXT NOT NULL,
    scopes_override TEXT[],         -- NULL = use all granted scopes; set = restrict
    PRIMARY KEY (agent_id, user_id, integration_id)
);
```

---

## 6. Credential Management & Security

### 6.1 Security Principles

1. **Credentials never leave the server** — tokens are stored server-side, injected into requests at execution time
2. **Least privilege** — agents only get the scopes they need
3. **Automatic rotation** — OAuth refresh tokens are rotated on each use
4. **Audit trail** — every credential access is logged
5. **User revocation** — one-click disconnect that immediately invalidates all tokens

### 6.2 Token Lifecycle

```
┌────────────┐     ┌────────────┐     ┌────────────┐
│   User     │     │  Operator  │     │  External  │
│  Dashboard │     │  Backend   │     │  Service   │
└─────┬──────┘     └─────┬──────┘     └─────┬──────┘
      │                  │                   │
      │  Click "Connect" │                   │
      │─────────────────▶│                   │
      │                  │  OAuth redirect   │
      │  ◀───────────────│──────────────────▶│
      │  Browser redirect│                   │
      │─────────────────▶│                   │
      │                  │  Exchange code    │
      │                  │──────────────────▶│
      │                  │  Access + Refresh │
      │                  │◀──────────────────│
      │                  │                   │
      │                  │  Encrypt & store  │
      │                  │  ┌──────────┐     │
      │                  │──│  Vault   │     │
      │                  │  └──────────┘     │
      │  "Connected! ✅" │                   │
      │◀─────────────────│                   │
      │                  │                   │
      │     (Later: Agent needs Shopify)     │
      │                  │                   │
      │                  │  Decrypt token    │
      │                  │  ┌──────────┐     │
      │                  │──│  Vault   │     │
      │                  │  └──────────┘     │
      │                  │                   │
      │                  │  API call with    │
      │                  │  injected token   │
      │                  │──────────────────▶│
      │                  │  Response         │
      │                  │◀──────────────────│
```

### 6.3 Token Refresh Strategy

```go
type TokenManager struct {
    vault     CredentialVault
    mu        sync.RWMutex
    refreshing map[string]bool // prevent concurrent refresh storms
}

func (tm *TokenManager) GetAccessToken(userID, integrationID string) (string, error) {
    cred, err := tm.vault.Get(userID, integrationID, "oauth_access")
    if err != nil {
        return "", err
    }
    
    // If token is still valid (with 5-minute buffer), return it
    if cred.ExpiresAt.After(time.Now().Add(5 * time.Minute)) {
        return cred.Value, nil
    }
    
    // Need refresh — use mutex to prevent concurrent refresh for same integration
    refreshKey := fmt.Sprintf("%s:%s", userID, integrationID)
    tm.mu.Lock()
    if tm.refreshing[refreshKey] {
        tm.mu.Unlock()
        // Wait and retry
        time.Sleep(2 * time.Second)
        return tm.GetAccessToken(userID, integrationID)
    }
    tm.refreshing[refreshKey] = true
    tm.mu.Unlock()
    
    defer func() {
        tm.mu.Lock()
        delete(tm.refreshing, refreshKey)
        tm.mu.Unlock()
    }()
    
    // Perform refresh
    refreshToken, _ := tm.vault.Get(userID, integrationID, "oauth_refresh")
    newTokens, err := refreshOAuthToken(integrationID, refreshToken.Value)
    if err != nil {
        return "", fmt.Errorf("token refresh failed: %w", err)
    }
    
    // Store new tokens
    tm.vault.Set(userID, integrationID, "oauth_access", newTokens.AccessToken, newTokens.ExpiresAt)
    if newTokens.RefreshToken != "" {
        tm.vault.Set(userID, integrationID, "oauth_refresh", newTokens.RefreshToken, time.Time{})
    }
    
    return newTokens.AccessToken, nil
}
```

---

## 7. OAuth vs API Key Strategies

### 7.1 Decision Matrix

| Service | Auth Method | Reason |
|---|---|---|
| **Shopify** | OAuth 2.0 | Official API, granular scopes |
| **Gmail / Google Drive** | OAuth 2.0 | Google requires OAuth for user data |
| **Twitter/X** | OAuth 2.0 | Required for posting, reading DMs |
| **Instagram** | OAuth 2.0 (Meta Graph) | Required by Meta |
| **Slack** | OAuth 2.0 | Workspace-level permissions |
| **GitHub** | OAuth 2.0 or PAT | OAuth preferred, PAT for advanced users |
| **Stripe** | API Key (Restricted) | Stripe's recommended approach for server apps |
| **OpenAI / Anthropic** | API Key | No OAuth available |
| **Notion** | OAuth 2.0 | User-specific page access |
| **Airtable** | OAuth 2.0 or PAT | OAuth preferred |
| **SendGrid** | API Key | Simple, scoped keys available |

### 7.2 OAuth Implementation

```
┌──────────────────────────────────────────────────────┐
│              OAuth 2.0 Flow (PKCE)                    │
│                                                        │
│  1. User clicks "Connect [Service]"                    │
│  2. Backend generates:                                 │
│     • state (CSRF protection, contains user_id)        │
│     • code_verifier + code_challenge (PKCE)            │
│  3. Redirect to service authorization URL              │
│  4. User grants permissions on service's consent page  │
│  5. Service redirects back with authorization code      │
│  6. Backend exchanges code for tokens (using PKCE)     │
│  7. Tokens encrypted and stored in vault               │
│  8. Integration status set to 'active'                 │
│  9. User returned to dashboard with success message    │
└──────────────────────────────────────────────────────┘
```

**Why PKCE everywhere:** Even for server-side apps, PKCE prevents authorization code interception. It's now recommended by OAuth 2.1 for all client types.

### 7.3 API Key Handling

For services using API keys:

```
┌─────────────────────────────────────────────────┐
│  Connect Stripe                                  │
│                                                  │
│  Enter your Stripe restricted API key:           │
│  [rk_live_*********************************]     │
│                                                  │
│  ℹ️ Create a restricted key at                   │
│     stripe.com/dashboard/apikeys                 │
│                                                  │
│  Required permissions:                           │
│  ✅ Charges (Read)                               │
│  ✅ Customers (Read + Write)                     │
│  ✅ Products (Read)                              │
│  ❌ Everything else should be disabled           │
│                                                  │
│  [Connect] [Cancel]                              │
│                                                  │
│  🔒 Your key is encrypted and never exposed.     │
│     You can revoke access anytime.               │
└─────────────────────────────────────────────────┘
```

**API Key validation flow:**
1. User enters API key
2. Backend makes a test API call to verify the key works
3. Backend checks which permissions/scopes the key has
4. Warn if permissions are too broad ("This key has write access to charges. We recommend a restricted key.")
5. Encrypt and store
6. Show "Connected ✅"

---

## 8. Permission & Scope Management

### 8.1 Three-Level Permission Model

```
┌──────────────────────────────────────────────────┐
│                Permission Hierarchy                │
│                                                    │
│  Level 1: Platform Permissions                     │
│  ┌─────────────────────────────────────┐           │
│  │ What Operator OS can do:            │           │
│  │ • Access your connected services    │           │
│  │ • Execute agent tasks               │           │
│  │ • Store conversation history        │           │
│  └─────────────────────────────────────┘           │
│           │                                        │
│  Level 2: Integration Scopes                       │
│  ┌─────────────────────────────────────┐           │
│  │ What each service connection allows:│           │
│  │ Shopify: read_orders, read_products │           │
│  │ Gmail: read, send                   │           │
│  │ Calendar: read, write               │           │
│  └─────────────────────────────────────┘           │
│           │                                        │
│  Level 3: Agent Permissions                        │
│  ┌─────────────────────────────────────┐           │
│  │ What each agent can use:            │           │
│  │ "Store Bot": Shopify (all)          │           │
│  │ "Email Bot": Gmail (read only)      │           │
│  │ "Custom":    No integrations        │           │
│  └─────────────────────────────────────┘           │
└──────────────────────────────────────────────────┘
```

### 8.2 Scope Display

When connecting a service, show clear, human-readable permissions:

```
┌─────────────────────────────────────────────────┐
│  Shopify wants to allow Operator OS to:          │
│                                                  │
│  ✅ View your orders and order history           │
│  ✅ View and update your products                │
│  ✅ View customer information                    │
│  ⚠️ Create and modify draft orders               │
│                                                  │
│  ❌ Operator OS will NOT be able to:             │
│  • Process payments or refunds                   │
│  • Delete products or orders                     │
│  • Change store settings                         │
│  • Access your billing information               │
│                                                  │
│  [Allow Access]  [Customize Permissions]          │
└─────────────────────────────────────────────────┘
```

### 8.3 Scope Narrowing

Users can restrict what agents can actually use, even below what was granted:

```
┌─────────────────────────────────────────────────┐
│  Agent: "Analytics Bot"                          │
│  Shopify Integration: Active                     │
│                                                  │
│  Permissions for this agent:                     │
│  ✅ View orders (read_orders)                    │
│  ✅ View products (read_products)                │
│  ❌ Modify products (write_products)  [disabled] │
│  ✅ View customers (read_customers)              │
│  ❌ Create draft orders (write_drafts) [disabled]│
│                                                  │
│  This agent can read but not modify your store.  │
│                                                  │
│  [Save]                                          │
└─────────────────────────────────────────────────┘
```

---

## 9. Agent Identity & Access Boundaries

### 9.1 Agent-Level Isolation

Each agent operates within defined boundaries:

```go
type AgentBoundary struct {
    AgentID         string
    UserID          string
    
    // What this agent can access
    Integrations    map[string][]string  // integration_id → allowed scopes
    Tools           []string             // allowed tool names
    Skills          []string             // allowed skill IDs
    
    // What this agent cannot do
    DeniedActions   []string             // e.g., "delete_files", "send_email"
    
    // Resource limits
    MaxTokensPerRequest  int
    MaxToolIterations    int
    MaxConcurrentTasks   int
}
```

### 9.2 Agent Actions Are Attributed

Every action an agent takes is logged with full attribution:

```sql
CREATE TABLE agent_actions (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL,
    agent_id        TEXT NOT NULL,
    action_type     TEXT NOT NULL,          -- 'tool_exec', 'api_call', 'message_sent'
    integration_id  TEXT,
    tool_name       TEXT,
    input_summary   TEXT,                   -- truncated input for audit
    output_summary  TEXT,                   -- truncated output for audit
    success         BOOLEAN,
    duration_ms     INT,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

### 9.3 Safety Rails

| Rail | Description | Default |
|---|---|---|
| **Confirmation for destructive actions** | Agent asks user before deleting, canceling, or refunding | ON |
| **Daily spending cap** | Maximum cost of LLM calls per agent per day | $5 |
| **Rate limiting** | Maximum API calls to external services per minute | 10/min |
| **Scope freeze** | Agent cannot escalate its own permissions | ON |
| **Conversation boundaries** | Agent cannot access other agents' conversations | ON |
| **PII handling** | Agent does not log sensitive data (credit cards, SSNs) | ON |

---

## 10. User Dashboard & Integration Management

### 10.1 Dashboard Layout

```
┌──────────────────────────────────────────────────────┐
│  🏠 Dashboard                                         │
│                                                        │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐               │
│  │ Agents   │ │ Messages │ │ Tokens   │               │
│  │    3     │ │  1,247   │ │  42.3K   │               │
│  │ active   │ │ this mo  │ │ this mo  │               │
│  └──────────┘ └──────────┘ └──────────┘               │
│                                                        │
│  📋 My Agents                                          │
│  ┌─────────────────────────────────────────────────┐   │
│  │ 🛍️ Store Manager    ● Active   │ Chat │ Config │   │
│  │ 📱 Social Media Bot  ● Active   │ Chat │ Config │   │
│  │ 📚 Research Helper   ○ Paused   │ Chat │ Config │   │
│  └─────────────────────────────────────────────────┘   │
│                                                        │
│  🔗 Connected Services                                 │
│  ┌─────────────────────────────────────────────────┐   │
│  │ Shopify     ● Connected  │ 2 agents │ Manage │   │
│  │ Gmail       ● Connected  │ 1 agent  │ Manage │   │
│  │ + Connect a service                           │   │
│  └─────────────────────────────────────────────────┘   │
│                                                        │
│  📊 Usage This Month                                   │
│  ┌─────────────────────────────────────────────────┐   │
│  │ ████████████████████░░░░░  80%  (4,012 / 5,000) │   │
│  │ Messages                                        │   │
│  │ ██████████░░░░░░░░░░░░░░  40%  (2.1K / 5K)    │   │
│  │ LLM Tokens (thousands)                          │   │
│  └─────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────┘
```

### 10.2 Integration Management Page

```
┌──────────────────────────────────────────────────────┐
│  🔗 Manage: Shopify                                   │
│                                                        │
│  Status: ● Connected                                   │
│  Connected: 2026-03-01                                 │
│  Last used: 2 hours ago                                │
│  Store: mystore.myshopify.com                          │
│                                                        │
│  Permissions Granted:                                  │
│  ✅ Read orders     ✅ Read products                    │
│  ✅ Write products  ✅ Read customers                   │
│                                                        │
│  Used by Agents:                                       │
│  • Store Manager (full access)                         │
│  • Analytics Bot (read-only)                           │
│                                                        │
│  Recent Activity:                                      │
│  │ 14:30 │ Store Manager read 5 orders                 │
│  │ 13:15 │ Analytics Bot fetched product list          │
│  │ 11:00 │ Store Manager updated product price         │
│                                                        │
│  [Refresh Credentials]  [Disconnect]                   │
└──────────────────────────────────────────────────────┘
```

---

## 11. Integration Marketplace

### 11.1 Marketplace Concept

A curated directory of integrations, organized by use case:

```
┌──────────────────────────────────────────────────────┐
│  🏪 Integration Marketplace                           │
│                                                        │
│  Search: [________________________] 🔍                 │
│                                                        │
│  Categories:                                           │
│  [All] [E-Commerce] [Communication] [Productivity]     │
│  [Social Media] [Analytics] [Developer Tools]          │
│                                                        │
│  ┌──────────────────┐  ┌──────────────────┐           │
│  │ 🛍️ Shopify         │  │ 📧 Gmail          │          │
│  │ Manage your store │  │ Read and send    │          │
│  │ ⭐ 4.8 (234 users)│  │ ⭐ 4.9 (891 users)│          │
│  │ [Connect]         │  │ [Connect]        │          │
│  └──────────────────┘  └──────────────────┘           │
│  ┌──────────────────┐  ┌──────────────────┐           │
│  │ 📊 Google Sheets  │  │ 📸 Instagram      │          │
│  │ Read and write   │  │ Post and analyze │          │
│  │ ⭐ 4.7 (156 users)│  │ ⭐ 4.5 (412 users)│          │
│  │ [Connect]         │  │ [Connect]        │          │
│  └──────────────────┘  └──────────────────┘           │
│                                                        │
│  Can't find what you need?                             │
│  [Request an Integration]  [Build Your Own →]          │
└──────────────────────────────────────────────────────┘
```

### 11.2 Integration Categories (Launch Priority)

**Phase 1 — Core (Launch)**
| Integration | Category | Auth | Use Case |
|---|---|---|---|
| Shopify | E-Commerce | OAuth | Store management |
| Gmail | Communication | OAuth | Email reading + sending |
| Google Calendar | Productivity | OAuth | Scheduling, reminders |
| Google Drive | Productivity | OAuth | File access, document management |
| Notion | Productivity | OAuth | Notes, databases |

**Phase 2 — Growth**
| Integration | Category | Auth | Use Case |
|---|---|---|---|
| Twitter/X | Social Media | OAuth | Posting, monitoring |
| Instagram (Meta) | Social Media | OAuth | Content scheduling, analytics |
| Slack | Communication | OAuth | Team messaging |
| Stripe | Finance | API Key | Payment tracking, invoices |
| Airtable | Productivity | OAuth | Data management |

**Phase 3 — Expansion**
| Integration | Category | Auth | Use Case |
|---|---|---|---|
| WooCommerce | E-Commerce | API Key | Store management |
| Mailchimp | Marketing | OAuth | Email campaigns |
| Trello | Productivity | OAuth | Task management |
| GitHub | Developer | OAuth | Issue tracking, PRs |
| Zapier | Automation | API Key | Cross-service workflows |
| HubSpot | CRM | OAuth | Contact management |

### 11.3 Community Integrations

Allow developers to publish integrations:

```json
{
  "manifest_version": 1,
  "id": "com.example.custom-crm",
  "name": "Custom CRM Connector",
  "author": "developer@example.com",
  "version": "1.0.0",
  "description": "Connect your Custom CRM to Operator OS",
  "auth_type": "api_key",
  "api_key_config": {
    "label": "API Key",
    "validation_endpoint": "https://api.custom-crm.com/v1/me",
    "validation_header": "Authorization: Bearer {key}"
  },
  "tools": [
    {
      "name": "crm_get_contacts",
      "endpoint": "GET https://api.custom-crm.com/v1/contacts",
      "description": "List contacts from your CRM",
      "parameters": {
        "type": "object",
        "properties": {
          "query": { "type": "string", "description": "Search query" },
          "limit": { "type": "integer", "default": 20 }
        }
      }
    }
  ]
}
```

---

## 12. Agent Templates & Preconfigured Workflows

### 12.1 Template Structure

```json
{
  "id": "shopify-store-manager",
  "name": "Shopify Store Manager",
  "icon": "🛍️",
  "description": "Manages your Shopify store: orders, inventory, customers.",
  "category": "ecommerce",
  "required_plan": "starter",
  
  "agent_config": {
    "persona": {
      "name": "Store Assistant",
      "soul": "You are a friendly, efficient e-commerce assistant. You help manage a Shopify store by tracking orders, monitoring inventory, and providing sales insights. You're proactive about flagging issues like low stock or unfulfilled orders."
    },
    "model": "gpt-4.1",
    "max_tool_iterations": 20,
    "temperature": 0.5
  },
  
  "required_integrations": ["shopify"],
  "optional_integrations": ["gmail", "google_sheets"],
  
  "skills": ["shopify-analytics", "customer-service"],
  
  "suggested_cron": [
    {
      "name": "Daily Order Summary",
      "schedule": "0 9 * * *",
      "task": "Check for unfulfilled orders and send me a morning summary."
    },
    {
      "name": "Low Stock Alert",
      "schedule": "0 */6 * * *",
      "task": "Check inventory levels. Alert me if any product has fewer than 5 units."
    }
  ],
  
  "onboarding_questions": [
    "What's your Shopify store URL?",
    "Would you like daily order summaries?",
    "What's your low stock threshold?"
  ]
}
```

### 12.2 Template Library (Launch)

| Template | Target User | Key Integrations |
|---|---|---|
| 🛍️ **Shopify Store Manager** | E-commerce entrepreneurs | Shopify, Gmail |
| 📱 **Social Media Manager** | Content creators, small businesses | Twitter, Instagram |
| 📚 **Research & Study Assistant** | Students, researchers | Google Drive, Notion |
| 💼 **Business Operations** | Small business owners | Gmail, Calendar, Sheets |
| 🏋️ **Coaching & Wellness** | Coaches, trainers | Calendar, Notion |
| 📝 **Content Writer** | Bloggers, marketers | Google Drive, WordPress |
| 💰 **Finance Tracker** | Freelancers, small businesses | Stripe, Sheets |
| 🎯 **Project Manager** | Team leads | Trello/Notion, Slack |

### 12.3 Workflow Presets

Each template can include preconfigured workflows:

```
┌─────────────────────────────────────────────────┐
│  🛍️ Shopify Store Manager — Workflows             │
│                                                   │
│  ✅ Daily Morning Briefing (9 AM)                  │
│     Summarize yesterday's sales, new orders,      │
│     and low-stock items.                          │
│                                                   │
│  ✅ New Order Notification (real-time)              │
│     Notify me when a new order comes in.          │
│                                                   │
│  ⬜ Weekly Sales Report (Monday 8 AM)              │
│     Generate a weekly sales comparison report.    │
│                                                   │
│  ⬜ Customer Follow-Up (3 days post-delivery)      │
│     Draft a follow-up email to recent customers.  │
│                                                   │
│  [Enable Selected] [Skip for Now]                  │
└─────────────────────────────────────────────────┘
```

---

## 13. Developer Experience for New Integrations

### 13.1 Integration SDK

Provide a Go SDK for building integrations:

```go
package integration

// Integration defines a third-party service connection.
type Integration struct {
    ID          string
    Name        string
    Category    string
    AuthConfig  AuthConfig
    Tools       []ToolDef
    Webhooks    []WebhookDef
}

// AuthConfig defines how users authenticate with this service.
type AuthConfig struct {
    Type        string        // "oauth2", "api_key", "basic"
    OAuth       *OAuthConfig
    APIKey      *APIKeyConfig
}

type OAuthConfig struct {
    AuthURL      string
    TokenURL     string
    Scopes       []ScopeDef
    PKCERequired bool
    DynamicParams map[string]ParamDef  // e.g., shop URL for Shopify
}

type ScopeDef struct {
    ID          string
    Name        string
    Description string
    Required    bool
}

// ToolDef defines a tool this integration provides to agents.
type ToolDef struct {
    Name        string
    Description string
    Parameters  json.RawMessage  // JSON Schema
    Handler     ToolHandler
}

// ToolHandler executes the tool with user credentials injected.
type ToolHandler func(ctx context.Context, creds *Credentials, args map[string]any) (*ToolResult, error)
```

### 13.2 Integration Development Workflow

```
1. Create integration manifest (JSON)
2. Implement tool handlers (Go or declarative HTTP)
3. Test with local Operator instance
4. Submit to integration registry
5. Review and approval
6. Published in marketplace
```

### 13.3 Declarative HTTP Integrations (No Code)

For simple REST API integrations, tools can be defined entirely in the manifest:

```json
{
  "name": "weather_get_forecast",
  "description": "Get weather forecast for a location",
  "method": "GET",
  "url": "https://api.weatherapi.com/v1/forecast.json",
  "auth_injection": {
    "type": "query_param",
    "param": "key",
    "credential": "api_key"
  },
  "parameters": {
    "type": "object",
    "properties": {
      "q": {
        "type": "string",
        "description": "Location (city name, zip code, or lat/lon)"
      },
      "days": {
        "type": "integer",
        "default": 3
      }
    },
    "required": ["q"]
  },
  "response_transform": {
    "jq": ".forecast.forecastday[] | {date: .date, condition: .day.condition.text, high: .day.maxtemp_f, low: .day.mintemp_f}"
  }
}
```

This allows non-developers to create simple integrations by just describing the API.

---

## 14. Reducing Cognitive Overhead

### 14.1 Design Principles for Non-Technical Users

1. **No jargon** — "Connect your store" not "Configure OAuth 2.0 authorization flow"
2. **Progressive disclosure** — Show simple options first, advanced settings on demand
3. **Safe defaults** — Everything works out of the box with minimal configuration
4. **Undo everything** — Every action is reversible with a clear undo path
5. **Show, don't explain** — Use visual indicators, not walls of text
6. **One primary action per screen** — Reduce decision paralysis

### 14.2 Error Handling UX

When things go wrong, the interface should:

```
┌─────────────────────────────────────────────────┐
│  ⚠️ Connection Issue                              │
│                                                  │
│  Your Shopify connection needs attention.        │
│  Your store changed its access permissions.      │
│                                                  │
│  [Reconnect Shopify]                             │
│                                                  │
│  Your agents will continue working with cached   │
│  data until you reconnect.                       │
└─────────────────────────────────────────────────┘
```

**Not:**
```
Error: OAuth token refresh failed. Status 401.
Invalid grant: The provided authorization grant is expired.
Please re-authenticate with the service provider.
```

### 14.3 Notification Strategy

| Event | Urgency | Channel | Message |
|---|---|---|---|
| Integration disconnected | High | In-app + email | "Your Shopify connection needs attention" |
| Usage at 80% | Medium | In-app | "You've used 80% of your monthly messages" |
| Agent completed task | Low | In-app | "Store Manager found 3 low-stock items" |
| New integration available | Low | In-app (weekly digest) | "New: Connect your Instagram" |
| Plan renewal | Medium | Email | "Your Pro plan renews in 3 days" |

### 14.4 Contextual Help

Every integration page includes:
- **Quick Start Guide** — 3-step setup with screenshots
- **FAQ** — Common questions and troubleshooting
- **Video walkthrough** — 60-second setup video
- **Live chat** — Access to support if stuck

### 14.5 Safe Default Configurations

Every template ships with safe defaults that minimize risk:

| Setting | Default | Why |
|---|---|---|
| Agent model | GPT-4.1 (latest stable) | Best balance of quality and cost |
| Temperature | 0.5 | Reliable but not robotic |
| Max tool iterations | 20 | Prevents runaway loops |
| Destructive action confirmation | ON | Prevents accidental deletions |
| Daily spending cap | $5 | Prevents cost surprises |
| Session history retention | 30 days | Balances utility and storage |
| Notification frequency | Batch (every 4 hours) | Not overwhelming |

Users can change everything, but the defaults should make the system safe and useful out of the box.

---

## Summary

The path from Operator OS's current state to a user-friendly managed platform requires building three major systems:

1. **Identity & Access** — User accounts, authentication, authorization, multi-tenancy
2. **Integration Platform** — OAuth framework, credential vault, tool injection, marketplace
3. **User Experience** — Dashboard, templates, onboarding flows, error handling

The existing codebase provides a solid foundation. The agent loop, provider system, tool registry, and skill system are well-designed and extensible. The integration platform should be built as a layer on top of these existing abstractions, not as a replacement.

**Critical first step:** Build the user management and credential vault. Everything else depends on knowing who the user is and securely storing their service credentials.

---

*This document is designed to guide product and engineering decisions for Operator OS's evolution from a self-hosted tool to a managed platform serving everyday users.*
