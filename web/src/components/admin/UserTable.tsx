// ============================================================================
// Operator OS — Admin User Table
// Responsive user list with role/status badges and action menus.
// ============================================================================

import { memo, useState, useRef, useEffect, useCallback } from 'react'
import {
  DotsThreeVertical,
  UserCircle,
  ShieldCheck,
  Prohibit,
  CheckCircle,
  Trash,
  Crown,
  User as UserIcon,
  EnvelopeSimple,
} from '@phosphor-icons/react'
import { Badge } from '../shared/Badge'
import type { AdminUser } from '../../types/api'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface UserTableProps {
  users: AdminUser[]
  loading: boolean
  actionLoading: string | null
  onSuspend: (id: string) => void
  onActivate: (id: string) => void
  onSetRole: (id: string, role: 'user' | 'admin') => void
  onDelete: (id: string) => void
}

interface UserRowProps {
  user: AdminUser
  actionLoading: string | null
  onSuspend: (id: string) => void
  onActivate: (id: string) => void
  onSetRole: (id: string, role: 'user' | 'admin') => void
  onDelete: (id: string) => void
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function statusVariant(status: string): 'success' | 'warning' | 'error' | 'default' {
  switch (status) {
    case 'active': return 'success'
    case 'pending_verification': return 'warning'
    case 'suspended': return 'error'
    default: return 'default'
  }
}

function statusLabel(status: string): string {
  switch (status) {
    case 'pending_verification': return 'Pending'
    case 'active': return 'Active'
    case 'suspended': return 'Suspended'
    case 'deleted': return 'Deleted'
    default: return status
  }
}

function formatDate(dateStr: string): string {
  try {
    return new Date(dateStr).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    })
  } catch {
    return dateStr
  }
}

// ---------------------------------------------------------------------------
// User Row
// ---------------------------------------------------------------------------

