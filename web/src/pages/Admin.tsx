import { ShieldCheck } from '@phosphor-icons/react'

export function AdminPage() {
  return (
    <div className="h-full flex flex-col items-center justify-center text-text-dim">
      <ShieldCheck size={48} weight="thin" className="mb-4 text-accent-text" />
      <h2 className="text-lg font-semibold text-text mb-1">Admin</h2>
      <p className="text-sm">Admin panel coming in Phase 5</p>
    </div>
  )
}
