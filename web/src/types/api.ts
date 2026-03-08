// ============================================================================
// Operator OS — API Types
// Generated from OpenAPI spec (pkg/openapi/spec.json)
// ============================================================================

// ---------------------------------------------------------------------------
// Common
// ---------------------------------------------------------------------------

/** Standard API error response */
export interface ApiError {
  error: string
  code?: string
  message?: string
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
  user: UserProfile
}

export interface RegisterRequest {
  email: string
  password: string
  display_name?: string
}

export interface UserProfile {
  id: string
  email: string
  display_name: string
  status: UserStatus
  email_verified: boolean
}

export type UserStatus = 'pending_verification' | 'active' | 'suspended' | 'deleted'

export interface RefreshRequest {
  refresh_token: string
}

export interface VerifyEmailRequest {
  token: string
}

export interface ResendVerificationRequest {
  email: string
}

// ---------------------------------------------------------------------------
// Agents
// ---------------------------------------------------------------------------

export interface AgentIntegrationScope {
  integration_id: string
  allowed_tools?: string[]
  allowed_scopes?: string[]
}

export interface Agent {
  id: string
  name: string
  description: string
  system_prompt: string
  model: string
  model_fallbacks: string[]
  tools: string[]
  skills: string[]
  max_tokens: number
  temperature: number
  max_iterations: number
  is_default: boolean
  status: AgentStatus
  allowed_integrations: AgentIntegrationScope[]
  created_at: string
  updated_at: string
}

export type AgentStatus = 'active' | 'archived'

export interface CreateAgentRequest {
  name: string
  description?: string
  system_prompt?: string
  model?: string
  model_fallbacks?: string[]
  tools?: string[]
  skills?: string[]
  max_tokens?: number
  temperature?: number
  max_iterations?: number
  is_default?: boolean
  allowed_integrations?: AgentIntegrationScope[]
}

export interface UpdateAgentRequest {
  name?: string
  description?: string
  system_prompt?: string
  model?: string
  model_fallbacks?: string[]
  tools?: string[]
  skills?: string[]
  max_tokens?: number
  temperature?: number
  max_iterations?: number
  status?: AgentStatus
  allowed_integrations?: AgentIntegrationScope[]
}

// ---------------------------------------------------------------------------
// Billing
// ---------------------------------------------------------------------------

export type PlanId = 'free' | 'starter' | 'pro' | 'enterprise'
export type BillingInterval = 'monthly' | 'yearly'

export interface PlanLimits {
  max_agents: number
  max_messages_per_month: number
  max_tokens_per_month: number
  max_integrations: number
  max_storage_mb: number
  max_team_members: number
  allowed_models: string[]
  custom_skills: boolean
  api_access: boolean
  rate_limit_rpm: number
  rate_limit_burst: number
  rate_limit_daily: number
}

export interface Plan {
  id: PlanId
  name: string
  price_monthly_cents: number
  price_yearly_cents: number
  active: boolean
  limits: PlanLimits
}

export interface Subscription {
  id: string
  user_id: string
  plan_id: string
  status: SubscriptionStatus
  interval: BillingInterval
  current_period_start: string
  current_period_end: string
  cancel_at_period_end: boolean
  plan: Plan
}

export type SubscriptionStatus =
  | 'active'
  | 'trialing'
  | 'past_due'
  | 'canceled'
  | 'expired'
  | 'paused'

export interface CheckoutRequest {
  plan_id: string
  interval?: BillingInterval
  success_url: string
  cancel_url: string
  trial_days?: number
}

export interface CheckoutResponse {
  url: string
}

export interface PortalResponse {
  url: string
}

export interface PlanChangeRequest {
  plan_id: string
  interval?: BillingInterval
  mode?: 'immediate' | 'at_period_end'
}

export interface PlanChangeResult {
  direction: 'upgrade' | 'downgrade' | 'same'
  mode: 'immediate' | 'at_period_end'
  previous_plan: string
  new_plan: string
  effective_at: string
  subscription: Subscription
  proration_amount: number
}

// ---------------------------------------------------------------------------
// Usage
// ---------------------------------------------------------------------------

