// ============================================================================
// Operator OS — Bottom Tab Navigation (Mobile)
// Fixed bottom tabs for mobile devices with safe area support.
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
    <nav className="md:hidden fixed bottom-0 left-0 right-0 z-80 bg-glass-bg backdrop-blur-[20px] saturate-[1.4] border-t border-glass-border pb-[var(--safe-b)]">
      <div className="flex items-center justify-around px-2 py-2">
        {tabs.map((item) => {
          const Icon = item.icon
          return (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                `flex flex-col items-center gap-0.5 px-3 py-1 rounded-lg text-[10px] font-medium transition-colors duration-200 ${
                  isActive ? 'text-accent-text' : 'text-text-dim'
                }`
              }
            >
              {({ isActive }) => (
                <>
                  <Icon size={20} weight={isActive ? 'fill' : 'regular'} />
                  {item.label}
                </>
              )}
            </NavLink>
          )
        })}
      </div>
    </nav>
  )
}
