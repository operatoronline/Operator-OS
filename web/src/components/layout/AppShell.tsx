import { Outlet, NavLink, useLocation } from 'react-router-dom'
import {
  ChatCircle,
  Robot,
  Plugs,
  CreditCard,
  Gear,
  ShieldCheck,
  Sun,
  Moon,
} from '@phosphor-icons/react'
import { useUIStore } from '../../stores/uiStore'

const navItems = [
  { to: '/chat', label: 'Chat', icon: ChatCircle },
  { to: '/agents', label: 'Agents', icon: Robot },
  { to: '/integrations', label: 'Integrations', icon: Plugs },
  { to: '/billing', label: 'Billing', icon: CreditCard },
  { to: '/settings', label: 'Settings', icon: Gear },
  { to: '/admin', label: 'Admin', icon: ShieldCheck },
]

export function AppShell() {
  const { theme, toggleTheme } = useUIStore()
  const location = useLocation()

  return (
    <div className="h-full flex flex-col">
      {/* ─── Floating Nav (desktop) ─── */}
      <nav className="fixed top-4 left-1/2 -translate-x-1/2 z-80 hidden md:flex items-center gap-1 bg-glass-bg backdrop-blur-[20px] saturate-[1.4] border border-glass-border rounded-full px-1 py-1 shadow-[0_4px_24px_var(--glass-shadow)]">
        {navItems.map((item) => {
          const Icon = item.icon
          const isActive = location.pathname.startsWith(item.to)
          return (
            <NavLink
              key={item.to}
              to={item.to}
              className={`flex items-center gap-2 px-4 py-2 rounded-full text-[13px] font-medium transition-all duration-200 select-none whitespace-nowrap ${
                isActive
                  ? 'bg-surface-2 text-text shadow-[0_1px_4px_var(--glass-shadow)]'
                  : 'text-text-dim hover:text-text-secondary'
              }`}
            >
              <Icon size={16} weight={isActive ? 'fill' : 'regular'} />
              {item.label}
            </NavLink>
          )
        })}

        <div className="w-px h-5 bg-border mx-1 shrink-0" />

        <button
          onClick={toggleTheme}
          className="flex items-center justify-center p-2 rounded-full text-text-dim hover:text-text-secondary transition-colors duration-200"
          aria-label="Toggle theme"
        >
          {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
        </button>
      </nav>

      {/* ─── Main content ─── */}
      <main className="flex-1 relative overflow-hidden">
        <Outlet />
      </main>

      {/* ─── Bottom tabs (mobile) ─── */}
      <nav className="md:hidden fixed bottom-0 left-0 right-0 z-80 bg-glass-bg backdrop-blur-[20px] saturate-[1.4] border-t border-glass-border pb-[var(--safe-b)]">
        <div className="flex items-center justify-around px-2 py-2">
          {navItems.slice(0, 5).map((item) => {
            const Icon = item.icon
            const isActive = location.pathname.startsWith(item.to)
            return (
              <NavLink
                key={item.to}
                to={item.to}
                className={`flex flex-col items-center gap-0.5 px-3 py-1 rounded-lg text-[10px] font-medium transition-colors duration-200 ${
                  isActive ? 'text-accent-text' : 'text-text-dim'
                }`}
              >
                <Icon size={20} weight={isActive ? 'fill' : 'regular'} />
                {item.label}
              </NavLink>
            )
          })}
        </div>
      </nav>
    </div>
  )
}
