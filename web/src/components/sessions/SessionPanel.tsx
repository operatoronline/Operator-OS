// ============================================================================
// Operator OS — Session Panel
// Sidebar panel listing chat sessions with create, rename, pin, delete.
// Appears as a left panel on desktop, slide-over on mobile.
// ============================================================================

import { useEffect, useCallback, useState, useMemo } from 'react'
import {
  Plus,
  MagnifyingGlass,
  ChatCircleDots,
  PushPin,
  X,
  SpinnerGap,
  Archive,
  Funnel,
  CalendarBlank,
  Robot,
} from '@phosphor-icons/react'
import { useSessionStore } from '../../stores/sessionStore'
import { useChatStore } from '../../stores/chatStore'
import { useAgentStore } from '../../stores/agentStore'
import { SessionItem } from './SessionItem'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import { exportAsMarkdown, exportAsJSON } from '../../utils/exportSession'

interface SessionPanelProps {
  /** Whether the panel is visible (controls mobile slide-over) */
  open?: boolean
  /** Callback to close the panel (mobile) */
  onClose?: () => void
  /** Whether to render as a mobile overlay vs inline */
  mobile?: boolean
}

export function SessionPanel({ open = true, onClose, mobile = false }: SessionPanelProps) {
  const sessions = useSessionStore((s) => s.sessions)
  const activeSessionId = useSessionStore((s) => s.activeSessionId)
  const loading = useSessionStore((s) => s.loading)
  const renaming = useSessionStore((s) => s.renaming)
  const fetchSessions = useSessionStore((s) => s.fetchSessions)
  const createSession = useSessionStore((s) => s.createSession)
  const updateSession = useSessionStore((s) => s.updateSession)
  const deleteSession = useSessionStore((s) => s.deleteSession)
  const selectSession = useSessionStore((s) => s.selectSession)
  const setRenaming = useSessionStore((s) => s.setRenaming)

  const setActiveSession = useChatStore((s) => s.setActiveSession)
  const setActiveAgent = useChatStore((s) => s.setActiveAgent)

  const agents = useAgentStore((s) => s.agents)
  const fetchAgents = useAgentStore((s) => s.fetchAgents)

  const [search, setSearch] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)
  const [showArchived, setShowArchived] = useState(false)
  const [showFilters, setShowFilters] = useState(false)
  const [agentFilter, setAgentFilter] = useState<string>('')
  const [dateFrom, setDateFrom] = useState<string>('')
  const [dateTo, setDateTo] = useState<string>('')
  const [_exportingId, setExportingId] = useState<string | null>(null)

  const hasActiveFilters = !!agentFilter || !!dateFrom || !!dateTo

  // Fetch sessions and agents on mount
  useEffect(() => {
    fetchSessions()
    fetchAgents()
  }, [fetchSessions, fetchAgents])

  // Filter sessions by search, archived state, agent, and date range
  const filtered = useMemo(() => {
    return sessions.filter((s) => {
      // Archived filter
      if (showArchived) {
        if (!s.archived) return false
      } else {
        if (s.archived) return false
      }

      // Text search
      if (search.trim()) {
        if (!s.name.toLowerCase().includes(search.toLowerCase())) return false
      }

      // Agent filter
      if (agentFilter) {
        if (s.agent_id !== agentFilter) return false
      }

      // Date range filter
      if (dateFrom) {
        const sessionDate = new Date(s.created_at)
        const fromDate = new Date(dateFrom)
        if (sessionDate < fromDate) return false
      }
      if (dateTo) {
        const sessionDate = new Date(s.created_at)
        const toDate = new Date(dateTo + 'T23:59:59')
        if (sessionDate > toDate) return false
      }

      return true
    })
  }, [sessions, search, showArchived, agentFilter, dateFrom, dateTo])

  const pinned = filtered.filter((s) => s.pinned)
  const unpinned = filtered.filter((s) => !s.pinned)

  // Session selection — sync with chatStore
  const handleSelect = useCallback(
    (id: string) => {
      if (id === activeSessionId) return
      selectSession(id)
      setActiveSession(id)
      // Also update the agent from session metadata
      const session = sessions.find((s) => s.id === id)
      if (session?.agent_id) {
        setActiveAgent(session.agent_id)
      }
      onClose?.()
    },
    [activeSessionId, selectSession, setActiveSession, setActiveAgent, sessions, onClose],
  )

  // Create new session
  const handleCreate = useCallback(async () => {
    try {
      const session = await createSession({ name: 'New Chat' })
      setActiveSession(session.id)
      if (session.agent_id) {
        setActiveAgent(session.agent_id)
      }
      onClose?.()
    } catch {
      // Error handled in store
    }
  }, [createSession, setActiveSession, setActiveAgent, onClose])

  // Rename
  const handleRename = useCallback(
    async (id: string, name: string) => {
      try {
        await updateSession(id, { name })
      } catch {
        // Error handled in store
      }
    },
    [updateSession],
  )

  // Pin/unpin
  const handleTogglePin = useCallback(
    async (id: string, pinned: boolean) => {
      try {
        await updateSession(id, { pinned })
      } catch {
        // Error handled in store
      }
    },
    [updateSession],
  )

  // Archive/unarchive
  const handleToggleArchive = useCallback(
    async (id: string, archived: boolean) => {
      try {
        await updateSession(id, { archived })
      } catch {
        // Error handled in store
      }
    },
    [updateSession],
  )

  // Export
  const handleExportMarkdown = useCallback(
    async (id: string) => {
      const session = sessions.find((s) => s.id === id)
      if (!session) return
      setExportingId(id)
      try {
        await exportAsMarkdown(session)
      } catch {
        // silently fail
      }
      setExportingId(null)
    },
    [sessions],
  )

  const handleExportJSON = useCallback(
    async (id: string) => {
      const session = sessions.find((s) => s.id === id)
      if (!session) return
      setExportingId(id)
      try {
        await exportAsJSON(session)
      } catch {
        // silently fail
      }
      setExportingId(null)
    },
    [sessions],
  )

  // Clear all filters
  const handleClearFilters = useCallback(() => {
    setAgentFilter('')
    setDateFrom('')
    setDateTo('')
    setSearch('')
  }, [])

  // Delete
  const handleDeleteConfirm = useCallback(async () => {
    if (!deleteTarget) return
    try {
      await deleteSession(deleteTarget)
    } catch {
      // Error handled in store
    }
    setDeleteTarget(null)
  }, [deleteTarget, deleteSession])

  // Panel content
  const panelContent = (
    <div className="flex flex-col h-full">
      {/* ─── Header ─── */}
      <div className="flex items-center justify-between px-3 py-3 border-b border-[var(--border-subtle)] shrink-0">
        <h2 className="text-sm font-semibold text-[var(--text)]">
          {showArchived ? 'Archived' : 'Sessions'}
        </h2>
        <div className="flex items-center gap-0.5">
          {/* Archive toggle */}
          <button
            onClick={() => setShowArchived(!showArchived)}
            className={`p-1.5 rounded-lg transition-colors cursor-pointer
              ${showArchived
                ? 'text-[var(--accent-text)] bg-[var(--accent-subtle)]'
                : 'text-[var(--text-dim)] hover:text-[var(--text-secondary)] hover:bg-[var(--surface-2)]'
              }`}
            aria-label={showArchived ? 'Show active' : 'Show archived'}
            title={showArchived ? 'Show active' : 'Show archived'}
          >
            <Archive size={15} weight={showArchived ? 'fill' : 'regular'} />
          </button>
          {/* Filter toggle */}
          <button
            onClick={() => setShowFilters(!showFilters)}
            className={`p-1.5 rounded-lg transition-colors cursor-pointer
              ${showFilters || hasActiveFilters
                ? 'text-[var(--accent-text)] bg-[var(--accent-subtle)]'
                : 'text-[var(--text-dim)] hover:text-[var(--text-secondary)] hover:bg-[var(--surface-2)]'
              }`}
            aria-label="Filters"
            title="Filters"
          >
            <Funnel size={15} weight={hasActiveFilters ? 'fill' : 'regular'} />
          </button>
          <button
            onClick={handleCreate}
            className="p-1.5 rounded-lg text-[var(--text-dim)] hover:text-[var(--accent-text)]
              hover:bg-[var(--accent-subtle)] transition-colors cursor-pointer"
            aria-label="New session"
            title="New session"
          >
            <Plus size={16} weight="bold" />
          </button>
          {mobile && onClose && (
            <button
              onClick={onClose}
              className="p-1.5 rounded-lg text-[var(--text-dim)] hover:text-[var(--text)]
                hover:bg-[var(--surface-2)] transition-colors cursor-pointer md:hidden"
              aria-label="Close sessions"
            >
              <X size={16} weight="bold" />
            </button>
          )}
        </div>
      </div>

      {/* ─── Search ─── */}
      <div className="px-3 pt-2 pb-1 shrink-0">
        <div className="relative">
          <MagnifyingGlass
            size={14}
            className="absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--text-dim)]"
          />
          <input
            type="text"
            placeholder="Search sessions…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full bg-[var(--surface-2)] text-[var(--text)] text-xs
              pl-8 pr-3 py-2 rounded-lg border border-[var(--border-subtle)]
              outline-none focus:border-[var(--accent)]
              placeholder:text-[var(--text-dim)]
              font-[family-name:var(--font)]
              transition-colors"
          />
          {search && (
            <button
              onClick={() => setSearch('')}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--text-dim)]
                hover:text-[var(--text)] cursor-pointer"
            >
              <X size={12} />
            </button>
          )}
        </div>
      </div>

      {/* ─── Filters ─── */}
      {showFilters && (
        <div className="px-3 py-2 border-b border-[var(--border-subtle)] space-y-2 shrink-0 animate-fade-slide">
          {/* Agent filter */}
          <div>
            <label className="flex items-center gap-1 text-[10px] font-medium text-[var(--text-dim)] uppercase tracking-wider mb-1">
              <Robot size={10} />
              Agent
            </label>
            <select
              value={agentFilter}
              onChange={(e) => setAgentFilter(e.target.value)}
              className="w-full bg-[var(--surface-2)] text-[var(--text)] text-xs
                px-2 py-1.5 rounded-lg border border-[var(--border-subtle)]
                outline-none focus:border-[var(--accent)]
                font-[family-name:var(--font)] transition-colors cursor-pointer"
            >
              <option value="">All agents</option>
              {agents.filter((a) => a.status === 'active').map((a) => (
                <option key={a.id} value={a.id}>{a.name}</option>
              ))}
            </select>
          </div>

          {/* Date range */}
          <div className="flex gap-2">
            <div className="flex-1">
              <label className="flex items-center gap-1 text-[10px] font-medium text-[var(--text-dim)] uppercase tracking-wider mb-1">
                <CalendarBlank size={10} />
                From
              </label>
              <input
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
                className="w-full bg-[var(--surface-2)] text-[var(--text)] text-xs
                  px-2 py-1.5 rounded-lg border border-[var(--border-subtle)]
                  outline-none focus:border-[var(--accent)]
                  font-[family-name:var(--font)] transition-colors"
              />
            </div>
            <div className="flex-1">
              <label className="text-[10px] font-medium text-[var(--text-dim)] uppercase tracking-wider mb-1 block">
                To
              </label>
              <input
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
                className="w-full bg-[var(--surface-2)] text-[var(--text)] text-xs
                  px-2 py-1.5 rounded-lg border border-[var(--border-subtle)]
                  outline-none focus:border-[var(--accent)]
                  font-[family-name:var(--font)] transition-colors"
              />
            </div>
          </div>

          {/* Clear filters */}
          {hasActiveFilters && (
            <button
              onClick={handleClearFilters}
              className="text-[10px] text-[var(--accent-text)] hover:underline cursor-pointer"
            >
              Clear all filters
            </button>
          )}
        </div>
      )}

      {/* ─── Session list ─── */}
      <div className="flex-1 overflow-y-auto scroll-touch px-2 py-1">
        {loading && sessions.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-[var(--text-dim)]">
            <SpinnerGap size={24} className="animate-spin mb-2" />
            <span className="text-xs">Loading sessions…</span>
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <ChatCircleDots
              size={32}
              weight="thin"
              className="text-[var(--text-dim)] mb-3"
            />
            {search ? (
              <>
                <p className="text-xs text-[var(--text-dim)] mb-1">No matching sessions</p>
                <button
                  onClick={() => setSearch('')}
                  className="text-[10px] text-[var(--accent-text)] hover:underline cursor-pointer"
                >
                  Clear search
                </button>
              </>
            ) : (
              <>
                <p className="text-xs text-[var(--text-dim)] mb-2">No conversations yet</p>
                <button
                  onClick={handleCreate}
                  className="flex items-center gap-1.5 text-xs text-[var(--accent-text)]
                    hover:underline cursor-pointer"
                >
                  <Plus size={12} weight="bold" />
                  Start a new chat
                </button>
              </>
            )}
          </div>
        ) : (
          <div className="flex flex-col gap-0.5">
            {/* Pinned section */}
            {pinned.length > 0 && (
              <>
                <div className="flex items-center gap-1.5 px-3 pt-2 pb-1">
                  <PushPin size={10} weight="fill" className="text-[var(--text-dim)]" />
                  <span className="text-[10px] font-medium text-[var(--text-dim)] uppercase tracking-wider">
                    Pinned
                  </span>
                </div>
                {pinned.map((session) => (
                  <SessionItem
                    key={session.id}
                    id={session.id}
                    name={session.name}
                    messageCount={session.message_count}
                    lastMessageAt={session.last_message_at}
                    pinned={session.pinned}
                    archived={session.archived}
                    isActive={session.id === activeSessionId}
                    isRenaming={renaming === session.id}
                    onSelect={handleSelect}
                    onRename={handleRename}
                    onTogglePin={handleTogglePin}
                    onToggleArchive={handleToggleArchive}
                    onExportMarkdown={handleExportMarkdown}
                    onExportJSON={handleExportJSON}
                    onDelete={(id) => setDeleteTarget(id)}
                    onStartRename={(id) => setRenaming(id)}
                    onCancelRename={() => setRenaming(null)}
                  />
                ))}
              </>
            )}

            {/* Recent section */}
            {unpinned.length > 0 && (
              <>
                {pinned.length > 0 && (
                  <div className="flex items-center gap-1.5 px-3 pt-3 pb-1">
                    <span className="text-[10px] font-medium text-[var(--text-dim)] uppercase tracking-wider">
                      Recent
                    </span>
                  </div>
                )}
                {unpinned.map((session) => (
                  <SessionItem
                    key={session.id}
                    id={session.id}
                    name={session.name}
                    messageCount={session.message_count}
                    lastMessageAt={session.last_message_at}
                    pinned={session.pinned}
                    archived={session.archived}
                    isActive={session.id === activeSessionId}
                    isRenaming={renaming === session.id}
                    onSelect={handleSelect}
                    onRename={handleRename}
                    onTogglePin={handleTogglePin}
                    onToggleArchive={handleToggleArchive}
                    onExportMarkdown={handleExportMarkdown}
                    onExportJSON={handleExportJSON}
                    onDelete={(id) => setDeleteTarget(id)}
                    onStartRename={(id) => setRenaming(id)}
                    onCancelRename={() => setRenaming(null)}
                  />
                ))}
              </>
            )}
          </div>
        )}
      </div>

      {/* ─── Delete confirmation ─── */}
      <ConfirmDialog
        open={!!deleteTarget}
        title="Delete Session"
        message="This will permanently delete this conversation and all its messages. This action cannot be undone."
        confirmLabel="Delete"
        variant="danger"
        onConfirm={handleDeleteConfirm}
        onClose={() => setDeleteTarget(null)}
      />
    </div>
  )

  // Mobile: slide-over overlay
  if (mobile) {
    return (
      <>
        {/* Backdrop */}
        {open && (
          <div
            className="fixed inset-0 bg-black/40 backdrop-blur-sm z-40 md:hidden animate-fade-in"
            onClick={onClose}
          />
        )}

        {/* Panel */}
        <div
          className={`
            fixed top-0 left-0 bottom-0 w-72 z-50
            bg-[var(--surface)] border-r border-[var(--border)]
            transition-transform duration-300 ease-out md:hidden
            ${open ? 'translate-x-0' : '-translate-x-full'}
          `}
          style={{ paddingTop: 'var(--safe-t)', paddingBottom: 'var(--bottom-tabs-h)' }}
        >
          {panelContent}
        </div>
      </>
    )
  }

  // Desktop: inline panel
  return (
    <div className="hidden md:flex w-64 shrink-0 border-r border-[var(--border-subtle)] bg-[var(--surface)] h-full">
      {panelContent}
    </div>
  )
}
