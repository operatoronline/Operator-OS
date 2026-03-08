// ============================================================================
// Operator OS — Bottom Tab Navigation (Mobile)
// Fixed bottom tabs for mobile devices. Proper safe area insets, 44px min
// touch targets, glass morphism background matching the legacy pill nav.
// ============================================================================

import { NavLink } from 'react-router-dom'
import {
  ChatCircle,
  Robot,
  Plugs,
  CreditCard,
  Gear,
} from '@phosphor-icons/react'

const tabs = [
  { to: '/chat', label: 'Chat', icon: ChatCircle },
  { to: '/agents', label: 'Agents', icon: Robot },
  { to: '/integrations', label: 'Integrations', icon: Plugs },
  { to: '/billing', label: 'Billing', icon: CreditCard },
  { to: '/settings', label: 'Settings', icon: Gear },
]

export function BottomTabs() {
  return (
    <nav
      aria-label="Main navigation"
      className="md:hidden fixed bottom-0 left-0 right-0 z-80
        bg-glass-bg backdrop-blur-[20px] saturate-[1.4]
        border-t border-glass-border"
      style={{ paddingBottom: 'var(--safe-b)' }}
    >
      <div className="flex items-center justify-around px-1">
        {tabs.map((item) => {
          const Icon = item.icon
          return (
            <NavLink
              key={item.to}
              to={item.to}
              aria-label={item.label}
              className={({ isActive }) =>
                `flex flex-col items-center justify-center gap-0.5
                 min-w-[44px] min-h-[44px] px-2 py-1
                 rounded-lg text-[10px] font-medium
                 transition-colors duration-200 select-none
                 active:scale-95 active:opacity-80
                 focus-ring
                 ${isActive ? 'text-accent-text' : 'text-text-dim'}`
              }
            >
              {({ isActive }) => (
                <>
                  <Icon size={22} weight={isActive ? 'fill' : 'regular'} aria-hidden="true" />
                  <span className="leading-none" aria-hidden="true">{item.label}</span>
                </>
              )}
            </NavLink>
          )
        })}
      </div>
    </nav>
  )
}
