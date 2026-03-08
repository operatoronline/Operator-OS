// ============================================================================
// Operator OS — Daily Usage Chart
// Pure CSS bar chart showing daily token usage over time.
// No chart library — lightweight, OKLCH-themed, responsive.
// ============================================================================

import { memo, useMemo } from 'react'
import type { DailyUsage } from '../../types/api'

interface DailyChartProps {
  daily: DailyUsage[]
  loading: boolean
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

function formatTokens(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toString()
}

export const DailyChart = memo(function DailyChart({
  daily,
  loading,
}: DailyChartProps) {
  const { maxTokens, displayDays } = useMemo(() => {
    // Show last 14 days max
    const slice = daily.slice(-14)
    const max = Math.max(...slice.map((d) => d.total_tokens), 1)
    return { maxTokens: max, displayDays: slice }
  }, [daily])

  if (loading) {
    return (
      <div className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius-md)] p-5">
        <div className="h-4 bg-surface-2 rounded w-32 mb-6 animate-pulse" />
        <div className="flex items-end gap-2 h-48">
          {Array.from({ length: 10 }).map((_, i) => (
            <div
              key={i}
              className="flex-1 bg-surface-2 rounded-t animate-pulse"
              style={{ height: `${20 + Math.random() * 60}%` }}
            />
          ))}
        </div>
      </div>
    )
  }

  if (displayDays.length === 0) {
    return (
      <div className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius-md)] p-5">
        <h3 className="text-sm font-semibold text-text mb-4">Daily Usage</h3>
        <div className="flex items-center justify-center h-48 text-sm text-text-dim">
          No usage data for this period
        </div>
      </div>
    )
  }

  return (
    <div className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius-md)] p-5">
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-sm font-semibold text-text">Daily Usage</h3>
        <span className="text-xs text-text-dim">
          Last {displayDays.length} day{displayDays.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Y-axis scale + bars */}
      <div className="flex gap-3">
        {/* Y-axis labels */}
        <div className="flex flex-col justify-between h-48 text-right shrink-0 w-10">
          <span className="text-[10px] text-text-dim leading-none">
            {formatTokens(maxTokens)}
          </span>
          <span className="text-[10px] text-text-dim leading-none">
            {formatTokens(Math.round(maxTokens / 2))}
          </span>
          <span className="text-[10px] text-text-dim leading-none">0</span>
        </div>

        {/* Bar container */}
        <div className="flex-1 flex items-end gap-[3px] h-48 relative">
          {/* Grid lines */}
          <div className="absolute inset-0 flex flex-col justify-between pointer-events-none">
            <div className="border-b border-border-subtle/30" />
            <div className="border-b border-border-subtle/30" />
            <div className="border-b border-border-subtle/30" />
          </div>

          {/* Bars */}
          {displayDays.map((day) => {
            const inputPct = (day.input_tokens / maxTokens) * 100
            const outputPct = (day.output_tokens / maxTokens) * 100
            return (
              <div
                key={day.date}
                className="flex-1 flex flex-col justify-end relative group min-w-0"
              >
                {/* Tooltip */}
                <div className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 hidden group-hover:block z-10 pointer-events-none">
                  <div className="glass rounded-[var(--radius-xs)] px-3 py-2 text-xs whitespace-nowrap shadow-lg">
                    <p className="font-medium text-text">{formatDate(day.date)}</p>
                    <p className="text-text-secondary mt-1">
                      {formatTokens(day.total_tokens)} tokens
                    </p>
                    <div className="flex gap-3 mt-1 text-text-dim">
                      <span>{formatTokens(day.input_tokens)} in</span>
                      <span>{formatTokens(day.output_tokens)} out</span>
                    </div>
                    {day.requests > 0 && (
                      <p className="text-text-dim mt-0.5">{day.requests} requests</p>
                    )}
                  </div>
                </div>

                {/* Stacked bar: input (bottom) + output (top) */}
                <div
                  className="w-full rounded-t transition-all duration-300 ease-out"
                  style={{ height: `${inputPct + outputPct}%`, minHeight: day.total_tokens > 0 ? '2px' : '0' }}
                >
                  {/* Output (top portion) */}
                  <div
                    className="w-full rounded-t bg-warning/60 group-hover:bg-warning/80 transition-colors"
                    style={{
                      height: outputPct > 0 ? `${(outputPct / (inputPct + outputPct)) * 100}%` : '0',
                      minHeight: day.output_tokens > 0 ? '1px' : '0',
                    }}
                  />
                  {/* Input (bottom portion) */}
                  <div
                    className="w-full bg-accent/60 group-hover:bg-accent/80 transition-colors"
                    style={{
                      height: inputPct > 0 ? `${(inputPct / (inputPct + outputPct)) * 100}%` : '0',
                      minHeight: day.input_tokens > 0 ? '1px' : '0',
                    }}
                  />
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* X-axis date labels — show every other for compactness */}
      <div className="flex gap-[3px] mt-2 ml-[52px]">
        {displayDays.map((day, i) => (
          <div key={day.date} className="flex-1 min-w-0">
            {i % Math.max(1, Math.floor(displayDays.length / 7)) === 0 && (
              <span className="text-[10px] text-text-dim block truncate text-center">
                {formatDate(day.date)}
              </span>
            )}
          </div>
        ))}
      </div>

      {/* Legend */}
      <div className="flex gap-4 mt-3 ml-[52px]">
        <div className="flex items-center gap-1.5">
          <div className="w-2.5 h-2.5 rounded-sm bg-accent/60" />
          <span className="text-[10px] text-text-dim">Input</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="w-2.5 h-2.5 rounded-sm bg-warning/60" />
          <span className="text-[10px] text-text-dim">Output</span>
        </div>
      </div>
    </div>
  )
})
