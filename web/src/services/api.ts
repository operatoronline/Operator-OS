// ============================================================================
// Operator OS — API Client
// Typed fetch wrapper with JWT auth, auto-refresh, and error normalization.
// ============================================================================

import type {
  ApiError,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  UserProfile,
  RefreshRequest,
  Agent,
  CreateAgentRequest,
  UpdateAgentRequest,
  Session,
  CreateSessionRequest,
  UpdateSessionRequest,
  SessionMessage,
  Plan,
  Subscription,
  CheckoutRequest,
  CheckoutResponse,
  PortalResponse,
  PlanChangeRequest,
  PlanChangeResult,
  UsageSummary,
  DailyUsage,
  ModelUsage,
  UsageLimits,
  UsageEvent,
  OverageStatus,
  IntegrationSummary,
  IntegrationStatus,
  ConnectRequest,
  ConnectResponse,
  UserIntegration,
  OAuthProvider,
  AdminUser,
  PlatformStats,
  SetRoleRequest,
  AuditEvent,
  AuditCountResponse,
  DataSubjectRequest,
  RetentionPolicy,
  RateLimitStatus,
  DetailedHealth,
  UpdateProfileRequest,
  ChangePasswordRequest,
  NotificationPreferences,
  ApiKey,
  CreateApiKeyRequest,
  CreateApiKeyResponse,
  SecurityAuditReport,
} from '../types/api'

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

const API_BASE = '/api/v1'
const TOKEN_KEY = 'os-access-token'
const REFRESH_KEY = 'os-refresh-token'

// ---------------------------------------------------------------------------
// Token storage
// ---------------------------------------------------------------------------

export const tokenStore = {
  getAccess: (): string | null => localStorage.getItem(TOKEN_KEY),
  getRefresh: (): string | null => localStorage.getItem(REFRESH_KEY),
  set(access: string, refresh: string) {
    localStorage.setItem(TOKEN_KEY, access)
    localStorage.setItem(REFRESH_KEY, refresh)
  },
  clear() {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(REFRESH_KEY)
  },
}

// ---------------------------------------------------------------------------
// Error class
// ---------------------------------------------------------------------------

export class ApiRequestError extends Error {
  status: number
  code: string
  body: ApiError

  constructor(status: number, body: ApiError) {
    super(body.message || body.error)
    this.name = 'ApiRequestError'
    this.status = status
    this.code = body.code || body.error
    this.body = body
  }
}

// ---------------------------------------------------------------------------
// Rate limit header extraction
// ---------------------------------------------------------------------------

/**
 * Extract X-RateLimit-* headers and forward to the rate limit store.
 * Uses lazy import to avoid circular dependency at module load time.
 */
function _extractRateLimitHeaders(headers: Headers) {
  // Only process if rate limit headers are present
  if (!headers.has('X-RateLimit-Limit')) return

  // Lazy dynamic import avoids circular dep (store imports api)
  import('../stores/rateLimitStore').then(({ useRateLimitStore }) => {
    useRateLimitStore.getState().updateFromHeaders(headers)
  })
}

// ---------------------------------------------------------------------------
// Core fetch wrapper
// ---------------------------------------------------------------------------

/** Pending refresh promise — deduplicates concurrent refresh calls */
let refreshPromise: Promise<boolean> | null = null

