// ============================================================================
// Operator OS — Admin Stats Cards
// Platform-level stat cards: total users, active, pending, suspended.
// ============================================================================

import { memo } from 'react'
import { Users, UserCheck, Clock, UserMinus } from '@phosphor-icons/react'
import type { PlatformStats } from '../../types/api'

interface StatsCardsProps {
  stats: PlatformStats | null
  loading: boolean
}

interface StatItem {
  label: string
  value: number
  icon: React.ReactNode
  color: string
  bgColor: string
}

export const StatsCards = memo(function StatsCards({ stats, loading }: StatsCardsProps) {
  const items: StatItem[] = [
    {
      label: 'Total Users',
      value: stats?.total_users ?? 0,
      icon: <Users size={20} weight="fill" />,
      color: 'var(--accent-text)',
      bgColor: 'var(--accent-subtle)',
    },
    {
      label: 'Active',
      value: stats?.active_users ?? 0,
      icon: <UserCheck size={20} weight="fill" />,
      color: 'var(--success)',
      bgColor: 'oklch(0.85 0.08 145)',
    },
    {
      label: 'Pending',
      value: stats?.pending_users ?? 0,
      icon: <Clock size={20} weight="fill" />,
      color: 'var(--warning)',
      bgColor: 'oklch(0.90 0.08 85)',
    },
    {
      label: 'Suspended',
      value: stats?.suspended_users ?? 0,
      icon: <UserMinus size={20} weight="fill" />,
      color: 'var(--error)',
      bgColor: 'var(--error-subtle)',
    },
  ]

  if (loading) {
    return (
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="h-[88px] rounded-[var(--radius-md)] bg-[var(--surface-2)] animate-pulse"
          />
        ))}
      </div>
    )
  }

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
      {items.map((item) => (
        <div
          key={item.label}
          className="px-4 py-4 rounded-[var(--radius-md)]
            bg-[var(--surface-2)] border border-[var(--border-subtle)]
            flex items-center gap-3"
        >
          <div
            className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0"
            style={{ backgroundColor: item.bgColor, color: item.color }}
          >
            {item.icon}
          </div>
          <div className="min-w-0">
            <div className="text-xl font-bold text-[var(--text)] tabular-nums">
              {item.value.toLocaleString()}
            </div>
            <div className="text-xs text-[var(--text-dim)] truncate">{item.label}</div>
          </div>
        </div>
      ))}
    </div>
  )
})
