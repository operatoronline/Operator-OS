// ============================================================================
// Operator OS — useWebSocket Hook
// Connects/disconnects the WebSocket based on auth state.
// Drop this into any component that needs the chat connection active.
// ============================================================================

import { useEffect } from 'react'
import { useAuthStore } from '../stores/authStore'
import { useChatStore } from '../stores/chatStore'

/**
 * Manages WebSocket lifecycle tied to authentication state.
 *
 * - Connects when the user is authenticated
 * - Disconnects on logout
 * - Auto-reconnects are handled internally by WebSocketManager
 *
 * Usage: call `useWebSocket()` once in a top-level authenticated component
 * (e.g., Chat page or AppShell).
 */
export function useWebSocket(): void {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const isInitialized = useAuthStore((s) => s.isInitialized)
  const connect = useChatStore((s) => s.connect)
  const disconnect = useChatStore((s) => s.disconnect)

  useEffect(() => {
    if (!isInitialized) return

    if (isAuthenticated) {
      connect()
    } else {
      disconnect()
    }

    return () => {
      disconnect()
    }
  }, [isAuthenticated, isInitialized, connect, disconnect])
}
