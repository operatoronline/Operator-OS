// ============================================================================
// Operator OS — PlanChangePreview
// Modal showing proration preview before confirming a plan change.
// ============================================================================

import { ArrowRight, Warning, Info } from '@phosphor-icons/react'
import { Modal, Button, Badge } from '../shared'
import type { PlanChangeResult } from '../../types/api'

interface PlanChangePreviewProps {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  preview: PlanChangeResult | null
  loading: boolean
}

function formatCents(cents: number): string {
  const abs = Math.abs(cents)
  const str = `$${(abs / 100).toFixed(2)}`
  return cents < 0 ? `-${str}` : str
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

export function PlanChangePreview({ open, onClose, onConfirm, preview, loading }: PlanChangePreviewProps) {
  if (!preview) return null

  const isUpgrade = preview.direction === 'upgrade'
  const isDowngrade = preview.direction === 'downgrade'
  const isImmediate = preview.mode === 'immediate'

  return (
    <Modal open={open} onClose={onClose} title="Confirm Plan Change" maxWidth="max-w-md">
      <div className="space-y-5">
        {/* Direction indicator */}
        <div className="flex items-center justify-center gap-3 py-3">
          <div className="text-center">
            <p className="text-xs text-text-dim mb-1">From</p>
            <p className="text-sm font-semibold text-text">{preview.previous_plan}</p>
          </div>
          <ArrowRight size={20} className="text-accent-text" />
          <div className="text-center">
            <p className="text-xs text-text-dim mb-1">To</p>
            <p className="text-sm font-semibold text-text">{preview.new_plan}</p>
          </div>
        </div>

        {/* Direction badge */}
        <div className="flex justify-center">
          <Badge variant={isUpgrade ? 'accent' : isDowngrade ? 'warning' : 'default'}>
            {preview.direction.toUpperCase()}
          </Badge>
        </div>

        {/* Proration */}
        {preview.proration_amount !== 0 && (
          <div className="bg-surface-2 rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm text-text-secondary">Proration adjustment</span>
              <span className={`text-sm font-semibold ${preview.proration_amount > 0 ? 'text-text' : 'text-success'}`}>
                {formatCents(preview.proration_amount)}
              </span>
            </div>
            <p className="text-xs text-text-dim">
              {preview.proration_amount > 0
                ? 'This amount will be charged to your payment method.'
                : 'This credit will be applied to your next invoice.'}
            </p>
          </div>
        )}

        {/* Timing */}
        <div className="flex items-start gap-2.5 p-3 rounded-lg bg-[var(--surface-2)]">
          <Info size={16} className="text-accent-text shrink-0 mt-0.5" />
          <p className="text-xs text-text-secondary leading-relaxed">
            {isImmediate
              ? `Change takes effect immediately.`
              : `Change takes effect on ${formatDate(preview.effective_at)}.`}
          </p>
        </div>

        {/* Downgrade warning */}
        {isDowngrade && (
          <div className="flex items-start gap-2.5 p-3 rounded-lg bg-warning-subtle border border-warning/20">
            <Warning size={16} className="text-warning shrink-0 mt-0.5" />
            <p className="text-xs text-warning leading-relaxed">
              Downgrading may reduce your available agents, integrations, and token limits.
              Existing data won't be deleted, but you may lose access until you upgrade again.
            </p>
          </div>
        )}

        {/* Actions */}
        <div className="flex items-center gap-3 pt-2">
          <Button variant="ghost" size="md" onClick={onClose} className="flex-1">
            Cancel
          </Button>
          <Button
            variant={isDowngrade ? 'danger' : 'primary'}
            size="md"
            onClick={onConfirm}
            loading={loading}
            className="flex-1"
          >
            {isUpgrade ? 'Upgrade Now' : isDowngrade ? 'Confirm Downgrade' : 'Confirm Change'}
          </Button>
        </div>
      </div>
    </Modal>
  )
}
