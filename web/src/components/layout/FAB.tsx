// ============================================================================
// Operator OS — Floating Action Button (FAB)
// Quick action button for starting new chats or creating agents.
// Positioned above bottom tabs on mobile, bottom-right on desktop.
// Glass morphism treatment, 44px minimum touch target.
// ============================================================================

import { useState, useRef, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  Plus,
  ChatCircle,
  Robot,
  X,
} from '@phosphor-icons/react'

const actions = [
  { id: 'chat', label: 'New Chat', icon: ChatCircle, to: '/chat' },
  { id: 'agent', label: 'New Agent', icon: Robot, to: '/agents' },
] as const

export function FAB() {
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()
  const location = useLocation()

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
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [open])

  // Close on route change
  useEffect(() => {
    setOpen(false)
  }, [location.pathname])

  return (
    <div
      ref={containerRef}
      className="fixed z-80 right-4 bottom-[calc(var(--bottom-tabs-h)+12px)] md:bottom-6 md:right-6"
    >
      {/* Speed dial actions */}
      {open && (
        <div className="absolute bottom-full right-0 mb-2 flex flex-col gap-2 items-end">
          {actions.map((action, i) => {
            const Icon = action.icon
            return (
              <button
                key={action.id}
                onClick={() => {
                  setOpen(false)
                  navigate(action.to, { state: { action: 'create' } })
                }}
                className="flex items-center gap-2.5 px-3.5 py-2.5 rounded-xl
                  bg-surface border border-border
                  shadow-[0_4px_16px_var(--glass-shadow)]
                  text-[13px] font-medium text-text
                  hover:bg-surface-2 active:scale-[0.97]
                  transition-all duration-150 focus-ring
                  animate-fade-slide"
                style={{ animationDelay: `${i * 50}ms` }}
                aria-label={action.label}
              >
                <Icon size={18} weight="regular" aria-hidden="true" />
                <span className="whitespace-nowrap">{action.label}</span>
              </button>
            )
          })}
        </div>
      )}

      {/* Main FAB button */}
      <button
        onClick={() => setOpen(!open)}
        className={`
          flex items-center justify-center
          w-12 h-12 rounded-2xl
          bg-accent text-white
          shadow-[0_4px_16px_var(--glass-shadow)]
          hover:opacity-90 active:scale-[0.95]
          transition-all duration-200 focus-ring
          ${open ? 'rotate-45' : ''}
        `}
        aria-label={open ? 'Close quick actions' : 'Open quick actions'}
        aria-expanded={open}
        aria-haspopup="true"
      >
        {open ? <X size={22} weight="bold" /> : <Plus size={22} weight="bold" />}
      </button>
    </div>
  )
}
