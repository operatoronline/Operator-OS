// ============================================================================
// Operator OS — ProfileForm
// Display name edit + password change. Email is read-only.
// ============================================================================

import { useState, useCallback } from 'react'
import {
  User,
  EnvelopeSimple,
  Lock,
  Eye,
  EyeSlash,
  CheckCircle,
} from '@phosphor-icons/react'
import { useAuthStore } from '../../stores/authStore'
import { useSettingsStore } from '../../stores/settingsStore'
import { Button } from '../shared/Button'
import { Input } from '../shared/Input'

export function ProfileForm() {
  const user = useAuthStore((s) => s.user)
  const {
    profileLoading,
    profileError,
    profileSuccess,
    passwordLoading,
    passwordError,
    passwordSuccess,
    updateProfile,
    changePassword,
    clearMessages,
  } = useSettingsStore()

  // ─── Profile state ───
  const [displayName, setDisplayName] = useState(user?.display_name || '')
  const profileDirty = displayName.trim() !== (user?.display_name || '')

  // ─── Password state ───
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showCurrent, setShowCurrent] = useState(false)
  const [showNew, setShowNew] = useState(false)

  const passwordMismatch =
    confirmPassword.length > 0 && newPassword !== confirmPassword
  const passwordTooShort = newPassword.length > 0 && newPassword.length < 8
  const canChangePassword =
    currentPassword.length > 0 &&
    newPassword.length >= 8 &&
    newPassword === confirmPassword

  // ─── Handlers ───
  const handleProfileSave = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault()
      clearMessages()
      try {
        await updateProfile(displayName.trim())
      } catch {
        // error shown via store
      }
    },
    [displayName, updateProfile, clearMessages],
  )

  const handlePasswordChange = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault()
      clearMessages()
      try {
        await changePassword(currentPassword, newPassword)
        setCurrentPassword('')
        setNewPassword('')
        setConfirmPassword('')
      } catch {
        // error shown via store
      }
    },
    [currentPassword, newPassword, changePassword, clearMessages],
  )

  return (
    <div className="space-y-8">
      {/* ─── Profile Section ─── */}
      <form onSubmit={handleProfileSave}>
        <div className="flex items-center gap-3 mb-5">
          <div className="w-9 h-9 rounded-xl bg-accent-subtle flex items-center justify-center">
            <User size={18} weight="duotone" className="text-accent-text" />
          </div>
          <div>
            <h3 className="text-[15px] font-semibold text-[var(--text)]">Profile</h3>
            <p className="text-xs text-[var(--text-dim)]">Your public display information</p>
          </div>
        </div>

        <div className="space-y-4">
          {/* Email — read only */}
          <div className="flex flex-col gap-1.5">
            <label className="text-[13px] font-medium text-text-secondary">
              Email
            </label>
            <div className="relative">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-text-dim pointer-events-none">
                <EnvelopeSimple size={16} />
              </span>
              <input
                type="email"
                value={user?.email || ''}
                disabled
                className="w-full pl-10 opacity-60 cursor-not-allowed"
              />
            </div>
            <p className="text-[11px] text-[var(--text-dim)]">
              Email cannot be changed.{' '}
              {user?.email_verified ? (
                <span className="text-success">Verified ✓</span>
              ) : (
                <span className="text-warning">Unverified</span>
              )}
            </p>
          </div>

          {/* Display name */}
          <Input
            label="Display name"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="How you appear in the platform"
            icon={<User size={16} />}
          />

          {/* Feedback */}
          {profileError && (
            <p className="text-xs text-error" role="alert">{profileError}</p>
          )}
          {profileSuccess && (
            <p className="text-xs text-success flex items-center gap-1" role="status">
              <CheckCircle size={14} weight="fill" />
              {profileSuccess}
            </p>
          )}

          <Button
            type="submit"
            size="sm"
            disabled={!profileDirty}
            loading={profileLoading}
          >
            Save profile
          </Button>
        </div>
      </form>

      {/* ─── Divider ─── */}
      <div className="border-t border-[var(--border-subtle)]" />

      {/* ─── Password Section ─── */}
      <form onSubmit={handlePasswordChange}>
        <div className="flex items-center gap-3 mb-5">
          <div className="w-9 h-9 rounded-xl bg-warning-subtle flex items-center justify-center">
            <Lock size={18} weight="duotone" className="text-warning" />
          </div>
          <div>
            <h3 className="text-[15px] font-semibold text-[var(--text)]">Password</h3>
            <p className="text-xs text-[var(--text-dim)]">Change your account password</p>
          </div>
        </div>

        <div className="space-y-4">
          {/* Current password */}
          <div className="flex flex-col gap-1.5">
            <label className="text-[13px] font-medium text-text-secondary">
              Current password
            </label>
            <div className="relative">
              <input
                type={showCurrent ? 'text' : 'password'}
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                placeholder="Enter current password"
                className="w-full pr-10"
                autoComplete="current-password"
              />
              <button
                type="button"
                onClick={() => setShowCurrent(!showCurrent)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-text-dim
                  hover:text-text-secondary transition-colors cursor-pointer"
                tabIndex={-1}
              >
                {showCurrent ? <EyeSlash size={16} /> : <Eye size={16} />}
              </button>
            </div>
          </div>

          {/* New password */}
          <div className="flex flex-col gap-1.5">
            <label className="text-[13px] font-medium text-text-secondary">
              New password
            </label>
            <div className="relative">
              <input
                type={showNew ? 'text' : 'password'}
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                placeholder="Minimum 8 characters"
                className={`w-full pr-10 ${passwordTooShort ? 'border-warning!' : ''}`}
                autoComplete="new-password"
              />
              <button
                type="button"
                onClick={() => setShowNew(!showNew)}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-text-dim
                  hover:text-text-secondary transition-colors cursor-pointer"
                tabIndex={-1}
              >
                {showNew ? <EyeSlash size={16} /> : <Eye size={16} />}
              </button>
            </div>
            {passwordTooShort && (
              <p className="text-[11px] text-warning">
                Must be at least 8 characters
              </p>
            )}
          </div>

          {/* Confirm password */}
          <Input
            label="Confirm new password"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            placeholder="Re-enter new password"
            error={passwordMismatch ? 'Passwords do not match' : undefined}
            autoComplete="new-password"
          />

          {/* Feedback */}
          {passwordError && (
            <p className="text-xs text-error" role="alert">{passwordError}</p>
          )}
          {passwordSuccess && (
            <p className="text-xs text-success flex items-center gap-1" role="status">
              <CheckCircle size={14} weight="fill" />
              {passwordSuccess}
            </p>
          )}

          <Button
            type="submit"
            size="sm"
            variant="secondary"
            disabled={!canChangePassword}
            loading={passwordLoading}
          >
            Change password
          </Button>
        </div>
      </form>
    </div>
  )
}
