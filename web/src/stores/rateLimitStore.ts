// ============================================================================
// Operator OS — Rate Limit Store
// Tracks API rate limit state from response headers + status endpoint.
// ============================================================================

import { create } from 'zustand'
import { api, ApiRequestError } from '../services/api'
import type { RateLimitStatus, RateLimitBucket } from '../types/api'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface RateLimitState {
  /** Full status from /rate-limit/status endpoint */
  status: RateLimitStatus | null

  /** Rate limit buckets extracted from X-RateLimit-* response headers */
  headerBucket: RateLimitBucket | null

  /** Last time we fetched from the API endpoint */
  lastFetch: number

  /** Loading state */
  loading: boolean

  /** Error message (cleared on next success) */
  error: string | null

  // Actions
  fetchStatus: () => Promise<void>
  updateFromHeaders: (headers: Headers) => void
  reset: () => void
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Severity level based on remaining/limit ratio */
export type RateLimitSeverity = 'ok' | 'caution' | 'warning' | 'critical'

export function getBucketSeverity(bucket: RateLimitBucket): RateLimitSeverity {
  if (bucket.limit === 0) return 'ok'
  const pctUsed = 1 - bucket.remaining / bucket.limit
  if (pctUsed >= 0.95) return 'critical'
  if (pctUsed >= 0.80) return 'warning'
  if (pctUsed >= 0.60) return 'caution'
  return 'ok'
}

/** Returns the worst severity across all known buckets */
export function getOverallSeverity(state: RateLimitState): RateLimitSeverity {
  const buckets: RateLimitBucket[] = []
  if (state.status) {
    buckets.push(state.status.per_minute, state.status.daily)
  }
  if (state.headerBucket) {
    buckets.push(state.headerBucket)
  }
  if (buckets.length === 0) return 'ok'

  const severityOrder: RateLimitSeverity[] = ['ok', 'caution', 'warning', 'critical']
  let worst = 0
  for (const b of buckets) {
    const s = severityOrder.indexOf(getBucketSeverity(b))
    if (s > worst) worst = s
  }
  return severityOrder[worst]
}

/** Time until bucket resets, human-readable */
export function timeUntilReset(resetsAt?: string): string {
  if (!resetsAt) return ''
  const ms = new Date(resetsAt).getTime() - Date.now()
  if (ms <= 0) return 'now'
  if (ms < 60_000) return `${Math.ceil(ms / 1000)}s`
  if (ms < 3600_000) return `${Math.ceil(ms / 60_000)}m`
  return `${Math.ceil(ms / 3600_000)}h`
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useRateLimitStore = create<RateLimitState>((set, get) => ({
  status: null,
  headerBucket: null,
  lastFetch: 0,
  loading: false,
  error: null,

  fetchStatus: async () => {
    // Throttle: at most once per 30 seconds
    if (Date.now() - get().lastFetch < 30_000) return
    set({ loading: true, error: null })

    try {
      const status = await api.rateLimit.status()
      set({ status, lastFetch: Date.now(), loading: false })
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to fetch rate limits'
      set({ error: msg, loading: false })
    }
  },

  updateFromHeaders: (headers: Headers) => {
    const limit = headers.get('X-RateLimit-Limit')
    const remaining = headers.get('X-RateLimit-Remaining')
    const reset = headers.get('X-RateLimit-Reset')

    if (limit === null || remaining === null) return

    const bucket: RateLimitBucket = {
      limit: parseInt(limit, 10),
      remaining: parseInt(remaining, 10),
      resets_at: reset
        ? new Date(parseInt(reset, 10) * 1000).toISOString()
        : undefined,
    }

    // Only update if values are valid numbers
    if (isNaN(bucket.limit) || isNaN(bucket.remaining)) return

    set({ headerBucket: bucket })

    // If remaining is low, auto-fetch full status for context
    const severity = getBucketSeverity(bucket)
    if (severity === 'warning' || severity === 'critical') {
      get().fetchStatus()
    }
  },

  reset: () =>
    set({ status: null, headerBucket: null, lastFetch: 0, error: null }),
}))
