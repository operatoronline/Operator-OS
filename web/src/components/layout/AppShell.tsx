// ============================================================================
// Operator OS — App Shell
// Root layout: Sidebar (desktop) + TopBar + Content + BottomTabs (mobile).
// Includes mobile slide-over sidebar with backdrop.
// ============================================================================

import { useEffect } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { Sidebar } from './Sidebar'
import { TopBar } from './TopBar'
import { BottomTabs } from './BottomTabs'
import { MobileSidebar } from './MobileSidebar'
import { OfflineBanner } from '../shared/OfflineBanner'
import { ToastContainer } from '../shared/ToastContainer'
import { useUIStore } from '../../stores/uiStore'
import { useFocusOnNavigate } from '../../hooks/useFocusOnNavigate'

export function AppShell() {
  const sidebarOpen = useUIStore((s) => s.sidebarOpen)
  const setSidebarOpen = useUIStore((s) => s.setSidebarOpen)
  const location = useLocation()

  // Focus management on route changes (WCAG 2.1)
  useFocusOnNavigate()

  // Close mobile sidebar on route change
  useEffect(() => {
    // Only close on mobile (sidebar is repurposed as overlay on small screens)
    if (window.innerWidth < 768) {
      setSidebarOpen(false)
    }
  }, [location.pathname, setSidebarOpen])

  return (
    <div className="h-full flex">
      {/* ─── Desktop sidebar ─── */}
      <Sidebar />

      {/* ─── Mobile sidebar overlay ─── */}
      <MobileSidebar open={sidebarOpen} onClose={() => setSidebarOpen(false)} />

      {/* ─── Main area ─── */}
      <div className="flex-1 flex flex-col min-w-0 h-full">
        <OfflineBanner />
        <TopBar />

        {/* ─── Page content ─── */}
        {/* pb on mobile to clear fixed BottomTabs */}
        <main
          id="main-content"
          className="flex-1 relative overflow-hidden pb-[var(--bottom-tabs-h)] md:pb-0"
          aria-label="Page content"
        >
          <Outlet />
        </main>
      </div>

      {/* ─── Bottom tabs (mobile) ─── */}
      <BottomTabs />

      {/* ─── Toast notifications ─── */}
      <ToastContainer />
    </div>
  )
}
