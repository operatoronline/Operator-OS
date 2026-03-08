// ============================================================================
// Operator OS — Agent List
// Grid of AgentCards with create button and empty state.
// ============================================================================

import { Robot, Plus, WarningCircle } from '@phosphor-icons/react'
import { AgentCard } from './AgentCard'
import { Button } from '../shared/Button'
import { EmptyState } from '../shared/EmptyState'
import type { Agent } from '../../types/api'

interface AgentListProps {
  agents: Agent[]
  loading: boolean
  error: string | null
  menuOpenId: string | null
  onToggleMenu: (id: string | null) => void
  onCreate: () => void
  onEdit: (agent: Agent) => void
  onDelete: (agent: Agent) => void
  onSetDefault: (agent: Agent) => void
  onRetry: () => void
}

export function AgentList({
  agents,
  loading,
  error,
  menuOpenId,
  onToggleMenu,
  onCreate,
  onEdit,
  onDelete,
  onSetDefault,
  onRetry,
}: AgentListProps) {
  // ─── Loading skeleton ───
  if (loading && agents.length === 0) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 p-6">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="h-48 rounded-[var(--radius)] bg-[var(--surface-2)] animate-pulse"
          />
        ))}
      </div>
    )
  }

  // ─── Error state ───
  if (error && agents.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-center px-4">
        <WarningCircle size={48} weight="thin" className="text-[var(--error)] mb-4" />
        <h2 className="text-lg font-semibold text-[var(--text)] mb-1">
          Failed to load agents
        </h2>
        <p className="text-sm text-[var(--text-secondary)] mb-4">{error}</p>
        <Button variant="secondary" size="sm" onClick={onRetry}>
          Try again
        </Button>
      </div>
    )
  }

  // ─── Empty state ───
  if (agents.length === 0) {
    return (
      <EmptyState
        icon={Robot}
        title="No agents yet"
        description="Create your first agent to start conversations. Agents define the AI model, personality, and tools available in a chat session."
        action={{ label: 'Create Agent', onClick: onCreate, icon: Plus }}
        className="h-full"
      />
    )
  }

  // ─── Agent grid ───
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      {agents.map((agent) => (
        <AgentCard
          key={agent.id}
          agent={agent}
          onEdit={onEdit}
          onDelete={onDelete}
          onSetDefault={onSetDefault}
          menuOpenId={menuOpenId}
          onToggleMenu={onToggleMenu}
        />
      ))}
    </div>
  )
}
