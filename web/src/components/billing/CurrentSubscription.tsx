// ============================================================================
// Operator OS — CurrentSubscription
// Displays active subscription details with manage/portal actions.
// ============================================================================

import { CreditCard, ArrowSquareOut, CalendarBlank, Warning } from '@phosphor-icons/react'
import { Button, Badge } from '../shared'
import type { Subscription } from '../../types/api'

interface CurrentSubscriptionProps {
  subscription: Subscription | null
  onManageClick: () => void
  loadingPortal: boolean
}

const statusVariants: Record<string, 'success' | 'warning' | 'error' | 'accent' | 'default'> = {
  active: 'success',
  trialing: 'accent',
  past_due: 'warning',
  canceled: 'error',
  expired: 'error',
  paused: 'default',
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

export function CurrentSubscription({ subscription, onManageClick, loadingPortal }: CurrentSubscriptionProps) {
  if (!subscription) {
    return (
      <div className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius)] p-5">
        <div className="flex items-center gap-3 mb-2">
          <CreditCard size={20} weight="duotone" className="text-text-dim" />
          <h3 className="text-sm font-semibold text-text">Current Plan</h3>
        </div>
        <p className="text-sm text-text-secondary mb-1">
          You're on the <span className="font-semibold text-text">Free</span> plan.
        </p>
        <p className="text-xs text-text-dim">
          Upgrade to unlock more agents, integrations, and higher limits.
        </p>
      </div>
    )
  }

  const { plan, status, interval, current_period_end, cancel_at_period_end } = subscription

  return (
    <div className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius)] p-5">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1">
          <div className="flex items-center gap-3 mb-2">
            <CreditCard size={20} weight="duotone" className="text-accent-text" />
            <h3 className="text-sm font-semibold text-text">Current Plan</h3>
            <Badge variant={statusVariants[status] || 'default'} dot>
              {status.replace('_', ' ')}
            </Badge>
          </div>

          <p className="text-lg font-semibold text-text mb-1">{plan.name}</p>

          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-text-dim">
            <span className="flex items-center gap-1">
              <CalendarBlank size={13} />
              Billed {interval}
            </span>
            <span>
              Renews {formatDate(current_period_end)}
            </span>
          </div>

          {cancel_at_period_end && (
            <div className="flex items-center gap-2 mt-3 p-2.5 rounded-lg bg-warning-subtle border border-warning/20">
              <Warning size={16} className="text-warning shrink-0" />
              <span className="text-xs text-warning">
                Cancels at end of period ({formatDate(current_period_end)})
              </span>
            </div>
          )}

          {status === 'past_due' && (
            <div className="flex items-center gap-2 mt-3 p-2.5 rounded-lg bg-error-subtle border border-error/20">
              <Warning size={16} className="text-error shrink-0" />
              <span className="text-xs text-error">
                Payment past due — please update your billing info.
              </span>
            </div>
          )}
        </div>

        <Button
          variant="secondary"
          size="sm"
          icon={<ArrowSquareOut size={14} />}
          onClick={onManageClick}
          loading={loadingPortal}
        >
          Manage
        </Button>
      </div>
    </div>
  )
}
