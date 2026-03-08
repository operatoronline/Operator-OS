// ============================================================================
// Operator OS — OAuth Flow Component
// Scope consent dialog + popup authorization for OAuth integrations.
// Shows available scopes, handles the popup lifecycle, and reports results.
// ============================================================================

import { memo, useState, useEffect, useCallback, useMemo } from 'react'
import {
  ShieldCheck,
  ArrowSquareOut,
  CheckCircle,
  XCircle,
  CircleNotch,
  Eye,
  PencilSimple,
  Trash,
  Folder,
  UserCircle,
  Envelope,
  Database,
  Globe,
  Warning,
  ArrowsClockwise,
} from '@phosphor-icons/react'
import { Modal } from '../shared/Modal'
import { Button } from '../shared/Button'
import { Badge } from '../shared/Badge'
import { openOAuthPopup, type OAuthPopupResult } from '../../utils/oauthPopup'
import { api } from '../../services/api'
import type { IntegrationSummary, OAuthProvider } from '../../types/api'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface OAuthFlowProps {
  open: boolean
  onClose: () => void
  integration: IntegrationSummary | null
  /** Called after successful or failed OAuth */
  onComplete: (result: OAuthPopupResult) => void
  /** Is this a reconnect (expired/revoked token)? */
  reconnect?: boolean
}

type FlowStep = 'consent' | 'authorizing' | 'success' | 'error'

interface ScopeInfo {
  id: string
  label: string
  description: string
  icon: React.ReactNode
  category: 'read' | 'write' | 'delete' | 'admin'
}

// ---------------------------------------------------------------------------
// Scope helpers
// ---------------------------------------------------------------------------

/** Categorize a scope string into a human-readable object */
function parseScopeInfo(scope: string): ScopeInfo {
  const lower = scope.toLowerCase()

  // Determine category
  let category: ScopeInfo['category'] = 'read'
  if (lower.includes('write') || lower.includes('edit') || lower.includes('create') || lower.includes('manage')) {
    category = 'write'
  } else if (lower.includes('delete') || lower.includes('remove')) {
    category = 'delete'
  } else if (lower.includes('admin') || lower.includes('full')) {
    category = 'admin'
  }

  // Icon based on scope content
  let icon: React.ReactNode = <Globe size={16} />
  if (lower.includes('profile') || lower.includes('user')) icon = <UserCircle size={16} />
  else if (lower.includes('email') || lower.includes('mail')) icon = <Envelope size={16} />
  else if (lower.includes('file') || lower.includes('drive') || lower.includes('storage')) icon = <Folder size={16} />
  else if (lower.includes('data') || lower.includes('database')) icon = <Database size={16} />

  // Action icon overlay
  if (category === 'write') icon = <PencilSimple size={16} />
  else if (category === 'delete') icon = <Trash size={16} />
  else if (category === 'read') icon = <Eye size={16} />

  // Human-readable label
  const label = scope
    .replace(/[._-]/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase())
    .replace(/Readonly/i, 'Read Only')

  // Description
  const descriptions: Record<string, string> = {
    'profile': 'View your basic profile information',
    'email': 'View your email address',
    'openid': 'Verify your identity',
    'read': 'Read access to your data',
    'write': 'Create and modify your data',
    'offline_access': 'Stay connected when you\'re not actively using the app',
  }

  let description = descriptions[lower] || `Access to ${label.toLowerCase()}`
  if (category === 'write') description = `Create and modify: ${label.toLowerCase()}`
  if (category === 'delete') description = `Delete: ${label.toLowerCase()}`

  return { id: scope, label, description, icon, category }
}

/** Category color mapping */
function categoryColor(cat: ScopeInfo['category']): string {
  switch (cat) {
    case 'read': return 'var(--success)'
    case 'write': return 'var(--accent-text)'
    case 'delete': return 'var(--error)'
    case 'admin': return 'var(--warning)'
  }
}

