// ============================================================================
// Operator OS — Model Breakdown
// Table showing token usage per model with proportional bars.
// ============================================================================

import { memo, useMemo } from 'react'
import { Cube } from '@phosphor-icons/react'
import type { ModelUsage } from '../../types/api'

interface ModelBreakdownProps {
  models: ModelUsage[]
  loading: boolean
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toLocaleString()
}

function formatCost(cents: number): string {
  if (cents === 0) return '—'
  return `$${(cents / 100).toFixed(2)}`
}

// Consistent colors for models using OKLCH hue rotation
const MODEL_HUES = [260, 145, 85, 25, 310, 200, 55, 170]

export const ModelBreakdown = memo(function ModelBreakdown({
  models,
  loading,
}: ModelBreakdownProps) {
  const { sorted, maxTokens } = useMemo(() => {
    const s = [...models].sort((a, b) => b.total_tokens - a.total_tokens)
    const max = Math.max(...s.map((m) => m.total_tokens), 1)
    return { sorted: s, maxTokens: max }
  }, [models])

  if (loading) {
    return (
      <div className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius-md)] p-5">
        <div className="h-4 bg-surface-2 rounded w-36 mb-5 animate-pulse" />
        <div className="space-y-4">
          {[1, 2, 3].map((i) => (
            <div key={i} className="animate-pulse">
              <div className="flex justify-between mb-2">
                <div className="h-3 bg-surface-2 rounded w-28" />
                <div className="h-3 bg-surface-2 rounded w-16" />
              </div>
              <div className="h-3 bg-surface-2 rounded w-full" />
            </div>
          ))}
        </div>
      </div>
    )
  }

  if (sorted.length === 0) {
    return (
      <div className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius-md)] p-5">
        <h3 className="text-sm font-semibold text-text mb-4">By Model</h3>
        <div className="flex flex-col items-center justify-center py-8 text-text-dim">
          <Cube size={32} weight="duotone" className="mb-2 opacity-40" />
          <p className="text-sm">No model data yet</p>
        </div>
      </div>
    )
  }

  return (
    <div className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius-md)] p-5">
      <h3 className="text-sm font-semibold text-text mb-5">By Model</h3>

      <div className="space-y-4">
        {sorted.map((model, idx) => {
          const pct = (model.total_tokens / maxTokens) * 100
          const hue = MODEL_HUES[idx % MODEL_HUES.length]
          return (
            <div key={model.model}>
              {/* Model name + token count */}
              <div className="flex items-center justify-between mb-1.5">
                <div className="flex items-center gap-2 min-w-0">
                  <div
                    className="w-2 h-2 rounded-full shrink-0"
                    style={{ backgroundColor: `oklch(0.6 0.14 ${hue})` }}
                  />
                  <span className="text-xs font-medium text-text truncate">
                    {model.model}
                  </span>
                </div>
                <span className="text-xs text-text-secondary tabular-nums shrink-0 ml-2">
                  {formatNumber(model.total_tokens)}
                </span>
              </div>

              {/* Progress bar */}
              <div className="h-2 bg-surface-2 rounded-full overflow-hidden">
                <div
                  className="h-full rounded-full transition-all duration-500 ease-out"
                  style={{
                    width: `${pct}%`,
                    backgroundColor: `oklch(0.55 0.12 ${hue})`,
                  }}
                />
              </div>

              {/* Detail row */}
              <div className="flex gap-4 mt-1 text-[10px] text-text-dim">
                <span>{formatNumber(model.input_tokens)} in</span>
                <span>{formatNumber(model.output_tokens)} out</span>
                <span>{model.requests} req</span>
                {model.cost > 0 && <span>{formatCost(model.cost)}</span>}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
})
