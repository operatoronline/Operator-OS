// ============================================================================
// Operator OS — Agents Page
// Full agent management: list, create, edit, delete, set default.
// ============================================================================

import { useEffect, useState, useCallback } from 'react'
import { Plus } from '@phosphor-icons/react'
import { useAgentStore } from '../stores/agentStore'
import { AgentList } from '../components/agents/AgentList'
import { AgentEditor } from '../components/agents/AgentEditor'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { Button } from '../components/shared/Button'
import { Badge } from '../components/shared/Badge'
import type { Agent, CreateAgentRequest, UpdateAgentRequest } from '../types/api'

type FilterStatus = 'all' | 'active' | 'archived'

export function AgentsPage() {
  const agents = useAgentStore((s) => s.agents)
  const loading = useAgentStore((s) => s.loading)
  const error = useAgentStore((s) => s.error)
  const fetchAgents = useAgentStore((s) => s.fetchAgents)
  const createAgent = useAgentStore((s) => s.createAgent)
  const updateAgent = useAgentStore((s) => s.updateAgent)
  const deleteAgent = useAgentStore((s) => s.deleteAgent)
  const setDefault = useAgentStore((s) => s.setDefault)

  // Local UI state
  const [editorOpen, setEditorOpen] = useState(false)
  const [editingAgent, setEditingAgent] = useState<Agent | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Agent | null>(null)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [menuOpenId, setMenuOpenId] = useState<string | null>(null)
  const [filterStatus, setFilterStatus] = useState<FilterStatus>('all')

  // Fetch on mount
  useEffect(() => {
    fetchAgents()
  }, [fetchAgents])

  // Close menus on outside click
  useEffect(() => {
    if (!menuOpenId) return
    const handler = () => setMenuOpenId(null)
    document.addEventListener('click', handler)
    return () => document.removeEventListener('click', handler)
  }, [menuOpenId])

  // ─── Handlers ───

  const handleCreate = useCallback(() => {
    setEditingAgent(null)
    setEditorOpen(true)
  }, [])

  const handleEdit = useCallback((agent: Agent) => {
    setEditingAgent(agent)
    setEditorOpen(true)
  }, [])

  const handleSave = useCallback(
    async (data: CreateAgentRequest | UpdateAgentRequest) => {
      setSaving(true)
      try {
        if (editingAgent) {
          await updateAgent(editingAgent.id, data as UpdateAgentRequest)
        } else {
          await createAgent(data as CreateAgentRequest)
        }
        setEditorOpen(false)
        setEditingAgent(null)
      } finally {
        setSaving(false)
      }
    },
    [editingAgent, createAgent, updateAgent],
  )

  const handleDeleteConfirm = useCallback(async () => {
    if (!deleteTarget) return
    setDeleting(true)
    try {
      await deleteAgent(deleteTarget.id)
      setDeleteTarget(null)
    } finally {
      setDeleting(false)
    }
  }, [deleteTarget, deleteAgent])

  const handleSetDefault = useCallback(
    async (agent: Agent) => {
      await setDefault(agent.id)
    },
    [setDefault],
  )

  // ─── Filtering ───
  const filteredAgents =
    filterStatus === 'all'
      ? agents
      : agents.filter((a) => a.status === filterStatus)

  const activeCount = agents.filter((a) => a.status === 'active').length
  const archivedCount = agents.filter((a) => a.status === 'archived').length

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* ─── Header ─── */}
      <div className="flex items-center justify-between gap-3 px-4 sm:px-6 py-3 sm:py-4 border-b border-[var(--border-subtle)] shrink-0">
        <div className="flex items-center gap-2 sm:gap-3 min-w-0">
          <h1 className="text-sm sm:text-base font-semibold text-[var(--text)] truncate">
            Agents
          </h1>
          {agents.length > 0 && (
            <Badge variant="default">{agents.length}</Badge>
          )}
        </div>

        <div className="flex items-center gap-2">
          {/* Filter pills — scrollable on mobile */}
          {agents.length > 0 && (
            <div className="flex items-center gap-1 mr-1 sm:mr-2 overflow-x-auto scrollbar-none">
              <FilterPill
                label="All"
                count={agents.length}
                active={filterStatus === 'all'}
                onClick={() => setFilterStatus('all')}
              />
              <FilterPill
                label="Active"
                count={activeCount}
                active={filterStatus === 'active'}
                onClick={() => setFilterStatus('active')}
              />
              {archivedCount > 0 && (
                <FilterPill
                  label="Archived"
                  count={archivedCount}
                  active={filterStatus === 'archived'}
                  onClick={() => setFilterStatus('archived')}
                />
              )}
            </div>
          )}

          <Button
            size="sm"
            icon={<Plus size={16} weight="bold" />}
            onClick={handleCreate}
          >
            <span className="hidden sm:inline">Create Agent</span>
            <span className="sm:hidden">New</span>
          </Button>
        </div>
      </div>

      {/* ─── Error banner ─── */}
      {error && agents.length > 0 && (
        <div className="px-6 py-2 text-xs text-[var(--error)] bg-[var(--error-subtle)] border-b border-[var(--border-subtle)] shrink-0">
          {error}
        </div>
      )}

      {/* ─── Content ─── */}
      <div className="flex-1 overflow-y-auto scroll-touch p-4 sm:p-6">
        <AgentList
          agents={filteredAgents}
          loading={loading}
          error={error}
          menuOpenId={menuOpenId}
          onToggleMenu={setMenuOpenId}
          onCreate={handleCreate}
          onEdit={handleEdit}
          onDelete={setDeleteTarget}
          onSetDefault={handleSetDefault}
          onRetry={fetchAgents}
        />
      </div>

      {/* ─── Editor modal ─── */}
      <AgentEditor
        open={editorOpen}
        onClose={() => {
          setEditorOpen(false)
          setEditingAgent(null)
        }}
        onSave={handleSave}
        agent={editingAgent}
        loading={saving}
      />

      {/* ─── Delete confirmation ─── */}
      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDeleteConfirm}
        title="Delete Agent"
        message={`Are you sure you want to delete "${deleteTarget?.name}"? This action cannot be undone. All sessions using this agent will be affected.`}
        confirmLabel="Delete"
        variant="danger"
        loading={deleting}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Filter pill helper
// ---------------------------------------------------------------------------

function FilterPill({
  label,
  count,
  active,
  onClick,
}: {
  label: string
  count: number
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      onClick={onClick}
      className={`
        px-2.5 py-1 rounded-full text-[11px] font-medium
        transition-colors cursor-pointer
        ${active
          ? 'bg-[var(--accent-subtle)] text-[var(--accent-text)]'
          : 'text-[var(--text-dim)] hover:text-[var(--text-secondary)] hover:bg-[var(--surface-2)]'
        }
      `}
    >
      {label} {count > 0 && <span className="opacity-70">({count})</span>}
    </button>
  )
}
