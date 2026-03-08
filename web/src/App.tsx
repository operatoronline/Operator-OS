import { lazy, Suspense, useEffect } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { AppShell } from './components/layout/AppShell'
import { ProtectedRoute } from './components/shared/ProtectedRoute'
import { SkipToContent } from './components/shared/SkipToContent'
import { RouteAnnouncer } from './components/shared/RouteAnnouncer'
import { ErrorBoundary } from './components/shared/ErrorBoundary'
import { PageSpinner } from './components/shared/PageSpinner'
import { useAuthStore } from './stores/authStore'

// ─── Lazy-loaded pages (code-split per route) ───
const ChatPage = lazy(() => import('./pages/Chat').then(m => ({ default: m.ChatPage })))
const AgentsPage = lazy(() => import('./pages/Agents').then(m => ({ default: m.AgentsPage })))
const IntegrationsPage = lazy(() => import('./pages/Integrations').then(m => ({ default: m.IntegrationsPage })))
const BillingPage = lazy(() => import('./pages/Billing').then(m => ({ default: m.BillingPage })))
const SettingsPage = lazy(() => import('./pages/Settings').then(m => ({ default: m.SettingsPage })))
const AdminPage = lazy(() => import('./pages/Admin').then(m => ({ default: m.AdminPage })))
const LoginPage = lazy(() => import('./pages/Login').then(m => ({ default: m.LoginPage })))
const RegisterPage = lazy(() => import('./pages/Register').then(m => ({ default: m.RegisterPage })))
const VerifyPage = lazy(() => import('./pages/Verify').then(m => ({ default: m.VerifyPage })))
const OAuthCallbackPage = lazy(() => import('./pages/OAuthCallback').then(m => ({ default: m.OAuthCallbackPage })))

export default function App() {
  const initialize = useAuthStore((s) => s.initialize)

  // Initialize auth state on mount (check stored tokens)
  useEffect(() => {
    initialize()
  }, [initialize])

  return (
    <ErrorBoundary>
    <SkipToContent />
    <RouteAnnouncer />
    <Suspense fallback={<PageSpinner />}>
    <Routes>
      {/* ─── Public routes ─── */}
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route path="/verify" element={<VerifyPage />} />
      <Route path="/oauth/callback" element={<OAuthCallbackPage />} />

      {/* ─── Protected routes ─── */}
      <Route
        element={
          <ProtectedRoute>
            <AppShell />
          </ProtectedRoute>
        }
      >
        <Route path="/chat" element={<ChatPage />} />
        <Route path="/agents" element={<AgentsPage />} />
        <Route path="/integrations" element={<IntegrationsPage />} />
        <Route path="/billing" element={<BillingPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/admin" element={<AdminPage />} />
      </Route>

      {/* ─── Fallback ─── */}
      <Route path="*" element={<Navigate to="/chat" replace />} />
    </Routes>
    </Suspense>
    </ErrorBoundary>
  )
}
