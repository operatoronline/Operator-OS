// ============================================================================
// Operator OS — Session Item
// Individual session row in the session sidebar panel.
// Supports: selection, inline rename, context menu (pin, rename, delete).
// ============================================================================

import { memo, useState, useRef, useEffect, useCallback, type KeyboardEvent } from 'react'
import {
  ChatCircle,
  PushPin,
  PencilSimple,
  Trash,
  DotsThree,
  Check,
  X,
  Archive,
  ArrowUUpLeft,
  Export,
  FileText,
  FileCode,
} from '@phosphor-icons/react'

interface SessionItemProps {
  id: string
  name: string
  messageCount: number
  lastMessageAt: string
  pinned: boolean
  archived?: boolean
  isActive: boolean
  isRenaming: boolean
  onSelect: (id: string) => void
  onRename: (id: string, name: string) => void
  onTogglePin: (id: string, pinned: boolean) => void
  onToggleArchive: (id: string, archived: boolean) => void
  onExportMarkdown: (id: string) => void
  onExportJSON: (id: string) => void
  onDelete: (id: string) => void
  onStartRename: (id: string) => void
  onCancelRename: () => void
}

function formatRelativeTime(dateStr: string): string {
  if (!dateStr) return ''
  const now = Date.now()
  const date = new Date(dateStr).getTime()
  const diffMs = now - date
  const diffMin = Math.floor(diffMs / 60000)
  const diffHour = Math.floor(diffMs / 3600000)
  const diffDay = Math.floor(diffMs / 86400000)

  if (diffMin < 1) return 'now'
  if (diffMin < 60) return `${diffMin}m`
  if (diffHour < 24) return `${diffHour}h`
  if (diffDay < 7) return `${diffDay}d`
  if (diffDay < 30) return `${Math.floor(diffDay / 7)}w`
  return new Date(dateStr).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

export const SessionItem = memo(function SessionItem({
  id,
  name,
  messageCount,
  lastMessageAt,
  pinned,
  archived = false,
  isActive,
  isRenaming,
  onSelect,
  onRename,
  onTogglePin,
  onToggleArchive,
  onExportMarkdown,
  onExportJSON,
  onDelete,
  onStartRename,
  onCancelRename,
}: SessionItemProps) {
  const [exportOpen, setExportOpen] = useState(false)
  const [menuOpen, setMenuOpen] = useState(false)
  const [renameValue, setRenameValue] = useState(name)
  const renameRef = useRef<HTMLInputElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  // Focus rename input on enter
  useEffect(() => {
    if (isRenaming && renameRef.current) {
      setRenameValue(name)
      renameRef.current.focus()
      renameRef.current.select()
    }
  }, [isRenaming, name])

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

  const handleRenameSubmit = useCallback(() => {
    const trimmed = renameValue.trim()
    if (trimmed && trimmed !== name) {
      onRename(id, trimmed)
    } else {
      onCancelRename()
    }
  }, [id, name, renameValue, onRename, onCancelRename])

  const handleRenameKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault()
        handleRenameSubmit()
      } else if (e.key === 'Escape') {
        onCancelRename()
      }
    },
    [handleRenameSubmit, onCancelRename],
  )

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => !isRenaming && onSelect(id)}
      onKeyDown={(e) => {
        if (e.key === 'Enter' && !isRenaming) onSelect(id)
      }}
      className={`
        group relative flex items-center gap-2.5 px-3 py-2.5 rounded-lg
        transition-all duration-150 select-none cursor-pointer
        ${isActive
          ? 'bg-[var(--surface-2)] text-[var(--text)] shadow-[inset_0_0_0_1px_var(--border)]'
          : 'text-[var(--text-dim)] hover:text-[var(--text-secondary)] hover:bg-[var(--surface-2)]/50'
        }
      `}
      aria-selected={isActive}
      aria-label={`Session: ${name}`}
    >
      {/* Icon */}
      <div className="shrink-0 relative">
        <ChatCircle
          size={18}
          weight={isActive ? 'fill' : 'regular'}
          className={isActive ? 'text-[var(--accent-text)]' : ''}
        />
        {pinned && (
          <PushPin
            size={8}
            weight="fill"
            className="absolute -top-1 -right-1 text-[var(--accent-text)]"
          />
        )}
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        {isRenaming ? (
          <div className="flex items-center gap-1">
            <input
              ref={renameRef}
              type="text"
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onKeyDown={handleRenameKeyDown}
              onBlur={handleRenameSubmit}
              className="flex-1 min-w-0 bg-[var(--surface-3)] text-[var(--text)] text-[13px]
                px-1.5 py-0.5 rounded border border-[var(--border)] outline-none
                focus:border-[var(--accent)] font-[family-name:var(--font)]"
              maxLength={100}
            />
            <button
              onClick={(e) => {
                e.stopPropagation()
                handleRenameSubmit()
              }}
              className="shrink-0 p-0.5 rounded hover:bg-[var(--surface-3)] text-[var(--success)] cursor-pointer"
              aria-label="Confirm rename"
            >
              <Check size={14} weight="bold" />
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation()
                onCancelRename()
              }}
              className="shrink-0 p-0.5 rounded hover:bg-[var(--surface-3)] text-[var(--text-dim)] cursor-pointer"
              aria-label="Cancel rename"
            >
              <X size={14} weight="bold" />
            </button>
          </div>
        ) : (
          <>
            <div className="text-[13px] font-medium truncate leading-tight">
              {name}
            </div>
            <div className="flex items-center gap-1.5 mt-0.5">
              <span className="text-[10px] text-[var(--text-dim)]">
                {messageCount} msg{messageCount !== 1 ? 's' : ''}
              </span>
              {lastMessageAt && (
                <>
                  <span className="text-[10px] text-[var(--text-dim)]">·</span>
                  <span className="text-[10px] text-[var(--text-dim)]">
                    {formatRelativeTime(lastMessageAt)}
                  </span>
                </>
              )}
            </div>
          </>
        )}
      </div>

      {/* Context menu button */}
      {!isRenaming && (
        <div className="relative shrink-0" ref={menuRef}>
          <button
            onClick={(e) => {
              e.stopPropagation()
              setMenuOpen(!menuOpen)
            }}
            className={`
              p-1 rounded-md transition-all duration-150 cursor-pointer
              ${menuOpen
                ? 'bg-[var(--surface-3)] text-[var(--text)]'
                : 'opacity-0 group-hover:opacity-100 text-[var(--text-dim)] hover:text-[var(--text)] hover:bg-[var(--surface-3)]'
              }
            `}
            aria-label="Session options"
          >
            <DotsThree size={16} weight="bold" />
          </button>

          {menuOpen && (
            <div
              className="absolute right-0 top-full mt-1 w-44
                glass rounded-[var(--radius-md)] py-1 z-50
                animate-fade-slide shadow-[0_4px_12px_var(--glass-shadow)]"
            >
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  onTogglePin(id, !pinned)
                  setMenuOpen(false)
                }}
                className="w-full flex items-center gap-2 px-3 py-2 text-xs text-[var(--text-secondary)]
                  hover:text-[var(--text)] hover:bg-[var(--surface-2)]/60 transition-colors cursor-pointer"
              >
                <PushPin size={14} weight={pinned ? 'fill' : 'regular'} />
                {pinned ? 'Unpin' : 'Pin'}
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  onStartRename(id)
                  setMenuOpen(false)
                }}
                className="w-full flex items-center gap-2 px-3 py-2 text-xs text-[var(--text-secondary)]
                  hover:text-[var(--text)] hover:bg-[var(--surface-2)]/60 transition-colors cursor-pointer"
              >
                <PencilSimple size={14} />
                Rename
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  onToggleArchive(id, !archived)
                  setMenuOpen(false)
                }}
                className="w-full flex items-center gap-2 px-3 py-2 text-xs text-[var(--text-secondary)]
                  hover:text-[var(--text)] hover:bg-[var(--surface-2)]/60 transition-colors cursor-pointer"
              >
                {archived ? <ArrowUUpLeft size={14} /> : <Archive size={14} />}
                {archived ? 'Unarchive' : 'Archive'}
              </button>
              {/* Export submenu */}
              <div className="relative">
                <button
                  onClick={(e) => {
                    e.stopPropagation()
                    setExportOpen(!exportOpen)
                  }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-xs text-[var(--text-secondary)]
                    hover:text-[var(--text)] hover:bg-[var(--surface-2)]/60 transition-colors cursor-pointer"
                >
                  <Export size={14} />
                  Export
                </button>
                {exportOpen && (
                  <div className="pl-6">
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        onExportMarkdown(id)
                        setMenuOpen(false)
                        setExportOpen(false)
                      }}
                      className="w-full flex items-center gap-2 px-3 py-1.5 text-xs text-[var(--text-secondary)]
                        hover:text-[var(--text)] hover:bg-[var(--surface-2)]/60 transition-colors cursor-pointer"
                    >
                      <FileText size={13} />
                      Markdown
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        onExportJSON(id)
                        setMenuOpen(false)
                        setExportOpen(false)
                      }}
                      className="w-full flex items-center gap-2 px-3 py-1.5 text-xs text-[var(--text-secondary)]
                        hover:text-[var(--text)] hover:bg-[var(--surface-2)]/60 transition-colors cursor-pointer"
                    >
                      <FileCode size={13} />
                      JSON
                    </button>
                  </div>
                )}
              </div>
              <div className="my-1 border-t border-[var(--border-subtle)]" />
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  onDelete(id)
                  setMenuOpen(false)
                }}
                className="w-full flex items-center gap-2 px-3 py-2 text-xs text-[var(--error)]
                  hover:bg-[var(--error)]/10 transition-colors cursor-pointer"
              >
                <Trash size={14} />
                Delete
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  )
})
