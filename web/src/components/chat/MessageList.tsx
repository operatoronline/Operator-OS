// ============================================================================
// Operator OS — MessageList
// Scrollable container for chat messages. Handles auto-scroll, scroll-to-bottom
// button, timestamp grouping, and the typing indicator.
// ============================================================================

import { useRef, useEffect, useCallback, useState, useMemo } from 'react'
import type { ChatMessage } from '../../stores/chatStore'
import { MessageBubble } from './MessageBubble'
import { TypingIndicator } from './TypingIndicator'
import { ScrollToBottom } from './ScrollToBottom'

interface MessageListProps {
  messages: ChatMessage[]
  isTyping: boolean
  /** Set true while loading history */
  loading?: boolean
  /** ID of the currently streaming message (for auto-scroll during streaming) */
  streamingMessageId?: string | null
}

// Show timestamp if messages are >2 minutes apart or from different roles
function shouldShowTimestamp(
  current: ChatMessage,
  previous: ChatMessage | undefined,
): boolean {
  if (!previous) return true
  if (previous.role !== current.role) return true

  const gap =
    new Date(current.createdAt).getTime() - new Date(previous.createdAt).getTime()
  return gap > 2 * 60 * 1000 // 2 minutes
}

export function MessageList({ messages, isTyping, loading = false, streamingMessageId }: MessageListProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const bottomRef = useRef<HTMLDivElement>(null)
  const [isAtBottom, setIsAtBottom] = useState(true)
  const [unreadCount, setUnreadCount] = useState(0)

  // ── Detect if user is at bottom ──
  const checkIsAtBottom = useCallback(() => {
    const el = containerRef.current
    if (!el) return true
    // "At bottom" = within 80px of the end
    return el.scrollHeight - el.scrollTop - el.clientHeight < 80
  }, [])

  // ── Scroll to bottom (smooth) ──
  const scrollToBottom = useCallback((behavior: ScrollBehavior = 'smooth') => {
    bottomRef.current?.scrollIntoView({ behavior, block: 'end' })
    setIsAtBottom(true)
    setUnreadCount(0)
  }, [])

  // ── Handle scroll events ──
  const handleScroll = useCallback(() => {
    const atBottom = checkIsAtBottom()
    setIsAtBottom(atBottom)
    if (atBottom) setUnreadCount(0)
  }, [checkIsAtBottom])

  // ── Auto-scroll on new messages if already at bottom ──
  useEffect(() => {
    if (isAtBottom) {
      // Instant scroll for first render, smooth for subsequent
      const behavior = messages.length <= 1 ? 'instant' : 'smooth'
      scrollToBottom(behavior as ScrollBehavior)
    } else {
      // User is scrolled up — increment unread
      const lastMsg = messages[messages.length - 1]
      if (lastMsg && lastMsg.role !== 'user') {
        setUnreadCount((c) => c + 1)
      }
    }
    // Only react to message count changes
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [messages.length])

  // ── Auto-scroll when typing starts ──
  useEffect(() => {
    if (isTyping && isAtBottom) {
      scrollToBottom()
    }
  }, [isTyping, isAtBottom, scrollToBottom])

  // ── Auto-scroll during streaming content growth ──
  // Track content length of streaming message to scroll as tokens arrive
  const streamingContent = streamingMessageId
    ? messages.find((m) => m.id === streamingMessageId)?.content
    : undefined

  useEffect(() => {
    if (streamingContent !== undefined && isAtBottom) {
      // Use instant scroll for streaming to avoid jitter from queued smooth scrolls
      bottomRef.current?.scrollIntoView({ behavior: 'instant', block: 'end' })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [streamingContent?.length])

  // ── Initial scroll to bottom ──
  useEffect(() => {
    if (messages.length > 0) {
      scrollToBottom('instant' as ScrollBehavior)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Screen reader announcement for new messages
  const lastMessage = messages[messages.length - 1]
  const lastNonStreamingMsg = useMemo(() => {
    if (!lastMessage || lastMessage.streaming) return null
    const prefix = lastMessage.role === 'user' ? 'You' : 'Agent'
    return `${prefix}: ${lastMessage.content.slice(0, 200)}`
  }, [lastMessage?.id, lastMessage?.streaming]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="relative flex-1 min-h-0">
      {/* ─── Scrollable message area ─── */}
      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="h-full overflow-y-auto overscroll-contain
          [-webkit-overflow-scrolling:touch]
          [&::-webkit-scrollbar]:w-[3px]
          [&::-webkit-scrollbar-track]:bg-transparent
          [&::-webkit-scrollbar-thumb]:bg-[var(--scrollthumb)]
          [&::-webkit-scrollbar-thumb]:rounded-sm"
      >
        <div className="max-w-3xl mx-auto px-4 md:px-5 pt-6 pb-2 flex flex-col gap-5">
          {/* Loading skeleton placeholder */}
          {loading && (
            <div className="flex justify-center py-4">
              <div className="flex items-center gap-2 text-xs text-[var(--text-dim)]">
                <div className="w-4 h-4 border-2 border-[var(--border)] border-t-[var(--accent)] rounded-full animate-spin" />
                Loading history…
              </div>
            </div>
          )}

          {/* Messages */}
          <div role="list" aria-label="Chat messages">
            {messages.map((msg, i) => (
              <MessageBubble
                key={msg.id}
                message={msg}
                showTimestamp={shouldShowTimestamp(msg, messages[i - 1])}
              />
            ))}
          </div>

          {/* Typing indicator */}
          {isTyping && !messages.some((m) => m.streaming) && <TypingIndicator />}

          {/* Bottom anchor for auto-scroll */}
          <div ref={bottomRef} className="h-px shrink-0" />

          {/* Bottom spacer so messages don't hide behind composer */}
          <div className="h-2 shrink-0 pointer-events-none" />
        </div>
      </div>

      {/* ─── Scroll-to-bottom button ─── */}
      <ScrollToBottom
        visible={!isAtBottom}
        onClick={() => scrollToBottom()}
        unreadCount={unreadCount}
      />

      {/* ─── Screen reader live region for new messages ─── */}
      <div
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      >
        {lastNonStreamingMsg}
      </div>
    </div>
  )
}
