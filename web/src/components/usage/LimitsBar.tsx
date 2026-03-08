// ============================================================================
// Operator OS — Usage Limits Bar
// Shows current usage vs plan limits with progress bars and warnings.
// ============================================================================

import { memo } from 'react'
import { Warning, Infinity as InfinityIcon } from '@phosphor-icons/react'
import type { UsageLimits } from '../../types/api'

interface LimitsBarProps {
  limits: UsageLimits | null
  loading: boolean
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toLocaleString()
}

interface LimitRowProps {
  label: string
  used: number
  limit: number
  unlimited: boolean
}

function LimitRow({ label, used, limit, unlimited }: LimitRowProps) {
  const pct = unlimited ? 0 : Math.min((used / Math.max(limit, 1)) * 100, 100)
  const isWarning = !unlimited && pct >= 80
  const isDanger = !unlimited && pct >= 95

  return (
    <div>
      <div className="flex items-center justify-between mb-1.5">
        <span className="text-xs font-medium text-text">{label}</span>
        <span className="text-xs text-text-secondary tabular-nums">
          {formatNumber(used)}
          {unlimited ? (
            <span className="inline-flex items-center ml-1 text-text-dim">
              / <InfinityIcon size={12} className="ml-0.5" />
            </span>
          ) : (
            <span className="text-text-dim"> / {formatNumber(limit)}</span>
          )}
        </span>
      </div>

      <div className="h-2.5 bg-surface-2 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-500 ease-out ${
            isDanger
              ? 'bg-error'
              : isWarning
                ? 'bg-warning'
                : 'bg-accent'
          }`}
          style={{ width: unlimited ? '0%' : `${pct}%` }}
        />
      </div>

      {isWarning && !isDanger && (
        <div className="flex items-center gap-1 mt-1">
          <Warning size={10} weight="fill" className="text-warning" />
          <span className="text-[10px] text-warning">
            {Math.round(pct)}% used — approaching limit
          </span>
        </div>
      )}
      {isDanger && (
        <div className="flex items-center gap-1 mt-1">
          <Warning size={10} weight="fill" className="text-error" />
          <span className="text-[10px] text-error">
            {Math.round(pct)}% used — near limit
          </span>
        </div>
      )}
    </div>
  )
}

export const LimitsBar = memo(function LimitsBar({
  limits,
  loading,
}: LimitsBarProps) {
  if (loading || !limits) {
    return (
      <div className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius-md)] p-5">
        <div className="h-4 bg-surface-2 rounded w-28 mb-5 animate-pulse" />
        <div className="space-y-5">
          {[1, 2].map((i) => (
            <div key={i} className="animate-pulse">
              <div className="flex justify-between mb-2">
                <div className="h-3 bg-surface-2 rounded w-20" />
                <div className="h-3 bg-surface-2 rounded w-24" />
              </div>
              <div className="h-2.5 bg-surface-2 rounded-full" />
            </div>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius-md)] p-5">
      <div className="flex items-center justify-between mb-5">
        <h3 className="text-sm font-semibold text-text">Plan Limits</h3>
        <span className="text-[10px] text-text-dim uppercase tracking-wider">
          {limits.plan_id} plan
        </span>
      </div>

      <div className="space-y-5">
        <LimitRow
          label="Tokens"
          used={limits.tokens.used}
          limit={limits.tokens.limit}
          unlimited={limits.tokens.unlimited}
        />
        <LimitRow
          label="Messages"
          used={limits.messages.used}
          limit={limits.messages.limit}
          unlimited={limits.messages.unlimited}
        />
      </div>
    </div>
  )
})
