// ============================================================================
// Operator OS — Audit Log Viewer
// Filterable event log with expandable detail rows, categories, and CSV export.
// ============================================================================

import { memo, useCallback, useEffect, useMemo, useState } from 'react'
import {
  CaretDown,
  CaretRight,
  CaretLeft,
  CheckCircle,
  XCircle,
  Export,
  Funnel,
  ArrowCounterClockwise,
  Clock,
  User,
  Shield,
  CreditCard,
  Gear,
  Plug,
  Database,
  Globe,
} from '@phosphor-icons/react'
import { useAuditStore, type AuditFilters } from '../../stores/auditStore'
import { Button } from '../shared/Button'
import type { AuditEvent } from '../../types/api'

// ---------------------------------------------------------------------------
// Action category system
// ---------------------------------------------------------------------------

interface ActionCategory {
  label: string
  icon: React.ReactNode
  color: string
  actions: string[]
}

const ACTION_CATEGORIES: ActionCategory[] = [
  {
    label: 'Auth',
    icon: <Shield size={13} />,
    color: 'var(--accent-text)',
    actions: ['login', 'logout', 'register', 'verify', 'refresh', 'password_change'],
  },
  {
    label: 'Agent',
    icon: <Gear size={13} />,
    color: 'var(--success)',
    actions: ['agent_create', 'agent_update', 'agent_delete', 'agent_default'],
  },
  {
    label: 'Billing',
    icon: <CreditCard size={13} />,
    color: 'var(--warning)',
    actions: ['checkout', 'plan_change', 'subscription_cancel', 'payment'],
  },
  {
    label: 'Integration',
    icon: <Plug size={13} />,
    color: 'oklch(0.72 0.15 200)',
    actions: ['integration_connect', 'integration_disconnect', 'oauth_authorize'],
  },
  {
    label: 'Admin',
    icon: <User size={13} />,
    color: 'var(--error)',
    actions: ['user_suspend', 'user_activate', 'user_delete', 'role_change'],
  },
  {
    label: 'Data',
    icon: <Database size={13} />,
    color: 'oklch(0.72 0.15 280)',
    actions: ['gdpr_export', 'gdpr_erase', 'session_create', 'session_delete'],
  },
]

function getCategoryForAction(action: string): ActionCategory | null {
  return ACTION_CATEGORIES.find((c) => c.actions.includes(action)) ?? null
}

// ---------------------------------------------------------------------------
// CSV export
// ---------------------------------------------------------------------------

function exportAuditCsv(events: AuditEvent[]) {
  const headers = ['ID', 'Timestamp', 'Actor', 'Action', 'Resource', 'Resource ID', 'Status', 'IP Address', 'User Agent', 'Detail']
  const rows = events.map((e) => [
    e.id,
    e.created_at,
    e.actor_id,
    e.action,
    e.resource,
    e.resource_id,
    e.status,
    e.ip_address,
    e.user_agent,
    JSON.stringify(e.detail),
  ])

  const csv = [headers, ...rows]
    .map((row) => row.map((cell) => `"${String(cell).replace(/"/g, '""')}"`).join(','))
    .join('\n')

  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `audit-log-${new Date().toISOString().slice(0, 10)}.csv`
  a.click()
  URL.revokeObjectURL(url)
}

// ---------------------------------------------------------------------------
// Relative time helper
// ---------------------------------------------------------------------------

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const mins = Math.floor(diff / 60_000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  return new Date(iso).toLocaleDateString()
}