async function refreshTokens(): Promise<boolean> {
  const refreshToken = tokenStore.getRefresh()
  if (!refreshToken) return false

  try {
    const res = await fetch(`${API_BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken } satisfies RefreshRequest),
    })

    if (!res.ok) {
      tokenStore.clear()
      return false
    }

    const data: LoginResponse = await res.json()
    tokenStore.set(data.access_token, data.refresh_token)
    return true
  } catch {
    tokenStore.clear()
    return false
  }
}

interface RequestOptions extends Omit<RequestInit, 'body'> {
  /** Skip JWT auth header (for login/register) */
  noAuth?: boolean
  /** Query parameters */
  params?: Record<string, string | number | boolean | undefined>
  /** JSON body (auto-serialized) */
  body?: unknown
}

async function request<T>(
  method: string,
  path: string,
  opts: RequestOptions = {},
): Promise<T> {
  try {
    return await _request<T>(method, path, opts)
  } catch (err) {
    // Network errors (fetch itself failed — no response)
    if (err instanceof TypeError && !(err instanceof ApiRequestError)) {
      import('../stores/toastStore').then(({ toast }) => {
        toast.error('Network error', 'Unable to reach the server. Check your connection.')
      })
    }
    throw err
  }
}

async function _request<T>(
  method: string,
  path: string,
  opts: RequestOptions = {},
): Promise<T> {
  const { noAuth, params, body, ...fetchOpts } = opts

  // Build URL with query params
  let url = `${API_BASE}${path}`
  if (params) {
    const sp = new URLSearchParams()
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined) sp.set(k, String(v))
    }
    const qs = sp.toString()
    if (qs) url += `?${qs}`
  }

  // Build headers
  const headers = new Headers(fetchOpts.headers)
  if (body !== undefined) {
    headers.set('Content-Type', 'application/json')
  }
  if (!noAuth) {
    const token = tokenStore.getAccess()
    if (token) headers.set('Authorization', `Bearer ${token}`)
  }

  const res = await fetch(url, {
    method,
    ...fetchOpts,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  // Handle 401 — try refresh once, then retry
  if (res.status === 401 && !noAuth) {
    if (!refreshPromise) {
      refreshPromise = refreshTokens().finally(() => {
        refreshPromise = null
      })
    }
    const refreshed = await refreshPromise
    if (refreshed) {
      // Retry original request with new token
      const retryHeaders = new Headers(headers)
      retryHeaders.set('Authorization', `Bearer ${tokenStore.getAccess()}`)
      const retry = await fetch(url, {
        method,
        ...fetchOpts,
        headers: retryHeaders,
        body: body !== undefined ? JSON.stringify(body) : undefined,
      })
      if (retry.ok) {
        _extractRateLimitHeaders(retry.headers)
        if (retry.status === 204) return undefined as T
        return retry.json() as Promise<T>
      }
      const errBody = await retry.json().catch(() => ({ error: 'request_failed' }))
      throw new ApiRequestError(retry.status, errBody as ApiError)
    }

    // Refresh failed — emit event for auth store to handle redirect
    window.dispatchEvent(new CustomEvent('os:auth:expired'))
    const errBody = await res.json().catch(() => ({ error: 'unauthorized' }))
    throw new ApiRequestError(401, errBody as ApiError)
  }

  // Capture rate limit headers from every response
  _extractRateLimitHeaders(res.headers)

  // Handle other errors
  if (!res.ok) {
    const errBody = await res.json().catch(() => ({
      error: 'request_failed',
      message: res.statusText,
    }))
    const err = new ApiRequestError(res.status, errBody as ApiError)
    // Show toast for server errors (5xx) and rate limits (429)
    // Client errors (4xx) are typically handled by calling code
    if (res.status >= 500 || res.status === 429) {
      import('../stores/toastStore').then(({ toast }) => {
        if (res.status === 429) {
          toast.warning('Rate limited', 'Too many requests — please slow down.')
        } else {
          toast.error('Server error', err.message || 'Something went wrong. Please try again.')
        }
      })
    }
    throw err
  }

  // 204 No Content
  if (res.status === 204) return undefined as T

  return res.json() as Promise<T>
}

// Convenience methods
const get = <T>(path: string, opts?: RequestOptions) =>
  request<T>('GET', path, opts)

const post = <T>(path: string, body?: unknown, opts?: RequestOptions) =>
  request<T>('POST', path, { ...opts, body })

const put = <T>(path: string, body?: unknown, opts?: RequestOptions) =>
  request<T>('PUT', path, { ...opts, body })

const del = <T>(path: string, opts?: RequestOptions) =>
  request<T>('DELETE', path, opts)

/** Fetch with absolute URL (bypasses API_BASE prefix) */
async function fetchRaw<T>(method: string, absolutePath: string): Promise<T> {
  const res = await fetch(absolutePath, { method })
  if (!res.ok) {
    const errBody = await res.json().catch(() => ({ error: 'request_failed' }))
    throw new ApiRequestError(res.status, errBody as ApiError)
  }
  return res.json() as Promise<T>
}

// ============================================================================
// API Namespace — organized by domain
// ============================================================================

export const api = {
  // -------------------------------------------------------------------------
  // Auth
  // -------------------------------------------------------------------------
  auth: {
    login: (data: LoginRequest) =>
      post<LoginResponse>('/auth/login', data, { noAuth: true }),

    register: (data: RegisterRequest) =>
      post<UserProfile>('/auth/register', data, { noAuth: true }),

    verifyEmail: (token: string) =>
      post<void>('/auth/verify-email', { token }, { noAuth: true }),

    resendVerification: (email: string) =>
      post<void>('/auth/resend-verification', { email }, { noAuth: true }),

    refresh: (refreshToken: string) =>
      post<LoginResponse>('/auth/refresh', { refresh_token: refreshToken } satisfies RefreshRequest, { noAuth: true }),
  },

  // -------------------------------------------------------------------------
  // Agents
  // -------------------------------------------------------------------------
  agents: {
    list: () => get<Agent[]>('/agents'),

    get: (id: string) => get<Agent>(`/agents/${id}`),

    create: (data: CreateAgentRequest) => post<Agent>('/agents', data),

    update: (id: string, data: UpdateAgentRequest) =>
      put<Agent>(`/agents/${id}`, data),

    delete: (id: string) => del<void>(`/agents/${id}`),

    setDefault: (id: string) => post<void>(`/agents/${id}/default`),
  },

  // -------------------------------------------------------------------------
  // Sessions
  // -------------------------------------------------------------------------
  sessions: {
    list: (params?: { archived?: boolean; page?: number; per_page?: number }) =>
      get<Session[]>('/sessions', { params }),

    get: (id: string) => get<Session>(`/sessions/${id}`),

    create: (data: CreateSessionRequest) => post<Session>('/sessions', data),

    update: (id: string, data: UpdateSessionRequest) =>
      put<Session>(`/sessions/${id}`, data),

    delete: (id: string) => del<void>(`/sessions/${id}`),

    messages: (id: string, params?: { page?: number; per_page?: number; before?: string }) =>
      get<SessionMessage[]>(`/sessions/${id}/messages`, { params }),
  },

  // -------------------------------------------------------------------------
  // Billing
  // -------------------------------------------------------------------------
  billing: {
    plans: () => get<Plan[]>('/billing/plans'),

    plan: (id: string) => get<Plan>(`/billing/plans/${id}`),

    subscription: () => get<Subscription>('/billing/subscription'),

    checkout: (data: CheckoutRequest) =>
      post<CheckoutResponse>('/billing/checkout', data),

    portal: () => post<PortalResponse>('/billing/portal'),

    changePlan: (data: PlanChangeRequest) =>
      post<PlanChangeResult>('/billing/change-plan', data),

    previewChange: (data: PlanChangeRequest) =>
      post<PlanChangeResult>('/billing/preview-change', data),
  },

  // -------------------------------------------------------------------------
  // Usage
  // -------------------------------------------------------------------------
  usage: {
    summary: () => get<UsageSummary>('/billing/usage'),

    daily: (params?: { start?: string; end?: string }) =>
      get<DailyUsage[]>('/billing/usage/daily', { params }),

    byModel: () => get<ModelUsage[]>('/billing/usage/models'),

    limits: () => get<UsageLimits>('/billing/usage/limits'),

    events: (params?: { page?: number; per_page?: number; model?: string }) =>
      get<UsageEvent[]>('/billing/usage/events', { params }),

    overage: () => get<OverageStatus>('/billing/overage'),
  },

  // -------------------------------------------------------------------------
  // Integrations
  // -------------------------------------------------------------------------
  integrations: {
    list: () => get<IntegrationSummary[]>('/integrations'),

    get: (id: string) => get<IntegrationSummary>(`/integrations/${id}`),

    categories: () => get<string[]>('/integrations/categories'),

    connect: (data: ConnectRequest) =>
      post<ConnectResponse>('/manage/integrations/connect', data),

    disconnect: (data: { integration_id: string }) =>
      post<void>('/manage/integrations/disconnect', data),

    status: () => get<IntegrationStatus[]>('/manage/integrations/status'),

    integrationStatus: (id: string) =>
      get<IntegrationStatus>(`/manage/integrations/${id}/status`),

    enable: (id: string) =>
      post<void>(`/manage/integrations/${id}/enable`),

    disable: (id: string) =>
      post<void>(`/manage/integrations/${id}/disable`),

    reconnect: (id: string) =>
      post<void>(`/manage/integrations/${id}/reconnect`),

    updateConfig: (id: string, config: Record<string, string>) =>
      put<void>(`/manage/integrations/${id}/config`, config),
  },

  // -------------------------------------------------------------------------
  // User Integrations
  // -------------------------------------------------------------------------
  userIntegrations: {
    list: () => get<UserIntegration[]>('/user/integrations'),

    get: (id: string) => get<UserIntegration>(`/user/integrations/${id}`),

    create: (data: ConnectRequest) =>
      post<UserIntegration>('/user/integrations', data),

    delete: (id: string) => del<void>(`/user/integrations/${id}`),
  },

  // -------------------------------------------------------------------------
  // OAuth
  // -------------------------------------------------------------------------
  oauth: {
    providers: () => get<OAuthProvider[]>('/oauth/providers'),

    authorize: (data: { provider: string; scopes?: string[]; redirect_uri?: string }) =>
      post<{ auth_url: string }>('/oauth/authorize', data),

    refresh: (data: { provider: string; refresh_token: string }) =>
      post<void>('/oauth/refresh', data),
  },

  // -------------------------------------------------------------------------
  // User (profile, password, notifications, API keys)
  // -------------------------------------------------------------------------
  user: {
    profile: () => get<UserProfile>('/user/profile'),

    updateProfile: (data: UpdateProfileRequest) =>
      put<UserProfile>('/user/profile', data),

    changePassword: (data: ChangePasswordRequest) =>
      post<void>('/user/password', data),

    notifications: () => get<NotificationPreferences>('/user/notifications'),

    updateNotifications: (data: Partial<NotificationPreferences>) =>
      put<NotificationPreferences>('/user/notifications', data),

    apiKeys: () => get<ApiKey[]>('/user/api-keys'),

    createApiKey: (data: CreateApiKeyRequest) =>
      post<CreateApiKeyResponse>('/user/api-keys', data),

    deleteApiKey: (id: string) => del<void>(`/user/api-keys/${id}`),
  },

  // -------------------------------------------------------------------------
  // Admin
  // -------------------------------------------------------------------------
  admin: {
    users: (params?: { page?: number; per_page?: number; search?: string; status?: string }) =>
      get<AdminUser[]>('/admin/users', { params }),

    user: (id: string) => get<AdminUser>(`/admin/users/${id}`),

    updateUser: (id: string, data: Partial<AdminUser>) =>
      put<AdminUser>(`/admin/users/${id}`, data),

    deleteUser: (id: string) => del<void>(`/admin/users/${id}`),

    suspendUser: (id: string) =>
      post<void>(`/admin/users/${id}/suspend`),

    activateUser: (id: string) =>
      post<void>(`/admin/users/${id}/activate`),

    setRole: (id: string, data: SetRoleRequest) =>
      post<void>(`/admin/users/${id}/role`, data),

    stats: () => get<PlatformStats>('/admin/stats'),

    securityAudit: (params?: { categories?: string }) =>
      get<SecurityAuditReport>('/admin/security-audit', { params }),
  },

  // -------------------------------------------------------------------------
  // Audit
  // -------------------------------------------------------------------------
  audit: {
    events: (params?: {
      page?: number
      per_page?: number
      action?: string
      user_id?: string
      resource?: string
      start?: string
      end?: string
    }) => get<AuditEvent[]>('/audit/events', { params }),

    count: (params?: {
      action?: string
      user_id?: string
      resource?: string
      start?: string
      end?: string
    }) => get<AuditCountResponse>('/audit/events/count', { params }),
  },

  // -------------------------------------------------------------------------
  // GDPR
  // -------------------------------------------------------------------------
  gdpr: {
    export: () => post<DataSubjectRequest>('/gdpr/export'),

    erase: () => post<DataSubjectRequest>('/gdpr/erase'),

    requests: () => get<DataSubjectRequest[]>('/gdpr/requests'),

    request: (id: string) => get<DataSubjectRequest>(`/gdpr/requests/${id}`),

    cancelRequest: (id: string) => del<void>(`/gdpr/requests/${id}`),

    retention: () => get<RetentionPolicy>('/gdpr/retention'),
  },

  // -------------------------------------------------------------------------
  // Rate Limiting
  // -------------------------------------------------------------------------
  rateLimit: {
    status: () => get<RateLimitStatus>('/rate-limit/status'),
  },

  // -------------------------------------------------------------------------
  // Health (no auth)
  // -------------------------------------------------------------------------
  health: {
    live: () => fetchRaw<{ status: string }>('GET', '/health/live'),
    ready: () => fetchRaw<{ status: string }>('GET', '/health/ready'),
    detailed: () => fetchRaw<DetailedHealth>('GET', '/health/detailed'),
    component: (name: string) =>
      fetchRaw<Record<string, unknown>>('GET', `/health/component/${name}`),
  },
} as const
