// ============================================================================
// Operator OS — Route Announcer
// Announces page title changes to screen readers via an aria-live region.
// Visually hidden but read by assistive technology on SPA navigation.
// ============================================================================

import { useEffect, useState } from 'react'
import { useLocation } from 'react-router-dom'

const pageTitles: Record<string, string> = {
  '/chat': 'Chat',
  '/agents': 'Agents',
  '/integrations': 'Integrations',
  '/billing': 'Billing',
  '/settings': 'Settings',
  '/admin': 'Admin',
  '/login': 'Sign In',
  '/register': 'Create Account',
  '/verify': 'Verify Email',
}

export function RouteAnnouncer() {
  const location = useLocation()
  const [announcement, setAnnouncement] = useState('')

  useEffect(() => {
    const title = pageTitles[location.pathname] || 'Operator OS'
    // Update document title
    document.title = `${title} — Operator OS`
    // Announce to screen readers (slight delay so aria-live picks it up)
    const timer = setTimeout(() => {
      setAnnouncement(`Navigated to ${title}`)
    }, 100)
    return () => clearTimeout(timer)
  }, [location.pathname])

  return (
    <div
      role="status"
      aria-live="polite"
      aria-atomic="true"
      className="sr-only"
    >
      {announcement}
    </div>
  )
}
