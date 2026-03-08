// ============================================================================
// Operator OS — Integration Scope Selector
// Per-agent visual editor for allowed integrations, tools, and OAuth scopes.
// ============================================================================

import { useState, useCallback, memo } from 'react'
import {
  Plugs,
  CaretDown,
  CaretUp,
  CheckSquare,
  Square,
  Wrench,
  ShieldCheck,
  Warning,
  Spinner,
  MagnifyingGlass,
} from '@phosphor-icons/react'
import { Badge } from '../shared/Badge'
import type { AgentIntegrationScope, IntegrationSummary } from '../../types/api'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface ScopeSelectorProps {
  /** Currently selected integration scopes for this agent */
  value: AgentIntegrationScope[]
  /** Called when scopes change */
  onChange: (scopes: AgentIntegrationScope[]) => void
  /** Available integrations from the platform */
  integrations: IntegrationSummary[]
  /** Loading state while fetching integrations */
  loading?: boolean
  /** Error fetching integrations */
  error?: string | null
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export const ScopeSelector = memo(function ScopeSelector({
  value,
  onChange,
  integrations,
  loading,
  error,
}: ScopeSelectorProps) {
  const [search, setSearch] = useState('')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  // Filter integrations by search
  const filtered = integrations.filter((i) => {
    if (!search.trim()) return true
    const q = search.toLowerCase()
    return (
      i.name.toLowerCase().includes(q) ||
      i.category.toLowerCase().includes(q) ||
      i.description.toLowerCase().includes(q)
    )
  })

  // Group by category
  const byCategory = filtered.reduce<Record<string, IntegrationSummary[]>>(
    (acc, i) => {
      const cat = i.category || 'Other'
      ;(acc[cat] ??= []).push(i)
      return acc
    },
    {},
  )
  const categoryKeys = Object.keys(byCategory).sort()

  // Helpers
  const findScope = useCallback(
    (integrationId: string) => value.find((s) => s.integration_id === integrationId),
    [value],
  )

  const isEnabled = useCallback(
    (integrationId: string) => !!findScope(integrationId),
    [findScope],
  )

  const toggleIntegration = useCallback(
    (integration: IntegrationSummary) => {
      if (isEnabled(integration.id)) {
        onChange(value.filter((s) => s.integration_id !== integration.id))
      } else {
        // Enable with all tools and scopes by default
        const newScope: AgentIntegrationScope = {
          integration_id: integration.id,
          allowed_tools: integration.tools?.map((t) => t.name),
          allowed_scopes: [], // No OAuth scope restrictions by default
        }
        onChange([...value, newScope])
      }
    },
    [value, onChange, isEnabled],
  )

  const toggleTool = useCallback(
    (integrationId: string, toolName: string) => {
      const scope = findScope(integrationId)
      if (!scope) return

      const tools = scope.allowed_tools ?? []
      const next = tools.includes(toolName)
        ? tools.filter((t) => t !== toolName)
        : [...tools, toolName]

      onChange(
        value.map((s) =>
          s.integration_id === integrationId
            ? { ...s, allowed_tools: next }
            : s,
        ),
      )
    },
    [value, onChange, findScope],
  )

  const toggleOAuthScope = useCallback(
    (integrationId: string, scopeName: string) => {
      const scope = findScope(integrationId)
      if (!scope) return

      const scopes = scope.allowed_scopes ?? []
      const next = scopes.includes(scopeName)
        ? scopes.filter((s) => s !== scopeName)
        : [...scopes, scopeName]

      onChange(
        value.map((s) =>
          s.integration_id === integrationId
            ? { ...s, allowed_scopes: next }
            : s,
        ),
      )
    },
    [value, onChange, findScope],
  )

  const toggleAllTools = useCallback(
    (integration: IntegrationSummary) => {
      const scope = findScope(integration.id)
      if (!scope || !integration.tools?.length) return

      const allNames = integration.tools.map((t) => t.name)
      const allSelected = allNames.every((n) =>
        (scope.allowed_tools ?? []).includes(n),
      )

      onChange(
        value.map((s) =>
          s.integration_id === integration.id
            ? { ...s, allowed_tools: allSelected ? [] : allNames }
            : s,
        ),
      )
    },
    [value, onChange, findScope],
  )

  // ─── Loading / Error states ───
  if (loading) {
    return (
      <div className="flex items-center justify-center gap-2 py-8 text-[var(--text-dim)] text-sm">
        <Spinner size={18} className="animate-spin" />
        Loading integrations…
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center gap-2 py-6 px-4 rounded-lg
        bg-[var(--error-subtle)] text-[var(--error)] text-sm">
        <Warning size={18} />
        {error}
      </div>
    )
  }

  if (integrations.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 py-8 text-[var(--text-dim)] text-sm">
        <Plugs size={32} weight="thin" />
        <p>No integrations available</p>
        <p className="text-[11px]">Connect integrations in the Integrations page first</p>
      </div>
    )
  }

  // ─── Render ───
  return (
    <div className="flex flex-col gap-3">
      {/* Header with count */}
      <div className="flex items-center justify-between">
        <span className="text-[13px] font-medium text-[var(--text-secondary)]">
          Integration Permissions
        </span>
        {value.length > 0 && (
          <Badge variant="accent">
            {value.length} enabled
          </Badge>
        )}
      </div>

      {/* Search */}
      {integrations.length > 4 && (
        <div className="relative">
          <MagnifyingGlass
            size={15}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-dim)]"
          />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Filter integrations…"
            className="w-full pl-9 pr-3 py-2 text-xs rounded-lg
              bg-[var(--surface-2)] border border-[var(--border-subtle)]
              text-[var(--text)] placeholder:text-[var(--text-dim)]
              focus:outline-none focus:border-[var(--accent-text)]
              transition-colors"
          />
        </div>
      )}

      {/* Integration list grouped by category */}
      <div className="flex flex-col gap-1 max-h-[320px] overflow-y-auto
        -mx-1 px-1 scrollbar-thin">
        {categoryKeys.map((category) => (
          <div key={category}>
            {categoryKeys.length > 1 && (
              <div className="text-[10px] uppercase tracking-wider font-semibold
                text-[var(--text-dim)] px-1 pt-2 pb-1">
                {category}
              </div>
            )}
            {byCategory[category].map((integration) => (
              <IntegrationRow
                key={integration.id}
                integration={integration}
                scope={findScope(integration.id)}
                enabled={isEnabled(integration.id)}
                expanded={expandedId === integration.id}
                onToggle={() => toggleIntegration(integration)}
                onExpand={() =>
                  setExpandedId(
                    expandedId === integration.id ? null : integration.id,
                  )
                }
                onToggleTool={(tool) => toggleTool(integration.id, tool)}
                onToggleScope={(scope) => toggleOAuthScope(integration.id, scope)}
                onToggleAllTools={() => toggleAllTools(integration)}
              />
            ))}
          </div>
        ))}

        {filtered.length === 0 && search && (
          <p className="text-center text-xs text-[var(--text-dim)] py-4">
            No integrations match "{search}"
          </p>
        )}
      </div>
    </div>
  )
})

