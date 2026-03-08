// ============================================================================
// Operator OS — Page Spinner
// Full-page loading spinner for Suspense fallback during code-split loading.
// ============================================================================

export function PageSpinner() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--bg)]">
      <div className="flex flex-col items-center gap-4">
        {/* Spinner */}
        <div
          className="w-8 h-8 rounded-full border-2 border-[var(--border)] border-t-[var(--accent)] animate-spin"
          role="status"
          aria-label="Loading page"
        />
        <span className="text-sm text-[var(--text-dim)] sr-only">Loading…</span>
      </div>
    </div>
  )
}
