// ============================================================================
// Operator OS — Top Bar
// Persistent header with page title, theme toggle, and user menu.
// ============================================================================

import { useState, useRef, useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import {
  Sun,
  Moon,
  SignOut,
  User,
  List,
  CaretDown,
} from '@phosphor-icons/react'
import { useTheme } from '../../hooks/useTheme'
import { useAuthStore } from '../../stores/authStore'
import { useUIStore } from '../../stores/uiStore'
import { RateLimitIndicator } from '../shared/RateLimitIndicator'

const pageTitles: Record<string, string> = {
  '/chat': 'Chat',
  '/agents': 'Agents',
  '/integrations': 'Integrations',
  '/billing': 'Billing',
  '/settings': 'Settings',
  '/admin': 'Admin',
}

export function TopBar() {
  const { isDark, toggleTheme } = useTheme()
  const { user, logout } = useAuthStore()
  const location = useLocation()
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  const title = pageTitles[location.pathname] || 'Operator OS'

  // Close menu on outside click or Escape key
  useEffect(() => {
    if (!menuOpen) return
    const handleClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setMenuOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    document.addEventListener('keydown', handleKey)
    return () => {
      document.removeEventListener('mousedown', handleClick)
      document.removeEventListener('keydown', handleKey)
    }
  }, [menuOpen])

  return (
    <header
      role="banner"
      className="flex items-center justify-between h-14 px-4 md:px-6 border-b border-border-subtle bg-surface/50 backdrop-blur-sm shrink-0 z-40"
      style={{ paddingTop: 'var(--safe-t)' }}
    >
      {/* ─── Left: mobile menu + page title ─── */}
      <div className="flex items-center gap-3">
        {/* Mobile hamburger — toggles sidebar (handled via mobile overlay in future) */}
        <button
          className="md:hidden flex items-center justify-center p-1.5 rounded-lg text-text-dim hover:text-text-secondary hover:bg-surface-2/50 transition-colors focus-ring"
          aria-label="Open navigation menu"
          onClick={() => useUIStore.getState().toggleSidebar()}
        >
          <List size={20} />
        </button>

        <h1 className="text-[15px] font-semibold text-text">{title}</h1>
      </div>

      {/* ─── Right: theme toggle + user menu ─── */}
      <div className="flex items-center gap-1">
        {/* Rate limit indicator */}
        <RateLimitIndicator />

        {/* Theme toggle */}
        <button
          onClick={toggleTheme}
          className="flex items-center justify-center w-9 h-9 rounded-lg text-text-dim hover:text-text-secondary hover:bg-surface-2/50 transition-colors duration-150 focus-ring"
          aria-label={`Switch to ${isDark ? 'light' : 'dark'} mode`}
        >
          {isDark ? <Sun size={18} /> : <Moon size={18} />}
        </button>

        {/* User menu */}
        <div className="relative" ref={menuRef}>
          <button
            onClick={() => setMenuOpen(!menuOpen)}
            className="flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-text-dim hover:text-text-secondary hover:bg-surface-2/50 transition-colors duration-150 focus-ring"
            aria-label="User menu"
            aria-expanded={menuOpen}
            aria-haspopup="true"
          >
            <div className="w-7 h-7 rounded-full bg-accent-subtle flex items-center justify-center">
              <User size={14} weight="bold" className="text-accent-text" />
            </div>
            {user && (
              <span className="hidden sm:block text-[13px] font-medium text-text max-w-[120px] truncate">
                {user.display_name || user.email}
              </span>
            )}
            <CaretDown
              size={12}
              className={`transition-transform duration-150 ${menuOpen ? 'rotate-180' : ''}`}
            />
          </button>

          {/* Dropdown */}
          {menuOpen && (
            <div
              role="menu"
              aria-label="User actions"
              className="absolute right-0 top-full mt-1.5 w-56 bg-surface border border-border rounded-xl shadow-[0_8px_32px_var(--glass-shadow)] overflow-hidden animate-fade-slide z-50"
            >
              {/* User info */}
              {user && (
                <div className="px-4 py-3 border-b border-border-subtle">
                  <p className="text-sm font-medium text-text truncate">
                    {user.display_name || user.email}
                  </p>
                  <p className="text-xs text-text-dim truncate mt-0.5">
                    {user.email}
                  </p>
                </div>
              )}

              {/* Actions */}
              <div className="py-1.5">
                <button
                  role="menuitem"
                  onClick={() => {
                    setMenuOpen(false)
                    logout()
                  }}
                  className="flex items-center gap-3 w-full px-4 py-2.5 text-[13px] font-medium text-text-dim hover:text-error hover:bg-error-subtle/50 transition-colors duration-150 focus-ring"
                >
                  <SignOut size={16} aria-hidden="true" />
                  Sign out
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}
