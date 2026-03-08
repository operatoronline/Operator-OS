// ============================================================================
// Operator OS — Chat Store
// Zustand store for WebSocket connection state, messages, and sessions.
// Bridges the WebSocketManager events into reactive React state.
// ============================================================================

import { create } from 'zustand'
import { wsManager } from '../services/ws'
import type { ConnectionState } from '../types/ws'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ChatMessage {
  id: string
  role: 'user' | 'agent' | 'system'
  content: string
  sessionId: string
  agentId?: string
  model?: string
  createdAt: string
  /** true while the message is still streaming */
  streaming?: boolean
}

interface ChatState {
  // Connection
  connectionState: ConnectionState
  reconnectVisible: boolean

  // Messages
  messages: ChatMessage[]
  isTyping: boolean

  // Active session/agent
  activeSessionId: string | null
  activeAgentId: string | null

  // Actions
  connect: () => void
  disconnect: () => void
  sendMessage: (content: string) => void
  cancelGeneration: () => void
  setActiveSession: (sessionId: string | null) => void
  setActiveAgent: (agentId: string | null) => void
  clearMessages: () => void
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

let listenersBound = false
let cleanupFns: (() => void)[] = []

function generateLocalId(): string {
  return `local-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useChatStore = create<ChatState>((set, get) => {
  // -------------------------------------------------------------------
  // Bind WebSocket event listeners (once)
  // -------------------------------------------------------------------
  function bindListeners() {
    if (listenersBound) return
    listenersBound = true

    // Connection state changes
    cleanupFns.push(
      wsManager.onStateChange((state) => {
        set({ connectionState: state, reconnectVisible: state === 'reconnecting' })
      }),
    )

    // message.create — new complete message from server
    cleanupFns.push(
      wsManager.on('message.create', (msg) => {
        const { messages } = get()
        const existing = messages.find((m) => m.id === msg.payload.message_id)

        if (existing) {
          // Finalize a streaming message
          set({
            messages: messages.map((m) =>
              m.id === msg.payload.message_id
                ? { ...m, content: msg.payload.content, streaming: false }
                : m,
            ),
            isTyping: false,
          })
        } else {
          // New message
          const newMsg: ChatMessage = {
            id: msg.payload.message_id,
            role: msg.payload.role,
            content: msg.payload.content,
            sessionId: msg.payload.session_id,
            agentId: msg.payload.agent_id,
            model: msg.payload.model,
            createdAt: msg.payload.created_at || new Date().toISOString(),
            streaming: false,
          }
          set({
            messages: [...messages, newMsg],
            isTyping: false,
          })
        }
      }),
    )

    // message.update — streaming token updates
    cleanupFns.push(
      wsManager.on('message.update', (msg) => {
        const { messages } = get()
        const existing = messages.find((m) => m.id === msg.payload.message_id)

        if (existing) {
          set({
            messages: messages.map((m) =>
              m.id === msg.payload.message_id
                ? {
                    ...m,
                    content: msg.payload.content,
                    streaming: !msg.payload.done,
                  }
                : m,
            ),
            isTyping: !msg.payload.done,
          })
        } else {
          // First streaming chunk — create a new message entry
          const newMsg: ChatMessage = {
            id: msg.payload.message_id,
            role: 'agent',
            content: msg.payload.content,
            sessionId: msg.payload.session_id,
            createdAt: new Date().toISOString(),
            streaming: !msg.payload.done,
          }
          set({
            messages: [...messages, newMsg],
            isTyping: !msg.payload.done,
          })
        }
      }),
    )

    // typing.start / typing.stop
    cleanupFns.push(
      wsManager.on('typing.start', () => set({ isTyping: true })),
    )
    cleanupFns.push(
      wsManager.on('typing.stop', () => set({ isTyping: false })),
    )

    // error — append as system message
    cleanupFns.push(
      wsManager.on('error', (msg) => {
        const { messages, activeSessionId } = get()
        const errMsg: ChatMessage = {
          id: generateLocalId(),
          role: 'system',
          content: `Error: ${msg.payload.message || 'Unknown error'}`,
          sessionId: activeSessionId || '',
          createdAt: new Date().toISOString(),
        }
        set({
          messages: [...messages, errMsg],
          isTyping: false,
        })
      }),
    )

    // session.created — update active session
    cleanupFns.push(
      wsManager.on('session.created', (msg) => {
        set({
          activeSessionId: msg.payload.session_id,
          activeAgentId: msg.payload.agent_id,
        })
      }),
    )
  }

  return {
    // State
    connectionState: 'disconnected',
    reconnectVisible: false,
    messages: [],
    isTyping: false,
    activeSessionId: null,
    activeAgentId: null,

    // -------------------------------------------------------------------
    // Connect
    // -------------------------------------------------------------------
    connect: () => {
      bindListeners()
      wsManager.connect()
    },

    // -------------------------------------------------------------------
    // Disconnect
    // -------------------------------------------------------------------
    disconnect: () => {
      wsManager.disconnect()
      // Cleanup listeners
      for (const fn of cleanupFns) fn()
      cleanupFns = []
      listenersBound = false
      set({
        connectionState: 'disconnected',
        reconnectVisible: false,
        isTyping: false,
      })
    },

    // -------------------------------------------------------------------
    // Send message
    // -------------------------------------------------------------------
    sendMessage: (content: string) => {
      const { messages, activeSessionId, activeAgentId } = get()
      const trimmed = content.trim()
      if (!trimmed) return

      // Optimistic user message
      const userMsg: ChatMessage = {
        id: generateLocalId(),
        role: 'user',
        content: trimmed,
        sessionId: activeSessionId || '',
        createdAt: new Date().toISOString(),
      }
      set({ messages: [...messages, userMsg] })

      // Send over WebSocket
      const sent = wsManager.send({
        type: 'message.send',
        payload: {
          content: trimmed,
          session_id: activeSessionId || undefined,
          agent_id: activeAgentId || undefined,
        },
      })

      if (!sent) {
        // Append error if send failed
        set((state) => ({
          messages: [
            ...state.messages,
            {
              id: generateLocalId(),
              role: 'system' as const,
              content: 'Failed to send — not connected.',
              sessionId: activeSessionId || '',
              createdAt: new Date().toISOString(),
            },
          ],
        }))
      }
    },

    // -------------------------------------------------------------------
    // Cancel generation
    // -------------------------------------------------------------------
    cancelGeneration: () => {
      wsManager.send({ type: 'message.cancel', payload: {} })
      set({ isTyping: false })
    },

    // -------------------------------------------------------------------
    // Session / agent selection
    // -------------------------------------------------------------------
    setActiveSession: (sessionId) => {
      set({ activeSessionId: sessionId, messages: [], isTyping: false })
    },

    setActiveAgent: (agentId) => {
      set({ activeAgentId: agentId })
    },

    // -------------------------------------------------------------------
    // Clear messages
    // -------------------------------------------------------------------
    clearMessages: () => {
      set({ messages: [], isTyping: false })
    },
  }
})
