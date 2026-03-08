import { useEffect } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { AppShell } from './components/layout/AppShell'
import { ProtectedRoute } from './components/shared/ProtectedRoute'
import { ChatPage } from './pages/Chat'
import { AgentsPage } from './pages/Agents'
import { IntegrationsPage } from './pages/Integrations'
import { BillingPage } from './pages/Billing'
import { SettingsPage } from './pages/Settings'
import { AdminPage } from './pages/Admin'
import { LoginPage } from './pages/Login'
import { RegisterPage } from './pages/Register'
import { VerifyPage } from './pages/Verify'
import { useAuthStore } from './stores/authStore'

export default function App() {
  const initialize = useAuthStore((s) => s.initialize)

  // Initialize auth state on mount (check stored tokens)
  useEffect(() => {
    initialize()
  }, [initialize])

  return (
    <Routes>
      {/* ─── Public routes ─── */}
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route path="/verify" element={<VerifyPage />} />

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
  )
}
