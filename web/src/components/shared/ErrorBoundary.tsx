// ============================================================================
// Operator OS — Error Boundary
// Global error boundary with recovery UI. Catches render errors in children.
// ============================================================================

import { Component, type ErrorInfo, type ReactNode } from 'react'
import { ArrowCounterClockwise, Warning } from '@phosphor-icons/react'

interface Props {
  children: ReactNode
  fallback?: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
  errorInfo: ErrorInfo | null
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null, errorInfo: null }
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.setState({ errorInfo })
    // Log to console in development
    console.error('[ErrorBoundary]', error, errorInfo)
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null, errorInfo: null })
  }

  handleReload = () => {
    window.location.reload()
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback
      }

      return (
        <div className="min-h-screen flex items-center justify-center bg-[var(--bg)] p-6">
          <div className="max-w-md w-full text-center space-y-6">
            {/* Icon */}
            <div className="mx-auto w-16 h-16 rounded-2xl bg-[oklch(0.85_0.15_25)] flex items-center justify-center">
              <Warning size={32} weight="fill" className="text-[oklch(0.55_0.2_25)]" />
            </div>

            {/* Heading */}
            <div className="space-y-2">
              <h1 className="text-xl font-semibold text-[var(--text)]">
                Something went wrong
              </h1>
              <p className="text-sm text-[var(--text-dim)]">
                An unexpected error occurred. You can try recovering or reload the page.
              </p>
            </div>

            {/* Error details (collapsed) */}
            {this.state.error && (
              <details className="text-left rounded-xl bg-[var(--surface-1)] border border-[var(--border)] p-4">
                <summary className="text-xs font-medium text-[var(--text-dim)] cursor-pointer select-none">
                  Error details
                </summary>
                <pre className="mt-3 text-xs text-[var(--text-dim)] font-mono whitespace-pre-wrap break-words overflow-x-auto max-h-48 overflow-y-auto">
                  {this.state.error.message}
                  {this.state.errorInfo?.componentStack && (
                    <>
                      {'\n\nComponent stack:'}
                      {this.state.errorInfo.componentStack}
                    </>
                  )}
                </pre>
              </details>
            )}

            {/* Actions */}
            <div className="flex items-center justify-center gap-3">
              <button
                onClick={this.handleReset}
                className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium
                  bg-[var(--accent)] text-white hover:opacity-90 transition-opacity focus-ring"
              >
                <ArrowCounterClockwise size={16} weight="bold" />
                Try again
              </button>
              <button
                onClick={this.handleReload}
                className="inline-flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium
                  bg-[var(--surface-2)] text-[var(--text)] hover:bg-[var(--surface-3)] transition-colors focus-ring"
              >
                Reload page
              </button>
            </div>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}
