// ============================================================================
// Operator OS — Overage Warning Banner
// Shows overage status with severity levels and actionable guidance.
// ============================================================================

import { memo } from 'react'
import {
  Warning,
  ShieldWarning,
  Prohibit,
  ArrowUp,
} from '@phosphor-icons/react'
import { Button } from '../shared'
import type { OverageStatus, OverageLevel } from '../../types/api'

interface OverageWarningProps {
  overage: OverageStatus | null
  loading: boolean
}

const levelConfig: Record<
  OverageLevel,
  {
    icon: React.ReactNode
    bg: string
    border: string
    text: string
    label: string
    showUpgrade: boolean
  }
> = {
  none: {
    icon: null,
    bg: '',
    border: '',
    text: '',
    label: '',
    showUpgrade: false,
  },
  warning: {
    icon: <Warning size={18} weight="fill" className="text-warning" />,
    bg: 'bg-warning-subtle',
    border: 'border-warning/20',
    text: 'text-warning',
    label: 'Approaching limits',
    showUpgrade: true,
  },
  soft_cap: {
    icon: <ShieldWarning size={18} weight="fill" className="text-warning" />,
    bg: 'bg-warning-subtle',
    border: 'border-warning/30',
    text: 'text-warning',
    label: 'Soft cap reached',
    showUpgrade: true,
  },
  hard_cap: {
    icon: <Prohibit size={18} weight="fill" className="text-error" />,
    bg: 'bg-error-subtle',
    border: 'border-error/20',
    text: 'text-error',
    label: 'Hard cap reached',
    showUpgrade: true,
  },
  blocked: {
    icon: <Prohibit size={18} weight="fill" className="text-error" />,
    bg: 'bg-error-subtle',
    border: 'border-error/30',
    text: 'text-error',
    label: 'Usage blocked',
    showUpgrade: true,
  },
}

export const OverageWarning = memo(function OverageWarning({
  overage,
  loading,
}: OverageWarningProps) {
  if (loading || !overage || overage.overall_level === 'none') return null

  const config = levelConfig[overage.overall_level]
  if (!config.icon) return null

  return (
    <div
      className={`${config.bg} border ${config.border} rounded-[var(--radius-md)] p-4 animate-fade-slide`}
    >
      <div className="flex items-start gap-3">
        <div className="shrink-0 mt-0.5">{config.icon}</div>

        <div className="flex-1 min-w-0">
          <p className={`text-sm font-medium ${config.text}`}>{config.label}</p>

          {/* Resource details */}
          <div className="mt-2 space-y-1.5">
            {overage.resources.map((r) => (
              <div key={r.resource} className="flex items-center gap-2 text-xs">
                <span className={`font-medium ${config.text}`}>
                  {r.resource}:
                </span>
                <span className="text-text-secondary">
                  {Math.round(r.percent)}% used
                </span>
                <span className="text-text-dim">—</span>
                <span className="text-text-dim">{r.message}</span>
              </div>
            ))}
          </div>
        </div>

        {config.showUpgrade && (
          <Button
            variant="primary"
            size="sm"
            icon={<ArrowUp size={14} />}
            onClick={() => (window.location.href = '/billing')}
          >
            Upgrade
          </Button>
        )}
      </div>
    </div>
  )
})