// ---------------------------------------------------------------------------
// Integration row — shows toggle, expand for tools/scopes
// ---------------------------------------------------------------------------

interface IntegrationRowProps {
  integration: IntegrationSummary
  scope: AgentIntegrationScope | undefined
  enabled: boolean
  expanded: boolean
  onToggle: () => void
  onExpand: () => void
  onToggleTool: (tool: string) => void
  onToggleScope: (scope: string) => void
  onToggleAllTools: () => void
}

const IntegrationRow = memo(function IntegrationRow({
  integration,
  scope,
  enabled,
  expanded,
  onToggle,
  onExpand,
  onToggleTool,
  onToggleScope: _onToggleScope,
  onToggleAllTools,
}: IntegrationRowProps) {
  const hasTools = integration.tools && integration.tools.length > 0
  const hasDetails = hasTools // Could add OAuth scope list here too

  return (
    <div
      className={`
        rounded-lg border transition-all duration-200
        ${enabled
          ? 'border-[var(--accent-text)]/30 bg-[var(--accent-subtle)]'
          : 'border-[var(--border-subtle)] bg-[var(--surface)]'
        }
      `}
    >
      {/* ─── Main row ─── */}
      <div className="flex items-center gap-3 px-3 py-2.5">
        {/* Toggle checkbox */}
        <button
          onClick={onToggle}
          className="shrink-0 text-[var(--accent-text)] hover:opacity-80
            transition-opacity cursor-pointer"
          aria-label={enabled ? `Disable ${integration.name}` : `Enable ${integration.name}`}
        >
          {enabled ? (
            <CheckSquare size={20} weight="fill" />
          ) : (
            <Square size={20} className="text-[var(--text-dim)]" />
          )}
        </button>

        {/* Icon + info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-[var(--text)] truncate">
              {integration.icon && (
                <span className="mr-1.5">{integration.icon}</span>
              )}
              {integration.name}
            </span>
            {integration.auth_type === 'oauth2' && (
              <ShieldCheck
                size={14}
                className="text-[var(--text-dim)] shrink-0"
                aria-label="OAuth 2.0"
              />
            )}
          </div>
          <p className="text-[11px] text-[var(--text-dim)] truncate mt-0.5">
            {integration.description}
          </p>
        </div>

        {/* Tool count + expand */}
        <div className="flex items-center gap-1.5 shrink-0">
          {enabled && hasTools && (
            <span className="text-[10px] text-[var(--text-dim)] font-mono">
              {scope?.allowed_tools?.length ?? 0}/{integration.tools!.length}
            </span>
          )}
          {enabled && hasDetails && (
            <button
              onClick={onExpand}
              className="p-1 rounded text-[var(--text-dim)]
                hover:text-[var(--text)] hover:bg-[var(--surface-2)]
                transition-colors cursor-pointer"
              aria-label={expanded ? 'Collapse' : 'Expand'}
            >
              {expanded ? <CaretUp size={14} /> : <CaretDown size={14} />}
            </button>
          )}
        </div>
      </div>

      {/* ─── Expanded detail: tools + scopes ─── */}
      {enabled && expanded && hasDetails && (
        <div className="px-3 pb-3 pt-0">
          <div className="border-t border-[var(--border-subtle)] pt-2.5 flex flex-col gap-2">
            {/* Tools section */}
            {hasTools && (
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <span className="text-[11px] font-semibold text-[var(--text-secondary)] flex items-center gap-1">
                    <Wrench size={12} />
                    Tools
                  </span>
                  <button
                    onClick={onToggleAllTools}
                    className="text-[10px] text-[var(--accent-text)]
                      hover:underline cursor-pointer"
                  >
                    {scope?.allowed_tools?.length === integration.tools!.length
                      ? 'Deselect all'
                      : 'Select all'}
                  </button>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {integration.tools!.map((tool) => {
                    const selected = (scope?.allowed_tools ?? []).includes(
                      tool.name,
                    )
                    return (
                      <button
                        key={tool.name}
                        onClick={() => onToggleTool(tool.name)}
                        title={tool.description}
                        className={`
                          inline-flex items-center gap-1 px-2 py-1
                          rounded-md text-[11px] font-mono
                          border transition-all cursor-pointer
                          ${selected
                            ? 'bg-[var(--accent-subtle)] border-[var(--accent-text)]/40 text-[var(--accent-text)]'
                            : 'bg-[var(--surface-2)] border-[var(--border-subtle)] text-[var(--text-dim)] hover:text-[var(--text-secondary)]'
                          }
                        `}
                      >
                        {selected ? (
                          <CheckSquare size={12} weight="fill" />
                        ) : (
                          <Square size={12} />
                        )}
                        {tool.name}
                      </button>
                    )
                  })}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
})
