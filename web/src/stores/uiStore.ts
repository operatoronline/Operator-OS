// ============================================================================
// Operator OS — UI Store
// Zustand store for theme, sidebar state, and UI preferences.
// ============================================================================

import { create } from 'zustand'

type Theme = 'dark' | 'light'

interface UIState {
  theme: Theme
  sidebarOpen: boolean     // Desktop: expanded (true) / collapsed (false). Mobile: overlay open/closed.
  toggleTheme: () => void
  setTheme: (theme: Theme) => void
  toggleSidebar: () => void
  setSidebarOpen: (open: boolean) => void
}

const getInitialTheme = (): Theme => {
  const stored = localStorage.getItem('os-theme')
  if (stored === 'light' || stored === 'dark') return stored
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
}

const getInitialSidebar = (): boolean => {
  const stored = localStorage.getItem('os-sidebar')
  if (stored === 'true' || stored === 'false') return stored === 'true'
  return true // default expanded on desktop
}

let transitionTimer: ReturnType<typeof setTimeout> | undefined

const applyTheme = (theme: Theme) => {
  const root = document.documentElement

  // Enable smooth theme transition
  root.classList.add('theme-transitioning')
  clearTimeout(transitionTimer)
  transitionTimer = setTimeout(() => root.classList.remove('theme-transitioning'), 350)

  root.classList.remove('dark', 'light')
  root.classList.add(theme)
  localStorage.setItem('os-theme', theme)

  // Update meta theme-color for mobile browser chrome
  const meta = document.querySelector('meta[name="theme-color"]')
  if (meta) {
    meta.setAttribute('content', theme === 'dark' ? 'oklch(0.13 0 0)' : 'oklch(0.96 0 0)')
  }

  // Update color-scheme for native form controls
  root.style.colorScheme = theme
}

export const useUIStore = create<UIState>((set) => {
  const initial = getInitialTheme()
  applyTheme(initial)

  return {
    theme: initial,
    sidebarOpen: getInitialSidebar(),
    toggleTheme: () =>
      set((state) => {
        const next = state.theme === 'dark' ? 'light' : 'dark'
        applyTheme(next)
        return { theme: next }
      }),
    setTheme: (theme) => {
      applyTheme(theme)
      set({ theme })
    },
    toggleSidebar: () =>
      set((state) => {
        const next = !state.sidebarOpen
        // Only persist on desktop
        if (window.innerWidth >= 768) {
          localStorage.setItem('os-sidebar', String(next))
        }
        return { sidebarOpen: next }
      }),
    setSidebarOpen: (open) => set({ sidebarOpen: open }),
  }
})
