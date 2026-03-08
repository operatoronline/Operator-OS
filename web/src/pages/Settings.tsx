import { Gear } from '@phosphor-icons/react'

export function SettingsPage() {
  return (
    <div className="h-full flex flex-col items-center justify-center text-text-dim">
      <Gear size={48} weight="thin" className="mb-4 text-accent-text" />
      <h2 className="text-lg font-semibold text-text mb-1">Settings</h2>
      <p className="text-sm">User settings coming in Phase 5</p>
    </div>
  )
}
