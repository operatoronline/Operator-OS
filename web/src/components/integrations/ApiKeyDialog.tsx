// ============================================================================
// Operator OS — API Key Dialog
// Modal for entering an API key to connect an integration.
// ============================================================================

import { useState } from 'react'
import { Key, Eye, EyeSlash } from '@phosphor-icons/react'
import { Modal } from '../shared/Modal'
import { Button } from '../shared/Button'
import type { IntegrationSummary } from '../../types/api'

interface ApiKeyDialogProps {
  open: boolean
  onClose: () => void
  integration: IntegrationSummary | null
  onSubmit: (apiKey: string) => void
  loading?: boolean
  error?: string | null
}

export function ApiKeyDialog({
  open,
  onClose,
  integration,
  onSubmit,
  loading,
  error,
}: ApiKeyDialogProps) {
  const [apiKey, setApiKey] = useState('')
  const [showKey, setShowKey] = useState(false)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (apiKey.trim()) {
      onSubmit(apiKey.trim())
    }
  }

  const handleClose = () => {
    setApiKey('')
    setShowKey(false)
    onClose()
  }

  if (!integration) return null

  return (
    <Modal open={open} onClose={handleClose} title={`Connect ${integration.name}`}>
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        {/* Description */}
        <p className="text-sm text-[var(--text-secondary)] leading-relaxed">
          Enter your API key to connect <strong className="text-[var(--text)]">{integration.name}</strong>.
          Your key is encrypted and stored securely.
        </p>

        {/* API Key Input */}
        <div className="flex flex-col gap-1.5">
          <label
            htmlFor="api-key-input"
            className="text-xs font-medium text-[var(--text-secondary)]"
          >
            API Key
          </label>
          <div className="relative">
            <div className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-dim)]">
              <Key size={16} />
            </div>
            <input
              id="api-key-input"
              type={showKey ? 'text' : 'password'}
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="sk-..."
              autoFocus
              className="w-full h-10 pl-9 pr-10
                bg-[var(--surface-2)] border border-[var(--border-subtle)]
                rounded-[var(--radius-md)] text-sm text-[var(--text)] font-mono
                placeholder:text-[var(--text-dim)]
                focus:border-[var(--accent)] focus:outline-none
                transition-colors"
            />
            <button
              type="button"
              onClick={() => setShowKey(!showKey)}
              className="absolute right-3 top-1/2 -translate-y-1/2
                text-[var(--text-dim)] hover:text-[var(--text)]
                transition-colors cursor-pointer"
              aria-label={showKey ? 'Hide key' : 'Show key'}
            >
              {showKey ? <EyeSlash size={16} /> : <Eye size={16} />}
            </button>
          </div>
        </div>

        {/* Error */}
        {error && (
          <div className="text-xs text-[var(--error)] bg-[var(--error-subtle)] px-3 py-2 rounded-lg">
            {error}
          </div>
        )}

        {/* Actions */}
        <div className="flex items-center justify-end gap-2 pt-2">
          <Button variant="ghost" size="sm" type="button" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            variant="primary"
            size="sm"
            type="submit"
            loading={loading}
            disabled={!apiKey.trim()}
          >
            Connect
          </Button>
        </div>
      </form>
    </Modal>
  )
}
