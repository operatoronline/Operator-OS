// ============================================================================
// Operator OS — useTheme Hook
// Reactive theme management with system preference detection.
// Wraps uiStore for convenience and adds OS-level media query listener.
// ============================================================================

import { useEffect } from 'react'
import { useUIStore } from '../stores/uiStore'

/**
 * useTheme — convenience hook for theme state + system preference sync.
 *
 * Features:
 *   - Exposes { theme, isDark, isLight, toggleTheme, setTheme }
 *   - Listens for `prefers-color-scheme` changes (user toggles OS dark mode)
 *     and follows them **only if** no explicit preference was saved.
 */
export function useTheme() {
  const theme = useUIStore((s) => s.theme)
  const toggleTheme = useUIStore((s) => s.toggleTheme)
  const setTheme = useUIStore((s) => s.setTheme)

  // ── Listen for OS-level theme changes ──
  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)')

    const handler = (e: MediaQueryListEvent) => {
      // Only follow OS if user hasn't explicitly chosen a theme
      const explicit = localStorage.getItem('os-theme')
      if (!explicit) {
        setTheme(e.matches ? 'dark' : 'light')
      }
    }

    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [setTheme])

  return {
    theme,
    isDark: theme === 'dark',
    isLight: theme === 'light',
    toggleTheme,
    setTheme,
    /** Clear explicit preference and follow system */
    followSystem: () => {
      localStorage.removeItem('os-theme')
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      setTheme(prefersDark ? 'dark' : 'light')
    },
  }
}
