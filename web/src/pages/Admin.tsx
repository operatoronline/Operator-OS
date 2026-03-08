// ============================================================================
// Operator OS — Admin Page
// User management, platform stats, role/status actions. Requires admin role.
// ============================================================================

import { useEffect, useState, useCallback, useRef } from 'react'
import {
  ShieldCheck,
  ShieldWarning,
  MagnifyingGlass,
  ArrowCounterClockwise,
  CaretLeft,
  CaretRight,
  Prohibit,
  Users,
  ClockCounterClockwise,
} from '@phosphor-icons/react'
import { useAdminStore } from '../stores/adminStore'
import { StatsCards } from '../components/admin/StatsCards'
import { UserTable } from '../components/admin/UserTable'
import { AuditLog } from '../components/admin/AuditLog'
import { SecurityDashboard } from '../components/admin/SecurityDashboard'
import { ConfirmDialog } from '../components/shared/ConfirmDialog'
import { Button } from '../components/shared/Button'

// ---------------------------------------------------------------------------
// Tab type
// ---------------------------------------------------------------------------

type AdminTab = 'users' | 'audit' | 'security'

// ---------------------------------------------------------------------------
// Status filter options
// ---------------------------------------------------------------------------

const STATUS_OPTIONS = [
  { value: '', label: 'All' },
  { value: 'active', label: 'Active' },
  { value: 'pending_verification', label: 'Pending' },
  { value: 'suspended', label: 'Suspended' },
]

// ---------------------------------------------------------------------------
// Admin Page
// ---------------------------------------------------------------------------

