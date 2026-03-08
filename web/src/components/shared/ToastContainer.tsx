// ============================================================================
// Operator OS — Toast Container
// Renders floating toast notifications. Positioned bottom-right (desktop) or
// top-center (mobile). Animated entry/exit with OKLCH-themed variants.
// ============================================================================

import { memo, useEffect, useState } from 'react'
import {
  CheckCircle,
  WarningCircle,
  Warning,
  Info,
  X,
} from '@phosphor-icons/react'
import { useToastStore, type Toast, type ToastVariant } from '../../stores/toastStore'

// ---------------------------------------------------------------------------
// Variant config
// ---------------------------------------------------------------------------

const VARIANT_CONFIG: Record<
  ToastVariant,
  { icon: typeof CheckCircle; bg: string; border: string; iconColor: string }
> = {
  success: {
    icon: CheckCircle,
    bg: 'bg-[oklch(0.25_0.04_145)]',
    border: 'border-[oklch(0.45_0.12_145)]',
    iconColor: 'text-[oklch(0.72_0.17_145)]',
  },
  error: {
    icon: WarningCircle,
    bg: 'bg-[oklch(0.25_0.04_25)]',
    border: 'border-[oklch(0.45_0.12_25)]',
    iconColor: 'text-[oklch(0.72_0.17_25)]',
  },
  warning: {
    icon: Warning,
    bg: 'bg-[oklch(0.28_0.04_80)]',
    border: 'border-[oklch(0.5_0.1_80)]',
    iconColor: 'text-[oklch(0.78_0.15_80)]',
  },
  info: {
    icon: Info,
    bg: 'bg-[oklch(0.25_0.04_250)]',
    border: 'border-[oklch(0.45_0.1_250)]',
    iconColor: 'text-[oklch(0.72_0.14_250)]',
  },
}

// Light theme overrides
const VARIANT_CONFIG_LIGHT: Record<
  ToastVariant,
  { bg: string; border: string; iconColor: string }
> = {
  success: {
    bg: 'bg-[oklch(0.97_0.01_145)]',
    border: 'border-[oklch(0.85_0.08_145)]',
    iconColor: 'text-[oklch(0.55_0.18_145)]',
  },
  error: {
    bg: 'bg-[oklch(0.97_0.01_25)]',
    border: 'border-[oklch(0.85_0.08_25)]',
    iconColor: 'text-[oklch(0.55_0.18_25)]',
  },
  warning: {
    bg: 'bg-[oklch(0.97_0.01_80)]',
    border: 'border-[oklch(0.85_0.08_80)]',
    iconColor: 'text-[oklch(0.6_0.15_80)]',
  },
  info: {
    bg: 'bg-[oklch(0.97_0.01_250)]',
    border: 'border-[oklch(0.85_0.08_250)]',
    iconColor: 'text-[oklch(0.5_0.15_250)]',
  },
}

// ---------------------------------------------------------------------------
// Single toast item
// ---------------------------------------------------------------------------

const ToastItem = memo(function ToastItem({ toast: t }: { toast: Toast }) {
  const dismiss = useToastStore((s) => s.dismiss)
  const [exiting, setExiting] = useState(false)
  const [visible, setVisible] = useState(false)

  // Entrance animation
  useEffect(() => {
    const raf = requestAnimationFrame(() => setVisible(true))
    return () => cancelAnimationFrame(raf)
  }, [])

  // Exit animation before dismiss
  const handleDismiss = () => {
    setExiting(true)
    setTimeout(() => dismiss(t.id), 200)
  }

  // Auto-dismiss: start exit slightly before store removes it
  useEffect(() => {
    if (t.duration && t.duration > 0) {
      const timer = setTimeout(() => setExiting(true), t.duration - 200)
      return () => clearTimeout(timer)
    }
  }, [t.duration])

  const dark = VARIANT_CONFIG[t.variant]
  const light = VARIANT_CONFIG_LIGHT[t.variant]
  const Icon = dark.icon

  return (
    <div
      role="alert"
      aria-live="assertive"
      className={`
        relative flex items-start gap-3 px-4 py-3 rounded-xl border shadow-lg
        max-w-sm w-full backdrop-blur-md
        transition-all duration-200 ease-out
        ${visible && !exiting ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'}
        dark:${dark.bg} dark:${dark.border}
        ${light.bg} ${light.border}
        dark:${dark.bg} dark:${dark.border}
      `}
      style={{
        // Inline fallback for dark/light since Tailwind dark: prefix may not work with class strategy
        // We use CSS custom properties to handle this more reliably
      }}
    >
      <Icon
        size={20}
        weight="fill"
        className={`shrink-0 mt-0.5 ${dark.iconColor}`}
        aria-hidden="true"
      />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-[var(--text)]">{t.title}</p>
        {t.message && (
          <p className="text-xs text-[var(--text-dim)] mt-0.5 line-clamp-2">{t.message}</p>
        )}
      </div>
      {t.dismissible && (
        <button
          onClick={handleDismiss}
          className="shrink-0 p-0.5 rounded-md text-[var(--text-dim)] hover:text-[var(--text)]
            hover:bg-[var(--surface-2)] transition-colors cursor-pointer"
          aria-label="Dismiss notification"
        >
          <X size={14} />
        </button>
      )}
    </div>
  )
})

// ---------------------------------------------------------------------------
// Container
// ---------------------------------------------------------------------------

export function ToastContainer() {
  const toasts = useToastStore((s) => s.toasts)

  if (toasts.length === 0) return null

  return (
    <div
      aria-label="Notifications"
      className="fixed z-[100] bottom-20 md:bottom-6 right-4 md:right-6
        flex flex-col-reverse gap-2 items-end
        pointer-events-none"
    >
      {toasts.slice(-5).map((t) => (
        <div key={t.id} className="pointer-events-auto">
          <ToastItem toast={t} />
        </div>
      ))}
    </div>
  )
}
