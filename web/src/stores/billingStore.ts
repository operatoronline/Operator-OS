// ============================================================================
// Operator OS — Billing Store
// Zustand store for plans, subscription, checkout, and plan changes.
// ============================================================================

import { create } from 'zustand'
import { api, ApiRequestError } from '../services/api'
import type {
  Plan,
  PlanId,
  Subscription,
  BillingInterval,
  PlanChangeResult,
} from '../types/api'

interface BillingState {
  // Data
  plans: Plan[]
  subscription: Subscription | null
  previewResult: PlanChangeResult | null

  // Loading states
  loadingPlans: boolean
  loadingSubscription: boolean
  loadingCheckout: boolean
  loadingChange: boolean
  loadingPreview: boolean
  loadingPortal: boolean

  // Errors
  plansError: string | null
  subscriptionError: string | null
  checkoutError: string | null
  changeError: string | null

  // Actions
  fetchPlans: () => Promise<void>
  fetchSubscription: () => Promise<void>
  checkout: (planId: string, interval?: BillingInterval) => Promise<void>
  openPortal: () => Promise<void>
  previewChange: (planId: string, interval?: BillingInterval) => Promise<PlanChangeResult>
  changePlan: (planId: string, interval?: BillingInterval, mode?: 'immediate' | 'at_period_end') => Promise<void>
  clearPreview: () => void
  clearErrors: () => void

  // Derived helpers
  currentPlanId: () => PlanId
  isFreePlan: () => boolean
}

function extractError(err: unknown): string {
  if (err instanceof ApiRequestError) return err.message
  if (err instanceof Error) return err.message
  return 'Something went wrong'
}

export const useBillingStore = create<BillingState>((set, get) => ({
  // Data
  plans: [],
  subscription: null,
  previewResult: null,

  // Loading
  loadingPlans: false,
  loadingSubscription: false,
  loadingCheckout: false,
  loadingChange: false,
  loadingPreview: false,
  loadingPortal: false,

  // Errors
  plansError: null,
  subscriptionError: null,
  checkoutError: null,
  changeError: null,

  // ─── Fetch Plans ───
  fetchPlans: async () => {
    set({ loadingPlans: true, plansError: null })
    try {
      const plans = await api.billing.plans()
      // Sort: free → starter → pro → enterprise
      const order: Record<string, number> = { free: 0, starter: 1, pro: 2, enterprise: 3 }
      plans.sort((a, b) => (order[a.id] ?? 99) - (order[b.id] ?? 99))
      set({ plans, loadingPlans: false })
    } catch (err) {
      set({ plansError: extractError(err), loadingPlans: false })
    }
  },

  // ─── Fetch Subscription ───
  fetchSubscription: async () => {
    set({ loadingSubscription: true, subscriptionError: null })
    try {
      const subscription = await api.billing.subscription()
      set({ subscription, loadingSubscription: false })
    } catch (err) {
      // 404 = no subscription (free plan)
      if (err instanceof ApiRequestError && err.status === 404) {
        set({ subscription: null, loadingSubscription: false })
        return
      }
      set({ subscriptionError: extractError(err), loadingSubscription: false })
    }
  },

  // ─── Checkout (redirect to Stripe) ───
  checkout: async (planId, interval = 'monthly') => {
    set({ loadingCheckout: true, checkoutError: null })
    try {
      const { url } = await api.billing.checkout({
        plan_id: planId,
        interval,
        success_url: `${window.location.origin}/billing?success=true`,
        cancel_url: `${window.location.origin}/billing?canceled=true`,
      })
      // Redirect to Stripe Checkout
      window.location.href = url
    } catch (err) {
      set({ checkoutError: extractError(err), loadingCheckout: false })
    }
  },

  // ─── Open Billing Portal ───
  openPortal: async () => {
    set({ loadingPortal: true })
    try {
      const { url } = await api.billing.portal()
      window.location.href = url
    } catch (err) {
      set({ checkoutError: extractError(err), loadingPortal: false })
    }
  },

  // ─── Preview Plan Change ───
  previewChange: async (planId, interval = 'monthly') => {
    set({ loadingPreview: true, changeError: null })
    try {
      const result = await api.billing.previewChange({ plan_id: planId, interval })
      set({ previewResult: result, loadingPreview: false })
      return result
    } catch (err) {
      set({ changeError: extractError(err), loadingPreview: false })
      throw err
    }
  },

  // ─── Change Plan ───
  changePlan: async (planId, interval = 'monthly', mode = 'immediate') => {
    set({ loadingChange: true, changeError: null })
    try {
      await api.billing.changePlan({ plan_id: planId, interval, mode })
      // Refresh subscription data
      await get().fetchSubscription()
      set({ loadingChange: false, previewResult: null })
    } catch (err) {
      set({ changeError: extractError(err), loadingChange: false })
    }
  },

  clearPreview: () => set({ previewResult: null }),
  clearErrors: () => set({ plansError: null, subscriptionError: null, checkoutError: null, changeError: null }),

  // Derived
  currentPlanId: () => {
    const sub = get().subscription
    return (sub?.plan_id as PlanId) || 'free'
  },

  isFreePlan: () => {
    return get().currentPlanId() === 'free'
  },
}))
