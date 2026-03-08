// ============================================================================
// Operator OS — Chat Page
// Main chat interface with message thread, connection state, and composer stub.
// ============================================================================

import { ChatCircle, ArrowsClockwise, WifiSlash, Stop } from '@phosphor-icons/react'
import { useWebSocket } from '../hooks/useWebSocket'
import { useChatStore } from '../stores/chatStore'
import { ConnectionStatus } from '../components/chat/ConnectionStatus'
import { MessageList } from '../components/chat/MessageList'
import { Button } from '../components/shared'

export function ChatPage() {
  // Activate WebSocket connection while on this page
  useWebSocket()

  const connectionState = useChatStore((s) => s.connectionState)
  const connect = useChatStore((s) => s.connect)
  const messages = useChatStore((s) => s.messages)
  const isTyping = useChatStore((s) => s.isTyping)
  const streamingMessageId = useChatStore((s) => s.streamingMessageId)
  const cancelGeneration = useChatStore((s) => s.cancelGeneration)

  return (
    <div className="h-full flex flex-col">
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
      {messages.length === 0 && !isTyping ? (
        /* ─── Welcome / empty state ─── */
        <div className="flex-1 flex flex-col items-center justify-center overflow-y-auto">
          <div className="flex flex-col items-center text-text-dim px-4 text-center">
            <div className="w-16 h-16 rounded-2xl bg-[var(--accent-subtle)] flex items-center justify-center mb-4">
              <ChatCircle size={32} weight="thin" className="text-[var(--accent-text)]" />
            </div>
            <h2 className="text-lg font-semibold text-[var(--text)] mb-1">
              Operator OS
            </h2>
            <p className="text-sm text-[var(--text-secondary)] mb-4">
              Your AI-powered workspace. Start a conversation.
            </p>
            <ConnectionStatus showLabel />
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

      {/* ─── Composer (C10 will upgrade to full composer with file upload) ─── */}
      <div className="border-t border-[var(--border-subtle)] bg-[var(--surface)] p-3 shrink-0">
        <ComposerStub />
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Minimal composer for testing (replaced in C10 with full Composer component)
// ---------------------------------------------------------------------------

function ComposerStub() {
  const sendMessage = useChatStore((s) => s.sendMessage)
  const connectionState = useChatStore((s) => s.connectionState)
  const streamingMessageId = useChatStore((s) => s.streamingMessageId)

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const form = e.currentTarget
    const input = form.elements.namedItem('message') as HTMLInputElement
    if (!input.value.trim()) return
    sendMessage(input.value)
    input.value = ''
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      const form = e.currentTarget.form
      if (form) form.requestSubmit()
    }
  }

  const disabled = connectionState !== 'connected' || !!streamingMessageId

  return (
    <div className="max-w-3xl mx-auto w-full">
      <form onSubmit={handleSubmit} className="flex items-end gap-2">
        <textarea
          name="message"
          rows={1}
          placeholder={
            streamingMessageId
              ? 'Waiting for response…'
              : disabled
                ? 'Connecting…'
                : 'Message Operator OS…'
          }
          disabled={disabled}
          onKeyDown={handleKeyDown}
          className="flex-1 resize-none bg-[var(--surface-2)] text-[var(--text)] text-sm rounded-xl px-4 py-2.5 border border-[var(--border-subtle)] focus:border-[var(--accent)] focus:outline-none placeholder:text-[var(--text-dim)] disabled:opacity-50 transition-colors"
          style={{ maxHeight: 120 }}
        />
        <Button
          type="submit"
          size="sm"
          disabled={disabled}
        >
          Send
        </Button>
      </form>
    </div>
  )
}
