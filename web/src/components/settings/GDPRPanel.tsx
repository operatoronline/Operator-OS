// ============================================================================
// Operator OS — GDPRPanel
// Data export, account deletion requests, and request status.
// ============================================================================

import { useEffect, useState, useCallback, memo } from 'react'
import {
  Download,
  Trash,
  ShieldCheck,
  Clock,
  CheckCircle,
  XCircle,
  Spinner,
} from '@phosphor-icons/react'
import { useSettingsStore } from '../../stores/settingsStore'
import { Button } from '../shared/Button'
import { Badge } from '../shared/Badge'
import { ConfirmDialog } from '../shared/ConfirmDialog'
import type { DataSubjectRequest } from '../../types/api'

// ---------------------------------------------------------------------------
// Status badge mapping
// ---------------------------------------------------------------------------

const statusConfig: Record<string, { variant: 'default' | 'accent' | 'success' | 'warning' | 'error'; icon: typeof Clock }> = {
  pending: { variant: 'warning', icon: Clock },
  processing: { variant: 'accent', icon: Spinner },
  completed: { variant: 'success', icon: CheckCircle },
  failed: { variant: 'error', icon: XCircle },
  canceled: { variant: 'default', icon: XCircle },
}

// ---------------------------------------------------------------------------
// Request Row
// ---------------------------------------------------------------------------

const RequestRow = memo(function RequestRow({
  request,
  onCancel,
}: {
  request: DataSubjectRequest
  onCancel: (id: string) => void
}) {
  const config = statusConfig[request.status] || statusConfig.pending
  const Icon = config.icon
  const canCancel = request.status === 'pending'

  return (
    <div className="flex items-center gap-3 p-3 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--surface-2)]">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-[var(--text)] capitalize">
            {request.type === 'export' ? 'Data export' : 'Account deletion'}
          </span>
          <Badge variant={config.variant} dot>
            <Icon
              size={12}
              weight={request.status === 'processing' ? 'bold' : 'fill'}
              className={request.status === 'processing' ? 'animate-spin' : ''}
            />
            {request.status}
          </Badge>
        </div>
        <div className="text-[11px] text-[var(--text-dim)] mt-1">
          Requested {new Date(request.created_at).toLocaleDateString()}
          {request.completed_at && (
            <> · Completed {new Date(request.completed_at).toLocaleDateString()}</>
          )}
          {request.error_message && (
            <span className="text-error"> · {request.error_message}</span>
          )}
        </div>
      </div>

      {canCancel && (
        <button
          onClick={() => onCancel(request.id)}
          className="text-[11px] text-[var(--text-dim)] hover:text-error
            transition-colors cursor-pointer underline"
        >
          Cancel
        </button>
      )}
    </div>
  )
})

// ---------------------------------------------------------------------------
// Main Component
// ---------------------------------------------------------------------------

export function GDPRPanel() {
  const {
    gdprRequests,
    gdprLoading,
    gdprError,
    gdprSuccess,
    fetchGdprRequests,
    requestDataExport,
    requestAccountDeletion,
    cancelGdprRequest,
  } = useSettingsStore()

  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  useEffect(() => {
    fetchGdprRequests()
  }, [fetchGdprRequests])

  const handleExport = useCallback(() => {
    requestDataExport()
  }, [requestDataExport])

  const handleDeletion = useCallback(() => {
    requestAccountDeletion()
    setShowDeleteConfirm(false)
  }, [requestAccountDeletion])

  const hasPendingExport = gdprRequests.some(
    (r) => r.type === 'export' && (r.status === 'pending' || r.status === 'processing'),
  )
  const hasPendingDeletion = gdprRequests.some(
    (r) => r.type === 'erasure' && (r.status === 'pending' || r.status === 'processing'),
  )

  return (
    <div>
      <div className="flex items-center gap-3 mb-5">
        <div className="w-9 h-9 rounded-xl bg-accent-subtle flex items-center justify-center">
          <ShieldCheck size={18} weight="duotone" className="text-accent-text" />
        </div>
        <div>
          <h3 className="text-[15px] font-semibold text-[var(--text)]">Data & Privacy</h3>
          <p className="text-xs text-[var(--text-dim)]">
            Manage your personal data under GDPR
          </p>
        </div>
      </div>

      {gdprError && (
        <p className="text-xs text-error mb-4" role="alert">{gdprError}</p>
      )}
      {gdprSuccess && (
        <p className="text-xs text-success flex items-center gap-1 mb-4" role="status">
          <CheckCircle size={14} weight="fill" />
          {gdprSuccess}
        </p>
      )}

      {/* ─── Actions ─── */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-6">
        {/* Export */}
        <div className="p-4 rounded-[var(--radius)] border border-[var(--border-subtle)] bg-[var(--surface-2)]">
          <div className="flex items-center gap-2 mb-2">
            <Download size={18} weight="duotone" className="text-accent-text" />
            <h4 className="text-sm font-semibold text-[var(--text)]">Export data</h4>
          </div>
          <p className="text-[11px] text-[var(--text-dim)] mb-3">
            Download a copy of all your data — profile, sessions, usage, and settings.
          </p>
          <Button
            size="sm"
            variant="secondary"
            onClick={handleExport}
            disabled={hasPendingExport}
            loading={gdprLoading}
            icon={<Download size={14} />}
          >
            {hasPendingExport ? 'Export in progress' : 'Request export'}
          </Button>
        </div>

        {/* Delete */}
        <div className="p-4 rounded-[var(--radius)] border border-error/20 bg-error-subtle/30">
          <div className="flex items-center gap-2 mb-2">
            <Trash size={18} weight="duotone" className="text-error" />
            <h4 className="text-sm font-semibold text-[var(--text)]">Delete account</h4>
          </div>
          <p className="text-[11px] text-[var(--text-dim)] mb-3">
            Permanently delete your account and all associated data. This cannot be undone.
          </p>
          <Button
            size="sm"
            variant="danger"
            onClick={() => setShowDeleteConfirm(true)}
            disabled={hasPendingDeletion}
            icon={<Trash size={14} />}
          >
            {hasPendingDeletion ? 'Deletion pending' : 'Delete account'}
          </Button>
        </div>
      </div>

      {/* ─── Request History ─── */}
      {gdprRequests.length > 0 && (
        <div>
          <h4 className="text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider mb-3">
            Request history
          </h4>
          <div className="space-y-2">
            {gdprRequests.map((req) => (
              <RequestRow key={req.id} request={req} onCancel={cancelGdprRequest} />
            ))}
          </div>
        </div>
      )}

      {/* ─── Delete Confirmation ─── */}
      <ConfirmDialog
        open={showDeleteConfirm}
        onClose={() => setShowDeleteConfirm(false)}
        onConfirm={handleDeletion}
        title="Delete your account?"
        message="This will permanently erase all your data, including sessions, agents, integrations, and billing history. Active subscriptions will be canceled. This action cannot be undone."
        confirmLabel="Yes, delete my account"
        variant="danger"
      />
    </div>
  )
}
