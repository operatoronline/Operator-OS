import { Plugs } from '@phosphor-icons/react'

export function IntegrationsPage() {
  return (
    <div className="h-full flex flex-col items-center justify-center text-text-dim">
      <Plugs size={48} weight="thin" className="mb-4 text-accent-text" />
      <h2 className="text-lg font-semibold text-text mb-1">Integrations</h2>
      <p className="text-sm">Integration marketplace coming in Phase 4</p>
    </div>
  )
}
