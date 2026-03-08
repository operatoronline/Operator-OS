// ============================================================================
// Operator OS — Security Audit Store
// Zustand store for security audit dashboard: run audits, filter results.
// ============================================================================

import { create } from 'zustand'
import { api, ApiRequestError } from '../services/api'
import type {
  SecurityAuditReport,
  SecurityFinding,
  AuditCategory,
  AuditSeverity,
} from '../types/api'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface SecurityAuditState {
  // Data
  report: SecurityAuditReport | null
  lastRunAt: string | null

  // UI state
  loading: boolean
  error: string | null
  forbidden: boolean

  // Filters
  severityFilter: AuditSeverity | 'all'
  categoryFilter: AuditCategory | 'all'
  statusFilter: 'all' | 'passed' | 'failed'
  expandedFindingId: string | null

  // Computed helpers
  filteredFindings: () => SecurityFinding[]

  // Actions
  runAudit: (categories?: AuditCategory[]) => Promise<void>
  setSeverityFilter: (severity: AuditSeverity | 'all') => void
  setCategoryFilter: (category: AuditCategory | 'all') => void
  setStatusFilter: (status: 'all' | 'passed' | 'failed') => void
  toggleFinding: (id: string) => void
  clearError: () => void
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useSecurityAuditStore = create<SecurityAuditState>((set, get) => ({
  report: null,
  lastRunAt: null,
  loading: false,
  error: null,
  forbidden: false,
  severityFilter: 'all',
  categoryFilter: 'all',
  statusFilter: 'all',
  expandedFindingId: null,

  filteredFindings: () => {
    const { report, severityFilter, categoryFilter, statusFilter } = get()
    if (!report) return []

    return report.findings.filter((f) => {
      if (severityFilter !== 'all' && f.severity !== severityFilter) return false
      if (categoryFilter !== 'all' && f.category !== categoryFilter) return false
      if (statusFilter === 'passed' && !f.passed) return false
      if (statusFilter === 'failed' && f.passed) return false
      return true
    })
  },

  runAudit: async (categories?: AuditCategory[]) => {
    set({ loading: true, error: null })
    try {
      const params = categories?.length
        ? { categories: categories.join(',') }
        : undefined
      const report = await api.admin.securityAudit(params)
      set({
        report,
        lastRunAt: report.timestamp,
        loading: false,
        forbidden: false,
      })
    } catch (err) {
      if (err instanceof ApiRequestError && err.status === 403) {
        set({ loading: false, forbidden: true, error: null })
      } else {
        set({
          loading: false,
          error: err instanceof Error ? err.message : 'Failed to run security audit',
        })
      }
    }
  },

  setSeverityFilter: (severity) => set({ severityFilter: severity }),
  setCategoryFilter: (category) => set({ categoryFilter: category }),
  setStatusFilter: (status) => set({ statusFilter: status }),

  toggleFinding: (id) =>
    set((s) => ({
      expandedFindingId: s.expandedFindingId === id ? null : id,
    })),

  clearError: () => set({ error: null }),
}))