export interface UsageSummary {
  total_input_tokens: number
  total_output_tokens: number
  total_tokens: number
  total_requests: number
  total_cost: number
}

export interface DailyUsage {
  date: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  requests: number
  cost: number
}

export interface ModelUsage {
  model: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  requests: number
  cost: number
}

export interface UsageLimits {
  plan_id: string
  tokens: { used: number; limit: number; unlimited: boolean }
  messages: { used: number; limit: number; unlimited: boolean }
}

export interface UsageEvent {
  id: string
  user_id: string
  model: string
  provider: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  session_key: string
  agent_id: string
  duration_ms: number
  estimated_cost: number
  created_at: string
}

export type OverageLevel = 'none' | 'warning' | 'soft_cap' | 'hard_cap' | 'blocked'
export type OverageAction = 'none' | 'warn' | 'downgrade_model' | 'throttle' | 'block'

export interface OverageResource {
  resource: string
  level: string
  action: string
  message: string
  usage: number
  limit: number
  percent: number
}

export interface OverageStatus {
  overall_level: OverageLevel
  overall_action: OverageAction
  resources: OverageResource[]
}

// ---------------------------------------------------------------------------
// Integrations
// ---------------------------------------------------------------------------

export interface IntegrationTool {
  name: string
  description: string
}

export interface IntegrationSummary {
  id: string
  name: string
  icon: string
  category: string
  description: string
  auth_type: 'oauth2' | 'api_key' | 'none'
  status: string
  required_plan: string
  tools: IntegrationTool[]
}

export interface IntegrationTokenStatus {
  has_access_token: boolean
  has_refresh_token: boolean
  expires_at: string
  is_expired: boolean
  needs_refresh: boolean
  token_status: string
}

export interface IntegrationRefreshStatus {
  retry_count: number
  max_retries: number
  last_error: string
  next_retry: string
  exhausted: boolean
}

export interface IntegrationStatus {
  id: string
  integration_id: string
  name: string
  category: string
  auth_type: string
  status: string
  token_status: IntegrationTokenStatus
  refresh_status: IntegrationRefreshStatus
}

export interface ConnectRequest {
  integration_id: string
  config?: Record<string, string>
  scopes?: string[]
  redirect_after?: string
  api_key?: string
}

export interface ConnectResponse {
  integration_id: string
  status: string
  auth_url: string
  message: string
}

export interface UserIntegration {
  id: string
  user_id: string
  integration_id: string
  status: 'pending' | 'active' | 'failed' | 'revoked' | 'disabled'
  config: Record<string, string>
  scopes: string[]
  error_message: string
  last_used_at: string
  created_at: string
  updated_at: string
}

// ---------------------------------------------------------------------------
// OAuth
// ---------------------------------------------------------------------------

export interface OAuthProvider {
  id: string
  name: string
  auth_url: string
  scopes: string[]
  use_pkce: boolean
}

export interface OAuthAuthorizeRequest {
  provider: string
  scopes?: string[]
  redirect_uri?: string
}

export interface OAuthTokenResponse {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
  scope: string
  id_token: string
}

// ---------------------------------------------------------------------------
// Admin
// ---------------------------------------------------------------------------

export interface AdminUser {
  id: string
  email: string
  display_name: string
  role: 'user' | 'admin'
  status: string
  email_verified: boolean
  created_at: string
  updated_at: string
}

export interface PlatformStats {
  total_users: number
  active_users: number
  pending_users: number
  suspended_users: number
}

export interface SetRoleRequest {
  role: 'user' | 'admin'
}

// ---------------------------------------------------------------------------
// Audit
// ---------------------------------------------------------------------------

export interface AuditEvent {
  id: string
  user_id: string
  actor_id: string
  action: string
  resource: string
  resource_id: string
  status: 'success' | 'failure'
  detail: Record<string, unknown>
  ip_address: string
  user_agent: string
  created_at: string
}

export interface AuditCountResponse {
  count: number
}

// ---------------------------------------------------------------------------
// User Profile
// ---------------------------------------------------------------------------

