// ============================================================================
// Operator OS — Login Page
// Premium centered card with branding and social login placeholder.
// ============================================================================

import { useState } from 'react'
import { useNavigate, useLocation, Link } from 'react-router-dom'
import { Eye, EyeSlash, SignIn } from '@phosphor-icons/react'
import { useAuthStore } from '../stores/authStore'
import { Button } from '../components/shared/Button'
import { Input } from '../components/shared/Input'

export function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const { login, isLoading, error, clearError } = useAuthStore()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)

  const from = (location.state as { from?: string })?.from || '/chat'

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!email || !password) return

    try {
      await login({ email, password })
      navigate(from, { replace: true })
    } catch {
      // Error is captured in store
    }
  }

  return (
    <div className="h-full flex items-center justify-center bg-bg">
      <div className="w-full max-w-[400px] mx-4 animate-fade-slide">
        {/* ─── Card ─── */}
        <div className="bg-surface border border-border rounded-2xl p-8 shadow-[0_4px_24px_var(--glass-shadow)]">
          {/* Logo + Brand */}
          <div className="flex flex-col items-center mb-8">
            <div className="w-12 h-12 rounded-2xl bg-accent flex items-center justify-center mb-4">
              <span className="text-white text-lg font-bold leading-none">OS</span>
            </div>
            <h1 className="text-xl font-bold text-text tracking-tight">
              Welcome back
            </h1>
            <p className="text-sm text-text-secondary mt-1">
              Sign in to Operator OS
            </p>
          </div>

          {/* Error banner */}
          {error && (
            <div className="mb-5 px-4 py-3 bg-error-subtle border border-error/20 rounded-[var(--radius-sm)] text-sm text-error flex items-start gap-2 animate-fade-slide">
              <span className="flex-1">{error}</span>
              <button
                onClick={clearError}
                className="shrink-0 text-error/60 hover:text-error transition-colors"
                aria-label="Dismiss error"
              >
                <span aria-hidden="true">&times;</span>
              </button>
            </div>
          )}

          {/* Form */}
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <Input
              label="Email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              autoFocus
              required
              disabled={isLoading}
              placeholder="you@example.com"
            />

            <div className="relative">
              <Input
                label="Password"
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
                required
                disabled={isLoading}
                placeholder="Enter your password"
              />
              <button
                type="button"
                onClick={() => setShowPassword(!showPassword)}
                className="absolute right-3 top-[34px] text-text-dim hover:text-text-secondary transition-colors"
                aria-label={showPassword ? 'Hide password' : 'Show password'}
                tabIndex={-1}
              >
                {showPassword ? <EyeSlash size={16} /> : <Eye size={16} />}
              </button>
            </div>

            <Button
              type="submit"
              variant="primary"
              size="lg"
              loading={isLoading}
              disabled={!email || !password}
              icon={!isLoading ? <SignIn size={18} weight="bold" /> : undefined}
              className="w-full mt-1"
            >
              {isLoading ? 'Signing in…' : 'Sign In'}
            </Button>
          </form>

          {/* Divider */}
          <div className="flex items-center gap-3 my-6">
            <div className="flex-1 h-px bg-border" />
            <span className="text-[11px] text-text-dim font-medium uppercase tracking-wider">or</span>
            <div className="flex-1 h-px bg-border" />
          </div>

          {/* Social login placeholder */}
          <div className="flex flex-col gap-2.5">
            <button
              type="button"
              disabled
              className="w-full py-2.5 px-4 bg-surface-2 border border-border rounded-[var(--radius-sm)]
                text-sm font-medium text-text-secondary
                hover:bg-surface-3 transition-colors
                disabled:opacity-40 disabled:cursor-not-allowed
                flex items-center justify-center gap-2"
            >
              Continue with Google
            </button>
            <button
              type="button"
              disabled
              className="w-full py-2.5 px-4 bg-surface-2 border border-border rounded-[var(--radius-sm)]
                text-sm font-medium text-text-secondary
                hover:bg-surface-3 transition-colors
                disabled:opacity-40 disabled:cursor-not-allowed
                flex items-center justify-center gap-2"
            >
              Continue with GitHub
            </button>
          </div>
        </div>

        {/* Footer */}
        <p className="text-center text-xs text-text-dim mt-6">
          Don&apos;t have an account?{' '}
          <Link to="/register" className="text-accent-text hover:underline font-medium">
            Create one
          </Link>
        </p>
      </div>
    </div>
  )
}
