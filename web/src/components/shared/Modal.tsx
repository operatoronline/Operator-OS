// ============================================================================
// Operator OS — Modal
// Dialog overlay with backdrop blur, ESC to close, focus trap basics.
// ============================================================================

import { useEffect, useRef, type ReactNode } from 'react'
import { X } from '@phosphor-icons/react'
import { useFocusTrap } from '../../hooks/useFocusTrap'

interface ModalProps {
  open: boolean
  onClose: () => void
  title: string
  children: ReactNode
  /** Maximum width class — defaults to max-w-lg */
  maxWidth?: string
}

export function Modal({ open, onClose, title, children, maxWidth = 'max-w-lg' }: ModalProps) {
  const overlayRef = useRef<HTMLDivElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)

  // True focus trap: Tab cycles within the modal, restores focus on close
  useFocusTrap(panelRef, open)

  // Close on Escape
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, onClose])

  // Lock body scroll
  useEffect(() => {
    if (!open) return
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = ''
    }
  }, [open])

  if (!open) return null

  return (
    <div
      ref={overlayRef}
      className="fixed inset-0 z-50 flex items-center justify-center p-4
        bg-black/50 backdrop-blur-sm animate-fade-in"
      onClick={(e) => {
        if (e.target === overlayRef.current) onClose()
      }}
    >
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        tabIndex={-1}
        className={`
          w-full ${maxWidth}
          bg-[var(--surface)] border border-[var(--border)]
          rounded-[var(--radius)] shadow-2xl
          animate-scale-in
          flex flex-col max-h-[85vh]
          outline-none
        `}
      >
        {/* ─── Header ─── */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-subtle)] shrink-0">
          <h2 className="text-base font-semibold text-[var(--text)]">{title}</h2>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-[var(--text-dim)] hover:text-[var(--text)]
              hover:bg-[var(--surface-2)] transition-colors cursor-pointer"
            aria-label="Close"
          >
            <X size={18} weight="bold" />
          </button>
        </div>

        {/* ─── Body ─── */}
        <div className="flex-1 overflow-y-auto px-6 py-4">
          {children}
        </div>
      </div>
    </div>
  )
}
