// ============================================================================
// Operator OS — Register Page
// Email + password + display name registration with verification redirect.
// ============================================================================

import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useAuthStore } from '../stores/authStore'

export function RegisterPage() {
  const navigate = useNavigate()
  const { register, isLoading, error, clearError } = useAuthStore()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [localError, setLocalError] = useState<string | null>(null)

  const combinedError = localError || error

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLocalError(null)

    if (!email || !password) return

    if (password.length < 8) {
      setLocalError('Password must be at least 8 characters')
      return
    }

    if (password !== confirmPassword) {
      setLocalError('Passwords do not match')
      return
    }

    try {
      await register({
        email,
        password,
        display_name: displayName || undefined,
      })
      // Redirect to verify page with email for resend functionality
      navigate('/verify', { state: { email }, replace: true })
    } catch {
      // Error is captured in store
    }
  }

  return (
    <div className="h-full flex items-center justify-center bg-bg">
      <div className="w-full max-w-sm mx-4 animate-fade-slide">
        {/* Header */}
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-text tracking-tight">
            Create Account
          </h1>
          <p className="text-sm text-text-secondary mt-1">
            Get started with Operator OS
          </p>
        </div>

        {/* Error banner */}
        {combinedError && (
          <div className="mb-4 px-4 py-3 bg-error-subtle border border-error/20 rounded-[var(--radius-sm)] text-sm text-error flex items-start gap-2">
            <span className="shrink-0 mt-0.5">⚠</span>
            <span className="flex-1">{combinedError}</span>
            <button
              onClick={() => {
                setLocalError(null)
                clearError()
              }}
              className="shrink-0 text-error/60 hover:text-error transition-colors"
              aria-label="Dismiss error"
            >
              ✕
            </button>
          </div>
        )}

        {/* Form */}
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <input
            type="text"
            placeholder="Display name (optional)"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            autoComplete="name"
            autoFocus
            disabled={isLoading}
            className="w-full px-4 py-3 bg-surface border border-border rounded-[var(--radius-sm)] text-text text-sm placeholder:text-text-dim outline-none focus:border-accent transition-colors disabled:opacity-50"
          />
          <input
            type="email"
            placeholder="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoComplete="email"
            required
            disabled={isLoading}
            className="w-full px-4 py-3 bg-surface border border-border rounded-[var(--radius-sm)] text-text text-sm placeholder:text-text-dim outline-none focus:border-accent transition-colors disabled:opacity-50"
          />
          <input
            type="password"
            placeholder="Password (min. 8 characters)"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="new-password"
            required
            disabled={isLoading}
            className="w-full px-4 py-3 bg-surface border border-border rounded-[var(--radius-sm)] text-text text-sm placeholder:text-text-dim outline-none focus:border-accent transition-colors disabled:opacity-50"
          />
          <input
            type="password"
            placeholder="Confirm password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            autoComplete="new-password"
            required
            disabled={isLoading}
            className="w-full px-4 py-3 bg-surface border border-border rounded-[var(--radius-sm)] text-text text-sm placeholder:text-text-dim outline-none focus:border-accent transition-colors disabled:opacity-50"
          />
          <button
            type="submit"
            disabled={isLoading || !email || !password || !confirmPassword}
            className="w-full py-3 bg-accent text-white text-sm font-semibold rounded-[var(--radius-sm)] hover:opacity-90 transition-opacity mt-2 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            {isLoading ? (
              <>
                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                Creating account…
              </>
            ) : (
              'Create Account'
            )}
          </button>
        </form>

        {/* Footer */}
        <p className="text-center text-xs text-text-dim mt-6">
          Already have an account?{' '}
          <Link
            to="/login"
            className="text-accent-text hover:underline"
          >
            Sign in
          </Link>
        </p>
      </div>
    </div>
  )
}
