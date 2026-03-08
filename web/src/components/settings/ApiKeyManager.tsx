// ============================================================================
// Operator OS — ApiKeyManager
// Create, view, and delete API keys. Shows secret once on creation.
// ============================================================================

import { useEffect, useState, useCallback, memo } from 'react'
import {
  Key,
  Plus,
  Trash,
  Copy,
  Check,
  Eye,
  EyeSlash,
  Warning,
  Clock,
} from '@phosphor-icons/react'
import { useSettingsStore } from '../../stores/settingsStore'
import { Button } from '../shared/Button'
import { Input } from '../shared/Input'
import { Badge } from '../shared/Badge'
import { Modal } from '../shared/Modal'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import type { ApiKey } from '../../types/api'

// ---------------------------------------------------------------------------
// Key Row
// ---------------------------------------------------------------------------

const KeyRow = memo(function KeyRow({
  apiKey,
  onDelete,
}: {
  apiKey: ApiKey
  onDelete: (id: string) => void
}) {
  const isExpired = apiKey.expires_at && new Date(apiKey.expires_at) < new Date()
  const lastUsed = apiKey.last_used_at
    ? new Date(apiKey.last_used_at).toLocaleDateString()
    : 'Never'

  return (
    <div className="flex items-center gap-3 p-4 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--surface-2)]">
      <div className="w-8 h-8 rounded-lg bg-[var(--surface-3)] flex items-center justify-center shrink-0">
        <Key size={16} className="text-[var(--text-dim)]" />
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-[var(--text)] truncate">
            {apiKey.name}
          </span>
          {isExpired && <Badge variant="error">Expired</Badge>}
        </div>
        <div className="flex items-center gap-3 mt-1 text-[11px] text-[var(--text-dim)]">
          <span className="font-mono">{apiKey.prefix}•••</span>
          <span className="flex items-center gap-1">
            <Clock size={12} />
            Used: {lastUsed}
          </span>
          {apiKey.expires_at && (
            <span>
              Expires: {new Date(apiKey.expires_at).toLocaleDateString()}
            </span>
          )}
        </div>
      </div>

      <button
        onClick={() => onDelete(apiKey.id)}
        className="p-2 rounded-lg text-[var(--text-dim)] hover:text-error
          hover:bg-error-subtle transition-colors cursor-pointer"
        aria-label={`Delete ${apiKey.name}`}
      >
        <Trash size={16} />
      </button>
    </div>
  )
})

// ---------------------------------------------------------------------------
// Create Key Dialog
// ---------------------------------------------------------------------------

function CreateKeyDialog({
  open,
  onClose,
}: {
  open: boolean
  onClose: () => void
}) {
  const { createApiKey, apiKeysLoading } = useSettingsStore()
  const [name, setName] = useState('')
  const [expiry, setExpiry] = useState('90') // days

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    try {
      await createApiKey(name.trim(), expiry ? Number(expiry) : undefined)
      setName('')
      setExpiry('90')
      onClose()
    } catch {
      // error in store
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Create API key">
      <form onSubmit={handleCreate} className="space-y-4">
        <Input
          label="Key name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. Production, CI/CD"
          autoFocus
        />

        <div className="flex flex-col gap-1.5">
          <label className="text-[13px] font-medium text-text-secondary">
            Expiration
          </label>
          <select
            value={expiry}
            onChange={(e) => setExpiry(e.target.value)}
            className="w-full rounded-[var(--radius-md)] bg-[var(--surface-2)] border border-[var(--border)]
              px-3 py-2 text-sm text-[var(--text)] focus-ring"
          >
            <option value="30">30 days</option>
            <option value="90">90 days</option>
            <option value="180">180 days</option>
            <option value="365">1 year</option>
            <option value="">No expiration</option>
          </select>
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <Button variant="ghost" size="sm" type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button size="sm" type="submit" disabled={!name.trim()} loading={apiKeysLoading}>
            Create key
          </Button>
        </div>
      </form>
    </Modal>
  )
}

// ---------------------------------------------------------------------------
// Secret Banner
// ---------------------------------------------------------------------------

