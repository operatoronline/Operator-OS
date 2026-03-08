// ============================================================================
// Operator OS — useMobile hook
// React hook that tracks mobile/desktop breakpoint (md: 768px).
// Uses matchMedia with live listener — no resize polling.
// ============================================================================

import { useSyncExternalStore } from 'react'

const MQ = '(max-width: 767px)'

function subscribe(cb: () => void): () => void {
  const mql = window.matchMedia(MQ)
  mql.addEventListener('change', cb)
  return () => mql.removeEventListener('change', cb)
}

function getSnapshot(): boolean {
  return window.matchMedia(MQ).matches
}

function getServerSnapshot(): boolean {
  return false // SSR: assume desktop
}

/**
 * Returns `true` when the viewport is below the `md` breakpoint (768px).
 * Re-renders only when the breakpoint is crossed.
 */
export function useMobile(): boolean {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)
}
