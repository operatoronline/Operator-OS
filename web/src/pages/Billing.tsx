// ============================================================================
// Operator OS — Billing Page
// Plan comparison, subscription management, and Stripe checkout.
// ============================================================================

import { useEffect, useState, useCallback } from 'react'
import { ArrowClockwise, CheckCircle, XCircle, Warning } from '@phosphor-icons/react'
import { useBillingStore } from '../stores/billingStore'
import { PlanCard, CurrentSubscription, PlanChangePreview, IntervalToggle } from '../components/billing'
import { Button } from '../components/shared'
import type { BillingInterval } from '../types/api'

export function BillingPage() {
  const {
    plans,
    subscription,
    previewResult,
    loadingPlans,
    loadingSubscription,
    loadingCheckout,
    loadingChange,
    loadingPreview,
    loadingPortal,
    plansError,
    subscriptionError,
    checkoutError,
    changeError,
    fetchPlans,
    fetchSubscription,
    checkout,
    openPortal,
    previewChange,
    changePlan,
    clearPreview,
    clearErrors,
    currentPlanId,
    isFreePlan,
  } = useBillingStore()

  const [interval, setInterval] = useState<BillingInterval>('monthly')
  const [pendingPlanId, setPendingPlanId] = useState<string | null>(null)
  const [showPreview, setShowPreview] = useState(false)

  // URL params for success/cancel from Stripe redirect
  const [checkoutStatus, setCheckoutStatus] = useState<'success' | 'canceled' | null>(null)

  // Fetch data on mount
  useEffect(() => {
    fetchPlans()
    fetchSubscription()

    // Check URL for Stripe redirect status
    const params = new URLSearchParams(window.location.search)
    if (params.get('success') === 'true') {
      setCheckoutStatus('success')
      // Clean URL
      window.history.replaceState({}, '', '/billing')
      // Refresh subscription after a delay (Stripe webhook processing)
      setTimeout(() => fetchSubscription(), 2000)
    } else if (params.get('canceled') === 'true') {
      setCheckoutStatus('canceled')
      window.history.replaceState({}, '', '/billing')
    }

    return () => clearErrors()
  }, [fetchPlans, fetchSubscription, clearErrors])

  // Handle plan selection
  const handleSelectPlan = useCallback(async (planId: string) => {
    // Enterprise → contact sales
    if (planId === 'enterprise') {
      window.open('mailto:sales@operator.onl?subject=Enterprise Plan Inquiry', '_blank')
      return
    }

    const current = currentPlanId()

    // Same plan — no action
    if (planId === current) return

    // Free plan user → go to Stripe checkout
    if (isFreePlan()) {
      if (planId === 'free') return
      checkout(planId, interval)
      return
    }

    // Paid user switching plans → show preview
    setPendingPlanId(planId)
    try {
      await previewChange(planId, interval)
      setShowPreview(true)
    } catch {
      // Error already in store
    }
  }, [currentPlanId, isFreePlan, checkout, previewChange, interval])

  // Confirm plan change from preview
  const handleConfirmChange = useCallback(async () => {
    if (!pendingPlanId) return
    await changePlan(pendingPlanId, interval)
    setShowPreview(false)
    setPendingPlanId(null)
  }, [pendingPlanId, changePlan, interval])

  // Close preview
  const handleClosePreview = useCallback(() => {
    setShowPreview(false)
    setPendingPlanId(null)
    clearPreview()
  }, [clearPreview])

  // Error display
  const error = plansError || subscriptionError || checkoutError || changeError

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-5xl mx-auto px-4 sm:px-6 py-8">
        {/* ─── Header ─── */}
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-text mb-2">Plans & Billing</h1>
          <p className="text-sm text-text-secondary">
            Choose the plan that fits your needs. Upgrade or downgrade anytime.
          </p>
        </div>

        {/* ─── Stripe redirect banners ─── */}
        {checkoutStatus === 'success' && (
          <div className="flex items-center gap-3 p-4 mb-6 rounded-[var(--radius)] bg-success-subtle border border-success/20 animate-fade-slide">
            <CheckCircle size={20} weight="fill" className="text-success shrink-0" />
            <div>
              <p className="text-sm font-medium text-success">Payment successful!</p>
              <p className="text-xs text-success/70">Your subscription is being activated. This may take a moment.</p>
            </div>
            <button
              onClick={() => setCheckoutStatus(null)}
              className="ml-auto text-success/50 hover:text-success cursor-pointer"
              aria-label="Dismiss"
            >
              <XCircle size={18} />
            </button>
          </div>
        )}

        {checkoutStatus === 'canceled' && (
          <div className="flex items-center gap-3 p-4 mb-6 rounded-[var(--radius)] bg-warning-subtle border border-warning/20 animate-fade-slide">
            <Warning size={20} weight="fill" className="text-warning shrink-0" />
            <div>
              <p className="text-sm font-medium text-warning">Checkout canceled</p>
              <p className="text-xs text-warning/70">No changes were made to your plan.</p>
            </div>
            <button
              onClick={() => setCheckoutStatus(null)}
              className="ml-auto text-warning/50 hover:text-warning cursor-pointer"
              aria-label="Dismiss"
            >
              <XCircle size={18} />
            </button>
          </div>
        )}

        {/* ─── Error banner ─── */}
        {error && (
          <div className="flex items-center gap-3 p-4 mb-6 rounded-[var(--radius)] bg-error-subtle border border-error/20 animate-fade-slide">
            <Warning size={20} weight="fill" className="text-error shrink-0" />
            <p className="text-sm text-error flex-1">{error}</p>
            <Button variant="ghost" size="sm" onClick={clearErrors}>
              Dismiss
            </Button>
          </div>
        )}

        {/* ─── Current Subscription ─── */}
        {!loadingSubscription && (
          <div className="mb-8 animate-fade-slide">
            <CurrentSubscription
              subscription={subscription}
              onManageClick={openPortal}
              loadingPortal={loadingPortal}
            />
          </div>
        )}

        {/* ─── Interval Toggle ─── */}
        <div className="flex justify-center mb-8">
          <IntervalToggle value={interval} onChange={setInterval} />
        </div>

        {/* ─── Plan Cards ─── */}
        {loadingPlans ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5">
            {[1, 2, 3, 4].map((i) => (
              <div
                key={i}
                className="bg-[var(--surface)] border border-border-subtle rounded-[var(--radius)] p-6 animate-pulse"
              >
                <div className="h-6 bg-surface-2 rounded w-1/2 mb-4" />
                <div className="h-10 bg-surface-2 rounded w-2/3 mb-6" />
                <div className="space-y-3">
                  {[1, 2, 3, 4, 5].map((j) => (
                    <div key={j} className="h-4 bg-surface-2 rounded w-full" />
                  ))}
                </div>
                <div className="h-12 bg-surface-2 rounded mt-6" />
              </div>
            ))}
          </div>
        ) : plansError ? (
          <div className="text-center py-16">
            <p className="text-sm text-text-dim mb-4">Failed to load plans</p>
            <Button variant="secondary" size="sm" icon={<ArrowClockwise size={14} />} onClick={fetchPlans}>
              Retry
            </Button>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5 animate-fade-slide">
            {plans.map((plan) => (
              <PlanCard
                key={plan.id}
                plan={plan}
                currentPlanId={currentPlanId()}
                interval={interval}
                onSelect={handleSelectPlan}
                loading={
                  (loadingCheckout && pendingPlanId === plan.id) ||
                  (loadingPreview && pendingPlanId === plan.id)
                }
                recommended={plan.id === 'pro'}
              />
            ))}
          </div>
        )}

        {/* ─── Footer info ─── */}
        <div className="mt-10 text-center">
          <p className="text-xs text-text-dim">
            All plans include SSL encryption, 99.9% uptime SLA, and email support.
            <br />
            Prices shown in USD. Taxes may apply.
          </p>
        </div>

        {/* ─── Plan Change Preview Modal ─── */}
        <PlanChangePreview
          open={showPreview}
          onClose={handleClosePreview}
          onConfirm={handleConfirmChange}
          preview={previewResult}
          loading={loadingChange}
        />
      </div>
    </div>
  )
}
