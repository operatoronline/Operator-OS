// ============================================================================
// Operator OS — OAuth Callback Page
// Handles the redirect from OAuth providers. Parses the result from URL params
// and posts it back to the opener window (popup flow) or renders inline.
// ============================================================================

import { useEffect, useState, useRef } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { CheckCircle, XCircle, CircleNotch } from '@phosphor-icons/react'
import type { OAuthPopupResult } from '../utils/oauthPopup'

type CallbackState = 'processing' | 'success' | 'error'

export function OAuthCallbackPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [state, setState] = useState<CallbackState>('processing')
  const [message, setMessage] = useState('')
  const [integrationName, setIntegrationName] = useState('')
  const processed = useRef(false)

  useEffect(() => {
    if (processed.current) return
    processed.current = true

    // Parse URL params from OAuth callback
    const code = searchParams.get('code')
    const error = searchParams.get('error')
    const errorDesc = searchParams.get('error_description')
    const integrationId = searchParams.get('integration_id') || searchParams.get('state')
    const provider = searchParams.get('provider')
    const name = searchParams.get('name') || provider || 'Integration'
    const status = searchParams.get('status') // Backend may set this after processing

    setIntegrationName(name)

    let result: OAuthPopupResult

    if (error) {
      // OAuth error
      result = {
        success: false,
        integration_id: integrationId ?? undefined,
        provider: provider ?? undefined,
        error: errorDesc || error,
      }
      setState('error')
      setMessage(errorDesc || error)
    } else if (status === 'success' || code) {
      // Successful authorization
      result = {
        success: true,
        integration_id: integrationId ?? undefined,
        provider: provider ?? undefined,
        code: code ?? undefined,
      }
      setState('success')
      setMessage(`${name} connected successfully`)
    } else {
      // Unknown state
      result = {
        success: false,
        error: 'Unexpected callback state',
      }
      setState('error')
      setMessage('Unexpected callback state. Please try again.')
    }

    // If opened as popup, post message to opener and close
    if (window.opener && !window.opener.closed) {
      window.opener.postMessage(
        { type: 'os:oauth:callback', result },
        window.location.origin,
      )

      // Auto-close after brief delay so the user sees the result
      setTimeout(() => {
        window.close()
      }, 1500)
    } else {
      // Not a popup — redirect to integrations page after delay
      setTimeout(() => {
        navigate('/integrations', { replace: true })
      }, 3000)
    }
  }, [searchParams, navigate])

  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--bg)] p-4">
      <div
        className="w-full max-w-sm bg-[var(--surface)] border border-[var(--border-subtle)]
          rounded-2xl p-8 text-center space-y-4 shadow-xl animate-scale-in"
      >
        {/* ─── Icon ─── */}
        {state === 'processing' && (
          <CircleNotch
            size={48}
            weight="bold"
            className="mx-auto text-[var(--accent-text)] animate-spin"
          />
        )}
        {state === 'success' && (
          <CheckCircle
            size={48}
            weight="fill"
            className="mx-auto text-[var(--success)]"
          />
        )}
        {state === 'error' && (
          <XCircle
            size={48}
            weight="fill"
            className="mx-auto text-[var(--error)]"
          />
        )}

        {/* ─── Title ─── */}
        <h1 className="text-base font-bold text-[var(--text)]">
          {state === 'processing' && 'Connecting…'}
          {state === 'success' && `${integrationName} Connected`}
          {state === 'error' && 'Connection Failed'}
        </h1>

        {/* ─── Message ─── */}
        <p className="text-sm text-[var(--text-secondary)]">
          {state === 'processing' && 'Processing authorization…'}
          {state === 'success' && (
            <>
              {message}
              <br />
              <span className="text-xs text-[var(--text-dim)] mt-1 block">
                {window.opener ? 'This window will close shortly.' : 'Redirecting…'}
              </span>
            </>
          )}
          {state === 'error' && (
            <>
              {message}
              <br />
              <span className="text-xs text-[var(--text-dim)] mt-1 block">
                {window.opener ? 'You can close this window.' : 'Redirecting to integrations…'}
              </span>
            </>
          )}
        </p>
      </div>
    </div>
  )
}
