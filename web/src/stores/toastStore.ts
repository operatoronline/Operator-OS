// ============================================================================
// Operator OS — Toast Store
// Zustand store for toast notifications. Supports success, error, warning, info.
// ============================================================================

import { create } from 'zustand'

export type ToastVariant = 'success' | 'error' | 'warning' | 'info'

export interface Toast {
  id: string
  variant: ToastVariant
  title: string
  message?: string
  duration?: number // ms, 0 = persistent
  dismissible?: boolean
}

interface ToastState {
  toasts: Toast[]
  add: (toast: Omit<Toast, 'id'>) => string
  dismiss: (id: string) => void
  clear: () => void
}

let counter = 0

const DEFAULT_DURATIONS: Record<ToastVariant, number> = {
  success: 4000,
  info: 5000,
  warning: 6000,
  error: 8000,
}

export const useToastStore = create<ToastState>((set, get) => ({
  toasts: [],

  add: (toast) => {
    const id = `toast-${++counter}-${Date.now()}`
    const duration = toast.duration ?? DEFAULT_DURATIONS[toast.variant]
    const entry: Toast = { ...toast, id, duration, dismissible: toast.dismissible ?? true }

    set((s) => ({ toasts: [...s.toasts, entry] }))

    // Auto-dismiss after duration (unless persistent)
    if (duration > 0) {
      setTimeout(() => {
        get().dismiss(id)
      }, duration)
    }

    return id
  },

  dismiss: (id) => {
    set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) }))
  },

  clear: () => set({ toasts: [] }),
}))

// ---------------------------------------------------------------------------
// Convenience helpers (importable without hooks)
// ---------------------------------------------------------------------------

export const toast = {
  success: (title: string, message?: string) =>
    useToastStore.getState().add({ variant: 'success', title, message }),
  error: (title: string, message?: string) =>
    useToastStore.getState().add({ variant: 'error', title, message }),
  warning: (title: string, message?: string) =>
    useToastStore.getState().add({ variant: 'warning', title, message }),
  info: (title: string, message?: string) =>
    useToastStore.getState().add({ variant: 'info', title, message }),
}
