import { CreditCard } from '@phosphor-icons/react'

export function BillingPage() {
  return (
    <div className="h-full flex flex-col items-center justify-center text-text-dim">
      <CreditCard size={48} weight="thin" className="mb-4 text-accent-text" />
      <h2 className="text-lg font-semibold text-text mb-1">Billing</h2>
      <p className="text-sm">Plans & usage coming in Phase 4</p>
    </div>
  )
}
