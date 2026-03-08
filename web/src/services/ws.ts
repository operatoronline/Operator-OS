// ============================================================================
// Operator OS — WebSocket Transport
// Manages the WebSocket connection lifecycle: connect, reconnect, ping,
// message dispatch. Designed for use with the Zustand chat store.
// ============================================================================

import { tokenStore } from './api'
import type {
  ConnectionState,
  WsOutbound,
  WsInbound,
  WsEventType,
  WsEventMap,
} from '../types/ws'

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

const WS_PATH = '/api/v1/ws'
const PING_INTERVAL_MS = 25_000
const RECONNECT_BASE_MS = 1_000
const RECONNECT_MAX_MS = 30_000
const RECONNECT_JITTER = 0.3 // ±30% jitter

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Listener<T extends WsEventType = WsEventType> = (msg: WsEventMap[T]) => void
type StateListener = (state: ConnectionState) => void

// ---------------------------------------------------------------------------
// WebSocket Manager (singleton-ish, but instantiable for testing)
// ---------------------------------------------------------------------------

export class WebSocketManager {
  private ws: WebSocket | null = null
  private state: ConnectionState = 'disconnected'
  private reconnectAttempt = 0
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private pingTimer: ReturnType<typeof setInterval> | null = null
  private listeners = new Map<WsEventType, Set<Listener>>()
  private stateListeners = new Set<StateListener>()
  private intentionalClose = false

  // -----------------------------------------------------------------------
  // Public API
  // -----------------------------------------------------------------------

  /** Current connection state */
  getState(): ConnectionState {
    return this.state
  }

  /**
   * Open a WebSocket connection.
   * JWT token is sent as a query param for the upgrade handshake.
   * If already connected/connecting, this is a no-op.
   */
  connect(): void {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    const token = tokenStore.getAccess()
    if (!token) {
      console.warn('[ws] No access token — cannot connect')
      this.setState('disconnected')
      return
    }

    this.intentionalClose = false
    this.setState(this.reconnectAttempt > 0 ? 'reconnecting' : 'connecting')

    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${proto}//${location.host}${WS_PATH}?token=${encodeURIComponent(token)}`

    try {
      this.ws = new WebSocket(url)
    } catch (err) {
      console.error('[ws] Failed to create WebSocket:', err)
      this.scheduleReconnect()
      return
    }

    this.ws.onopen = this.handleOpen
    this.ws.onclose = this.handleClose
    this.ws.onerror = this.handleError
    this.ws.onmessage = this.handleMessage
  }

  /**
   * Gracefully close the connection. No auto-reconnect.
   */
  disconnect(): void {
    this.intentionalClose = true
    this.clearTimers()
    if (this.ws) {
      this.ws.onclose = null // prevent reconnect handler
      this.ws.close(1000, 'client disconnect')
      this.ws = null
    }
    this.reconnectAttempt = 0
    this.setState('disconnected')
  }

  /**
   * Send a typed message over the WebSocket.
   * Returns false if the socket isn't open.
   */
  send(msg: WsOutbound): boolean {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('[ws] Cannot send — not connected')
      return false
    }
    try {
      this.ws.send(JSON.stringify(msg))
      return true
    } catch (err) {
      console.error('[ws] Send failed:', err)
      return false
    }
  }

  /**
   * Subscribe to a specific message type.
   * Returns an unsubscribe function.
   */
  on<T extends WsEventType>(type: T, listener: Listener<T>): () => void {
    if (!this.listeners.has(type)) {
      this.listeners.set(type, new Set())
    }
    const set = this.listeners.get(type)!
    set.add(listener as Listener)
    return () => set.delete(listener as Listener)
  }

  /**
   * Subscribe to connection state changes.
   * Returns an unsubscribe function.
   */
  onStateChange(listener: StateListener): () => void {
    this.stateListeners.add(listener)
    return () => this.stateListeners.delete(listener)
  }

  // -----------------------------------------------------------------------
  // Internal handlers
  // -----------------------------------------------------------------------

  private handleOpen = (): void => {
    this.reconnectAttempt = 0
    this.setState('connected')
    this.startPing()
  }

  private handleClose = (ev: CloseEvent): void => {
    this.ws = null
    this.stopPing()

    if (this.intentionalClose) {
      this.setState('disconnected')
      return
    }

    // Auth failure — don't reconnect, emit auth expired
    if (ev.code === 4001 || ev.code === 4003) {
      this.setState('disconnected')
      window.dispatchEvent(new CustomEvent('os:auth:expired'))
      return
    }

    this.scheduleReconnect()
  }

  private handleError = (): void => {
    // Error is always followed by close, so we just log
    console.error('[ws] Connection error')
  }

  private handleMessage = (ev: MessageEvent): void => {
    let msg: WsInbound
    try {
      msg = JSON.parse(ev.data as string)
    } catch {
      console.warn('[ws] Invalid JSON:', ev.data)
      return
    }

    // Dispatch to type-specific listeners
    const set = this.listeners.get(msg.type as WsEventType)
    if (set) {
      for (const listener of set) {
        try {
          listener(msg as never)
        } catch (err) {
          console.error(`[ws] Listener error for ${msg.type}:`, err)
        }
      }
    }
  }

  // -----------------------------------------------------------------------
  // Reconnect with exponential backoff + jitter
  // -----------------------------------------------------------------------

  private scheduleReconnect(): void {
    this.setState('reconnecting')

    const base = Math.min(
      RECONNECT_BASE_MS * Math.pow(2, this.reconnectAttempt),
      RECONNECT_MAX_MS,
    )
    const jitter = base * RECONNECT_JITTER * (Math.random() * 2 - 1)
    const delay = Math.round(base + jitter)

    this.reconnectAttempt++
    console.log(`[ws] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempt})`)

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, delay)
  }

  // -----------------------------------------------------------------------
  // Ping keepalive
  // -----------------------------------------------------------------------

  private startPing(): void {
    this.stopPing()
    this.pingTimer = setInterval(() => {
      this.send({ type: 'ping' })
    }, PING_INTERVAL_MS)
  }

  private stopPing(): void {
    if (this.pingTimer) {
      clearInterval(this.pingTimer)
      this.pingTimer = null
    }
  }

  // -----------------------------------------------------------------------
  // State management
  // -----------------------------------------------------------------------

  private setState(next: ConnectionState): void {
    if (this.state === next) return
    this.state = next
    for (const listener of this.stateListeners) {
      try {
        listener(next)
      } catch (err) {
        console.error('[ws] State listener error:', err)
      }
    }
  }

  private clearTimers(): void {
    this.stopPing()
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }
}

// ---------------------------------------------------------------------------
// Default singleton instance
// ---------------------------------------------------------------------------

export const wsManager = new WebSocketManager()
