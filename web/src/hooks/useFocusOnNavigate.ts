// ============================================================================
// Operator OS — useFocusOnNavigate
// Manages focus on route changes for accessibility.
// Moves focus to main content area (or page heading) after navigation,
// so screen readers announce the new page context.
// ============================================================================

import { useEffect, useRef } from 'react'
import { useLocation } from 'react-router-dom'

/**
 * On route change: focus the main content heading (h1) or a designated
 * target. Also announces the page title to screen readers via a live region.
 */
export function useFocusOnNavigate() {
  const location = useLocation()
  const prevPathRef = useRef(location.pathname)

  useEffect(() => {
    // Skip initial mount — only fire on actual navigation
    if (prevPathRef.current === location.pathname) return
    prevPathRef.current = location.pathname

    // Small delay to let the new page render
    requestAnimationFrame(() => {
      // Try to focus the main heading first
      const heading = document.querySelector('main h1, main [data-page-title]') as HTMLElement | null
      if (heading) {
        heading.setAttribute('tabindex', '-1')
        heading.focus({ preventScroll: true })
        return
      }

      // Fallback: focus the main element
      const main = document.querySelector('main') as HTMLElement | null
      if (main) {
        main.setAttribute('tabindex', '-1')
        main.focus({ preventScroll: true })
      }
    })
  }, [location.pathname])
}
