// ============================================================================
// Operator OS — Integrations Page
// Browse, connect, and manage integrations (marketplace view).
// ============================================================================

import { useEffect, useMemo, useState, useCallback } from 'react'
import {
  MagnifyingGlass,
  Plugs,
  PlugsConnected,
  Funnel,
  ArrowCounterClockwise,
} from '@phosphor-icons/react'
import { useIntegrationStore } from '../stores/integrationStore'
import { CategoryFilter } from '../components/integrations/CategoryFilter'
import { IntegrationGrid } from '../components/integrations/IntegrationGrid'
import { ApiKeyDialog } from '../components/integrations/ApiKeyDialog'
import { OAuthFlow } from '../components/integrations/OAuthFlow'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { Button } from '../components/shared/Button'
import type { IntegrationSummary } from '../types/api'
import type { OAuthPopupResult } from '../utils/oauthPopup'

type ViewFilter = 'all' | 'connected' | 'available'

export function IntegrationsPage() {
  const store = useIntegrationStore()
  const [viewFilter, setViewFilter] = useState<ViewFilter>('all')
  const [apiKeyTarget, setApiKeyTarget] = useState<IntegrationSummary | null>(null)
  const [oauthTarget, setOauthTarget] = useState<IntegrationSummary | null>(null)
  const [oauthReconnect, setOauthReconnect] = useState(false)
  const [disconnectTarget, setDisconnectTarget] = useState<string | null>(null)

  // Fetch on mount
  useEffect(() => {
    store.fetchAll()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Handle connect action
  const handleConnect = useCallback((integration: IntegrationSummary) => {
    if (integration.auth_type === 'api_key') {
      setApiKeyTarget(integration)
    } else if (integration.auth_type === 'oauth2') {
      setOauthTarget(integration)
      setOauthReconnect(false)
    } else {
      // No-auth — direct connect
      store.connect(integration.id)
    }
  }, [store])

  const handleApiKeySubmit = useCallback((apiKey: string) => {
    if (apiKeyTarget) {
      store.connect(apiKeyTarget.id, { apiKey }).then((result) => {
        if (result) setApiKeyTarget(null)
      })
    }
  }, [apiKeyTarget, store])

  const handleDisconnect = useCallback((integrationId: string) => {
    setDisconnectTarget(integrationId)
  }, [])

  const handleConfirmDisconnect = useCallback(() => {
    if (disconnectTarget) {
      store.disconnect(disconnectTarget)
      setDisconnectTarget(null)
    }
  }, [disconnectTarget, store])

  const handleReconnect = useCallback((integrationId: string) => {
    // For OAuth integrations, open the OAuth flow in reconnect mode
    const integration = store.integrations.find((i) => i.id === integrationId)
    if (integration?.auth_type === 'oauth2') {
      setOauthTarget(integration)
      setOauthReconnect(true)
    } else {
      store.reconnect(integrationId)
    }
  }, [store])

  const handleOAuthComplete = useCallback((result: OAuthPopupResult) => {
    if (result.success) {
      // Refresh integration statuses after successful OAuth
      store.fetchStatuses()
    }
    setOauthTarget(null)
    setOauthReconnect(false)
  }, [store])

  // Category counts
  const categoryCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const i of store.integrations) {
      counts[i.category] = (counts[i.category] ?? 0) + 1
    }
    return counts
  }, [store.integrations])

  // Filtered integrations
  const displayedIntegrations = useMemo(() => {
    let filtered = store.filteredIntegrations()

    if (viewFilter === 'connected') {
      filtered = filtered.filter((i) => store.isConnected(i.id))
    } else if (viewFilter === 'available') {
      filtered = filtered.filter((i) => !store.isConnected(i.id))
    }

    return filtered
  }, [store, viewFilter])

  // Stats
  const connectedCount = useMemo(() => {
    return store.userIntegrations.filter((ui) => ui.status === 'active').length
  }, [store.userIntegrations])

  const disconnectIntegrationName = useMemo(() => {
    if (!disconnectTarget) return ''
    return store.integrations.find((i) => i.id === disconnectTarget)?.name ?? 'this integration'
  }, [disconnectTarget, store.integrations])

  const hasError = store.integrationsError || store.statusError

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* ─── Header ─── */}
      <div className="shrink-0 px-4 md:px-6 pt-4 md:pt-6 pb-4 space-y-4">
        {/* Title row */}
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-lg font-bold text-[var(--text)] flex items-center gap-2">
              <Plugs size={22} weight="fill" className="text-[var(--accent-text)]" />
              Integrations
            </h1>
            <p className="text-xs text-[var(--text-dim)] mt-0.5">
              {store.integrations.length} available · {connectedCount} connected
            </p>
          </div>
        </div>

        {/* Error banner */}
        {hasError && (
          <div className="flex items-center gap-3 px-4 py-3
            bg-[var(--error-subtle)] border border-[var(--error)]/20
            rounded-[var(--radius-md)] text-sm text-[var(--error)]"
          >
            <span className="flex-1">{store.integrationsError || store.statusError}</span>
            <Button
              variant="ghost"
              size="sm"
              icon={<ArrowCounterClockwise size={14} />}
              onClick={() => { store.clearErrors(); store.fetchAll() }}
            >
              Retry
            </Button>
          </div>
        )}

        {/* Connect error toast */}
        {store.connectError && (
          <div className="flex items-center gap-3 px-4 py-3
            bg-[var(--error-subtle)] border border-[var(--error)]/20
            rounded-[var(--radius-md)] text-sm text-[var(--error)]"
          >
            <span className="flex-1">{store.connectError}</span>
            <button
              onClick={() => store.clearErrors()}
              className="text-[var(--error)] hover:opacity-70 text-xs font-medium cursor-pointer"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Search + view filter */}
        <div className="flex items-center gap-3 flex-wrap">
          {/* Search */}
          <div className="relative flex-1 min-w-[200px] max-w-sm">
            <MagnifyingGlass
              size={15}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-dim)]"
            />
            <input
              type="text"
              placeholder="Search integrations…"
              value={store.searchQuery}
              onChange={(e) => store.setSearch(e.target.value)}
              className="w-full h-9 pl-9 pr-3
                bg-[var(--surface-2)] border border-[var(--border-subtle)]
                rounded-full text-xs text-[var(--text)]
                placeholder:text-[var(--text-dim)]
                focus:border-[var(--accent)] focus:outline-none
                transition-colors"
            />
          </div>

          {/* View filter pills */}
          <div className="flex items-center gap-1 bg-[var(--surface-2)] rounded-full p-0.5">
            {(['all', 'connected', 'available'] as ViewFilter[]).map((filter) => (
              <button
                key={filter}
                onClick={() => setViewFilter(filter)}
                className={`
                  px-3 py-1.5 rounded-full text-xs font-medium capitalize
                  transition-all duration-150 cursor-pointer
                  ${viewFilter === filter
                    ? 'bg-[var(--surface)] text-[var(--text)] shadow-sm'
                    : 'text-[var(--text-dim)] hover:text-[var(--text-secondary)]'
                  }
                `}
              >
                {filter === 'connected' && <PlugsConnected size={12} className="inline mr-1 -mt-0.5" />}
                {filter === 'available' && <Funnel size={12} className="inline mr-1 -mt-0.5" />}
                {filter}
              </button>
            ))}
          </div>
        </div>

        {/* Category filter */}
        <CategoryFilter
          categories={store.categories}
          selected={store.selectedCategory}
          onSelect={store.setCategory}
          integrationCounts={categoryCounts}
        />
      </div>

      {/* ─── Grid ─── */}
      <div className="flex-1 overflow-y-auto scroll-touch px-4 md:px-6 pb-6">
        <IntegrationGrid
          integrations={displayedIntegrations}
          statuses={store.statuses}
          userIntegrations={store.userIntegrations}
          onConnect={handleConnect}
          onDisconnect={handleDisconnect}
          onReconnect={handleReconnect}
          connectingId={store.loadingConnect}
          disconnectingId={store.loadingDisconnect}
          loading={store.loadingIntegrations}
        />
      </div>

      {/* ─── API Key Dialog ─── */}
      <ApiKeyDialog
        open={!!apiKeyTarget}
        onClose={() => setApiKeyTarget(null)}
        integration={apiKeyTarget}
        onSubmit={handleApiKeySubmit}
        loading={!!store.loadingConnect}
        error={store.connectError}
      />

      {/* ─── OAuth Flow ─── */}
      <OAuthFlow
        open={!!oauthTarget}
        onClose={() => { setOauthTarget(null); setOauthReconnect(false) }}
        integration={oauthTarget}
        onComplete={handleOAuthComplete}
        reconnect={oauthReconnect}
      />

      {/* ─── Disconnect Confirmation ─── */}
      <ConfirmDialog
        open={!!disconnectTarget}
        onClose={() => setDisconnectTarget(null)}
        onConfirm={handleConfirmDisconnect}
        title="Disconnect integration"
        message={`Are you sure you want to disconnect ${disconnectIntegrationName}? Any agents using this integration will lose access to its tools.`}
        confirmLabel="Disconnect"
        variant="danger"
      />
    </div>
  )
}