function categoryLabel(cat: ScopeInfo['category']): string {
  switch (cat) {
    case 'read': return 'Read'
    case 'write': return 'Write'
    case 'delete': return 'Delete'
    case 'admin': return 'Admin'
  }
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export const OAuthFlow = memo(function OAuthFlow({
  open,
  onClose,
  integration,
  onComplete,
  reconnect = false,
}: OAuthFlowProps) {
  const [step, setStep] = useState<FlowStep>('consent')
  const [provider, setProvider] = useState<OAuthProvider | null>(null)
  const [selectedScopes, setSelectedScopes] = useState<Set<string>>(new Set())
  const [loadingProvider, setLoadingProvider] = useState(false)
  const [errorMessage, setErrorMessage] = useState('')
  const [resultMessage, setResultMessage] = useState('')

  // Reset when opening
  useEffect(() => {
    if (open && integration) {
      setStep('consent')
      setErrorMessage('')
      setResultMessage('')
      setLoadingProvider(true)

      // Fetch OAuth provider details for scope info
      api.oauth.providers()
        .then((providers) => {
          const match = providers.find(
            (p) =>
              p.name.toLowerCase() === integration.name.toLowerCase() ||
              integration.name.toLowerCase().includes(p.name.toLowerCase()),
          )
          if (match) {
            setProvider(match)
            setSelectedScopes(new Set(match.scopes))
          } else {
            // No provider match — use integration tools as proxy
            setProvider(null)
            setSelectedScopes(new Set())
          }
        })
        .catch(() => {
          setProvider(null)
        })
        .finally(() => setLoadingProvider(false))
    }
  }, [open, integration])

  // Parse scopes for display
  const scopeInfos = useMemo(() => {
    if (!provider) return []
    return provider.scopes.map(parseScopeInfo)
  }, [provider])

  // Group by category
  const scopesByCategory = useMemo(() => {
    const groups: Record<string, ScopeInfo[]> = {}
    for (const info of scopeInfos) {
      if (!groups[info.category]) groups[info.category] = []
      groups[info.category].push(info)
    }
    return groups
  }, [scopeInfos])

  const toggleScope = useCallback((scope: string) => {
    setSelectedScopes((prev) => {
      const next = new Set(prev)
      if (next.has(scope)) next.delete(scope)
      else next.add(scope)
      return next
    })
  }, [])

  // ─── Authorize ───
  const handleAuthorize = useCallback(async () => {
    if (!integration) return

    setStep('authorizing')
    setErrorMessage('')

    try {
      // Get the auth URL from backend
      const scopes = selectedScopes.size > 0 ? Array.from(selectedScopes) : undefined
      const { auth_url } = await api.oauth.authorize({
        provider: integration.name.toLowerCase(),
        scopes,
        redirect_uri: `${window.location.origin}/oauth/callback`,
      })

      // Open popup
      const result = await openOAuthPopup({
        url: auth_url,
        title: `Connect ${integration.name}`,
      })

      if (result.success) {
        setStep('success')
        setResultMessage(`${integration.name} has been connected successfully.`)
        // Delay before closing so user sees success state
        setTimeout(() => {
          onComplete(result)
          onClose()
        }, 1500)
      } else {
        setStep('error')
        setErrorMessage(result.error || 'Authorization was not completed.')
      }
    } catch (err) {
      setStep('error')
      setErrorMessage(
        err instanceof Error ? err.message : 'Failed to start authorization.',
      )
    }
  }, [integration, selectedScopes, onComplete, onClose])

  // ─── Retry ───
  const handleRetry = useCallback(() => {
    setStep('consent')
    setErrorMessage('')
  }, [])

  if (!integration) return null

  return (
    <Modal
      open={open}
      onClose={step === 'authorizing' ? (() => {}) : onClose}
      title={
        step === 'consent'
          ? `${reconnect ? 'Reconnect' : 'Connect'} ${integration.name}`
          : step === 'authorizing'
            ? 'Authorizing…'
            : step === 'success'
              ? 'Connected!'
              : 'Connection Failed'
      }
      maxWidth="md"
    >
      <div className="space-y-5">
        {/* ─── Step: Consent ─── */}
        {step === 'consent' && (
          <>
            {/* Reconnect warning */}
            {reconnect && (
              <div
                className="flex items-start gap-3 px-4 py-3
                  bg-[var(--warning-subtle)] border border-[var(--warning)]/20
                  rounded-[var(--radius-md)] text-sm"
              >
                <Warning size={18} className="text-[var(--warning)] shrink-0 mt-0.5" />
                <div>
                  <p className="text-[var(--text)] font-medium text-xs">Re-authorization required</p>
                  <p className="text-[var(--text-secondary)] text-xs mt-0.5">
                    Your previous authorization has expired or been revoked. Please authorize again to restore access.
                  </p>
                </div>
              </div>
            )}

            {/* Integration info */}
            <div className="flex items-center gap-3 p-3 bg-[var(--surface-2)] rounded-xl">
              <div className="w-10 h-10 rounded-xl bg-[var(--accent-subtle)] flex items-center justify-center">
                <ShieldCheck size={22} weight="fill" className="text-[var(--accent-text)]" />
              </div>
              <div>
                <p className="text-sm font-semibold text-[var(--text)]">{integration.name}</p>
                <p className="text-xs text-[var(--text-dim)]">{integration.description}</p>
              </div>
            </div>

            {/* PKCE indicator */}
            {provider?.use_pkce && (
              <div className="flex items-center gap-2 text-[11px] text-[var(--success)]">
                <ShieldCheck size={14} />
                <span>This connection uses PKCE for enhanced security</span>
              </div>
            )}

            {/* Scopes */}
            {loadingProvider ? (
              <div className="flex items-center justify-center py-8">
                <CircleNotch size={24} className="text-[var(--text-dim)] animate-spin" />
              </div>
            ) : scopeInfos.length > 0 ? (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <h3 className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wider">
                    Permissions requested
                  </h3>
                  <span className="text-[11px] text-[var(--text-dim)]">
                    {selectedScopes.size} of {scopeInfos.length} selected
                  </span>
                </div>

                {/* Grouped by category */}
                {Object.entries(scopesByCategory).map(([cat, scopes]) => (
                  <div key={cat} className="space-y-1.5">
                    <div className="flex items-center gap-2">
                      <span
                        className="w-1.5 h-1.5 rounded-full"
                        style={{ backgroundColor: categoryColor(cat as ScopeInfo['category']) }}
                      />
                      <span className="text-[11px] font-medium text-[var(--text-dim)] uppercase tracking-wider">
                        {categoryLabel(cat as ScopeInfo['category'])}
                      </span>
                    </div>

                    {scopes.map((scope) => {
                      const isSelected = selectedScopes.has(scope.id)
                      return (
                        <button
                          key={scope.id}
                          onClick={() => toggleScope(scope.id)}
                          className={`
                            w-full flex items-center gap-3 px-3 py-2.5
                            rounded-lg border text-left transition-all cursor-pointer
                            ${isSelected
                              ? 'bg-[var(--accent-subtle)] border-[var(--accent)]/30'
                              : 'bg-[var(--surface)] border-[var(--border-subtle)] hover:border-[var(--border)]'
                            }
                          `}
                        >
                          {/* Checkbox */}
                          <div
                            className={`
                              w-4 h-4 rounded border-2 shrink-0
                              flex items-center justify-center transition-all
                              ${isSelected
                                ? 'bg-[var(--accent)] border-[var(--accent)]'
                                : 'border-[var(--border)] bg-transparent'
                              }
                            `}
                          >
                            {isSelected && (
                              <svg width="10" height="8" viewBox="0 0 10 8" fill="none">
                                <path d="M1 4L3.5 6.5L9 1" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                              </svg>
                            )}
                          </div>

                          {/* Icon */}
                          <span
                            className="shrink-0"
                            style={{ color: categoryColor(scope.category) }}
                          >
                            {scope.icon}
                          </span>

                          {/* Label + description */}
                          <div className="min-w-0 flex-1">
                            <p className="text-xs font-medium text-[var(--text)]">
                              {scope.label}
                            </p>
                            <p className="text-[11px] text-[var(--text-dim)] truncate">
                              {scope.description}
                            </p>
                          </div>

                          {/* Category badge */}
                          <Badge
                            variant={
                              scope.category === 'read' ? 'success'
                                : scope.category === 'delete' ? 'error'
                                  : scope.category === 'admin' ? 'warning'
                                    : 'accent'
                            }
                          >
                            {categoryLabel(scope.category)}
                          </Badge>
                        </button>
                      )
                    })}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-sm text-[var(--text-dim)] text-center py-4">
                This integration will request access to your account.
              </p>
            )}

            {/* Privacy note */}
            <div className="text-[11px] text-[var(--text-dim)] bg-[var(--surface-2)] rounded-lg px-3 py-2">
              <ShieldCheck size={12} className="inline mr-1 -mt-0.5" />
              A popup window will open to authorize with {integration.name}. Your credentials are
              never shared with Operator OS. You can revoke access at any time.
            </div>

            {/* Actions */}
            <div className="flex items-center justify-end gap-3 pt-1">
              <Button variant="ghost" onClick={onClose}>Cancel</Button>
              <Button
                variant="primary"
                icon={reconnect ? <ArrowsClockwise size={15} /> : <ArrowSquareOut size={15} />}
                onClick={handleAuthorize}
                disabled={scopeInfos.length > 0 && selectedScopes.size === 0}
              >
                {reconnect ? 'Re-authorize' : 'Authorize'}
              </Button>
            </div>
          </>
        )}

        {/* ─── Step: Authorizing ─── */}
        {step === 'authorizing' && (
          <div className="text-center py-8 space-y-4">
            <CircleNotch
              size={40}
              weight="bold"
              className="mx-auto text-[var(--accent-text)] animate-spin"
            />
            <div>
              <p className="text-sm font-medium text-[var(--text)]">
                Waiting for authorization…
              </p>
              <p className="text-xs text-[var(--text-dim)] mt-1">
                Complete the sign-in in the popup window.
                <br />
                Don't see it? Check if popups are blocked.
              </p>
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setStep('error')
                setErrorMessage('Authorization was cancelled.')
              }}
            >
              Cancel
            </Button>
          </div>
        )}

        {/* ─── Step: Success ─── */}
        {step === 'success' && (
          <div className="text-center py-8 space-y-4">
            <CheckCircle
              size={48}
              weight="fill"
              className="mx-auto text-[var(--success)] animate-scale-in"
            />
            <div>
              <p className="text-sm font-medium text-[var(--text)]">
                {resultMessage}
              </p>
              <p className="text-xs text-[var(--text-dim)] mt-1">
                You can now use {integration.name} tools in your agents.
              </p>
            </div>
          </div>
        )}

        {/* ─── Step: Error ─── */}
        {step === 'error' && (
          <div className="text-center py-8 space-y-4">
            <XCircle
              size={48}
              weight="fill"
              className="mx-auto text-[var(--error)] animate-scale-in"
            />
            <div>
              <p className="text-sm font-medium text-[var(--text)]">
                Authorization failed
              </p>
              <p className="text-xs text-[var(--error)] mt-1">
                {errorMessage}
              </p>
            </div>
            <div className="flex items-center justify-center gap-3">
              <Button variant="ghost" onClick={onClose}>Close</Button>
              <Button variant="primary" onClick={handleRetry}>
                Try Again
              </Button>
            </div>
          </div>
        )}
      </div>
    </Modal>
  )
})
