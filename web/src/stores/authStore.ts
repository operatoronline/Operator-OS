// ============================================================================
// Operator OS — Auth Store
// Zustand store for authentication state, JWT tokens, and user profile.
// ============================================================================

import { create } from 'zustand'
import { api, tokenStore, ApiRequestError } from '../services/api'
import type { UserProfile, LoginRequest, RegisterRequest } from '../types/api'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface AuthState {
  user: UserProfile | null
  isAuthenticated: boolean
  isLoading: boolean
  isInitialized: boolean
  error: string | null

  // Actions
  login: (data: LoginRequest) => Promise<void>
  register: (data: RegisterRequest) => Promise<UserProfile>
  logout: () => void
  verifyEmail: (token: string) => Promise<void>
  resendVerification: (email: string) => Promise<void>
  initialize: () => void
  clearError: () => void
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Parse JWT payload without external library */
function parseJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const base64 = token.split('.')[1]
    if (!base64) return null
    const json = atob(base64.replace(/-/g, '+').replace(/_/g, '/'))
    return JSON.parse(json)
  } catch {
    return null
  }
}

/** Check if a JWT is expired (with 30s buffer) */
function isTokenExpired(token: string): boolean {
  const payload = parseJwtPayload(token)
  if (!payload || typeof payload.exp !== 'number') return true
  return payload.exp * 1000 < Date.now() + 30_000
}

/** Extract UserProfile from stored JWT if available */
function getUserFromToken(token: string): UserProfile | null {
  const payload = parseJwtPayload(token)
  if (!payload) return null
  return {
    id: payload.sub as string,
    email: payload.email as string,
    display_name: (payload.display_name as string) || (payload.email as string),
    status: (payload.status as UserProfile['status']) || 'active',
    email_verified: (payload.email_verified as boolean) ?? false,
  }
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useAuthStore = create<AuthState>((set, get) => {
  // Listen for auth expiry events from API client
  if (typeof window !== 'undefined') {
    window.addEventListener('os:auth:expired', () => {
      get().logout()
    })
  }

  return {
    user: null,
    isAuthenticated: false,
    isLoading: false,
    isInitialized: false,
    error: null,

    // -----------------------------------------------------------------------
    // Initialize — check for existing tokens on app startup
    // -----------------------------------------------------------------------
    initialize: () => {
      const accessToken = tokenStore.getAccess()
      if (!accessToken) {
        set({ isInitialized: true, isAuthenticated: false, user: null })
        return
      }

      // If token isn't expired, restore session from token payload
      if (!isTokenExpired(accessToken)) {
        const user = getUserFromToken(accessToken)
        set({
          isInitialized: true,
          isAuthenticated: !!user,
          user,
        })
        return
      }

      // Token expired — try refresh
      const refreshToken = tokenStore.getRefresh()
      if (!refreshToken) {
        tokenStore.clear()
        set({ isInitialized: true, isAuthenticated: false, user: null })
        return
      }

      // Async refresh
      set({ isLoading: true })
      api.auth
        .refresh(refreshToken)
        .then((res) => {
          tokenStore.set(res.access_token, res.refresh_token)
          set({
            isInitialized: true,
            isAuthenticated: true,
            isLoading: false,
            user: res.user,
          })
        })
        .catch(() => {
          tokenStore.clear()
          set({
            isInitialized: true,
            isAuthenticated: false,
            isLoading: false,
            user: null,
          })
        })
    },

    // -----------------------------------------------------------------------
    // Login
    // -----------------------------------------------------------------------
    login: async (data: LoginRequest) => {
      set({ isLoading: true, error: null })
      try {
        const res = await api.auth.login(data)
        tokenStore.set(res.access_token, res.refresh_token)
        set({
          user: res.user,
          isAuthenticated: true,
          isLoading: false,
          error: null,
        })
      } catch (err) {
        const message =
          err instanceof ApiRequestError
            ? err.status === 401
              ? 'Invalid email or password'
              : err.status === 403
                ? 'Please verify your email before signing in'
                : err.message
            : 'Something went wrong. Please try again.'
        set({ isLoading: false, error: message })
        throw err
      }
    },

    // -----------------------------------------------------------------------
    // Register
    // -----------------------------------------------------------------------
    register: async (data: RegisterRequest) => {
      set({ isLoading: true, error: null })
      try {
        const user = await api.auth.register(data)
        set({ isLoading: false, error: null })
        return user
      } catch (err) {
        const message =
          err instanceof ApiRequestError
            ? err.status === 409
              ? 'An account with this email already exists'
              : err.message
            : 'Something went wrong. Please try again.'
        set({ isLoading: false, error: message })
        throw err
      }
    },

    // -----------------------------------------------------------------------
    // Logout
    // -----------------------------------------------------------------------
    logout: () => {
      tokenStore.clear()
      set({
        user: null,
        isAuthenticated: false,
        error: null,
      })
    },

    // -----------------------------------------------------------------------
    // Email verification
    // -----------------------------------------------------------------------
    verifyEmail: async (token: string) => {
      set({ isLoading: true, error: null })
      try {
        await api.auth.verifyEmail(token)
        set({ isLoading: false })
      } catch (err) {
        const message =
          err instanceof ApiRequestError
            ? err.status === 400
              ? 'Invalid or expired verification link'
              : err.message
            : 'Verification failed. Please try again.'
        set({ isLoading: false, error: message })
        throw err
      }
    },

    resendVerification: async (email: string) => {
      set({ isLoading: true, error: null })
      try {
        await api.auth.resendVerification(email)
        set({ isLoading: false })
      } catch (err) {
        const message =
          err instanceof ApiRequestError
            ? err.message
            : 'Failed to resend verification email.'
        set({ isLoading: false, error: message })
        throw err
      }
    },

    // -----------------------------------------------------------------------
    // Clear error
    // -----------------------------------------------------------------------
    clearError: () => set({ error: null }),
  }
})
