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
import { useUIStore } from '../../stores/uiStore'

export function AppShell() {
  const sidebarOpen = useUIStore((s) => s.sidebarOpen)
  const setSidebarOpen = useUIStore((s) => s.setSidebarOpen)
  const location = useLocation()

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
        <TopBar />

        {/* ─── Page content ─── */}
        {/* pb on mobile to clear fixed BottomTabs */}
        <main className="flex-1 relative overflow-hidden pb-[var(--bottom-tabs-h)] md:pb-0">
          <Outlet />
        </main>
      </div>

      {/* ─── Bottom tabs (mobile) ─── */}
      <BottomTabs />
    </div>
  )
}
