// ============================================================================
// Operator OS — Integration Store
// Zustand store for integration marketplace, connection, and status management.
// ============================================================================

import { create } from 'zustand'
import { api, ApiRequestError } from '../services/api'
import type {
  IntegrationSummary,
  IntegrationStatus,
  UserIntegration,
  ConnectResponse,
} from '../types/api'

interface IntegrationState {
  // Data
  integrations: IntegrationSummary[]
  categories: string[]
  statuses: IntegrationStatus[]
  userIntegrations: UserIntegration[]

  // UI
  selectedCategory: string | null
  searchQuery: string

  // Loading
  loadingIntegrations: boolean
  loadingCategories: boolean
  loadingStatuses: boolean
  loadingConnect: string | null // integration_id being connected
  loadingDisconnect: string | null

  // Errors
  integrationsError: string | null
  connectError: string | null
  statusError: string | null

  // Actions
  fetchIntegrations: () => Promise<void>
  fetchCategories: () => Promise<void>
  fetchStatuses: () => Promise<void>
  fetchAll: () => Promise<void>
  connect: (integrationId: string, config?: { apiKey?: string; scopes?: string[] }) => Promise<ConnectResponse | null>
  disconnect: (integrationId: string) => Promise<void>
  reconnect: (integrationId: string) => Promise<void>
  setCategory: (category: string | null) => void
  setSearch: (query: string) => void
  clearErrors: () => void

  // Derived
  filteredIntegrations: () => IntegrationSummary[]
  getStatus: (integrationId: string) => IntegrationStatus | undefined
  getUserIntegration: (integrationId: string) => UserIntegration | undefined
  isConnected: (integrationId: string) => boolean
}

function extractError(err: unknown): string {
  if (err instanceof ApiRequestError) return err.message
  if (err instanceof Error) return err.message
  return 'Something went wrong'
}

export const useIntegrationStore = create<IntegrationState>((set, get) => ({
  // Data
  integrations: [],
  categories: [],
  statuses: [],
  userIntegrations: [],

  // UI
  selectedCategory: null,
  searchQuery: '',

  // Loading
  loadingIntegrations: false,
  loadingCategories: false,
  loadingStatuses: false,
  loadingConnect: null,
  loadingDisconnect: null,

  // Errors
  integrationsError: null,
  connectError: null,
  statusError: null,

  // ─── Fetch Integrations ───
  fetchIntegrations: async () => {
    set({ loadingIntegrations: true, integrationsError: null })
    try {
      const integrations = await api.integrations.list()
      set({ integrations, loadingIntegrations: false })
    } catch (err) {
      set({ integrationsError: extractError(err), loadingIntegrations: false })
    }
  },

  // ─── Fetch Categories ───
  fetchCategories: async () => {
    set({ loadingCategories: true })
    try {
      const categories = await api.integrations.categories()
      set({ categories, loadingCategories: false })
    } catch {
      set({ loadingCategories: false })
    }
  },

  // ─── Fetch Statuses ───
  fetchStatuses: async () => {
    set({ loadingStatuses: true, statusError: null })
    try {
      const [statuses, userIntegrations] = await Promise.all([
        api.integrations.status(),
        api.userIntegrations.list(),
      ])
      set({ statuses, userIntegrations, loadingStatuses: false })
    } catch (err) {
      set({ statusError: extractError(err), loadingStatuses: false })
    }
  },

  // ─── Fetch All ───
  fetchAll: async () => {
    const state = get()
    await Promise.all([
      state.fetchIntegrations(),
      state.fetchCategories(),
      state.fetchStatuses(),
    ])
  },

  // ─── Connect ───
  connect: async (integrationId, config) => {
    set({ loadingConnect: integrationId, connectError: null })
    try {
      const response = await api.integrations.connect({
        integration_id: integrationId,
        api_key: config?.apiKey,
        scopes: config?.scopes,
        redirect_after: window.location.href,
      })

      // If OAuth, redirect to auth URL
      if (response.auth_url) {
        window.location.href = response.auth_url
        return response
      }

      // API key or no-auth — refresh statuses
      await get().fetchStatuses()
      set({ loadingConnect: null })
      return response
    } catch (err) {
      set({ connectError: extractError(err), loadingConnect: null })
      return null
    }
  },

  // ─── Disconnect ───
  disconnect: async (integrationId) => {
    set({ loadingDisconnect: integrationId, connectError: null })
    try {
      await api.integrations.disconnect({ integration_id: integrationId })
      await get().fetchStatuses()
      set({ loadingDisconnect: null })
    } catch (err) {
      set({ connectError: extractError(err), loadingDisconnect: null })
    }
  },

  // ─── Reconnect ───
  reconnect: async (integrationId) => {
    set({ loadingConnect: integrationId, connectError: null })
    try {
      await api.integrations.reconnect(integrationId)
      await get().fetchStatuses()
      set({ loadingConnect: null })
    } catch (err) {
      set({ connectError: extractError(err), loadingConnect: null })
    }
  },

  // ─── UI ───
  setCategory: (category) => set({ selectedCategory: category }),
  setSearch: (query) => set({ searchQuery: query }),
  clearErrors: () => set({ integrationsError: null, connectError: null, statusError: null }),

  // ─── Derived ───
  filteredIntegrations: () => {
    const { integrations, selectedCategory, searchQuery } = get()
    let filtered = integrations

    if (selectedCategory) {
      filtered = filtered.filter((i) => i.category === selectedCategory)
    }

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase()
      filtered = filtered.filter(
        (i) =>
          i.name.toLowerCase().includes(q) ||
          i.description.toLowerCase().includes(q) ||
          i.category.toLowerCase().includes(q),
      )
    }

    return filtered
  },

  getStatus: (integrationId) => {
    return get().statuses.find((s) => s.integration_id === integrationId)
  },

  getUserIntegration: (integrationId) => {
    return get().userIntegrations.find((ui) => ui.integration_id === integrationId)
  },

  isConnected: (integrationId) => {
    const ui = get().getUserIntegration(integrationId)
    return ui?.status === 'active'
  },
}))
