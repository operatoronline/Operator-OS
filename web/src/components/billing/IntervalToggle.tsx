// ============================================================================
// Operator OS — IntervalToggle
// Monthly / Yearly billing interval toggle with savings indicator.
// ============================================================================

import type { BillingInterval } from '../../types/api'

interface IntervalToggleProps {
  value: BillingInterval
  onChange: (interval: BillingInterval) => void
}

export function IntervalToggle({ value, onChange }: IntervalToggleProps) {
  return (
    <div className="inline-flex items-center bg-surface-2 rounded-full p-1 border border-border-subtle">
      <button
        onClick={() => onChange('monthly')}
        className={`
          px-4 py-1.5 rounded-full text-sm font-medium transition-all duration-200 cursor-pointer
          ${value === 'monthly'
            ? 'bg-[var(--surface)] text-text shadow-sm'
            : 'text-text-dim hover:text-text-secondary'}
        `}
      >
        Monthly
      </button>
      <button
        onClick={() => onChange('yearly')}
        className={`
          px-4 py-1.5 rounded-full text-sm font-medium transition-all duration-200 cursor-pointer
          flex items-center gap-1.5
          ${value === 'yearly'
            ? 'bg-[var(--surface)] text-text shadow-sm'
            : 'text-text-dim hover:text-text-secondary'}
        `}
      >
        Yearly
        <span className="text-[10px] font-semibold text-success bg-success-subtle px-1.5 py-0.5 rounded-full">
          Save 20%
        </span>
      </button>
    </div>
  )
}