export interface UpdateProfileRequest {
  display_name?: string
}

export interface ChangePasswordRequest {
  current_password: string
  new_password: string
}

export interface NotificationPreferences {
  email_billing: boolean
  email_security: boolean
  email_product: boolean
  push_enabled: boolean
}

export interface ApiKey {
  id: string
  name: string
  prefix: string
  last_used_at: string
  created_at: string
  expires_at: string
}

export interface CreateApiKeyRequest {
  name: string
  expires_in_days?: number
}

export interface CreateApiKeyResponse {
  key: ApiKey
  secret: string
}

// ---------------------------------------------------------------------------
// GDPR
// ---------------------------------------------------------------------------

export interface DataSubjectRequest {
  id: string
  user_id: string
  type: 'export' | 'erasure'
  status: 'pending' | 'processing' | 'completed' | 'failed' | 'canceled'
  requested_by: string
  result: Record<string, unknown>
  error_message: string
  created_at: string
  completed_at: string
}

export interface RetentionPolicy {
  audit_logs_days: number
  usage_data_days: number
  session_data_days: number
  deleted_users_days: number
}

// ---------------------------------------------------------------------------
// Rate Limiting
// ---------------------------------------------------------------------------

export interface RateLimitBucket {
  limit: number
  remaining: number
  burst?: number
  resets_at?: string
}

export interface RateLimitStatus {
  plan: string
  per_minute: RateLimitBucket
  daily: RateLimitBucket
}

// ---------------------------------------------------------------------------
// Security Audit
// ---------------------------------------------------------------------------

export type AuditSeverity = 'critical' | 'high' | 'medium' | 'low' | 'info'

export type AuditCategory =
  | 'authentication'
  | 'authorization'
  | 'input_validation'
  | 'cryptography'
  | 'session_management'
  | 'api_security'
  | 'configuration'
  | 'data_protection'
  | 'rate_limiting'
  | 'security_headers'
  | 'injection'
  | 'compliance'

export interface SecurityFinding {
  id: string
  category: AuditCategory
  severity: AuditSeverity
  title: string
  description: string
  location?: string
  evidence?: string
  remediation?: string
  references?: string[]
  passed: boolean
}

export interface SecuritySummary {
  total: number
  critical: number
  high: number
  medium: number
  low: number
  info: number
  passed: number
  failed: number
}

export interface CategoryStats {
  total: number
  passed: number
  failed: number
}

export interface SecurityAuditReport {
  timestamp: string
  duration: number // nanoseconds
  checks_run: number
  findings: SecurityFinding[]
  summary: SecuritySummary
  risk_score: number   // 0-100, lower = better
  pass_rate: number    // 0-100%
  categories: Record<AuditCategory, CategoryStats>
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

export type HealthStatus = 'healthy' | 'degraded' | 'unhealthy'

export interface ComponentHealth {
  status: HealthStatus
  type: 'database' | 'cache' | 'messaging' | 'external' | 'internal'
  critical: boolean
  duration_ms: number
  details: Record<string, unknown>
}

export interface DetailedHealth {
  status: HealthStatus
  uptime: number
  timestamp: string
  components: Record<string, ComponentHealth>
  summary: {
    total: number
    healthy: number
    degraded: number
    unhealthy: number
  }
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

export interface Session {
  id: string
  user_id: string
  agent_id: string
  name: string
  created_at: string
  updated_at: string
  last_message_at: string
  message_count: number
  pinned: boolean
  archived: boolean
}

export interface CreateSessionRequest {
  agent_id?: string
  name?: string
}

export interface UpdateSessionRequest {
  name?: string
  pinned?: boolean
  archived?: boolean
}

export interface SessionMessage {
  id: string
  session_id: string
  role: 'user' | 'agent' | 'system'
  content: string
  agent_id?: string
  model?: string
  created_at: string
}

// ---------------------------------------------------------------------------
// Paginated list wrapper (common API pattern)
// ---------------------------------------------------------------------------

export interface PaginatedResponse<T> {
  data: T[]
  total?: number
  page?: number
  per_page?: number
  has_more?: boolean
}
