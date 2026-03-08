// ============================================================================
// Operator OS — Usage Summary Cards
// Four stat cards: total tokens, input/output split, requests, cost.
// ============================================================================

import { memo } from 'react'
import {
  Coins,
  ArrowFatLineDown,
  ArrowFatLineUp,
  Lightning,
} from '@phosphor-icons/react'
import type { UsageSummary } from '../../types/api'

interface SummaryCardsProps {
  summary: UsageSummary | null
  loading: boolean
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return n.toLocaleString()
}

function formatCost(cents: number): string {
  return `$${(cents / 100).toFixed(2)}`
}

interface CardData {
  label: string
  value: string
  sub?: string
  icon: React.ReactNode
  color: string
}

function StatCard({ label, value, sub, icon, color }: CardData) {
  return (
    <div className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius-md)] p-5 flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium text-text-dim uppercase tracking-wider">
          {label}
        </span>
        <div className={`p-2 rounded-[var(--radius-sm)] ${color}`}>{icon}</div>
      </div>
      <div>
        <p className="text-2xl font-bold text-text">{value}</p>
        {sub && <p className="text-xs text-text-dim mt-1">{sub}</p>}
      </div>
    </div>
  )
}

function SkeletonCard() {
  return (
    <div className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius-md)] p-5 animate-pulse">
      <div className="flex items-center justify-between mb-3">
        <div className="h-3 bg-surface-2 rounded w-20" />
        <div className="h-8 w-8 bg-surface-2 rounded-[var(--radius-sm)]" />
      </div>
      <div className="h-8 bg-surface-2 rounded w-24 mb-1" />
      <div className="h-3 bg-surface-2 rounded w-32" />
    </div>
  )
}

export const SummaryCards = memo(function SummaryCards({
  summary,
  loading,
}: SummaryCardsProps) {
  if (loading || !summary) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <SkeletonCard key={i} />
        ))}
      </div>
    )
  }

  const cards: CardData[] = [
    {
      label: 'Total Tokens',
      value: formatNumber(summary.total_tokens),
      sub: `${formatNumber(summary.total_input_tokens)} in · ${formatNumber(summary.total_output_tokens)} out`,
      icon: <Coins size={18} weight="duotone" className="text-accent-text" />,
      color: 'bg-accent-subtle',
    },
    {
      label: 'Input Tokens',
      value: formatNumber(summary.total_input_tokens),
      sub: `${((summary.total_input_tokens / Math.max(summary.total_tokens, 1)) * 100).toFixed(0)}% of total`,
      icon: <ArrowFatLineDown size={18} weight="duotone" className="text-success" />,
      color: 'bg-success-subtle',
    },
    {
      label: 'Output Tokens',
      value: formatNumber(summary.total_output_tokens),
      sub: `${((summary.total_output_tokens / Math.max(summary.total_tokens, 1)) * 100).toFixed(0)}% of total`,
      icon: <ArrowFatLineUp size={18} weight="duotone" className="text-warning" />,
      color: 'bg-warning-subtle',
    },
    {
      label: 'Requests',
      value: formatNumber(summary.total_requests),
      sub: summary.total_cost > 0 ? `Est. cost: ${formatCost(summary.total_cost)}` : undefined,
      icon: <Lightning size={18} weight="duotone" className="text-accent-text" />,
      color: 'bg-accent-subtle',
    },
  ]

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {cards.map((card) => (
        <StatCard key={card.label} {...card} />
      ))}
    </div>
  )
})