function formatTimestamp(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

// ---------------------------------------------------------------------------
// Event Row
// ---------------------------------------------------------------------------

interface EventRowProps {
  event: AuditEvent
  expanded: boolean
  onToggle: () => void
}

const EventRow = memo(function EventRow({ event, expanded, onToggle }: EventRowProps) {
  const category = getCategoryForAction(event.action)
  const isSuccess = event.status === 'success'

  return (
    <>
      {/* Main row */}
      <button
        onClick={onToggle}
        className="w-full flex items-center gap-3 px-4 py-3
          hover:bg-[var(--surface-2)]/50 transition-colors
          border-b border-[var(--border-subtle)]/50
          text-left cursor-pointer group"
      >
        {/* Expand chevron */}
        <span className="shrink-0 text-[var(--text-dim)] transition-transform">
          {expanded ? <CaretDown size={14} /> : <CaretRight size={14} />}
        </span>

        {/* Status dot */}
        <span
          className="shrink-0 w-2 h-2 rounded-full"
          style={{ background: isSuccess ? 'var(--success)' : 'var(--error)' }}
        />

        {/* Category badge */}
        {category ? (
          <span
            className="shrink-0 inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold uppercase tracking-wider"
            style={{
              color: category.color,
              background: `color-mix(in oklch, ${category.color} 12%, transparent)`,
            }}
          >
            {category.icon}
            {category.label}
          </span>
        ) : (
          <span className="shrink-0 inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold uppercase tracking-wider text-[var(--text-dim)] bg-[var(--surface-2)]">
            <Globe size={13} />
            Other
          </span>
        )}

        {/* Action */}
        <span className="text-xs font-medium text-[var(--text)] font-mono truncate min-w-0">
          {event.action}
        </span>

        {/* Resource */}
        <span className="hidden md:block text-xs text-[var(--text-dim)] truncate min-w-0 flex-1">
          {event.resource}
          {event.resource_id && (
            <span className="text-[var(--text-dim)]/60 ml-1">#{event.resource_id.slice(0, 8)}</span>
          )}
        </span>

        {/* Actor (truncated ID) */}
        <span className="hidden lg:block text-[11px] text-[var(--text-dim)] font-mono shrink-0">
          {event.actor_id.slice(0, 8)}…
        </span>

        {/* Status icon */}
        <span className="shrink-0">
          {isSuccess ? (
            <CheckCircle size={16} weight="fill" className="text-[var(--success)]" />
          ) : (
            <XCircle size={16} weight="fill" className="text-[var(--error)]" />
          )}
        </span>

        {/* Time */}
        <span className="shrink-0 text-[11px] text-[var(--text-dim)] tabular-nums whitespace-nowrap">
          {relativeTime(event.created_at)}
        </span>
      </button>

      {/* Expanded detail panel */}
      {expanded && (
        <div className="px-4 py-3 bg-[var(--surface-2)]/30 border-b border-[var(--border-subtle)]/50 animate-fadeIn">
          <div className="ml-8 grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-2">
            <DetailField label="Event ID" value={event.id} mono />
            <DetailField label="Timestamp" value={formatTimestamp(event.created_at)} />
            <DetailField label="Actor ID" value={event.actor_id} mono />
            <DetailField label="User ID" value={event.user_id} mono />
            <DetailField label="Action" value={event.action} mono />
            <DetailField label="Resource" value={`${event.resource}${event.resource_id ? ` #${event.resource_id}` : ''}`} />
            <DetailField label="Status" value={event.status} />
            <DetailField label="IP Address" value={event.ip_address || '—'} mono />
            <div className="md:col-span-2">
              <DetailField label="User Agent" value={event.user_agent || '—'} />
            </div>
            {event.detail && Object.keys(event.detail).length > 0 && (
              <div className="md:col-span-2">
                <p className="text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold mb-1">
                  Detail
                </p>
                <pre className="text-xs font-mono text-[var(--text-secondary)] bg-[var(--surface-2)] rounded-lg p-3 overflow-x-auto max-h-40">
                  {JSON.stringify(event.detail, null, 2)}
                </pre>
              </div>
            )}
          </div>
        </div>
      )}
    </>
  )
})

function DetailField({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <p className="text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold mb-0.5">
        {label}
      </p>
      <p className={`text-xs text-[var(--text-secondary)] break-all ${mono ? 'font-mono' : ''}`}>
        {value}
      </p>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Loading skeleton
// ---------------------------------------------------------------------------

function AuditSkeleton() {
  return (
    <div className="animate-pulse space-y-0">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 px-4 py-3 border-b border-[var(--border-subtle)]/50">
          <div className="w-3.5 h-3.5 rounded bg-[var(--surface-3)]" />
          <div className="w-2 h-2 rounded-full bg-[var(--surface-3)]" />
          <div className="w-14 h-4 rounded-full bg-[var(--surface-3)]" />
          <div className="w-24 h-3.5 rounded bg-[var(--surface-3)]" />
          <div className="flex-1 h-3.5 rounded bg-[var(--surface-3)] hidden md:block" />
          <div className="w-12 h-3.5 rounded bg-[var(--surface-3)]" />
        </div>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Filter bar
// ---------------------------------------------------------------------------

const RESOURCE_OPTIONS = [
  { value: '', label: 'All resources' },
  { value: 'user', label: 'User' },
  { value: 'agent', label: 'Agent' },
  { value: 'session', label: 'Session' },
  { value: 'subscription', label: 'Subscription' },
  { value: 'integration', label: 'Integration' },
]

interface FilterBarProps {
  filters: AuditFilters
  onFilterChange: (key: keyof Omit<AuditFilters, 'page' | 'perPage'>, value: string) => void
  onReset: () => void
  hasActiveFilters: boolean
}

function FilterBar({ filters, onFilterChange, onReset, hasActiveFilters }: FilterBarProps) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="space-y-3">
      {/* Toggle + category pills */}
      <div className="flex items-center gap-2 flex-wrap">
        <button
          onClick={() => setExpanded(!expanded)}
          className={`
            inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium
            transition-all duration-150 cursor-pointer
            ${expanded
              ? 'bg-[var(--accent-text)]/10 text-[var(--accent-text)]'
              : 'bg-[var(--surface-2)] text-[var(--text-dim)] hover:text-[var(--text-secondary)]'
            }
          `}
        >
          <Funnel size={13} />
          Filters
          {hasActiveFilters && (
            <span className="w-1.5 h-1.5 rounded-full bg-[var(--accent-text)]" />
          )}
        </button>

        {/* Quick category pills */}
        {ACTION_CATEGORIES.map((cat) => (
          <button
            key={cat.label}
            onClick={() => {
              // Toggle: if current filter matches any of this category's actions, clear it
              const currentAction = filters.action
              const isActive = cat.actions.includes(currentAction)
              onFilterChange('action', isActive ? '' : cat.actions[0])
            }}
            className={`
              inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[11px] font-medium
              transition-all duration-150 cursor-pointer
              ${cat.actions.includes(filters.action)
                ? 'shadow-sm'
                : 'opacity-60 hover:opacity-100'
              }
            `}
            style={{
              color: cat.color,
              background: cat.actions.includes(filters.action)
                ? `color-mix(in oklch, ${cat.color} 18%, transparent)`
                : `color-mix(in oklch, ${cat.color} 8%, transparent)`,
            }}
          >
            {cat.icon}
            {cat.label}
          </button>
        ))}

        {hasActiveFilters && (
          <button
            onClick={onReset}
            className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-[11px]
              text-[var(--text-dim)] hover:text-[var(--text-secondary)]
              bg-[var(--surface-2)] hover:bg-[var(--surface-3)]
              transition-colors cursor-pointer"
          >
            <ArrowCounterClockwise size={12} />
            Clear
          </button>
        )}
      </div>

      {/* Expanded filter fields */}
      {expanded && (
        <div className="flex items-end gap-3 flex-wrap animate-fadeSlideDown">
          {/* Action (text input for specific action) */}
          <div className="space-y-1">
            <label className="text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold">
              Action
            </label>
            <input
              type="text"
              placeholder="e.g. login"
              value={filters.action}
              onChange={(e) => onFilterChange('action', e.target.value)}
              className="h-8 w-32 px-2.5
                bg-[var(--surface-2)] border border-[var(--border-subtle)]
                rounded-lg text-xs text-[var(--text)]
                placeholder:text-[var(--text-dim)]
                focus:border-[var(--accent)] focus:outline-none
                transition-colors"
            />
          </div>

          {/* User ID */}
          <div className="space-y-1">
            <label className="text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold">
              User ID
            </label>
            <input
              type="text"
              placeholder="User or actor ID"
              value={filters.userId}
              onChange={(e) => onFilterChange('userId', e.target.value)}
              className="h-8 w-40 px-2.5
                bg-[var(--surface-2)] border border-[var(--border-subtle)]
                rounded-lg text-xs text-[var(--text)] font-mono
                placeholder:text-[var(--text-dim)] placeholder:font-sans
                focus:border-[var(--accent)] focus:outline-none
                transition-colors"
            />
          </div>

          {/* Resource */}
          <div className="space-y-1">
            <label className="text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold">
              Resource
            </label>
            <select
              value={filters.resource}
              onChange={(e) => onFilterChange('resource', e.target.value)}
              className="h-8 w-36 px-2.5
                bg-[var(--surface-2)] border border-[var(--border-subtle)]
                rounded-lg text-xs text-[var(--text)]
                focus:border-[var(--accent)] focus:outline-none
                transition-colors cursor-pointer"
            >
              {RESOURCE_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          {/* Date range */}
          <div className="space-y-1">
            <label className="text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold">
              From
            </label>
            <input
              type="date"
              value={filters.start}
              onChange={(e) => onFilterChange('start', e.target.value)}
              className="h-8 w-36 px-2.5
                bg-[var(--surface-2)] border border-[var(--border-subtle)]
                rounded-lg text-xs text-[var(--text)]
                focus:border-[var(--accent)] focus:outline-none
                transition-colors"
            />
          </div>
          <div className="space-y-1">
            <label className="text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold">
              To
            </label>
            <input
              type="date"
              value={filters.end}
              onChange={(e) => onFilterChange('end', e.target.value)}
              className="h-8 w-36 px-2.5
                bg-[var(--surface-2)] border border-[var(--border-subtle)]
                rounded-lg text-xs text-[var(--text)]
                focus:border-[var(--accent)] focus:outline-none
                transition-colors"
            />
          </div>
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// AuditLog Component
// ---------------------------------------------------------------------------

export const AuditLog = memo(function AuditLog() {
  const store = useAuditStore()
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())

  // Fetch on mount
  useEffect(() => {
    store.fetchAll()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const toggleExpand = useCallback((id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  const hasActiveFilters = useMemo(() => {
    const f = store.filters
    return !!(f.action || f.userId || f.resource || f.start || f.end)
  }, [store.filters])

  const handleExportCsv = useCallback(() => {
    if (store.events.length > 0) {
      exportAuditCsv(store.events)
    }
  }, [store.events])

  // Pagination
  const hasNextPage = store.events.length === store.filters.perPage
  const hasPrevPage = store.filters.page > 1
  const totalPages = store.totalCount !== null
    ? Math.ceil(store.totalCount / store.filters.perPage)
    : null

  return (
    <div className="space-y-4">
      {/* Filter bar + export */}
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <FilterBar
            filters={store.filters}
            onFilterChange={store.setFilter}
            onReset={store.resetFilters}
            hasActiveFilters={hasActiveFilters}
          />
        </div>
        <Button
          variant="secondary"
          size="sm"
          icon={<Export size={14} />}
          onClick={handleExportCsv}
          disabled={store.events.length === 0}
        >
          <span className="hidden sm:inline">Export CSV</span>
        </Button>
      </div>

      {/* Error banner */}
      {store.error && (
        <div
          className="flex items-center gap-3 px-4 py-3
            bg-[var(--error-subtle)] border border-[var(--error)]/20
            rounded-[var(--radius-md)] text-sm text-[var(--error)]"
        >
          <span className="flex-1">{store.error}</span>
          <Button
            variant="ghost"
            size="sm"
            icon={<ArrowCounterClockwise size={14} />}
            onClick={() => {
              store.clearError()
              store.fetchAll()
            }}
          >
            Retry
          </Button>
        </div>
      )}

      {/* Event list */}
      <div className="border border-[var(--border-subtle)] rounded-[var(--radius-md)] overflow-hidden">
        {/* Header */}
        <div className="flex items-center gap-3 px-4 py-2 bg-[var(--surface-2)]/50 border-b border-[var(--border-subtle)]">
          <span className="w-3.5" /> {/* chevron spacer */}
          <span className="w-2" /> {/* status dot spacer */}
          <span className="text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold">
            Category
          </span>
          <span className="text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold flex-1 min-w-0">
            Action
          </span>
          <span className="hidden md:block text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold flex-1">
            Resource
          </span>
          <span className="hidden lg:block text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold w-20">
            Actor
          </span>
          <span className="text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold w-4">
            {/* Status */}
          </span>
          <span className="text-[10px] uppercase tracking-wider text-[var(--text-dim)] font-semibold w-16 text-right">
            <Clock size={11} className="inline" />
          </span>
        </div>

        {/* Body */}
        {store.loading ? (
          <AuditSkeleton />
        ) : store.events.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-[var(--text-dim)]">
            <Clock size={32} weight="thin" className="mb-3 opacity-40" />
            <p className="text-sm font-medium text-[var(--text-secondary)] mb-1">
              {hasActiveFilters ? 'No events match your filters' : 'No audit events yet'}
            </p>
            <p className="text-xs">
              {hasActiveFilters
                ? 'Try adjusting your filters or date range.'
                : 'Events will appear here as platform actions are performed.'}
            </p>
          </div>
        ) : (
          <div>
            {store.events.map((event) => (
              <EventRow
                key={event.id}
                event={event}
                expanded={expandedIds.has(event.id)}
                onToggle={() => toggleExpand(event.id)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Pagination */}
      {!store.loading && store.events.length > 0 && (
        <div className="flex items-center justify-between px-1">
          <p className="text-xs text-[var(--text-dim)] tabular-nums">
            Page {store.filters.page}
            {totalPages !== null && <span> of {totalPages}</span>}
            {store.totalCount !== null && (
              <span className="ml-2">· {store.totalCount.toLocaleString()} total event{store.totalCount !== 1 ? 's' : ''}</span>
            )}
          </p>
          <div className="flex items-center gap-1">
            <button
              onClick={() => store.setPage(store.filters.page - 1)}
              disabled={!hasPrevPage}
              className="p-1.5 rounded-lg text-[var(--text-dim)]
                hover:text-[var(--text)] hover:bg-[var(--surface-2)]
                disabled:opacity-30 disabled:cursor-not-allowed
                transition-colors cursor-pointer"
              aria-label="Previous page"
            >
              <CaretLeft size={16} />
            </button>
            <button
              onClick={() => store.setPage(store.filters.page + 1)}
              disabled={!hasNextPage}
              className="p-1.5 rounded-lg text-[var(--text-dim)]
                hover:text-[var(--text)] hover:bg-[var(--surface-2)]
                disabled:opacity-30 disabled:cursor-not-allowed
                transition-colors cursor-pointer"
              aria-label="Next page"
            >
              <CaretRight size={16} />
            </button>
          </div>
        </div>
      )}
    </div>
  )
})