const UserRow = memo(function UserRow({
  user,
  actionLoading,
  onSuspend,
  onActivate,
  onSetRole,
  onDelete,
}: UserRowProps) {
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const isActing = actionLoading === user.id

  // Close menu on outside click
  useEffect(() => {
    if (!menuOpen) return
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [menuOpen])

  const handleAction = useCallback(
    (action: () => void) => {
      setMenuOpen(false)
      action()
    },
    [],
  )

  return (
    <div
      className={`
        flex items-center gap-3 px-4 py-3
        border-b border-[var(--border-subtle)] last:border-b-0
        hover:bg-[var(--surface-2)]/50 transition-colors
        ${isActing ? 'opacity-60 pointer-events-none' : ''}
      `}
    >
      {/* Avatar */}
      <div className="w-9 h-9 rounded-full bg-[var(--accent-subtle)] flex items-center justify-center shrink-0">
        <span className="text-sm font-semibold text-[var(--accent-text)] uppercase">
          {user.display_name?.[0] || user.email[0]}
        </span>
      </div>

      {/* User info */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-[var(--text)] truncate">
            {user.display_name || 'No name'}
          </span>
          {user.role === 'admin' && (
            <Crown size={13} weight="fill" className="text-[var(--accent-text)] shrink-0" />
          )}
          {/* Status inline on mobile (hidden on sm+ where column shows) */}
          <span className="sm:hidden">
            <Badge variant={statusVariant(user.status)} dot>
              {statusLabel(user.status)}
            </Badge>
          </span>
        </div>
        <div className="flex items-center gap-2 mt-0.5">
          <span className="text-xs text-[var(--text-dim)] truncate">{user.email}</span>
          {!user.email_verified && (
            <EnvelopeSimple
              size={12}
              className="text-[var(--warning)] shrink-0"
              aria-label="Email not verified"
            />
          )}
        </div>
      </div>

      {/* Status badge — hidden on small screens */}
      <div className="hidden sm:block shrink-0">
        <Badge variant={statusVariant(user.status)} dot>
          {statusLabel(user.status)}
        </Badge>
      </div>

      {/* Role badge — hidden on small screens */}
      <div className="hidden md:block shrink-0">
        <Badge variant={user.role === 'admin' ? 'accent' : 'default'}>
          {user.role}
        </Badge>
      </div>

      {/* Date — hidden on small screens */}
      <div className="hidden lg:block text-xs text-[var(--text-dim)] shrink-0 w-24 text-right tabular-nums">
        {formatDate(user.created_at)}
      </div>

      {/* Actions menu */}
      <div ref={menuRef} className="relative shrink-0">
        <button
          onClick={() => setMenuOpen((o) => !o)}
          className="p-2.5 md:p-1.5 rounded-lg text-[var(--text-dim)] hover:text-[var(--text)]
            hover:bg-[var(--surface-2)] transition-colors cursor-pointer
            active:scale-95 active:opacity-80"
          aria-label="User actions"
        >
          <DotsThreeVertical size={18} weight="bold" />
        </button>

        {menuOpen && (
          <div
            className="absolute right-0 top-full mt-1 z-30
              w-48 py-1 rounded-[var(--radius-md)]
              bg-[var(--surface)] border border-[var(--border)]
              shadow-lg animate-fade-slide-down"
          >
            {/* View profile */}
            <MenuButton
              icon={<UserCircle size={15} />}
              label="View profile"
              onClick={() => handleAction(() => {})}
            />

            {/* Role toggle */}
            {user.role === 'user' ? (
              <MenuButton
                icon={<ShieldCheck size={15} />}
                label="Promote to admin"
                onClick={() => handleAction(() => onSetRole(user.id, 'admin'))}
              />
            ) : (
              <MenuButton
                icon={<UserIcon size={15} />}
                label="Demote to user"
                onClick={() => handleAction(() => onSetRole(user.id, 'user'))}
              />
            )}

            {/* Suspend / Activate */}
            {user.status === 'active' ? (
              <MenuButton
                icon={<Prohibit size={15} />}
                label="Suspend user"
                onClick={() => handleAction(() => onSuspend(user.id))}
                danger
              />
            ) : user.status === 'suspended' ? (
              <MenuButton
                icon={<CheckCircle size={15} />}
                label="Activate user"
                onClick={() => handleAction(() => onActivate(user.id))}
              />
            ) : null}

            <div className="my-1 border-t border-[var(--border-subtle)]" />

            {/* Delete */}
            <MenuButton
              icon={<Trash size={15} />}
              label="Delete user"
              onClick={() => handleAction(() => onDelete(user.id))}
              danger
            />
          </div>
        )}
      </div>
    </div>
  )
})

// ---------------------------------------------------------------------------
// Menu Button
// ---------------------------------------------------------------------------

function MenuButton({
  icon,
  label,
  onClick,
  danger = false,
}: {
  icon: React.ReactNode
  label: string
  onClick: () => void
  danger?: boolean
}) {
  return (
    <button
      onClick={onClick}
      className={`
        w-full flex items-center gap-2.5 px-3 py-2 text-xs cursor-pointer
        transition-colors
        ${danger
          ? 'text-[var(--error)] hover:bg-[var(--error-subtle)]'
          : 'text-[var(--text)] hover:bg-[var(--surface-2)]'
        }
      `}
    >
      {icon}
      {label}
    </button>
  )
}

// ---------------------------------------------------------------------------
// Loading Skeleton
// ---------------------------------------------------------------------------

function UserTableSkeleton() {
  return (
    <div className="divide-y divide-[var(--border-subtle)]">
      {Array.from({ length: 6 }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 px-4 py-3">
          <div className="w-9 h-9 rounded-full bg-[var(--surface-2)] animate-pulse" />
          <div className="flex-1 space-y-1.5">
            <div className="h-3.5 w-32 rounded bg-[var(--surface-2)] animate-pulse" />
            <div className="h-3 w-48 rounded bg-[var(--surface-2)] animate-pulse" />
          </div>
          <div className="hidden sm:block h-5 w-16 rounded-full bg-[var(--surface-2)] animate-pulse" />
          <div className="hidden md:block h-5 w-12 rounded-full bg-[var(--surface-2)] animate-pulse" />
        </div>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// User Table
// ---------------------------------------------------------------------------

export const UserTable = memo(function UserTable({
  users,
  loading,
  actionLoading,
  onSuspend,
  onActivate,
  onSetRole,
  onDelete,
}: UserTableProps) {
  if (loading) {
    return (
      <div className="rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--surface)] overflow-hidden">
        <UserTableSkeleton />
      </div>
    )
  }

  if (users.length === 0) {
    return (
      <div className="rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--surface)]
        flex flex-col items-center justify-center py-16 text-center"
      >
        <UserIcon size={40} weight="thin" className="text-[var(--text-dim)] mb-3" />
        <p className="text-sm font-medium text-[var(--text)]">No users found</p>
        <p className="text-xs text-[var(--text-dim)] mt-1">
          Try adjusting your search or filters
        </p>
      </div>
    )
  }

  return (
    <div className="rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--surface)] overflow-hidden">
      {/* Column headers — desktop */}
      <div className="hidden sm:flex items-center gap-3 px-4 py-2
        border-b border-[var(--border-subtle)] bg-[var(--surface-2)]/50
        text-[10px] font-semibold text-[var(--text-dim)] uppercase tracking-wider"
      >
        <div className="w-9 shrink-0" /> {/* Avatar column */}
        <div className="flex-1">User</div>
        <div className="hidden sm:block w-20">Status</div>
        <div className="hidden md:block w-14">Role</div>
        <div className="hidden lg:block w-24 text-right">Joined</div>
        <div className="w-8 shrink-0" /> {/* Actions column */}
      </div>

      {/* Rows */}
      {users.map((user) => (
        <UserRow
          key={user.id}
          user={user}
          actionLoading={actionLoading}
          onSuspend={onSuspend}
          onActivate={onActivate}
          onSetRole={onSetRole}
          onDelete={onDelete}
        />
      ))}
    </div>
  )
})
