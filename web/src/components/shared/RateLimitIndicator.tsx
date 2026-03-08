// ============================================================================
// Operator OS — Rate Limit Indicator
// Visual indicator for API rate limit status. Lives in the TopBar.
// Shows a compact icon that expands to a detailed popover on click.
// ============================================================================

import { useState, useRef, useEffect, useCallback, memo } from 'react'
import { Lightning, Warning, Timer } from '@phosphor-icons/react'
import {
  useRateLimitStore,
  getOverallSeverity,
  getBucketSeverity,
  timeUntilReset,
  type RateLimitSeverity,
} from '../../stores/rateLimitStore'
import type { RateLimitBucket } from '../../types/api'

// ---------------------------------------------------------------------------
// Style mappings
// ---------------------------------------------------------------------------

const severityDot: Record<RateLimitSeverity, string> = {
  ok: 'bg-success',
  caution: 'bg-accent',
  warning: 'bg-warning',
  critical: 'bg-error',
}

const severityText: Record<RateLimitSeverity, string> = {
  ok: 'text-success',
  caution: 'text-accent-text',
  warning: 'text-warning',
  critical: 'text-error',
}

const severityBg: Record<RateLimitSeverity, string> = {
  ok: 'bg-success-subtle/50',
  caution: 'bg-accent-subtle/50',
  warning: 'bg-warning-subtle/50',
  critical: 'bg-error-subtle/50',
}

const severityLabel: Record<RateLimitSeverity, string> = {
  ok: 'Normal',
  caution: 'Moderate',
  warning: 'High usage',
  critical: 'Near limit',
}

// ---------------------------------------------------------------------------
// Bucket row
// ---------------------------------------------------------------------------

const BucketRow = memo(function BucketRow({
  label,
  bucket,
}: {
  label: string
  bucket: RateLimitBucket
}) {
  const severity = getBucketSeverity(bucket)
  const pctUsed = bucket.limit > 0 ? (1 - bucket.remaining / bucket.limit) * 100 : 0
  const reset = timeUntilReset(bucket.resets_at)

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <span className="text-[12px] font-medium text-text-secondary">{label}</span>
        <span className={`text-[11px] font-semibold ${severityText[severity]}`}>
          {bucket.remaining}/{bucket.limit}
        </span>
      </div>

      {/* Progress bar */}
      <div className="h-1.5 rounded-full bg-surface-2 overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-500 ${severityDot[severity]}`}
          style={{ width: `${Math.min(pctUsed, 100)}%` }}
        />
      </div>

      {/* Reset timer */}
      {reset && (
        <div className="flex items-center gap-1 text-[10px] text-text-dim">
          <Timer size={10} />
          <span>Resets in {reset}</span>
        </div>
      )}
    </div>
  )
})

// ---------------------------------------------------------------------------
// Main component
// ---------------------------------------------------------------------------

export const RateLimitIndicator = memo(function RateLimitIndicator() {
  const { status, headerBucket, fetchStatus } = useRateLimitStore()
  const [popoverOpen, setPopoverOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  // Fetch on mount and periodically (every 2 minutes)
  useEffect(() => {
    fetchStatus()
    const interval = setInterval(fetchStatus, 120_000)
    return () => clearInterval(interval)
  }, [fetchStatus])

  // Close popover on outside click
  useEffect(() => {
    if (!popoverOpen) return
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setPopoverOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [popoverOpen])

  // Close on Escape
  useEffect(() => {
    if (!popoverOpen) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setPopoverOpen(false)
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [popoverOpen])

  const storeState = useRateLimitStore.getState()
  const severity = getOverallSeverity(storeState)

  // Don't render anything if we have no data at all
  if (!status && !headerBucket) return null

  const handleToggle = useCallback(() => {
    setPopoverOpen((prev) => !prev)
    // Refresh status when opening
    if (!popoverOpen) fetchStatus()
  }, [popoverOpen, fetchStatus])

  return (
    <div className="relative" ref={containerRef}>
      {/* ─── Trigger button ─── */}
      <button
        onClick={handleToggle}
        className={`
          flex items-center justify-center gap-1.5 px-2 py-1.5 rounded-lg
          text-text-dim hover:text-text-secondary hover:bg-surface-2/50
          transition-colors duration-150
          ${severity === 'critical' ? 'animate-pulse-glow' : ''}
        `}
        aria-label={`API rate limit: ${severityLabel[severity]}`}
        aria-expanded={popoverOpen}
      >
        {severity === 'ok' || severity === 'caution' ? (
          <Lightning size={16} weight={severity === 'caution' ? 'fill' : 'regular'} />
        ) : (
          <Warning size={16} weight="fill" className={severityText[severity]} />
        )}

        {/* Status dot — only visible when not "ok" */}
        {severity !== 'ok' && (
          <span className={`w-1.5 h-1.5 rounded-full ${severityDot[severity]}`} />
        )}
      </button>

      {/* ─── Popover ─── */}
      {popoverOpen && (
        <div className="absolute right-0 top-full mt-1.5 w-72 bg-surface border border-border rounded-xl shadow-[0_8px_32px_var(--glass-shadow)] overflow-hidden animate-fade-slide z-50">
          {/* Header */}
          <div className={`px-4 py-3 border-b border-border-subtle ${severityBg[severity]}`}>
            <div className="flex items-center gap-2">
              <span className={`w-2 h-2 rounded-full ${severityDot[severity]}`} />
              <span className={`text-[13px] font-semibold ${severityText[severity]}`}>
                {severityLabel[severity]}
              </span>
            </div>
            {status?.plan && (
              <p className="text-[11px] text-text-dim mt-1">
                Plan: <span className="font-medium text-text-secondary capitalize">{status.plan}</span>
              </p>
            )}
          </div>

          {/* Buckets */}
          <div className="px-4 py-3 space-y-3">
            {status?.per_minute && (
              <BucketRow label="Per minute" bucket={status.per_minute} />
            )}
            {status?.daily && (
              <BucketRow label="Daily" bucket={status.daily} />
            )}
            {headerBucket && !status && (
              <BucketRow label="Current window" bucket={headerBucket} />
            )}
          </div>

          {/* Footer — critical warning */}
          {severity === 'critical' && (
            <div className="px-4 py-2.5 bg-error-subtle/30 border-t border-border-subtle">
              <p className="text-[11px] text-error font-medium">
                You're approaching your rate limit. Requests may be throttled.
              </p>
            </div>
          )}
          {severity === 'warning' && (
            <div className="px-4 py-2.5 bg-warning-subtle/30 border-t border-border-subtle">
              <p className="text-[11px] text-warning font-medium">
                Usage is elevated. Consider spacing out requests.
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  )
})
