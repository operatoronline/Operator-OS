// ============================================================================
// Operator OS — Agent Editor
// Create/edit agent modal with form fields for all agent properties.
// Includes per-agent integration scope editor (C14).
// ============================================================================

import { useState, useEffect, useCallback } from 'react'
import { Modal } from '../shared/Modal'
import { Button } from '../shared/Button'
import { Input } from '../shared/Input'
import { ScopeSelector } from './ScopeSelector'
import { api, ApiRequestError } from '../../services/api'
import type {
  Agent,
  AgentIntegrationScope,
  CreateAgentRequest,
  UpdateAgentRequest,
  IntegrationSummary,
} from '../../types/api'

// ---------------------------------------------------------------------------
// Available models (could later come from an API endpoint)
// ---------------------------------------------------------------------------

const AVAILABLE_MODELS = [
  'gpt-4o',
  'gpt-4o-mini',
  'claude-sonnet-4-20250514',
  'claude-haiku-3.5',
  'gemini-2.0-flash',
  'gemini-2.0-pro',
  'o3-mini',
  'deepseek-chat',
]

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface AgentEditorProps {
  open: boolean
  onClose: () => void
  onSave: (data: CreateAgentRequest | UpdateAgentRequest) => Promise<void>
  agent?: Agent | null
  loading?: boolean
}

interface FormState {
  name: string
  description: string
  system_prompt: string
  model: string
  temperature: string
  max_tokens: string
  max_iterations: string
  tools: string
  skills: string
  allowed_integrations: AgentIntegrationScope[]
}

const defaultForm: FormState = {
  name: '',
  description: '',
  system_prompt: '',
  model: AVAILABLE_MODELS[0],
  temperature: '0.7',
  max_tokens: '4096',
  max_iterations: '10',
  tools: '',
  skills: '',
  allowed_integrations: [],
}

function agentToForm(agent: Agent): FormState {
  return {
    name: agent.name,
    description: agent.description,
    system_prompt: agent.system_prompt,
    model: agent.model,
    temperature: String(agent.temperature),
    max_tokens: String(agent.max_tokens),
    max_iterations: String(agent.max_iterations),
    tools: agent.tools.join(', '),
    skills: agent.skills.join(', '),
    allowed_integrations: agent.allowed_integrations ?? [],
  }
}

function formToPayload(form: FormState): CreateAgentRequest {
  return {
    name: form.name.trim(),
    description: form.description.trim() || undefined,
    system_prompt: form.system_prompt.trim() || undefined,
    model: form.model,
    temperature: parseFloat(form.temperature) || 0.7,
    max_tokens: parseInt(form.max_tokens, 10) || 4096,
    max_iterations: parseInt(form.max_iterations, 10) || 10,
    tools: form.tools
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean),
    skills: form.skills
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean),
    allowed_integrations: form.allowed_integrations.length > 0
      ? form.allowed_integrations
      : undefined,
  }
}

// ============================================================================
// Component
// ============================================================================

