// ============================================================================
// Operator OS — Usage Store
// Zustand store for token usage, daily breakdown, model stats, limits & overage.
// ============================================================================

import { create } from 'zustand'
import { api, ApiRequestError } from '../services/api'
import type {
  UsageSummary,
  DailyUsage,
  ModelUsage,
  UsageLimits,
  OverageStatus,
} from '../types/api'

interface UsageState {
  // Data
  summary: UsageSummary | null
  daily: DailyUsage[]
  byModel: ModelUsage[]
  limits: UsageLimits | null
  overage: OverageStatus | null

  // Loading
  loadingSummary: boolean
  loadingDaily: boolean
  loadingModels: boolean
  loadingLimits: boolean
  loadingOverage: boolean

  // Errors
  summaryError: string | null
  dailyError: string | null
  modelsError: string | null
  limitsError: string | null
  overageError: string | null

  // Actions
  fetchSummary: () => Promise<void>
  fetchDaily: (start?: string, end?: string) => Promise<void>
  fetchByModel: () => Promise<void>
  fetchLimits: () => Promise<void>
  fetchOverage: () => Promise<void>
  fetchAll: () => Promise<void>
  clearErrors: () => void
}

function extractError(err: unknown): string {
  if (err instanceof ApiRequestError) return err.message
  if (err instanceof Error) return err.message
  return 'Something went wrong'
}

export const useUsageStore = create<UsageState>((set, get) => ({
  // Data
  summary: null,
  daily: [],
  byModel: [],
  limits: null,
  overage: null,

  // Loading
  loadingSummary: false,
  loadingDaily: false,
  loadingModels: false,
  loadingLimits: false,
  loadingOverage: false,

  // Errors
  summaryError: null,
  dailyError: null,
  modelsError: null,
  limitsError: null,
  overageError: null,

  // ─── Fetch Summary ───
  fetchSummary: async () => {
    set({ loadingSummary: true, summaryError: null })
    try {
      const summary = await api.usage.summary()
      set({ summary, loadingSummary: false })
    } catch (err) {
      set({ summaryError: extractError(err), loadingSummary: false })
    }
  },

  // ─── Fetch Daily ───
  fetchDaily: async (start?: string, end?: string) => {
    set({ loadingDaily: true, dailyError: null })
    try {
      const params: { start?: string; end?: string } = {}
      if (start) params.start = start
      if (end) params.end = end
      const daily = await api.usage.daily(params)
      set({ daily, loadingDaily: false })
    } catch (err) {
      set({ dailyError: extractError(err), loadingDaily: false })
    }
  },

  // ─── Fetch By Model ───
  fetchByModel: async () => {
    set({ loadingModels: true, modelsError: null })
    try {
      const byModel = await api.usage.byModel()
      set({ byModel, loadingModels: false })
    } catch (err) {
      set({ modelsError: extractError(err), loadingModels: false })
    }
  },

  // ─── Fetch Limits ───
  fetchLimits: async () => {
    set({ loadingLimits: true, limitsError: null })
    try {
      const limits = await api.usage.limits()
      set({ limits, loadingLimits: false })
    } catch (err) {
      set({ limitsError: extractError(err), loadingLimits: false })
    }
  },

  // ─── Fetch Overage ───
  fetchOverage: async () => {
    set({ loadingOverage: true, overageError: null })
    try {
      const overage = await api.usage.overage()
      set({ overage, loadingOverage: false })
    } catch (err) {
      set({ overageError: extractError(err), loadingOverage: false })
    }
  },

  // ─── Fetch All ───
  fetchAll: async () => {
    const { fetchSummary, fetchDaily, fetchByModel, fetchLimits, fetchOverage } = get()
    await Promise.allSettled([
      fetchSummary(),
      fetchDaily(),
      fetchByModel(),
      fetchLimits(),
      fetchOverage(),
    ])
  },

  clearErrors: () =>
    set({
      summaryError: null,
      dailyError: null,
      modelsError: null,
      limitsError: null,
      overageError: null,
    }),
}))
