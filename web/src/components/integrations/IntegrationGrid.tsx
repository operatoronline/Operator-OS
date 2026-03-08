// ============================================================================
// Operator OS — Integration Grid
// Category-filtered, searchable grid of integration cards.
// ============================================================================

import { memo } from 'react'
import { Plugs } from '@phosphor-icons/react'
import { IntegrationCard } from './IntegrationCard'
import { EmptyState } from '../shared/EmptyState'
import type { IntegrationSummary, IntegrationStatus, UserIntegration } from '../../types/api'

interface IntegrationGridProps {
  integrations: IntegrationSummary[]
  statuses: IntegrationStatus[]
  userIntegrations: UserIntegration[]
  onConnect: (integration: IntegrationSummary) => void
  onDisconnect: (integrationId: string) => void
  onReconnect: (integrationId: string) => void
  connectingId: string | null
  disconnectingId: string | null
  loading?: boolean
}

export const IntegrationGrid = memo(function IntegrationGrid({
  integrations,
  statuses,
  userIntegrations,
  onConnect,
  onDisconnect,
  onReconnect,
  connectingId,
  disconnectingId,
  loading,
}: IntegrationGridProps) {
  if (loading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {Array.from({ length: 6 }).map((_, i) => (
          <IntegrationSkeleton key={i} />
        ))}
      </div>
    )
  }

  if (integrations.length === 0) {
    return (
      <EmptyState
        icon={Plugs}
        title="No integrations found"
        description="Try adjusting your search or filter to find integrations."
      />
    )
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      {integrations.map((integration) => (
        <IntegrationCard
          key={integration.id}
          integration={integration}
          status={statuses.find((s) => s.integration_id === integration.id)}
          userIntegration={userIntegrations.find((ui) => ui.integration_id === integration.id)}
          onConnect={onConnect}
          onDisconnect={onDisconnect}
          onReconnect={onReconnect}
          connectingId={connectingId}
          disconnectingId={disconnectingId}
        />
      ))}
    </div>
  )
})

// ---------------------------------------------------------------------------
// Loading skeleton
// ---------------------------------------------------------------------------

function IntegrationSkeleton() {
  return (
    <div className="flex flex-col gap-3 p-4 bg-[var(--surface)] border border-[var(--border-subtle)] rounded-[var(--radius)] animate-pulse">
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 rounded-xl bg-[var(--surface-2)]" />
        <div className="flex-1">
          <div className="h-3.5 w-24 bg-[var(--surface-2)] rounded mb-1.5" />
          <div className="h-2.5 w-16 bg-[var(--surface-2)] rounded" />
        </div>
      </div>
      <div className="h-3 w-full bg-[var(--surface-2)] rounded" />
      <div className="h-3 w-3/4 bg-[var(--surface-2)] rounded" />
      <div className="h-8 w-full bg-[var(--surface-2)] rounded-lg mt-auto" />
    </div>
  )
}
