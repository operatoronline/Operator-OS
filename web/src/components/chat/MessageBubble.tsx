// ============================================================================
// Operator OS — MessageBubble
// Renders a single chat message: user, agent, or system.
// Ports visual treatment from legacy index.html (OKLCH tokens, border radii,
// font sizing, spacing). Agent messages are plain text for now — markdown
// rendering comes in C8.
// ============================================================================

import { memo } from 'react'
import type { ChatMessage } from '../../stores/chatStore'

interface MessageBubbleProps {
  message: ChatMessage
  /** Whether to show the timestamp (e.g. first in a group, or >2min gap) */
  showTimestamp?: boolean
}

function formatTime(iso: string): string {
  try {
    const d = new Date(iso)
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  } catch {
    return ''
  }
}

function MessageBubbleInner({ message, showTimestamp = false }: MessageBubbleProps) {
  const { role, content, streaming } = message

  // ── System message ──
  if (role === 'system') {
    return (
      <div className="flex justify-center animate-fade-slide">
        <div className="text-xs text-[var(--text-dim)] px-3 py-1 max-w-[90%] text-center">
          {content}
        </div>
      </div>
    )
  }

  // ── User message ──
  if (role === 'user') {
    return (
      <div className="flex flex-col items-end animate-fade-slide">
        <div
          className="max-w-[680px] w-fit bg-[var(--user-bg)] border border-[var(--user-border)]
            rounded-[var(--radius)] rounded-br-[var(--radius-xs)]
            px-4 py-3 text-sm leading-[1.6] whitespace-pre-wrap break-words text-[var(--text)]"
        >
          {content}
        </div>
        {showTimestamp && (
          <span className="text-[10px] text-[var(--text-dim)] mt-1 mr-1">
            {formatTime(message.createdAt)}
          </span>
        )}
      </div>
    )
  }

  // ── Agent message ──
  return (
    <div className="flex flex-col items-start animate-fade-slide">
      <div
        className={`max-w-[680px] w-fit text-sm leading-[1.7] text-[var(--text)]
          px-0 py-1 whitespace-pre-wrap break-words
          ${streaming ? 'animate-pulse-glow rounded-lg px-2' : ''}`}
      >
        {content}
        {streaming && (
          <span className="inline-block w-[2px] h-[1em] bg-[var(--accent)] ml-0.5 align-middle animate-blink" />
        )}
      </div>
      {showTimestamp && !streaming && (
        <span className="text-[10px] text-[var(--text-dim)] mt-1 ml-1">
          {formatTime(message.createdAt)}
          {message.model && (
            <span className="ml-1.5 text-[var(--text-dim)]">· {message.model}</span>
          )}
        </span>
      )}
    </div>
  )
}

export const MessageBubble = memo(MessageBubbleInner)
