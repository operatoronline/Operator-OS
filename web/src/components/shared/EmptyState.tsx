// ============================================================================
// Operator OS — Empty State
// Standardized empty state component for list views. Icon + title + description
// + optional CTA button. Consistent across all pages.
// ============================================================================

import { memo, type ReactNode } from 'react'
import type { Icon as PhosphorIcon } from '@phosphor-icons/react'
import { Button } from './Button'

interface EmptyStateProps {
  icon: PhosphorIcon
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
    icon?: PhosphorIcon
  }
  compact?: boolean // Smaller variant for inline panels
  className?: string
  children?: ReactNode
}

export const EmptyState = memo(function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  compact = false,
  className = '',
  children,
}: EmptyStateProps) {
  const iconSize = compact ? 28 : 36
  const iconBoxSize = compact ? 'w-12 h-12 rounded-xl' : 'w-16 h-16 rounded-2xl'

  return (
    <div
      className={`flex flex-col items-center justify-center text-center px-6
        ${compact ? 'py-8' : 'py-16'}
        ${className}`}
    >
      {/* Icon */}
      <div
        className={`${iconBoxSize} bg-[var(--accent-subtle)] flex items-center justify-center
          ${compact ? 'mb-3' : 'mb-4'}`}
      >
        <Icon
          size={iconSize}
          weight="thin"
          className="text-[var(--accent-text)]"
          aria-hidden="true"
        />
      </div>

      {/* Title */}
      <h3
        className={`font-semibold text-[var(--text)]
          ${compact ? 'text-sm mb-0.5' : 'text-base mb-1'}`}
      >
        {title}
      </h3>

      {/* Description */}
      {description && (
        <p
          className={`text-[var(--text-dim)] max-w-xs
            ${compact ? 'text-xs' : 'text-sm'}`}
        >
          {description}
        </p>
      )}

      {/* CTA */}
      {action && (
        <div className={compact ? 'mt-3' : 'mt-5'}>
          <Button
            variant="primary"
            size={compact ? 'sm' : 'md'}
            onClick={action.onClick}
          >
            {action.icon && <action.icon size={compact ? 14 : 16} weight="bold" />}
            {action.label}
          </Button>
        </div>
      )}

      {/* Custom children */}
      {children}
    </div>
  )
})