export function AgentEditor({ open, onClose, onSave, agent, loading }: AgentEditorProps) {
  const isEditing = !!agent
  const [form, setForm] = useState<FormState>(defaultForm)
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>({})

  // Integration data for ScopeSelector
  const [integrations, setIntegrations] = useState<IntegrationSummary[]>([])
  const [integrationsLoading, setIntegrationsLoading] = useState(false)
  const [integrationsError, setIntegrationsError] = useState<string | null>(null)

  // Reset form when modal opens / agent changes
  useEffect(() => {
    if (open) {
      setForm(agent ? agentToForm(agent) : defaultForm)
      setErrors({})
    }
  }, [open, agent])

  // Fetch available integrations when modal opens
  useEffect(() => {
    if (!open) return
    let cancelled = false
    setIntegrationsLoading(true)
    setIntegrationsError(null)

    api.integrations
      .list()
      .then((data) => {
        if (!cancelled) setIntegrations(data)
      })
      .catch((err) => {
        if (!cancelled) {
          setIntegrationsError(
            err instanceof ApiRequestError
              ? err.message
              : 'Failed to load integrations',
          )
        }
      })
      .finally(() => {
        if (!cancelled) setIntegrationsLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [open])

  const update = useCallback(
    (field: keyof FormState, value: string) => {
      setForm((prev) => ({ ...prev, [field]: value }))
      if (errors[field]) setErrors((prev) => ({ ...prev, [field]: undefined }))
    },
    [errors],
  )

  const validate = (): boolean => {
    const next: Partial<Record<keyof FormState, string>> = {}
    if (!form.name.trim()) next.name = 'Name is required'
    if (form.name.trim().length > 100) next.name = 'Name must be ≤ 100 characters'
    const temp = parseFloat(form.temperature)
    if (isNaN(temp) || temp < 0 || temp > 2) next.temperature = '0–2'
    const tokens = parseInt(form.max_tokens, 10)
    if (isNaN(tokens) || tokens < 1) next.max_tokens = 'Must be ≥ 1'
    setErrors(next)
    return Object.keys(next).length === 0
  }

  const handleSubmit = async () => {
    if (!validate()) return
    await onSave(formToPayload(form))
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEditing ? `Edit ${agent!.name}` : 'Create Agent'}
      maxWidth="max-w-xl"
    >
      <div className="flex flex-col gap-5">
        {/* ─── Name ─── */}
        <Input
          label="Name"
          placeholder="My Agent"
          value={form.name}
          onChange={(e) => update('name', e.target.value)}
          error={errors.name}
          autoFocus
        />

        {/* ─── Description ─── */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[13px] font-medium text-[var(--text-secondary)]">
            Description
          </label>
          <textarea
            value={form.description}
            onChange={(e) => update('description', e.target.value)}
            placeholder="What does this agent do?"
            rows={2}
            className="resize-none focus-ring"
          />
        </div>

        {/* ─── System Prompt ─── */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[13px] font-medium text-[var(--text-secondary)]">
            System Prompt
          </label>
          <textarea
            value={form.system_prompt}
            onChange={(e) => update('system_prompt', e.target.value)}
            placeholder="You are a helpful assistant…"
            rows={4}
            className="resize-none font-mono text-xs focus-ring"
          />
        </div>

        {/* ─── Model ─── */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[13px] font-medium text-[var(--text-secondary)]">
            Model
          </label>
          <select
            value={form.model}
            onChange={(e) => update('model', e.target.value)}
            className="focus-ring cursor-pointer"
          >
            {AVAILABLE_MODELS.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
            {/* Show current value if it's custom / not in list */}
            {form.model && !AVAILABLE_MODELS.includes(form.model) && (
              <option value={form.model}>{form.model} (custom)</option>
            )}
          </select>
        </div>

        {/* ─── Numeric params row ─── */}
        <div className="grid grid-cols-3 gap-4">
          <Input
            label="Temperature"
            type="number"
            step="0.1"
            min="0"
            max="2"
            value={form.temperature}
            onChange={(e) => update('temperature', e.target.value)}
            error={errors.temperature}
          />
          <Input
            label="Max Tokens"
            type="number"
            min="1"
            value={form.max_tokens}
            onChange={(e) => update('max_tokens', e.target.value)}
            error={errors.max_tokens}
          />
          <Input
            label="Max Iterations"
            type="number"
            min="1"
            value={form.max_iterations}
            onChange={(e) => update('max_iterations', e.target.value)}
            error={errors.max_iterations}
          />
        </div>

        {/* ─── Tools ─── */}
        <Input
          label="Tools (comma-separated)"
          placeholder="web_search, code_exec"
          value={form.tools}
          onChange={(e) => update('tools', e.target.value)}
        />

        {/* ─── Skills ─── */}
        <Input
          label="Skills (comma-separated)"
          placeholder="summarizer, coder"
          value={form.skills}
          onChange={(e) => update('skills', e.target.value)}
        />

        {/* ─── Integration Scopes (C14) ─── */}
        <ScopeSelector
          value={form.allowed_integrations}
          onChange={(scopes) =>
            setForm((prev) => ({ ...prev, allowed_integrations: scopes }))
          }
          integrations={integrations}
          loading={integrationsLoading}
          error={integrationsError}
        />

        {/* ─── Actions ─── */}
        <div className="flex justify-end gap-3 pt-2 border-t border-[var(--border-subtle)]">
          <Button variant="ghost" size="sm" onClick={onClose} disabled={loading}>
            Cancel
          </Button>
          <Button size="sm" onClick={handleSubmit} loading={loading}>
            {isEditing ? 'Save Changes' : 'Create Agent'}
          </Button>
        </div>
      </div>
    </Modal>
  )
}
