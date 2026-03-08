// ============================================================================
// Operator OS — Collapsible Sidebar
// Desktop navigation sidebar with collapse/expand toggle.
// ============================================================================

import { NavLink } from 'react-router-dom'
import {
  ChatCircle,
  Robot,
  Plugs,
  CreditCard,
  Gear,
  ShieldCheck,
  CaretLeft,
  CaretRight,
} from '@phosphor-icons/react'
import { useUIStore } from '../../stores/uiStore'
import { usePrefetch } from '../../hooks/usePrefetch'

const navItems = [
  { to: '/chat', label: 'Chat', icon: ChatCircle },
  { to: '/agents', label: 'Agents', icon: Robot },
  { to: '/integrations', label: 'Integrations', icon: Plugs },
  { to: '/billing', label: 'Billing', icon: CreditCard },
  { to: '/settings', label: 'Settings', icon: Gear },
  { to: '/admin', label: 'Admin', icon: ShieldCheck },
]

// Separate component so usePrefetch hook can be called per-item
function SidebarNavItem({ item, sidebarOpen }: { item: typeof navItems[number]; sidebarOpen: boolean }) {
  const prefetchProps = usePrefetch(item.to)
  const Icon = item.icon

  return (
    <NavLink
      to={item.to}
      aria-label={!sidebarOpen ? item.label : undefined}
      {...prefetchProps}
      className={({ isActive }) =>
        `group flex items-center gap-3 rounded-lg transition-all duration-150 select-none relative focus-ring ${
          sidebarOpen ? 'px-3 py-2.5' : 'px-0 py-2.5 justify-center'
        } ${
          isActive
            ? 'bg-surface-2 text-text shadow-[inset_0_0_0_1px_var(--border)]'
            : 'text-text-dim hover:text-text-secondary hover:bg-surface-2/50'
        }`
      }
    >
      {({ isActive }) => (
        <>
          <Icon
            size={20}
            weight={isActive ? 'fill' : 'regular'}
            className="shrink-0"
          />
          {sidebarOpen && (
            <span className="text-[13px] font-medium whitespace-nowrap overflow-hidden">
              {item.label}
            </span>
          )}
          {!sidebarOpen && (
            <span className="absolute left-full ml-2 px-2 py-1 rounded-md bg-surface-3 text-text text-xs font-medium whitespace-nowrap opacity-0 pointer-events-none group-hover:opacity-100 transition-opacity duration-150 z-50 shadow-[0_2px_8px_var(--glass-shadow)]">
              {item.label}
            </span>
          )}
        </>
      )}
    </NavLink>
  )
}

export function Sidebar() {
  const { sidebarOpen, toggleSidebar } = useUIStore()

  return (
    <aside
      aria-label="Sidebar navigation"
      className={`hidden md:flex flex-col shrink-0 h-full bg-surface border-r border-border transition-[width] duration-200 ease-out ${
        sidebarOpen ? 'w-52' : 'w-16'
      }`}
    >
      {/* ─── Logo / brand ─── */}
      <div className="flex items-center h-14 px-4 border-b border-border-subtle shrink-0">
        <div className="w-7 h-7 rounded-lg bg-accent flex items-center justify-center shrink-0">
          <span className="text-white text-xs font-bold leading-none">OS</span>
        </div>
        {sidebarOpen && (
          <span className="ml-3 text-sm font-semibold text-text whitespace-nowrap overflow-hidden animate-fade-slide">
            Operator OS
          </span>
        )}
      </div>

      {/* ─── Nav items ─── */}
      <nav className="flex-1 flex flex-col gap-0.5 px-2 py-3 overflow-y-auto">
        {navItems.map((item) => (
          <SidebarNavItem key={item.to} item={item} sidebarOpen={sidebarOpen} />
        ))}
      </nav>

      {/* ─── Collapse toggle ─── */}
      <div className="px-2 pb-3 pt-1 border-t border-border-subtle shrink-0">
        <button
          onClick={toggleSidebar}
          className={`flex items-center gap-3 w-full rounded-lg py-2.5 text-text-dim hover:text-text-secondary hover:bg-surface-2/50 transition-all duration-150 ${
            sidebarOpen ? 'px-3' : 'px-0 justify-center'
          }`}
          aria-label={sidebarOpen ? 'Collapse sidebar' : 'Expand sidebar'}
        >
          {sidebarOpen ? <CaretLeft size={18} /> : <CaretRight size={18} />}
          {sidebarOpen && (
            <span className="text-[13px] font-medium">Collapse</span>
          )}
        </button>
      </div>
    </aside>
  )
}