function SecretBanner({ secret, onDismiss }: { secret: string; onDismiss: () => void }) {
  const [copied, setCopied] = useState(false)
  const [visible, setVisible] = useState(false)

  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(secret)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [secret])

  return (
    <div className="p-4 rounded-[var(--radius)] border border-warning bg-warning-subtle mb-4 animate-fade-slide">
      <div className="flex items-start gap-3">
        <Warning size={20} weight="fill" className="text-warning shrink-0 mt-0.5" />
        <div className="flex-1 min-w-0">
          <p className="text-sm font-semibold text-[var(--text)] mb-1">
            Copy your API key now
          </p>
          <p className="text-[11px] text-[var(--text-dim)] mb-3">
            This is the only time you'll see the full key. Store it securely.
          </p>

          <div className="flex items-center gap-2">
            <code className="flex-1 px-3 py-2 rounded-lg bg-[var(--surface)] border border-[var(--border)]
              font-mono text-xs text-[var(--text)] truncate select-all">
              {visible ? secret : '•'.repeat(Math.min(secret.length, 40))}
            </code>
            <button
              onClick={() => setVisible(!visible)}
              className="p-2 rounded-lg hover:bg-[var(--surface-3)] text-[var(--text-dim)]
                hover:text-[var(--text)] transition-colors cursor-pointer"
              aria-label={visible ? 'Hide key' : 'Show key'}
            >
              {visible ? <EyeSlash size={16} /> : <Eye size={16} />}
            </button>
            <button
              onClick={handleCopy}
              className="p-2 rounded-lg hover:bg-[var(--surface-3)] text-[var(--text-dim)]
                hover:text-[var(--text)] transition-colors cursor-pointer"
              aria-label="Copy key"
            >
              {copied ? (
                <Check size={16} className="text-success" />
              ) : (
                <Copy size={16} />
              )}
            </button>
          </div>

          <button
            onClick={onDismiss}
            className="text-[11px] text-[var(--text-dim)] hover:text-[var(--text)]
              mt-2 underline cursor-pointer"
          >
            I've saved it, dismiss
          </button>
        </div>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Main Component
// ---------------------------------------------------------------------------

export function ApiKeyManager() {
  const {
    apiKeys,
    apiKeysLoading,
    apiKeysError,
    newKeySecret,
    fetchApiKeys,
    deleteApiKey,
    clearNewKeySecret,
  } = useSettingsStore()

  const [showCreate, setShowCreate] = useState(false)
  const [deleteId, setDeleteId] = useState<string | null>(null)

  useEffect(() => {
    fetchApiKeys()
  }, [fetchApiKeys])

  const handleDelete = useCallback(() => {
    if (deleteId) {
      deleteApiKey(deleteId)
      setDeleteId(null)
    }
  }, [deleteId, deleteApiKey])

  const deletingKey = apiKeys.find((k) => k.id === deleteId)

  return (
    <div>
      <div className="flex items-center justify-between mb-5">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-xl bg-accent-subtle flex items-center justify-center">
            <Key size={18} weight="duotone" className="text-accent-text" />
          </div>
          <div>
            <h3 className="text-[15px] font-semibold text-[var(--text)]">API Keys</h3>
            <p className="text-xs text-[var(--text-dim)]">Manage programmatic access to your account</p>
          </div>
        </div>

        <Button
          size="sm"
          variant="secondary"
          icon={<Plus size={14} />}
          onClick={() => setShowCreate(true)}
        >
          Create key
        </Button>
      </div>

      {/* New key secret banner */}
      {newKeySecret && (
        <SecretBanner secret={newKeySecret} onDismiss={clearNewKeySecret} />
      )}

      {apiKeysError && (
        <p className="text-xs text-error mb-4" role="alert">{apiKeysError}</p>
      )}

      {/* Key list */}
      {apiKeysLoading && apiKeys.length === 0 ? (
        <div className="space-y-3">
          {[1, 2].map((i) => (
            <div key={i} className="h-[72px] rounded-[var(--radius-md)] bg-[var(--surface-2)] animate-pulse" />
          ))}
        </div>
      ) : apiKeys.length === 0 ? (
        <div className="text-center py-8 text-[var(--text-dim)]">
          <Key size={32} weight="thin" className="mx-auto mb-2 text-[var(--text-dim)]" />
          <p className="text-sm">No API keys yet</p>
          <p className="text-[11px] mt-1">Create one to access the API programmatically</p>
        </div>
      ) : (
        <div className="space-y-2">
          {apiKeys.map((key) => (
            <KeyRow key={key.id} apiKey={key} onDelete={setDeleteId} />
          ))}
        </div>
      )}

      {/* Create dialog */}
      <CreateKeyDialog open={showCreate} onClose={() => setShowCreate(false)} />

      {/* Delete confirmation */}
      <ConfirmDialog
        open={!!deleteId}
        onClose={() => setDeleteId(null)}
        onConfirm={handleDelete}
        title="Delete API key"
        message={`Delete "${deletingKey?.name || 'this key'}"? Any applications using this key will lose access immediately.`}
        confirmLabel="Delete key"
        variant="danger"
      />
    </div>
  )
}
