// ============================================================================
// Operator OS — Skip to Content Link
// Visible only on keyboard focus. Allows keyboard users to bypass navigation.
// ============================================================================

export function SkipToContent() {
  return (
    <a
      href="#main-content"
      className="sr-only focus:not-sr-only focus:fixed focus:top-2 focus:left-2 focus:z-[200]
        focus:px-4 focus:py-2 focus:rounded-lg focus:bg-accent focus:text-white
        focus:text-sm focus:font-semibold focus:shadow-lg focus:outline-none
        focus:ring-2 focus:ring-white focus:ring-offset-2 focus:ring-offset-bg"
    >
      Skip to content
    </a>
  )
}
