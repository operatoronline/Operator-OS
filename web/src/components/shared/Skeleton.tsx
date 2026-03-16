// ============================================================================
// Operator OS — Skeleton
// Reusable loading placeholder with shimmer animation.
// Uses OKLCH token system for theme-aware skeleton rendering.
// ============================================================================

import { memo } from 'react'

interface SkeletonProps {
  /** Width — any CSS value or Tailwind class */
  width?: string
  /** Height — any CSS value or Tailwind class */
  height?: string
  /** Makes the skeleton a circle */
  circle?: boolean
  /** Number of repeated skeleton lines */
  count?: number
  /** Gap between repeated items */
  gap?: string
  className?: string
}

export const Skeleton = memo(function Skeleton({
  width,
  height = '1rem',
  circle = false,
  count = 1,
  gap = '0.5rem',
  className = '',
}: SkeletonProps) {
  const baseClass = `
    bg-surface-2 rounded-[var(--radius-sm)]
    animate-[shimmer_1.5s_ease-in-out_infinite]
    ${circle ? '!rounded-full' : ''}
    ${className}
  `.trim()

  if (count === 1) {
    return (
      <div
        className={baseClass}
        style={{ width, height: circle ? width : height }}
        role="status"
        aria-label="Loading"
      />
    )
  }

  return (
    <div className="flex flex-col" style={{ gap }} role="status" aria-label="Loading">
      {Array.from({ length: count }, (_, i) => (
        <div
          key={i}
          className={baseClass}
          style={{
            width: i === count - 1 ? '60%' : width,
            height,
          }}
        />
      ))}
    </div>
  )
})
