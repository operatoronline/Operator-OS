// ============================================================================
// Operator OS — Email Verification Page
// Handles two flows:
//   1. Post-registration: shows "check your email" with resend button
//   2. Token verification: auto-verifies when ?token=xxx is in URL
// ============================================================================

import { useEffect, useState } from 'react'
import { Link, useLocation, useSearchParams } from 'react-router-dom'
import { useAuthStore } from '../stores/authStore'

type VerifyState = 'pending' | 'verifying' | 'success' | 'error'

export function VerifyPage() {
  const [searchParams] = useSearchParams()
  const location = useLocation()
  const { verifyEmail, resendVerification, isLoading, error, clearError } =
    useAuthStore()

  const token = searchParams.get('token')
  const emailFromState = (location.state as { email?: string })?.email
  const [verifyState, setVerifyState] = useState<VerifyState>(
    token ? 'verifying' : 'pending',
  )
  const [resendSuccess, setResendSuccess] = useState(false)
  const [resendEmail, setResendEmail] = useState(emailFromState || '')

  // Auto-verify if token is present in URL
  useEffect(() => {
    if (!token) return
    let cancelled = false

    const verify = async () => {
      try {
        await verifyEmail(token)
        if (!cancelled) setVerifyState('success')
      } catch {
        if (!cancelled) setVerifyState('error')
      }
    }

    verify()
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token])

  const handleResend = async () => {
    if (!resendEmail) return
    setResendSuccess(false)
    try {
      await resendVerification(resendEmail)
      setResendSuccess(true)
    } catch {
      // Error captured in store
    }
  }

  // ─── Token verification flow ───
  if (token) {
    return (
      <div className="h-full flex items-center justify-center bg-bg">
        <div className="w-full max-w-sm mx-4 animate-fade-slide text-center">
          {verifyState === 'verifying' && (
            <>
              <div className="w-8 h-8 border-2 border-accent border-t-transparent rounded-full animate-spin mx-auto mb-4" />
              <h1 className="text-xl font-bold text-text">
                Verifying your email…
              </h1>
              <p className="text-sm text-text-secondary mt-2">
                This will only take a moment.
              </p>
            </>
          )}

          {verifyState === 'success' && (
            <>
              <div className="w-12 h-12 bg-success-subtle rounded-full flex items-center justify-center mx-auto mb-4">
                <span className="text-success text-xl">✓</span>
              </div>
              <h1 className="text-xl font-bold text-text">Email Verified</h1>
              <p className="text-sm text-text-secondary mt-2">
                Your account is now active. You can sign in.
              </p>
              <Link
                to="/login"
                className="inline-block mt-6 px-6 py-3 bg-accent text-white text-sm font-semibold rounded-[var(--radius-sm)] hover:opacity-90 transition-opacity"
              >
                Sign In
              </Link>
            </>
          )}

          {verifyState === 'error' && (
            <>
              <div className="w-12 h-12 bg-error-subtle rounded-full flex items-center justify-center mx-auto mb-4">
                <span className="text-error text-xl">✕</span>
              </div>
              <h1 className="text-xl font-bold text-text">
                Verification Failed
              </h1>
              <p className="text-sm text-text-secondary mt-2">
                {error || 'This link may be invalid or expired.'}
              </p>

              {/* Resend form */}
              <div className="mt-6 flex flex-col gap-3">
                <input
                  type="email"
                  placeholder="Enter your email to resend"
                  value={resendEmail}
                  onChange={(e) => setResendEmail(e.target.value)}
                  className="w-full px-4 py-3 bg-surface border border-border rounded-[var(--radius-sm)] text-text text-sm placeholder:text-text-dim outline-none focus:border-accent transition-colors"
                />
                <button
                  onClick={handleResend}
                  disabled={isLoading || !resendEmail}
                  className="w-full py-3 bg-surface-2 text-text text-sm font-medium rounded-[var(--radius-sm)] hover:bg-surface-3 transition-colors disabled:opacity-50"
                >
                  {isLoading ? 'Sending…' : 'Resend Verification Email'}
                </button>
              </div>

              <Link
                to="/login"
                className="inline-block mt-4 text-sm text-accent-text hover:underline"
              >
                Back to sign in
              </Link>
            </>
          )}
        </div>
      </div>
    )
  }

  // ─── Post-registration: check your email ───
  return (
    <div className="h-full flex items-center justify-center bg-bg">
      <div className="w-full max-w-sm mx-4 animate-fade-slide text-center">
        <div className="w-12 h-12 bg-accent-subtle rounded-full flex items-center justify-center mx-auto mb-4">
          <span className="text-accent-text text-xl">✉</span>
        </div>

        <h1 className="text-xl font-bold text-text">Check Your Email</h1>
        <p className="text-sm text-text-secondary mt-2">
          We sent a verification link to{' '}
          {emailFromState ? (
            <span className="text-text font-medium">{emailFromState}</span>
          ) : (
            'your email'
          )}
          . Click the link to activate your account.
        </p>

        {/* Resend */}
        <div className="mt-8">
          {resendSuccess ? (
            <p className="text-sm text-success">
              ✓ Verification email resent. Check your inbox.
            </p>
          ) : (
            <>
              {!emailFromState && (
                <input
                  type="email"
                  placeholder="Enter your email"
                  value={resendEmail}
                  onChange={(e) => setResendEmail(e.target.value)}
                  className="w-full px-4 py-3 bg-surface border border-border rounded-[var(--radius-sm)] text-text text-sm placeholder:text-text-dim outline-none focus:border-accent transition-colors mb-3"
                />
              )}
              <button
                onClick={handleResend}
                disabled={isLoading || !resendEmail}
                className="w-full py-3 bg-surface-2 text-text text-sm font-medium rounded-[var(--radius-sm)] hover:bg-surface-3 transition-colors disabled:opacity-50"
              >
                {isLoading ? 'Sending…' : "Didn't get the email? Resend"}
              </button>
            </>
          )}

          {error && (
            <p className="text-sm text-error mt-3">
              {error}
              <button
                onClick={clearError}
                className="ml-2 text-error/60 hover:text-error"
              >
                ✕
              </button>
            </p>
          )}
        </div>

        <Link
          to="/login"
          className="inline-block mt-6 text-sm text-accent-text hover:underline"
        >
          Back to sign in
        </Link>
      </div>
    </div>
  )
}
