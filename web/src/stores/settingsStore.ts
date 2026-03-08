// ============================================================================
// Operator OS — Settings Store
// Profile, password, notifications, API keys, GDPR data requests.
// ============================================================================

import { create } from 'zustand'
import { api, ApiRequestError } from '../services/api'
import type {
  NotificationPreferences,
  ApiKey,
  DataSubjectRequest,
} from '../types/api'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface SettingsState {
  // Profile
  profileLoading: boolean
  profileError: string | null
  profileSuccess: string | null

  // Password
  passwordLoading: boolean
  passwordError: string | null
  passwordSuccess: string | null

  // Notifications
  notifications: NotificationPreferences | null
  notificationsLoading: boolean
  notificationsError: string | null

  // API Keys
  apiKeys: ApiKey[]
  apiKeysLoading: boolean
  apiKeysError: string | null
  newKeySecret: string | null

  // GDPR
  gdprRequests: DataSubjectRequest[]
  gdprLoading: boolean
  gdprError: string | null
  gdprSuccess: string | null

  // Actions
  updateProfile: (displayName: string) => Promise<void>
  changePassword: (currentPassword: string, newPassword: string) => Promise<void>

  fetchNotifications: () => Promise<void>
  updateNotifications: (prefs: Partial<NotificationPreferences>) => Promise<void>

  fetchApiKeys: () => Promise<void>
  createApiKey: (name: string, expiresInDays?: number) => Promise<void>
  deleteApiKey: (id: string) => Promise<void>
  clearNewKeySecret: () => void

  fetchGdprRequests: () => Promise<void>
  requestDataExport: () => Promise<void>
  requestAccountDeletion: () => Promise<void>
  cancelGdprRequest: (id: string) => Promise<void>

  clearMessages: () => void
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function extractError(err: unknown, fallback: string): string {
  if (err instanceof ApiRequestError) {
    if (err.status === 401) return 'Current password is incorrect'
    if (err.status === 409) return 'Request already exists'
    if (err.status === 429) return 'Too many requests. Please wait.'
    return err.message
  }
  return fallback
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useSettingsStore = create<SettingsState>((set, get) => ({
  profileLoading: false,
  profileError: null,
  profileSuccess: null,

  passwordLoading: false,
  passwordError: null,
  passwordSuccess: null,

  notifications: null,
  notificationsLoading: false,
  notificationsError: null,

  apiKeys: [],
  apiKeysLoading: false,
  apiKeysError: null,
  newKeySecret: null,

  gdprRequests: [],
  gdprLoading: false,
  gdprError: null,
  gdprSuccess: null,

  // ─── Profile ───────────────────────────────────────────────────────────

  updateProfile: async (displayName: string) => {
    set({ profileLoading: true, profileError: null, profileSuccess: null })
    try {
      await api.user.updateProfile({ display_name: displayName })
      set({ profileLoading: false, profileSuccess: 'Profile updated' })

      // Update auth store user
      const { useAuthStore } = await import('./authStore')
      const authState = useAuthStore.getState()
      if (authState.user) {
        useAuthStore.setState({
          user: { ...authState.user, display_name: displayName },
        })
      }
    } catch (err) {
      set({
        profileLoading: false,
        profileError: extractError(err, 'Failed to update profile'),
      })
      throw err
    }
  },

  // ─── Password ──────────────────────────────────────────────────────────

  changePassword: async (currentPassword: string, newPassword: string) => {
    set({ passwordLoading: true, passwordError: null, passwordSuccess: null })
    try {
      await api.user.changePassword({
        current_password: currentPassword,
        new_password: newPassword,
      })
      set({ passwordLoading: false, passwordSuccess: 'Password changed successfully' })
    } catch (err) {
      set({
        passwordLoading: false,
        passwordError: extractError(err, 'Failed to change password'),
      })
      throw err
    }
  },

  // ─── Notifications ─────────────────────────────────────────────────────

  fetchNotifications: async () => {
    set({ notificationsLoading: true, notificationsError: null })
    try {
      const prefs = await api.user.notifications()
      set({ notifications: prefs, notificationsLoading: false })
    } catch (err) {
      set({
        notificationsLoading: false,
        notificationsError: extractError(err, 'Failed to load notification preferences'),
      })
    }
  },

  updateNotifications: async (prefs: Partial<NotificationPreferences>) => {
    const current = get().notifications
    // Optimistic update
    if (current) {
      set({ notifications: { ...current, ...prefs } })
    }
    try {
      const updated = await api.user.updateNotifications(prefs)
      set({ notifications: updated })
    } catch (err) {
      // Revert
      set({
        notifications: current,
        notificationsError: extractError(err, 'Failed to update notifications'),
      })
    }
  },

  // ─── API Keys ──────────────────────────────────────────────────────────

  fetchApiKeys: async () => {
    set({ apiKeysLoading: true, apiKeysError: null })
    try {
      const keys = await api.user.apiKeys()
      set({ apiKeys: keys, apiKeysLoading: false })
    } catch (err) {
      set({
        apiKeysLoading: false,
        apiKeysError: extractError(err, 'Failed to load API keys'),
      })
    }
  },

  createApiKey: async (name: string, expiresInDays?: number) => {
    set({ apiKeysLoading: true, apiKeysError: null, newKeySecret: null })
    try {
      const res = await api.user.createApiKey({
        name,
        expires_in_days: expiresInDays,
      })
      set((state) => ({
        apiKeys: [res.key, ...state.apiKeys],
        apiKeysLoading: false,
        newKeySecret: res.secret,
      }))
    } catch (err) {
      set({
        apiKeysLoading: false,
        apiKeysError: extractError(err, 'Failed to create API key'),
      })
      throw err
    }
  },

  deleteApiKey: async (id: string) => {
    const prev = get().apiKeys
    // Optimistic
    set({ apiKeys: prev.filter((k) => k.id !== id) })
    try {
      await api.user.deleteApiKey(id)
    } catch (err) {
      set({
        apiKeys: prev,
        apiKeysError: extractError(err, 'Failed to delete API key'),
      })
    }
  },

  clearNewKeySecret: () => set({ newKeySecret: null }),

  // ─── GDPR ──────────────────────────────────────────────────────────────

  fetchGdprRequests: async () => {
    set({ gdprLoading: true, gdprError: null })
    try {
      const requests = await api.gdpr.requests()
      set({ gdprRequests: requests, gdprLoading: false })
    } catch (err) {
      set({
        gdprLoading: false,
        gdprError: extractError(err, 'Failed to load data requests'),
      })
    }
  },

  requestDataExport: async () => {
    set({ gdprLoading: true, gdprError: null, gdprSuccess: null })
    try {
      const req = await api.gdpr.export()
      set((state) => ({
        gdprRequests: [req, ...state.gdprRequests],
        gdprLoading: false,
        gdprSuccess: 'Data export requested. You will be notified when ready.',
      }))
    } catch (err) {
      set({
        gdprLoading: false,
        gdprError: extractError(err, 'Failed to request data export'),
      })
    }
  },

  requestAccountDeletion: async () => {
    set({ gdprLoading: true, gdprError: null, gdprSuccess: null })
    try {
      const req = await api.gdpr.erase()
      set((state) => ({
        gdprRequests: [req, ...state.gdprRequests],
        gdprLoading: false,
        gdprSuccess: 'Account deletion requested. This may take up to 30 days.',
      }))
    } catch (err) {
      set({
        gdprLoading: false,
        gdprError: extractError(err, 'Failed to request account deletion'),
      })
    }
  },

  cancelGdprRequest: async (id: string) => {
    try {
      await api.gdpr.cancelRequest(id)
      set((state) => ({
        gdprRequests: state.gdprRequests.filter((r) => r.id !== id),
      }))
    } catch (err) {
      set({
        gdprError: extractError(err, 'Failed to cancel request'),
      })
    }
  },

  // ─── Utility ───────────────────────────────────────────────────────────

  clearMessages: () =>
    set({
      profileError: null,
      profileSuccess: null,
      passwordError: null,
      passwordSuccess: null,
      notificationsError: null,
      apiKeysError: null,
      gdprError: null,
      gdprSuccess: null,
    }),
}))
