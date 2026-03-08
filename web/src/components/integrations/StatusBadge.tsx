// ============================================================================
// Operator OS — Integration Status Badge
// Visual indicator for integration connection state + token health.
// ============================================================================

import { memo } from 'react'
import {
  CheckCircle,
  Warning,
  XCircle,
  ArrowsClockwise,
  Clock,
} from '@phosphor-icons/react'

type ConnectionStatus = 'active' | 'pending' | 'failed' | 'revoked' | 'disabled' | 'disconnected'

interface StatusBadgeProps {
  status: ConnectionStatus
  tokenExpired?: boolean
  needsRefresh?: boolean
  className?: string
}

const statusConfig: Record<
  ConnectionStatus,
  { icon: typeof CheckCircle; label: string; color: string; bg: string }
> = {
  active: {
    icon: CheckCircle,
    label: 'Connected',
    color: 'text-[var(--success)]',
    bg: 'bg-[var(--success-subtle)]',
  },
  pending: {
    icon: Clock,
    label: 'Pending',
    color: 'text-[var(--warning)]',
    bg: 'bg-[var(--warning-subtle)]',
  },
  failed: {
    icon: XCircle,
    label: 'Failed',
    color: 'text-[var(--error)]',
    bg: 'bg-[var(--error-subtle)]',
  },
  revoked: {
    icon: XCircle,
    label: 'Revoked',
    color: 'text-[var(--error)]',
    bg: 'bg-[var(--error-subtle)]',
  },
  disabled: {
    icon: XCircle,
    label: 'Disabled',
    color: 'text-[var(--text-dim)]',
    bg: 'bg-[var(--surface-2)]',
  },
  disconnected: {
    icon: XCircle,
    label: 'Not connected',
    color: 'text-[var(--text-dim)]',
    bg: 'bg-[var(--surface-2)]',
  },
}

export const StatusBadge = memo(function StatusBadge({
  status,
  tokenExpired,
  needsRefresh,
  className = '',
}: StatusBadgeProps) {
  // Override label for token issues on active connections
  const effectiveStatus = tokenExpired || needsRefresh
    ? status === 'active' ? 'active' : status
    : status

  const config = statusConfig[effectiveStatus] || statusConfig.disconnected
  const Icon = tokenExpired ? Warning : needsRefresh ? ArrowsClockwise : config.icon

  const label = tokenExpired
    ? 'Token expired'
    : needsRefresh
      ? 'Needs refresh'
      : config.label

  const color = tokenExpired || needsRefresh
    ? 'text-[var(--warning)]'
    : config.color

  const bg = tokenExpired || needsRefresh
    ? 'bg-[var(--warning-subtle)]'
    : config.bg

  return (
    <span
      className={`
        inline-flex items-center gap-1.5
        px-2 py-0.5 rounded-full
        text-[11px] font-semibold leading-none tracking-wide
        ${bg} ${color}
        ${className}
      `}
    >
      <Icon size={12} weight="fill" />
      {label}
    </span>
  )
})
