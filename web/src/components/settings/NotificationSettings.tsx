// ============================================================================
// Operator OS — NotificationSettings
// Toggle email and push notification preferences.
// ============================================================================

import { useEffect, useCallback } from 'react'
import { Bell, EnvelopeSimple, CreditCard, ShieldCheck, Megaphone } from '@phosphor-icons/react'
import { useSettingsStore } from '../../stores/settingsStore'

interface ToggleRowProps {
  icon: React.ReactNode
  label: string
  description: string
  checked: boolean
  onChange: (checked: boolean) => void
  disabled?: boolean
}

function ToggleRow({ icon, label, description, checked, onChange, disabled }: ToggleRowProps) {
  return (
    <label
      className={`
        flex items-center gap-4 p-4 rounded-[var(--radius-md)]
        border border-[var(--border-subtle)] bg-[var(--surface-2)]
        hover:border-[var(--border-hover)] transition-colors
        ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}
      `}
    >
      <div className="shrink-0 text-[var(--text-dim)]">{icon}</div>
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-[var(--text)]">{label}</div>
        <div className="text-[11px] text-[var(--text-dim)] mt-0.5">{description}</div>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={`
          relative shrink-0 w-10 h-[22px] rounded-full transition-colors duration-200
          ${checked ? 'bg-accent' : 'bg-[var(--surface-3)]'}
          ${disabled ? 'cursor-not-allowed' : 'cursor-pointer'}
        `}
      >
        <span
          className={`
            absolute top-[3px] w-4 h-4 rounded-full bg-white shadow-sm
            transition-transform duration-200
            ${checked ? 'translate-x-[22px]' : 'translate-x-[3px]'}
          `}
        />
      </button>
    </label>
  )
}

export function NotificationSettings() {
  const {
    notifications,
    notificationsLoading,
    notificationsError,
    fetchNotifications,
    updateNotifications,
  } = useSettingsStore()

  useEffect(() => {
    if (!notifications) fetchNotifications()
  }, [notifications, fetchNotifications])

  const handleToggle = useCallback(
    (key: string, value: boolean) => {
      updateNotifications({ [key]: value })
    },
    [updateNotifications],
  )

  const prefs = notifications

  return (
    <div>
      <div className="flex items-center gap-3 mb-5">
        <div className="w-9 h-9 rounded-xl bg-accent-subtle flex items-center justify-center">
          <Bell size={18} weight="duotone" className="text-accent-text" />
        </div>
        <div>
          <h3 className="text-[15px] font-semibold text-[var(--text)]">Notifications</h3>
          <p className="text-xs text-[var(--text-dim)]">Choose what you want to be notified about</p>
        </div>
      </div>

      {notificationsError && (
        <p className="text-xs text-error mb-4" role="alert">{notificationsError}</p>
      )}

      {notificationsLoading && !prefs ? (
        <div className="space-y-3">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-[72px] rounded-[var(--radius-md)] bg-[var(--surface-2)] animate-pulse" />
          ))}
        </div>
      ) : (
        <div className="space-y-3">
          <p className="text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider mb-2">
            Email notifications
          </p>

          <ToggleRow
            icon={<CreditCard size={20} weight="duotone" />}
            label="Billing & payments"
            description="Invoices, payment failures, and plan changes"
            checked={prefs?.email_billing ?? true}
            onChange={(v) => handleToggle('email_billing', v)}
          />

          <ToggleRow
            icon={<ShieldCheck size={20} weight="duotone" />}
            label="Security alerts"
            description="New sign-ins, password changes, and suspicious activity"
            checked={prefs?.email_security ?? true}
            onChange={(v) => handleToggle('email_security', v)}
          />

          <ToggleRow
            icon={<Megaphone size={20} weight="duotone" />}
            label="Product updates"
            description="New features, tips, and platform announcements"
            checked={prefs?.email_product ?? false}
            onChange={(v) => handleToggle('email_product', v)}
          />

          <div className="pt-3">
            <p className="text-xs font-medium text-[var(--text-secondary)] uppercase tracking-wider mb-2">
              Push notifications
            </p>

            <ToggleRow
              icon={<EnvelopeSimple size={20} weight="duotone" />}
              label="Browser push"
              description="Real-time notifications in your browser"
              checked={prefs?.push_enabled ?? false}
              onChange={(v) => handleToggle('push_enabled', v)}
            />
          </div>
        </div>
      )}
    </div>
  )
}
