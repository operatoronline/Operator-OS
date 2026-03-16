// ============================================================================
// Operator OS — Register Page
// Premium card design with password strength meter and step indicator.
// ============================================================================

import { useState, useMemo } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { Eye, EyeSlash, UserPlus } from '@phosphor-icons/react'
import { useAuthStore } from '../stores/authStore'
import { Button } from '../components/shared/Button'
import { Input } from '../components/shared/Input'

// ─── Password strength calculator ───
type StrengthLevel = 0 | 1 | 2 | 3 | 4

function calcPasswordStrength(pw: string): { level: StrengthLevel; label: string } {
  if (pw.length === 0) return { level: 0, label: '' }
  let score = 0
  if (pw.length >= 8) score++
  if (pw.length >= 12) score++
  if (/[A-Z]/.test(pw) && /[a-z]/.test(pw)) score++
  if (/\d/.test(pw)) score++
  if (/[^A-Za-z0-9]/.test(pw)) score++

  if (score <= 1) return { level: 1, label: 'Weak' }
  if (score === 2) return { level: 2, label: 'Fair' }
  if (score === 3) return { level: 3, label: 'Good' }
  return { level: 4, label: 'Strong' }
}

const strengthColors: Record<StrengthLevel, string> = {
  0: 'bg-border',
  1: 'bg-error',
  2: 'bg-warning',
  3: 'bg-accent',
  4: 'bg-success',
}

const strengthTextColors: Record<StrengthLevel, string> = {
  0: 'text-text-dim',
  1: 'text-error',
  2: 'text-warning',
  3: 'text-accent-text',
  4: 'text-success',
}

export function RegisterPage() {
  const navigate = useNavigate()
  const { register, isLoading, error, clearError } = useAuthStore()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [localError, setLocalError] = useState<string | null>(null)

  const combinedError = localError || error

  const strength = useMemo(() => calcPasswordStrength(password), [password])

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
      navigate('/verify', { state: { email }, replace: true })
    } catch {
      // Error is captured in store
    }
  }

  return (
    <div className="h-full flex items-center justify-center bg-bg overflow-y-auto">
      <div className="w-full max-w-[400px] mx-4 my-8 animate-fade-slide">
        {/* ─── Card ─── */}
        <div className="bg-surface border border-border rounded-2xl p-8 shadow-[0_4px_24px_var(--glass-shadow)]">
          {/* Logo + Brand */}
          <div className="flex flex-col items-center mb-8">
            <div className="w-12 h-12 rounded-2xl bg-accent flex items-center justify-center mb-4">
              <span className="text-white text-lg font-bold leading-none">OS</span>
            </div>
            <h1 className="text-xl font-bold text-text tracking-tight">
              Create Account
            </h1>
            <p className="text-sm text-text-secondary mt-1">
              Get started with Operator OS
            </p>
          </div>

          {/* Step indicator */}
          <div className="flex items-center gap-2 mb-6">
            <div className="flex-1 h-1 rounded-full bg-accent" />
            <div className="flex-1 h-1 rounded-full bg-border" />
          </div>

          {/* Error banner */}
          {combinedError && (
            <div className="mb-5 px-4 py-3 bg-error-subtle border border-error/20 rounded-[var(--radius-sm)] text-sm text-error flex items-start gap-2 animate-fade-slide">
              <span className="flex-1">{combinedError}</span>
              <button
                onClick={() => {
                  setLocalError(null)
                  clearError()
                }}
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
              label="Display name"
              type="text"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              autoComplete="name"
              autoFocus
              disabled={isLoading}
              placeholder="Optional"
            />

            <Input
              label="Email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              required
              disabled={isLoading}
              placeholder="you@example.com"
            />

            <div>
              <div className="relative">
                <Input
                  label="Password"
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="new-password"
                  required
                  disabled={isLoading}
                  placeholder="Min. 8 characters"
                  error={password.length > 0 && password.length < 8 ? 'Too short' : undefined}
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

              {/* Password strength meter */}
              {password.length > 0 && (
                <div className="mt-2 animate-fade-in">
                  <div className="flex gap-1">
                    {([1, 2, 3, 4] as const).map((seg) => (
                      <div
                        key={seg}
                        className={`h-1 flex-1 rounded-full transition-colors duration-200 ${
                          seg <= strength.level ? strengthColors[strength.level] : 'bg-border'
                        }`}
                      />
                    ))}
                  </div>
                  <p className={`text-[11px] mt-1 font-medium ${strengthTextColors[strength.level]}`}>
                    {strength.label}
                  </p>
                </div>
              )}
            </div>

            <Input
              label="Confirm password"
              type={showPassword ? 'text' : 'password'}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              autoComplete="new-password"
              required
              disabled={isLoading}
              placeholder="Re-enter password"
              error={
                confirmPassword.length > 0 && password !== confirmPassword
                  ? 'Passwords do not match'
                  : undefined
              }
            />

            <Button
              type="submit"
              variant="primary"
              size="lg"
              loading={isLoading}
              disabled={!email || !password || !confirmPassword}
              icon={!isLoading ? <UserPlus size={18} weight="bold" /> : undefined}
              className="w-full mt-1"
            >
              {isLoading ? 'Creating account…' : 'Create Account'}
            </Button>
          </form>
        </div>

        {/* Footer */}
        <p className="text-center text-xs text-text-dim mt-6">
          Already have an account?{' '}
          <Link to="/login" className="text-accent-text hover:underline font-medium">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  )
}
