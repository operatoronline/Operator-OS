// ============================================================================
// Operator OS — Session Store
// Zustand store for chat session CRUD, selection, and history management.
// ============================================================================

import { create } from 'zustand'
import { api, ApiRequestError } from '../services/api'
import type { Session, CreateSessionRequest, UpdateSessionRequest } from '../types/api'

interface SessionState {
  // Data
  sessions: Session[]
  activeSessionId: string | null
  loading: boolean
  error: string | null
  renaming: string | null // session id being renamed

  // Derived
  activeSession: Session | undefined
  pinnedSessions: Session[]
  unpinnedSessions: Session[]

  // Actions
  fetchSessions: () => Promise<void>
  createSession: (data?: CreateSessionRequest) => Promise<Session>
  updateSession: (id: string, data: UpdateSessionRequest) => Promise<Session>
  deleteSession: (id: string) => Promise<void>
  selectSession: (id: string | null) => void
  setRenaming: (id: string | null) => void
  clearError: () => void

  // Helpers
  getSession: (id: string) => Session | undefined
}

// Sort sessions: pinned first, then by last_message_at (newest first)
function sortSessions(sessions: Session[]): Session[] {
  return [...sessions].sort((a, b) => {
    if (a.pinned && !b.pinned) return -1
    if (!a.pinned && b.pinned) return 1
    const aTime = a.last_message_at || a.updated_at || a.created_at
    const bTime = b.last_message_at || b.updated_at || b.created_at
    return new Date(bTime).getTime() - new Date(aTime).getTime()
  })
}

export const useSessionStore = create<SessionState>((set, get) => ({
  sessions: [],
  activeSessionId: null,
  loading: false,
  error: null,
  renaming: null,

  get activeSession() {
    const { sessions, activeSessionId } = get()
    return sessions.find((s) => s.id === activeSessionId)
  },

  get pinnedSessions() {
    return get().sessions.filter((s) => s.pinned && !s.archived)
  },

  get unpinnedSessions() {
    return get().sessions.filter((s) => !s.pinned && !s.archived)
  },

  fetchSessions: async () => {
    set({ loading: true, error: null })
    try {
      const sessions = await api.sessions.list()
      set({ sessions: sortSessions(sessions), loading: false })
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to load sessions'
      set({ error: msg, loading: false })
    }
  },

  createSession: async (data) => {
    set({ error: null })
    try {
      const session = await api.sessions.create(data || {})
      set((s) => ({
        sessions: sortSessions([session, ...s.sessions]),
        activeSessionId: session.id,
      }))
      return session
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to create session'
      set({ error: msg })
      throw err
    }
  },

  updateSession: async (id, data) => {
    set({ error: null })
    try {
      const updated = await api.sessions.update(id, data)
      set((s) => ({
        sessions: sortSessions(s.sessions.map((sess) => (sess.id === id ? updated : sess))),
        renaming: s.renaming === id ? null : s.renaming,
      }))
      return updated
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to update session'
      set({ error: msg })
      throw err
    }
  },

  deleteSession: async (id) => {
    set({ error: null })
    try {
      await api.sessions.delete(id)
      set((s) => {
        const filtered = s.sessions.filter((sess) => sess.id !== id)
        return {
          sessions: filtered,
          activeSessionId: s.activeSessionId === id ? null : s.activeSessionId,
        }
      })
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to delete session'
      set({ error: msg })
      throw err
    }
  },

  selectSession: (id) => {
    set({ activeSessionId: id })
  },

  setRenaming: (id) => set({ renaming: id }),

  clearError: () => set({ error: null }),

  getSession: (id) => get().sessions.find((s) => s.id === id),
}))
