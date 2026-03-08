// ============================================================================
// Operator OS — PlanCard
// Individual plan card with features, pricing, and action button.
// ============================================================================

import { memo } from 'react'
import { Check, Star, Lightning, Crown, Rocket } from '@phosphor-icons/react'
import { Button } from '../shared'
import type { Plan, PlanId, BillingInterval } from '../../types/api'

interface PlanCardProps {
  plan: Plan
  currentPlanId: PlanId
  interval: BillingInterval
  onSelect: (planId: string) => void
  loading?: boolean
  recommended?: boolean
}

const planIcons: Record<string, React.ReactNode> = {
  free: <Star size={24} weight="duotone" className="text-text-secondary" />,
  starter: <Lightning size={24} weight="duotone" className="text-accent-text" />,
  pro: <Crown size={24} weight="duotone" className="text-warning" />,
  enterprise: <Rocket size={24} weight="duotone" className="text-success" />,
}

const planColors: Record<string, string> = {
  free: 'border-border',
  starter: 'border-accent/40',
  pro: 'border-warning/40',
  enterprise: 'border-success/40',
}

function formatPrice(cents: number): string {
  if (cents === 0) return 'Free'
  return `$${(cents / 100).toFixed(cents % 100 === 0 ? 0 : 2)}`
}

function formatLimit(value: number, suffix: string): string {
  if (value <= 0) return 'Unlimited'
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(value % 1_000_000 === 0 ? 0 : 1)}M ${suffix}`
  if (value >= 1_000) return `${(value / 1_000).toFixed(value % 1_000 === 0 ? 0 : 1)}K ${suffix}`
  return `${value} ${suffix}`
}

function PlanCardRaw({ plan, currentPlanId, interval, onSelect, loading, recommended }: PlanCardProps) {
  const isCurrent = plan.id === currentPlanId
  const price = interval === 'yearly' ? plan.price_yearly_cents : plan.price_monthly_cents
  const monthlyPrice = interval === 'yearly' ? Math.round(plan.price_yearly_cents / 12) : plan.price_monthly_cents
  const yearlyMonthly = Math.round(plan.price_yearly_cents / 12)
  const monthlyCost = plan.price_monthly_cents
  const savings = interval === 'yearly' && monthlyCost > 0
    ? Math.round(((monthlyCost - yearlyMonthly) / monthlyCost) * 100)
    : 0

  const isEnterprise = plan.id === 'enterprise'
  const isFree = plan.id === 'free'

  const features = [
    { label: `${plan.limits.max_agents} agent${plan.limits.max_agents !== 1 ? 's' : ''}`, included: true },
    { label: formatLimit(plan.limits.max_messages_per_month, 'messages/mo'), included: true },
    { label: formatLimit(plan.limits.max_tokens_per_month, 'tokens/mo'), included: true },
    { label: `${plan.limits.max_integrations} integration${plan.limits.max_integrations !== 1 ? 's' : ''}`, included: plan.limits.max_integrations > 0 },
    { label: `${plan.limits.max_storage_mb}MB storage`, included: plan.limits.max_storage_mb > 0 },
    { label: 'Custom skills', included: plan.limits.custom_skills },
    { label: 'API access', included: plan.limits.api_access },
    { label: `${plan.limits.max_team_members} team member${plan.limits.max_team_members !== 1 ? 's' : ''}`, included: plan.limits.max_team_members > 1 },
  ]

  // Determine button state
  let buttonText = 'Get Started'
  let buttonVariant: 'primary' | 'secondary' | 'ghost' = 'primary'
  let buttonDisabled = false

  if (isCurrent) {
    buttonText = 'Current Plan'
    buttonVariant = 'secondary'
    buttonDisabled = true
  } else if (isEnterprise) {
    buttonText = 'Contact Sales'
    buttonVariant = 'secondary'
  } else if (isFree && currentPlanId !== 'free') {
    buttonText = 'Downgrade'
    buttonVariant = 'ghost'
  }

  return (
    <div
      className={`
        relative flex flex-col
        bg-[var(--surface)] border-2 ${isCurrent ? 'border-accent' : recommended ? planColors[plan.id] || 'border-border' : 'border-border-subtle'}
        rounded-[var(--radius)] p-6
        transition-all duration-200
        ${!isCurrent ? 'hover:border-accent/60 hover:shadow-[0_4px_24px_var(--glass-shadow)]' : ''}
      `}
    >
      {/* Recommended badge */}
      {recommended && !isCurrent && (
        <div className="absolute -top-3 left-1/2 -translate-x-1/2">
          <span className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-[11px] font-semibold tracking-wide bg-accent text-white shadow-md">
            <Star size={12} weight="fill" />
            RECOMMENDED
          </span>
        </div>
      )}

      {/* Current badge */}
      {isCurrent && (
        <div className="absolute -top-3 left-1/2 -translate-x-1/2">
          <span className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-[11px] font-semibold tracking-wide bg-success text-white shadow-md">
            <Check size={12} weight="bold" />
            CURRENT
          </span>
        </div>
      )}

      {/* Header */}
      <div className="flex items-center gap-3 mb-4 mt-1">
        {planIcons[plan.id]}
        <div>
          <h3 className="text-lg font-semibold text-text">{plan.name}</h3>
        </div>
      </div>

      {/* Pricing */}
      <div className="mb-6">
        <div className="flex items-baseline gap-1">
          <span className="text-3xl font-bold text-text">
            {formatPrice(monthlyPrice)}
          </span>
          {!isFree && (
            <span className="text-sm text-text-dim">/month</span>
          )}
        </div>
        {interval === 'yearly' && !isFree && (
          <div className="flex items-center gap-2 mt-1">
            <span className="text-xs text-text-dim">
              {formatPrice(price)}/year
            </span>
            {savings > 0 && (
              <span className="text-[11px] font-semibold text-success bg-success-subtle px-1.5 py-0.5 rounded-full">
                Save {savings}%
              </span>
            )}
          </div>
        )}
      </div>

      {/* Features */}
      <ul className="flex-1 space-y-2.5 mb-6">
        {features.map((f) => (
          <li key={f.label} className="flex items-start gap-2.5">
            <Check
              size={16}
              weight="bold"
              className={`shrink-0 mt-0.5 ${f.included ? 'text-success' : 'text-text-dim/30'}`}
            />
            <span className={`text-sm ${f.included ? 'text-text-secondary' : 'text-text-dim/40 line-through'}`}>
              {f.label}
            </span>
          </li>
        ))}
      </ul>

      {/* Action */}
      <Button
        variant={buttonVariant}
        size="lg"
        disabled={buttonDisabled || loading}
        loading={loading}
        onClick={() => onSelect(plan.id)}
        className="w-full"
      >
        {buttonText}
      </Button>
    </div>
  )
}

export const PlanCard = memo(PlanCardRaw)
