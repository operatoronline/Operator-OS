// ============================================================================
// Operator OS — Protected Route
// Redirects unauthenticated users to /login with return-to support.
// ============================================================================

import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../../stores/authStore'

/** Loading spinner shown while auth state initializes */
function AuthLoading() {
  return (
    <div className="h-full flex items-center justify-center bg-bg">
      <div className="flex flex-col items-center gap-3">
        <div className="w-6 h-6 border-2 border-accent border-t-transparent rounded-full animate-spin" />
        <span className="text-xs text-text-dim">Loading…</span>
      </div>
    </div>
  )
}

interface ProtectedRouteProps {
  children: React.ReactNode
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const { isAuthenticated, isInitialized, isLoading } = useAuthStore()
  const location = useLocation()

  // Still initializing (checking stored tokens / refreshing)
  if (!isInitialized || isLoading) {
    return <AuthLoading />
  }

  // Not authenticated — redirect to login with return path
  if (!isAuthenticated) {
    return (
      <Navigate
        to="/login"
        state={{ from: location.pathname }}
        replace
      />
    )
  }

  return <>{children}</>
}
