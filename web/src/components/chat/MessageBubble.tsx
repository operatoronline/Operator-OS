// ============================================================================
// Operator OS — MessageBubble
// Renders a single chat message: user, agent, or system.
// Ports visual treatment from legacy index.html (OKLCH tokens, border radii,
// font sizing, spacing). Agent messages render markdown via MarkdownRenderer.
// ============================================================================

import { memo } from 'react'
import type { ChatMessage } from '../../stores/chatStore'
import { MarkdownRenderer } from './MarkdownRenderer'

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
  const { role, content, streaming, cancelled } = message

  // ── System message ──
  if (role === 'system') {
    return (
      <div className="flex justify-center animate-fade-slide" role="status">
        <div className="text-xs text-[var(--text-dim)] px-3 py-1 max-w-[90%] text-center">
          {content}
        </div>
      </div>
    )
  }

  // ── User message ──
  if (role === 'user') {
    return (
      <div className="flex flex-col items-end animate-fade-slide" role="listitem" aria-label={`You said: ${content.slice(0, 100)}`}>
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
    <div className="flex flex-col items-start animate-fade-slide" role="listitem" aria-label={`Agent said: ${content.slice(0, 100)}`}>
      <div
        className={`max-w-[680px] w-fit text-[var(--text)]
          px-0 py-1
          ${streaming ? 'animate-pulse-glow rounded-lg px-2' : ''}`}
      >
        <MarkdownRenderer content={content} streaming={streaming} />
        {streaming && (
          <span className="inline-block w-[2px] h-[1em] bg-[var(--accent)] ml-0.5 align-middle animate-blink" />
        )}
      </div>
      {(showTimestamp || cancelled) && !streaming && (
        <span className="text-[10px] text-[var(--text-dim)] mt-1 ml-1">
          {showTimestamp && formatTime(message.createdAt)}
          {message.model && showTimestamp && (
            <span className="ml-1.5 text-[var(--text-dim)]">· {message.model}</span>
          )}
          {cancelled && (
            <span className="ml-1.5 text-[var(--warning)] italic">· stopped</span>
          )}
        </span>
      )}
    </div>
  )
}

export const MessageBubble = memo(MessageBubbleInner)
