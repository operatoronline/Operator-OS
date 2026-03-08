// ============================================================================
// Operator OS — Audit Store
// Zustand store for audit log: filterable event log with pagination.
// ============================================================================

import { create } from 'zustand'
import { api, ApiRequestError } from '../services/api'
import type { AuditEvent } from '../types/api'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface AuditFilters {
  action: string   // '' = all
  userId: string   // '' = all
  resource: string // '' = all
  start: string    // ISO date or ''
  end: string      // ISO date or ''
  page: number
  perPage: number
}

interface AuditState {
  // Data
  events: AuditEvent[]
  totalCount: number | null

  // UI state
  filters: AuditFilters
  loading: boolean
  countLoading: boolean
  error: string | null
  forbidden: boolean

  // Actions
  fetchEvents: () => Promise<void>
  fetchCount: () => Promise<void>
  fetchAll: () => Promise<void>
  setFilter: (key: keyof Omit<AuditFilters, 'page' | 'perPage'>, value: string) => void
  setPage: (page: number) => void
  resetFilters: () => void
  clearError: () => void
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const DEFAULT_FILTERS: AuditFilters = {
  action: '',
  userId: '',
  resource: '',
  start: '',
  end: '',
  page: 1,
  perPage: 25,
}

function buildParams(filters: AuditFilters): Record<string, string | number> {
  const params: Record<string, string | number> = {
    page: filters.page,
    per_page: filters.perPage,
  }
  if (filters.action) params.action = filters.action
  if (filters.userId) params.user_id = filters.userId
  if (filters.resource) params.resource = filters.resource
  if (filters.start) params.start = filters.start
  if (filters.end) params.end = filters.end
  return params
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useAuditStore = create<AuditState>((set, get) => ({
  events: [],
  totalCount: null,
  filters: { ...DEFAULT_FILTERS },
  loading: false,
  countLoading: false,
  error: null,
  forbidden: false,

  fetchEvents: async () => {
    set({ loading: true, error: null })
    try {
      const { filters } = get()
      const events = await api.audit.events(buildParams(filters) as any)
      set({ events, loading: false, forbidden: false })
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 403) {
        set({ loading: false, forbidden: true, error: null })
      } else {
        set({
          loading: false,
          error: err instanceof Error ? err.message : 'Failed to load audit events',
        })
      }
    }
  },

  fetchCount: async () => {
    set({ countLoading: true })
    try {
      const { filters } = get()
      const params = buildParams(filters)
      delete params.page
      delete params.per_page
      const result = await api.audit.count(params as any)
      set({ totalCount: result.count, countLoading: false })
    } catch {
      set({ countLoading: false })
    }
  },

  fetchAll: async () => {
    const { fetchEvents, fetchCount } = get()
    await Promise.all([fetchEvents(), fetchCount()])
  },

  setFilter: (key, value) => {
    set((s) => ({
      filters: { ...s.filters, [key]: value, page: 1 },
    }))
    get().fetchAll()
  },

  setPage: (page) => {
    set((s) => ({ filters: { ...s.filters, page } }))
    get().fetchEvents()
  },

  resetFilters: () => {
    set({ filters: { ...DEFAULT_FILTERS } })
    get().fetchAll()
  },

  clearError: () => set({ error: null }),
}))
