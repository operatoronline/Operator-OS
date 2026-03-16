// ============================================================================
// Operator OS — Tooltip
// Hover/focus-triggered tooltip. Accessible, keyboard-friendly, theme-aware.
// Uses OKLCH surface-3 background with glass shadow.
// ============================================================================

import { useState, useRef, useCallback, type ReactNode } from 'react'

type Placement = 'top' | 'bottom' | 'left' | 'right'

interface TooltipProps {
  /** Tooltip text content */
  content: string
  /** Trigger element (must accept onMouseEnter/Leave/Focus/Blur) */
  children: ReactNode
  /** Placement relative to trigger */
  placement?: Placement
  /** Delay before showing (ms) */
  delay?: number
  className?: string
}

const placementClasses: Record<Placement, string> = {
  top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
  bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
  left: 'right-full top-1/2 -translate-y-1/2 mr-2',
  right: 'left-full top-1/2 -translate-y-1/2 ml-2',
}

export function Tooltip({
  content,
  children,
  placement = 'top',
  delay = 200,
  className = '',
}: TooltipProps) {
  const [visible, setVisible] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const show = useCallback(() => {
    timerRef.current = setTimeout(() => setVisible(true), delay)
  }, [delay])

  const hide = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current)
    setVisible(false)
  }, [])

  return (
    <div
      className={`relative inline-flex ${className}`}
      onMouseEnter={show}
      onMouseLeave={hide}
      onFocus={show}
      onBlur={hide}
    >
      {children}
      {visible && (
        <div
          role="tooltip"
          className={`absolute z-50 px-2.5 py-1.5 rounded-lg
            bg-surface-3 text-text text-xs font-medium
            whitespace-nowrap pointer-events-none
            shadow-[0_2px_8px_var(--glass-shadow)]
            animate-fade-in
            ${placementClasses[placement]}`}
        >
          {content}
        </div>
      )}
    </div>
  )
}
