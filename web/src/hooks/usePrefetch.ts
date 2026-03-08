// ============================================================================
// Operator OS — Route Prefetch Hook
// Prefetches lazy-loaded route chunks on hover/focus for instant navigation.
// ============================================================================

const prefetched = new Set<string>()

// Map route paths to their dynamic import functions
const routeImports: Record<string, () => Promise<unknown>> = {
  '/chat': () => import('../pages/Chat'),
  '/agents': () => import('../pages/Agents'),
  '/integrations': () => import('../pages/Integrations'),
  '/billing': () => import('../pages/Billing'),
  '/settings': () => import('../pages/Settings'),
  '/admin': () => import('../pages/Admin'),
}

/**
 * Returns an object with onMouseEnter/onFocus handlers that prefetch
 * the chunk for a given route path on hover or focus.
 */
export function usePrefetch(path: string) {
  const prefetch = () => {
    if (prefetched.has(path)) return
    const importFn = routeImports[path]
    if (importFn) {
      prefetched.add(path)
      importFn().catch(() => {
        // Failed to prefetch — remove from set so it can retry
        prefetched.delete(path)
      })
    }
  }

  return { onMouseEnter: prefetch, onFocus: prefetch }
}
