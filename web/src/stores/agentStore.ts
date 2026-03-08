// ============================================================================
// Operator OS — Agent Store
// Zustand store for agent CRUD, selection, and default management.
// ============================================================================

import { create } from 'zustand'
import { api, ApiRequestError } from '../services/api'
import type { Agent, CreateAgentRequest, UpdateAgentRequest } from '../types/api'

interface AgentState {
  // Data
  agents: Agent[]
  selectedAgentId: string | null
  loading: boolean
  error: string | null

  // Derived
  defaultAgent: Agent | undefined
  selectedAgent: Agent | undefined

  // Actions
  fetchAgents: () => Promise<void>
  createAgent: (data: CreateAgentRequest) => Promise<Agent>
  updateAgent: (id: string, data: UpdateAgentRequest) => Promise<Agent>
  deleteAgent: (id: string) => Promise<void>
  setDefault: (id: string) => Promise<void>
  selectAgent: (id: string | null) => void
  clearError: () => void
}

export const useAgentStore = create<AgentState>((set, get) => ({
  agents: [],
  selectedAgentId: null,
  loading: false,
  error: null,

  get defaultAgent() {
    return get().agents.find((a) => a.is_default)
  },

  get selectedAgent() {
    const { agents, selectedAgentId } = get()
    return agents.find((a) => a.id === selectedAgentId)
  },

  fetchAgents: async () => {
    set({ loading: true, error: null })
    try {
      const agents = await api.agents.list()
      set({ agents, loading: false })
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to load agents'
      set({ error: msg, loading: false })
    }
  },

  createAgent: async (data) => {
    set({ error: null })
    try {
      const agent = await api.agents.create(data)
      set((s) => ({ agents: [...s.agents, agent] }))
      return agent
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to create agent'
      set({ error: msg })
      throw err
    }
  },

  updateAgent: async (id, data) => {
    set({ error: null })
    try {
      const updated = await api.agents.update(id, data)
      set((s) => ({
        agents: s.agents.map((a) => (a.id === id ? updated : a)),
      }))
      return updated
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to update agent'
      set({ error: msg })
      throw err
    }
  },

  deleteAgent: async (id) => {
    set({ error: null })
    try {
      await api.agents.delete(id)
      set((s) => ({
        agents: s.agents.filter((a) => a.id !== id),
        selectedAgentId: s.selectedAgentId === id ? null : s.selectedAgentId,
      }))
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to delete agent'
      set({ error: msg })
      throw err
    }
  },

  setDefault: async (id) => {
    set({ error: null })
    try {
      await api.agents.setDefault(id)
      set((s) => ({
        agents: s.agents.map((a) => ({
          ...a,
          is_default: a.id === id,
        })),
      }))
    } catch (err) {
      const msg = err instanceof ApiRequestError ? err.message : 'Failed to set default agent'
      set({ error: msg })
      throw err
    }
  },

  selectAgent: (id) => set({ selectedAgentId: id }),

  clearError: () => set({ error: null }),
}))
