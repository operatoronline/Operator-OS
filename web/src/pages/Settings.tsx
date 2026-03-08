// ============================================================================
// Operator OS — Settings Page
// Profile, theme, notifications, API keys, and GDPR data management.
// ============================================================================

import { useState, useCallback } from 'react'
import {
  User,
  PaintBrush,
  Bell,
  Key,
  ShieldCheck,
} from '@phosphor-icons/react'
import { ProfileForm } from '../components/settings/ProfileForm'
import { ThemePreference } from '../components/settings/ThemePreference'
import { NotificationSettings } from '../components/settings/NotificationSettings'
import { ApiKeyManager } from '../components/settings/ApiKeyManager'
import { GDPRPanel } from '../components/settings/GDPRPanel'

// ---------------------------------------------------------------------------
// Tab config
// ---------------------------------------------------------------------------

type TabId = 'profile' | 'appearance' | 'notifications' | 'api-keys' | 'privacy'

const tabs: { id: TabId; label: string; icon: typeof User }[] = [
  { id: 'profile', label: 'Profile', icon: User },
  { id: 'appearance', label: 'Appearance', icon: PaintBrush },
  { id: 'notifications', label: 'Notifications', icon: Bell },
  { id: 'api-keys', label: 'API Keys', icon: Key },
  { id: 'privacy', label: 'Data & Privacy', icon: ShieldCheck },
]

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function SettingsPage() {
  const [activeTab, setActiveTab] = useState<TabId>('profile')

  const handleTabChange = useCallback((id: TabId) => {
    setActiveTab(id)
  }, [])

  return (
    <div className="h-full flex flex-col md:flex-row">
      {/* ─── Sidebar / Tab Nav ─── */}
      <nav className="shrink-0 md:w-56 border-b md:border-b-0 md:border-r border-[var(--border-subtle)]">
        {/* Mobile: horizontal scroll tabs */}
        <div className="flex md:hidden gap-1 px-4 py-3 overflow-x-auto scrollbar-none">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => handleTabChange(id)}
              className={`
                flex items-center gap-1.5 px-3 py-2 rounded-lg text-[13px] font-medium
                whitespace-nowrap transition-colors cursor-pointer
                ${activeTab === id
                  ? 'bg-accent-subtle text-accent-text'
                  : 'text-[var(--text-dim)] hover:text-[var(--text)] hover:bg-[var(--surface-2)]'
                }
              `}
            >
              <Icon size={16} weight={activeTab === id ? 'fill' : 'regular'} />
              {label}
            </button>
          ))}
        </div>

        {/* Desktop: vertical sidebar */}
        <div className="hidden md:flex flex-col gap-0.5 p-3">
          <h2 className="text-xs font-medium text-[var(--text-dim)] uppercase tracking-wider px-3 py-2">
            Settings
          </h2>
          {tabs.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => handleTabChange(id)}
              className={`
                flex items-center gap-2.5 px-3 py-2.5 rounded-[var(--radius-md)]
                text-[13px] font-medium transition-colors cursor-pointer text-left w-full
                ${activeTab === id
                  ? 'bg-accent-subtle text-accent-text'
                  : 'text-[var(--text-secondary)] hover:text-[var(--text)] hover:bg-[var(--surface-2)]'
                }
              `}
            >
              <Icon size={18} weight={activeTab === id ? 'fill' : 'regular'} />
              {label}
            </button>
          ))}
        </div>
      </nav>

      {/* ─── Content ─── */}
      <div className="flex-1 overflow-y-auto scroll-touch">
        <div className="max-w-2xl mx-auto p-4 sm:p-6 md:p-8">
          {activeTab === 'profile' && <ProfileForm />}
          {activeTab === 'appearance' && <ThemePreference />}
          {activeTab === 'notifications' && <NotificationSettings />}
          {activeTab === 'api-keys' && <ApiKeyManager />}
          {activeTab === 'privacy' && <GDPRPanel />}
        </div>
      </div>
    </div>
  )
}
