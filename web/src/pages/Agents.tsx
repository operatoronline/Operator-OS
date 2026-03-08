import { Robot } from '@phosphor-icons/react'

export function AgentsPage() {
  return (
    <div className="h-full flex flex-col items-center justify-center text-text-dim">
      <Robot size={48} weight="thin" className="mb-4 text-accent-text" />
      <h2 className="text-lg font-semibold text-text mb-1">Agents</h2>
      <p className="text-sm">Agent management coming in Phase 3</p>
    </div>
  )
}
