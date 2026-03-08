// ============================================================================
// Operator OS — Chat Page
// Main chat interface with session panel, message thread, and composer.
// ============================================================================

import { useState } from 'react'
import {
  ChatCircle,
  ArrowsClockwise,
  WifiSlash,
  Stop,
  List,
  WarningCircle,
} from '@phosphor-icons/react'
import { useWebSocket } from '../hooks/useWebSocket'
import { useChatStore } from '../stores/chatStore'
import { useSessionStore } from '../stores/sessionStore'
import { ConnectionStatus } from '../components/chat/ConnectionStatus'
import { MessageList } from '../components/chat/MessageList'
import { MessageSkeleton } from '../components/chat/MessageSkeleton'
import { Composer } from '../components/chat/Composer'
import { SessionPanel } from '../components/sessions/SessionPanel'

export function ChatPage() {
  // Activate WebSocket connection while on this page
  useWebSocket()

  const connectionState = useChatStore((s) => s.connectionState)
  const connect = useChatStore((s) => s.connect)
  const messages = useChatStore((s) => s.messages)
  const isTyping = useChatStore((s) => s.isTyping)
  const streamingMessageId = useChatStore((s) => s.streamingMessageId)
  const cancelGeneration = useChatStore((s) => s.cancelGeneration)
  const loadingHistory = useChatStore((s) => s.loadingHistory)
  const historyError = useChatStore((s) => s.historyError)

  const activeSessionId = useSessionStore((s) => s.activeSessionId)
  const activeSession = useSessionStore((s) => s.sessions.find((sess) => sess.id === s.activeSessionId))

  // Mobile session panel toggle
  const [mobileSessionsOpen, setMobileSessionsOpen] = useState(false)

  const hasSession = !!activeSessionId

  return (
    <div className="h-full flex overflow-hidden">
      {/* ─── Desktop session panel ─── */}
      <SessionPanel />

      {/* ─── Mobile session panel (overlay) ─── */}
      <SessionPanel
        mobile
        open={mobileSessionsOpen}
        onClose={() => setMobileSessionsOpen(false)}
      />

      {/* ─── Main chat area ─── */}
      <div className="flex-1 flex flex-col min-w-0 h-full">
        {/* ─── Chat header bar ─── */}
        <div className="flex items-center gap-3 px-4 h-11 border-b border-[var(--border-subtle)] shrink-0 bg-[var(--surface)]">
          {/* Mobile menu toggle */}
          <button
            onClick={() => setMobileSessionsOpen(true)}
            className="md:hidden p-1 rounded-md text-[var(--text-dim)] hover:text-[var(--text)]
              hover:bg-[var(--surface-2)] transition-colors cursor-pointer"
            aria-label="Open sessions"
          >
            <List size={18} />
          </button>

          {/* Session name */}
          <div className="flex-1 min-w-0">
            {activeSession ? (
              <span className="text-[13px] font-medium text-[var(--text)] truncate">
                {activeSession.name}
              </span>
            ) : (
              <span className="text-[13px] text-[var(--text-dim)]">
                Select or start a conversation
              </span>
            )}
          </div>

          {/* Connection indicator */}
          <ConnectionStatus />
        </div>

        {/* ─── Connection banner (shows when not connected) ─── */}
        {(connectionState === 'reconnecting' || connectionState === 'disconnected') && (
          <div className="flex items-center justify-center gap-2 px-4 py-2 text-xs bg-[var(--warning-subtle)] text-[var(--warning)] border-b border-[var(--border-subtle)] animate-fade-slide shrink-0">
            {connectionState === 'reconnecting' ? (
              <>
                <ArrowsClockwise size={14} weight="bold" className="animate-spin" />
                <span>Reconnecting to server…</span>
              </>
            ) : (
              <>
                <WifiSlash size={14} weight="bold" />
                <span>Disconnected</span>
                <button
                  onClick={() => connect()}
                  className="ml-2 underline hover:no-underline cursor-pointer"
                >
                  Retry
                </button>
              </>
            )}
          </div>
        )}

        {/* ─── Chat area ─── */}
        {!hasSession ? (
          /* ─── No session selected ─── */
          <div className="flex-1 flex flex-col items-center justify-center overflow-y-auto">
            <div className="flex flex-col items-center text-text-dim px-4 text-center">
              <div className="w-16 h-16 rounded-2xl bg-[var(--accent-subtle)] flex items-center justify-center mb-4">
                <ChatCircle size={32} weight="thin" className="text-[var(--accent-text)]" />
              </div>
              <h2 className="text-lg font-semibold text-[var(--text)] mb-1">
                Operator OS
              </h2>
              <p className="text-sm text-[var(--text-secondary)] mb-4">
                Select a conversation from the sidebar or start a new one.
              </p>
              <ConnectionStatus showLabel />
            </div>
          </div>
        ) : loadingHistory ? (
          /* ─── Loading history ─── */
          <div className="flex-1 overflow-y-auto">
            <MessageSkeleton />
          </div>
        ) : historyError ? (
          /* ─── History load error ─── */
          <div className="flex-1 flex flex-col items-center justify-center px-4">
            <WarningCircle size={32} weight="thin" className="text-[var(--error)] mb-3" />
            <p className="text-sm text-[var(--error)] mb-2">Failed to load messages</p>
            <p className="text-xs text-[var(--text-dim)] mb-3">{historyError}</p>
            <button
              onClick={() => useChatStore.getState().loadSessionHistory(activeSessionId!)}
              className="text-xs text-[var(--accent-text)] hover:underline cursor-pointer"
            >
              Try again
            </button>
          </div>
        ) : messages.length === 0 && !isTyping ? (
          /* ─── Empty session ─── */
          <div className="flex-1 flex flex-col items-center justify-center overflow-y-auto">
            <div className="flex flex-col items-center text-text-dim px-4 text-center">
              <div className="w-12 h-12 rounded-xl bg-[var(--accent-subtle)] flex items-center justify-center mb-3">
                <ChatCircle size={24} weight="thin" className="text-[var(--accent-text)]" />
              </div>
              <p className="text-sm text-[var(--text-secondary)] mb-1">
                Start the conversation
              </p>
              <p className="text-xs text-[var(--text-dim)]">
                Send a message to begin.
              </p>
            </div>
          </div>
        ) : (
          /* ─── Message thread ─── */
          <MessageList
            messages={messages}
            isTyping={isTyping}
            streamingMessageId={streamingMessageId}
          />
        )}

        {/* ─── Stop generating overlay ─── */}
        {streamingMessageId && (
          <div className="flex justify-center pb-2 shrink-0">
            <button
              onClick={cancelGeneration}
              className="flex items-center gap-1.5 px-4 py-1.5 rounded-full
                bg-[var(--surface-2)] border border-[var(--border)]
                text-xs text-[var(--text-secondary)] font-medium
                hover:bg-[var(--surface-3)] hover:text-[var(--text)] hover:border-[var(--border-subtle)]
                active:scale-95 transition-all duration-150 cursor-pointer
                animate-fade-slide"
            >
              <Stop size={14} weight="fill" />
              Stop generating
            </button>
          </div>
        )}

        {/* ─── Composer ─── */}
        {hasSession && (
          <div className="border-t border-[var(--border-subtle)] bg-[var(--surface)] p-3 shrink-0">
            <Composer />
          </div>
        )}
      </div>
    </div>
  )
}
