// ============================================================================
// Operator OS — Admin Store
// Zustand store for admin panel: user management + platform stats.
// ============================================================================

import { create } from 'zustand'
import { api, ApiRequestError } from '../services/api'
import type { AdminUser, PlatformStats, SetRoleRequest } from '../types/api'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface AdminFilters {
  search: string
  status: string // '' = all
  page: number
  perPage: number
}

interface AdminState {
  // Data
  users: AdminUser[]
  stats: PlatformStats | null

  // UI state
  filters: AdminFilters
  loadingUsers: boolean
  loadingStats: boolean
  actionLoading: string | null // user ID being acted on
  usersError: string | null
  statsError: string | null
  actionError: string | null
  forbidden: boolean // true if user is not admin

  // Actions
  fetchUsers: () => Promise<void>
  fetchStats: () => Promise<void>
  fetchAll: () => Promise<void>
  setSearch: (search: string) => void
  setStatusFilter: (status: string) => void
  setPage: (page: number) => void
  suspendUser: (id: string) => Promise<boolean>
  activateUser: (id: string) => Promise<boolean>
  setRole: (id: string, role: 'user' | 'admin') => Promise<boolean>
  deleteUser: (id: string) => Promise<boolean>
  clearErrors: () => void
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useAdminStore = create<AdminState>((set, get) => ({
  users: [],
  stats: null,
  filters: { search: '', status: '', page: 1, perPage: 20 },
  loadingUsers: false,
  loadingStats: false,
  actionLoading: null,
  usersError: null,
  statsError: null,
  actionError: null,
  forbidden: false,

  fetchUsers: async () => {
    set({ loadingUsers: true, usersError: null })
    try {
      const { filters } = get()
      const params: Record<string, string | number> = {
        page: filters.page,
        per_page: filters.perPage,
      }
      if (filters.search) params.search = filters.search
      if (filters.status) params.status = filters.status

      const users = await api.admin.users(params as any)
      set({ users, loadingUsers: false, forbidden: false })
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 403) {
        set({ loadingUsers: false, forbidden: true, usersError: null })
      } else {
        set({
          loadingUsers: false,
          usersError: err instanceof Error ? err.message : 'Failed to load users',
        })
      }
    }
  },

  fetchStats: async () => {
    set({ loadingStats: true, statsError: null })
    try {
      const stats = await api.admin.stats()
      set({ stats, loadingStats: false })
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 403) {
        set({ loadingStats: false, forbidden: true })
      } else {
        set({
          loadingStats: false,
          statsError: err instanceof Error ? err.message : 'Failed to load stats',
        })
      }
    }
  },

  fetchAll: async () => {
    const { fetchUsers, fetchStats } = get()
    await Promise.all([fetchUsers(), fetchStats()])
  },

  setSearch: (search: string) => {
    set((s) => ({ filters: { ...s.filters, search, page: 1 } }))
    // Debounced fetch is handled by the component
  },

  setStatusFilter: (status: string) => {
    set((s) => ({ filters: { ...s.filters, status, page: 1 } }))
    get().fetchUsers()
  },

  setPage: (page: number) => {
    set((s) => ({ filters: { ...s.filters, page } }))
    get().fetchUsers()
  },

  suspendUser: async (id: string) => {
    set({ actionLoading: id, actionError: null })
    try {
      await api.admin.suspendUser(id)
      // Update local state
      set((s) => ({
        users: s.users.map((u) => (u.id === id ? { ...u, status: 'suspended' } : u)),
        actionLoading: null,
        stats: s.stats
          ? {
              ...s.stats,
              active_users: s.stats.active_users - 1,
              suspended_users: s.stats.suspended_users + 1,
            }
          : null,
      }))
      return true
    } catch (err) {
      set({
        actionLoading: null,
        actionError: err instanceof Error ? err.message : 'Failed to suspend user',
      })
      return false
    }
  },

  activateUser: async (id: string) => {
    set({ actionLoading: id, actionError: null })
    try {
      await api.admin.activateUser(id)
      set((s) => ({
        users: s.users.map((u) => (u.id === id ? { ...u, status: 'active' } : u)),
        actionLoading: null,
        stats: s.stats
          ? {
              ...s.stats,
              active_users: s.stats.active_users + 1,
              suspended_users: Math.max(0, s.stats.suspended_users - 1),
            }
          : null,
      }))
      return true
    } catch (err) {
      set({
        actionLoading: null,
        actionError: err instanceof Error ? err.message : 'Failed to activate user',
      })
      return false
    }
  },

  setRole: async (id: string, role: 'user' | 'admin') => {
    set({ actionLoading: id, actionError: null })
    try {
      const data: SetRoleRequest = { role }
      await api.admin.setRole(id, data)
      set((s) => ({
        users: s.users.map((u) => (u.id === id ? { ...u, role } : u)),
        actionLoading: null,
      }))
      return true
    } catch (err) {
      set({
        actionLoading: null,
        actionError: err instanceof Error ? err.message : 'Failed to set role',
      })
      return false
    }
  },

  deleteUser: async (id: string) => {
    set({ actionLoading: id, actionError: null })
    try {
      await api.admin.deleteUser(id)
      set((s) => ({
        users: s.users.filter((u) => u.id !== id),
        actionLoading: null,
        stats: s.stats
          ? { ...s.stats, total_users: s.stats.total_users - 1 }
          : null,
      }))
      return true
    } catch (err) {
      set({
        actionLoading: null,
        actionError: err instanceof Error ? err.message : 'Failed to delete user',
      })
      return false
    }
  },

  clearErrors: () => set({ usersError: null, statsError: null, actionError: null }),
}))
