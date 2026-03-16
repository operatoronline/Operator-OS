// ============================================================================
// Operator OS — Dropdown Menu
// Reusable, accessible dropdown. Keyboard navigable (Arrow keys, Escape).
// Uses OKLCH token system. Anchors to a trigger element.
// ============================================================================

import {
  useState,
  useRef,
  useEffect,
  useCallback,
  type ReactNode,
  type KeyboardEvent,
} from 'react'

export interface DropdownItem {
  id: string
  label: string
  icon?: ReactNode
  danger?: boolean
  disabled?: boolean
  onClick: () => void
}

interface DropdownProps {
  /** Trigger element — rendered as-is, receives onClick */
  trigger: ReactNode
  /** Menu items */
  items: DropdownItem[]
  /** Alignment relative to trigger */
  align?: 'left' | 'right'
  /** Optional label for accessibility */
  label?: string
  className?: string
}

export function Dropdown({
  trigger,
  items,
  align = 'right',
  label = 'Menu',
  className = '',
}: DropdownProps) {
  const [open, setOpen] = useState(false)
  const [focusIndex, setFocusIndex] = useState(-1)
  const containerRef = useRef<HTMLDivElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  // Close on outside click
  useEffect(() => {
    if (!open) return
    const handleClick = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  // Close on Escape
  useEffect(() => {
    if (!open) return
    const handleKey = (e: globalThis.KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [open])

  // Focus management within menu
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const enabledItems = items.filter((i) => !i.disabled)
      if (!open || enabledItems.length === 0) return

      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setFocusIndex((prev) => (prev + 1) % enabledItems.length)
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setFocusIndex((prev) => (prev - 1 + enabledItems.length) % enabledItems.length)
      } else if (e.key === 'Enter' && focusIndex >= 0) {
        e.preventDefault()
        enabledItems[focusIndex].onClick()
        setOpen(false)
      }
    },
    [open, items, focusIndex],
  )

  // Focus the active item
  useEffect(() => {
    if (!open || focusIndex < 0) return
    const buttons = menuRef.current?.querySelectorAll('[role="menuitem"]:not([aria-disabled="true"])')
    if (buttons?.[focusIndex]) {
      ;(buttons[focusIndex] as HTMLElement).focus()
    }
  }, [focusIndex, open])

  const toggleMenu = useCallback(() => {
    setOpen((prev) => {
      if (!prev) setFocusIndex(-1)
      return !prev
    })
  }, [])

  return (
    <div ref={containerRef} className={`relative inline-flex ${className}`} onKeyDown={handleKeyDown}>
      {/* Trigger */}
      <div onClick={toggleMenu} aria-haspopup="true" aria-expanded={open}>
        {trigger}
      </div>

      {/* Menu */}
      {open && (
        <div
          ref={menuRef}
          role="menu"
          aria-label={label}
          className={`absolute top-full mt-1.5 min-w-[160px]
            bg-surface border border-border rounded-xl
            shadow-[0_8px_32px_var(--glass-shadow)]
            overflow-hidden animate-fade-slide z-50
            ${align === 'right' ? 'right-0' : 'left-0'}`}
        >
          <div className="py-1">
            {items.map((item) => (
              <button
                key={item.id}
                role="menuitem"
                aria-disabled={item.disabled || undefined}
                disabled={item.disabled}
                onClick={() => {
                  if (item.disabled) return
                  item.onClick()
                  setOpen(false)
                }}
                className={`flex items-center gap-2.5 w-full px-3.5 py-2
                  text-[13px] font-medium transition-colors duration-150 focus-ring
                  ${item.disabled
                    ? 'text-text-dim opacity-50 cursor-not-allowed'
                    : item.danger
                      ? 'text-text-dim hover:text-error hover:bg-error-subtle/50'
                      : 'text-text-secondary hover:text-text hover:bg-surface-2/50'
                  }`}
              >
                {item.icon && <span className="shrink-0" aria-hidden="true">{item.icon}</span>}
                {item.label}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