export function AdminPage() {
  const store = useAdminStore()
  const [activeTab, setActiveTab] = useState<AdminTab>('users')
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null)
  const [suspendTarget, setSuspendTarget] = useState<string | null>(null)
  const [roleTarget, setRoleTarget] = useState<{ id: string; role: 'user' | 'admin' } | null>(null)
  const searchTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined)

  // Fetch on mount
  useEffect(() => {
    store.fetchAll()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Debounced search
  const handleSearchChange = useCallback(
    (value: string) => {
      store.setSearch(value)
      clearTimeout(searchTimerRef.current)
      searchTimerRef.current = setTimeout(() => {
        store.fetchUsers()
      }, 300)
    },
    [store],
  )

  // Cleanup timer
  useEffect(() => {
    return () => clearTimeout(searchTimerRef.current)
  }, [])

  // Confirm handlers
  const handleConfirmDelete = useCallback(async () => {
    if (deleteTarget) {
      await store.deleteUser(deleteTarget)
      setDeleteTarget(null)
    }
  }, [deleteTarget, store])

  const handleConfirmSuspend = useCallback(async () => {
    if (suspendTarget) {
      await store.suspendUser(suspendTarget)
      setSuspendTarget(null)
    }
  }, [suspendTarget, store])

  const handleConfirmRole = useCallback(async () => {
    if (roleTarget) {
      await store.setRole(roleTarget.id, roleTarget.role)
      setRoleTarget(null)
    }
  }, [roleTarget, store])

  const handleActivate = useCallback(
    (id: string) => store.activateUser(id),
    [store],
  )

  // Pagination
  const hasNextPage = store.users.length === store.filters.perPage
  const hasPrevPage = store.filters.page > 1

  // ─── Forbidden (not admin) ───
  if (store.forbidden) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-[var(--text-dim)] px-4">
        <Prohibit size={48} weight="thin" className="mb-4 text-[var(--error)]" />
        <h2 className="text-lg font-semibold text-[var(--text)] mb-1">Access Denied</h2>
        <p className="text-sm text-center max-w-sm">
          You don't have permission to access the admin panel. Contact your administrator to request admin access.
        </p>
      </div>
    )
  }

  // Target user names for confirm dialogs
  const deleteUserName = deleteTarget
    ? store.users.find((u) => u.id === deleteTarget)?.display_name || 'this user'
    : ''
  const suspendUserName = suspendTarget
    ? store.users.find((u) => u.id === suspendTarget)?.display_name || 'this user'
    : ''
  const roleUserName = roleTarget
    ? store.users.find((u) => u.id === roleTarget.id)?.display_name || 'this user'
    : ''

  const hasErrors = store.usersError || store.statsError || store.actionError

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* ─── Header ─── */}
      <div className="shrink-0 px-4 md:px-6 pt-4 md:pt-6 pb-4 space-y-4">
        {/* Title */}
        <div className="flex items-center justify-between flex-wrap gap-3">
          <div>
            <h1 className="text-lg font-bold text-[var(--text)] flex items-center gap-2">
              <ShieldCheck size={22} weight="fill" className="text-[var(--accent-text)]" />
              Admin Panel
            </h1>
            <p className="text-xs text-[var(--text-dim)] mt-0.5">
              Manage users, roles, and platform settings
            </p>
          </div>

          {/* Tab switcher — scrollable on narrow screens */}
          <div className="flex items-center gap-1 bg-[var(--surface-2)] rounded-full p-0.5 overflow-x-auto scrollbar-none">
            <button
              onClick={() => setActiveTab('users')}
              className={`
                inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-full text-xs font-medium
                transition-all duration-150 cursor-pointer
                ${activeTab === 'users'
                  ? 'bg-[var(--surface)] text-[var(--text)] shadow-sm'
                  : 'text-[var(--text-dim)] hover:text-[var(--text-secondary)]'
                }
              `}
            >
              <Users size={14} />
              Users
            </button>
            <button
              onClick={() => setActiveTab('audit')}
              className={`
                inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-full text-xs font-medium
                transition-all duration-150 cursor-pointer
                ${activeTab === 'audit'
                  ? 'bg-[var(--surface)] text-[var(--text)] shadow-sm'
                  : 'text-[var(--text-dim)] hover:text-[var(--text-secondary)]'
                }
              `}
            >
              <ClockCounterClockwise size={14} />
              Audit Log
            </button>
            <button
              onClick={() => setActiveTab('security')}
              className={`
                inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-full text-xs font-medium
                transition-all duration-150 cursor-pointer
                ${activeTab === 'security'
                  ? 'bg-[var(--surface)] text-[var(--text)] shadow-sm'
                  : 'text-[var(--text-dim)] hover:text-[var(--text-secondary)]'
                }
              `}
            >
              <ShieldWarning size={14} />
              Security
            </button>
          </div>
        </div>

        {/* Error banners */}
        {hasErrors && (
          <div className="space-y-2">
            {[store.usersError, store.statsError, store.actionError]
              .filter(Boolean)
              .map((err, i) => (
                <div
                  key={i}
                  className="flex items-center gap-3 px-4 py-3
                    bg-[var(--error-subtle)] border border-[var(--error)]/20
                    rounded-[var(--radius-md)] text-sm text-[var(--error)]"
                >
                  <span className="flex-1">{err}</span>
                  <Button
                    variant="ghost"
                    size="sm"
                    icon={<ArrowCounterClockwise size={14} />}
                    onClick={() => {
                      store.clearErrors()
                      store.fetchAll()
                    }}
                  >
                    Retry
                  </Button>
                </div>
              ))}
          </div>
        )}

        {/* Stats cards (users tab only) */}
        {activeTab === 'users' && (
          <StatsCards stats={store.stats} loading={store.loadingStats} />
        )}

        {/* Search + filter bar (users tab only) */}
        {activeTab === 'users' && (
          <div className="flex items-center gap-3 flex-wrap">
            {/* Search */}
            <div className="relative flex-1 min-w-[200px] max-w-sm">
              <MagnifyingGlass
                size={15}
                className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-dim)]"
              />
              <input
                type="text"
                placeholder="Search users by name or email…"
                value={store.filters.search}
                onChange={(e) => handleSearchChange(e.target.value)}
                className="w-full h-9 pl-9 pr-3
                  bg-[var(--surface-2)] border border-[var(--border-subtle)]
                  rounded-full text-xs text-[var(--text)]
                  placeholder:text-[var(--text-dim)]
                  focus:border-[var(--accent)] focus:outline-none
                  transition-colors"
              />
            </div>

            {/* Status filter pills */}
            <div className="flex items-center gap-1 bg-[var(--surface-2)] rounded-full p-0.5">
              {STATUS_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => store.setStatusFilter(opt.value)}
                  className={`
                    px-3 py-1.5 rounded-full text-xs font-medium
                    transition-all duration-150 cursor-pointer
                    ${store.filters.status === opt.value
                      ? 'bg-[var(--surface)] text-[var(--text)] shadow-sm'
                      : 'text-[var(--text-dim)] hover:text-[var(--text-secondary)]'
                    }
                  `}
                >
                  {opt.value === '' && <Users size={12} className="inline mr-1 -mt-0.5" />}
                  {opt.label}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* ─── Tab content ─── */}
      <div className="flex-1 overflow-y-auto scroll-touch px-4 md:px-6 pb-4">
        {activeTab === 'security' ? (
          <SecurityDashboard />
        ) : activeTab === 'users' ? (
          <>
            <UserTable
              users={store.users}
              loading={store.loadingUsers}
              actionLoading={store.actionLoading}
              onSuspend={(id) => setSuspendTarget(id)}
              onActivate={handleActivate}
              onSetRole={(id, role) => setRoleTarget({ id, role })}
              onDelete={(id) => setDeleteTarget(id)}
            />

            {/* Pagination */}
            {!store.loadingUsers && store.users.length > 0 && (
              <div className="flex items-center justify-between mt-4 px-1">
                <p className="text-xs text-[var(--text-dim)] tabular-nums">
                  Page {store.filters.page}
                  {store.users.length > 0 && (
                    <span> · {store.users.length} user{store.users.length !== 1 ? 's' : ''}</span>
                  )}
                </p>
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => store.setPage(store.filters.page - 1)}
                    disabled={!hasPrevPage}
                    className="p-1.5 rounded-lg text-[var(--text-dim)]
                      hover:text-[var(--text)] hover:bg-[var(--surface-2)]
                      disabled:opacity-30 disabled:cursor-not-allowed
                      transition-colors cursor-pointer"
                    aria-label="Previous page"
                  >
                    <CaretLeft size={16} />
                  </button>
                  <button
                    onClick={() => store.setPage(store.filters.page + 1)}
                    disabled={!hasNextPage}
                    className="p-1.5 rounded-lg text-[var(--text-dim)]
                      hover:text-[var(--text)] hover:bg-[var(--surface-2)]
                      disabled:opacity-30 disabled:cursor-not-allowed
                      transition-colors cursor-pointer"
                    aria-label="Next page"
                  >
                    <CaretRight size={16} />
                  </button>
                </div>
              </div>
            )}
          </>
        ) : (
          <AuditLog />
        )}
      </div>

      {/* ─── Confirm Dialogs ─── */}
      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleConfirmDelete}
        title="Delete user"
        message={`Are you sure you want to permanently delete ${deleteUserName}? This action cannot be undone. All user data, sessions, and agent configurations will be removed.`}
        confirmLabel="Delete user"
        variant="danger"
      />

      <ConfirmDialog
        open={!!suspendTarget}
        onClose={() => setSuspendTarget(null)}
        onConfirm={handleConfirmSuspend}
        title="Suspend user"
        message={`Are you sure you want to suspend ${suspendUserName}? They will be logged out and unable to access the platform until reactivated.`}
        confirmLabel="Suspend"
        variant="danger"
      />

      <ConfirmDialog
        open={!!roleTarget}
        onClose={() => setRoleTarget(null)}
        onConfirm={handleConfirmRole}
        title={roleTarget?.role === 'admin' ? 'Promote to admin' : 'Demote to user'}
        message={
          roleTarget?.role === 'admin'
            ? `Promote ${roleUserName} to admin? They will gain access to the admin panel, user management, and platform settings.`
            : `Demote ${roleUserName} to regular user? They will lose admin panel access.`
        }
        confirmLabel={roleTarget?.role === 'admin' ? 'Promote' : 'Demote'}
        variant={roleTarget?.role === 'admin' ? 'primary' : 'danger'}
      />
    </div>
  )
}
