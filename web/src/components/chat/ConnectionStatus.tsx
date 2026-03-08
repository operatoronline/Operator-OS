// ============================================================================
// Operator OS — Connection Status Indicator
// Shows WebSocket connection state: connected (green), reconnecting (amber),
// disconnected (red). Mirrors the legacy nav-status dot pattern.
// ============================================================================

import { WifiHigh, WifiSlash, ArrowsClockwise } from '@phosphor-icons/react'
import { useChatStore } from '../../stores/chatStore'
import type { ConnectionState } from '../../types/ws'

const STATUS_CONFIG: Record<ConnectionState, {
  icon: typeof WifiHigh
  label: string
  dotClass: string
  textClass: string
}> = {
  connected: {
    icon: WifiHigh,
    label: 'Connected',
    dotClass: 'bg-[var(--success)]',
    textClass: 'text-[var(--success)]',
  },
  connecting: {
    icon: ArrowsClockwise,
    label: 'Connecting…',
    dotClass: 'bg-[var(--warning)]',
    textClass: 'text-[var(--warning)]',
  },
  reconnecting: {
    icon: ArrowsClockwise,
    label: 'Reconnecting…',
    dotClass: 'bg-[var(--warning)] animate-pulse',
    textClass: 'text-[var(--warning)]',
  },
  disconnected: {
    icon: WifiSlash,
    label: 'Disconnected',
    dotClass: 'bg-[var(--error)]',
    textClass: 'text-[var(--error)]',
  },
}

interface ConnectionStatusProps {
  /** Show text label alongside the dot/icon */
  showLabel?: boolean
  /** Compact mode — just the dot */
  compact?: boolean
}

export function ConnectionStatus({ showLabel = false, compact = false }: ConnectionStatusProps) {
  const connectionState = useChatStore((s) => s.connectionState)
  const config = STATUS_CONFIG[connectionState]
  const Icon = config.icon

  if (compact) {
    return (
      <span
        className={`inline-block w-2 h-2 rounded-full ${config.dotClass}`}
        title={config.label}
        role="status"
        aria-label={config.label}
      />
    )
  }

  return (
    <span
      className={`inline-flex items-center gap-1.5 text-xs ${config.textClass}`}
      role="status"
      aria-label={config.label}
    >
      <Icon
        size={14}
        weight="bold"
        className={connectionState === 'connecting' || connectionState === 'reconnecting' ? 'animate-spin' : ''}
      />
      {showLabel && <span>{config.label}</span>}
      {!showLabel && (
        <span
          className={`w-1.5 h-1.5 rounded-full ${config.dotClass}`}
        />
      )}
    </span>
  )
}
