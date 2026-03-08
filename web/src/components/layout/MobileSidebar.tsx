// ============================================================================
// Operator OS — Mobile Sidebar Overlay
// Slide-over navigation panel for mobile devices with backdrop.
// ============================================================================

import { useEffect, useRef } from 'react'
import { NavLink } from 'react-router-dom'
import {
  ChatCircle,
  Robot,
  Plugs,
  CreditCard,
  Gear,
  ShieldCheck,
  X,
} from '@phosphor-icons/react'
import { useFocusTrap } from '../../hooks/useFocusTrap'

const navItems = [
  { to: '/chat', label: 'Chat', icon: ChatCircle },
  { to: '/agents', label: 'Agents', icon: Robot },
  { to: '/integrations', label: 'Integrations', icon: Plugs },
  { to: '/billing', label: 'Billing', icon: CreditCard },
  { to: '/settings', label: 'Settings', icon: Gear },
  { to: '/admin', label: 'Admin', icon: ShieldCheck },
]

interface MobileSidebarProps {
  open: boolean
  onClose: () => void
}

export function MobileSidebar({ open, onClose }: MobileSidebarProps) {
  const panelRef = useRef<HTMLDivElement>(null)

  // Focus trap: Tab cycles within the mobile sidebar when open
  useFocusTrap(panelRef, open)

  // Lock body scroll when open
  useEffect(() => {
    if (open) {
      document.body.style.overflow = 'hidden'
      return () => {
        document.body.style.overflow = ''
      }
    }
  }, [open])

  // Close on Escape
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, onClose])

  return (
    <>
      {/* Backdrop */}
      <div
        className={`md:hidden fixed inset-0 z-90 bg-[oklch(0_0_0/0.5)] backdrop-blur-sm transition-opacity duration-200 ${
          open ? 'opacity-100' : 'opacity-0 pointer-events-none'
        }`}
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Panel */}
      <div
        ref={panelRef}
        className={`md:hidden fixed top-0 left-0 bottom-0 z-100 w-64 bg-surface border-r border-border shadow-[4px_0_24px_var(--glass-shadow)] transition-transform duration-200 ease-out ${
          open ? 'translate-x-0' : '-translate-x-full'
        }`}
        style={{ paddingTop: 'var(--safe-t)', paddingLeft: 'var(--safe-l)' }}
        role="dialog"
        aria-modal="true"
        aria-label="Navigation menu"
      >
        {/* Header */}
        <div className="flex items-center justify-between h-14 px-4 border-b border-border-subtle">
          <div className="flex items-center gap-3">
            <div className="w-7 h-7 rounded-lg bg-accent flex items-center justify-center">
              <span className="text-white text-xs font-bold leading-none">OS</span>
            </div>
            <span className="text-sm font-semibold text-text">Operator OS</span>
          </div>
          <button
            onClick={onClose}
            className="flex items-center justify-center w-8 h-8 rounded-lg text-text-dim hover:text-text-secondary hover:bg-surface-2/50 transition-colors"
            aria-label="Close menu"
          >
            <X size={18} />
          </button>
        </div>

        {/* Nav items */}
        <nav className="flex flex-col gap-0.5 px-2 py-3">
          {navItems.map((item) => {
            const Icon = item.icon
            return (
              <NavLink
                key={item.to}
                to={item.to}
                onClick={onClose}
                className={({ isActive }) =>
                  `flex items-center gap-3 px-3 py-3 rounded-lg text-[14px] font-medium
                   min-h-[44px] select-none active:scale-[0.98] active:opacity-80
                   transition-all duration-150 focus-ring ${
                    isActive
                      ? 'bg-surface-2 text-text shadow-[inset_0_0_0_1px_var(--border)]'
                      : 'text-text-dim hover:text-text-secondary hover:bg-surface-2/50'
                  }`
                }
              >
                {({ isActive }) => (
                  <>
                    <Icon size={20} weight={isActive ? 'fill' : 'regular'} />
                    {item.label}
                  </>
                )}
              </NavLink>
            )
          })}
        </nav>
      </div>
    </>
  )
}
