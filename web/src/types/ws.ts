// ============================================================================
// Operator OS — WebSocket Message Types
// Protocol types for real-time communication with the backend.
// Mirrors the legacy Pico protocol, adapted for /api/v1/ws + JWT auth.
// ============================================================================

// ---------------------------------------------------------------------------
// Connection states
// ---------------------------------------------------------------------------

export type ConnectionState =
  | 'disconnected'
  | 'connecting'
  | 'connected'
  | 'reconnecting'

// ---------------------------------------------------------------------------
// Outbound messages (client → server)
// ---------------------------------------------------------------------------

export interface WsSendMessage {
  type: 'message.send'
  payload: {
    content: string
    session_id?: string
    agent_id?: string
    attachments?: WsAttachment[]
  }
}

export interface WsPing {
  type: 'ping'
}

export interface WsCancel {
  type: 'message.cancel'
  payload: {
    message_id?: string
  }
}

export type WsOutbound = WsSendMessage | WsPing | WsCancel

// ---------------------------------------------------------------------------
// Inbound messages (server → client)
// ---------------------------------------------------------------------------

export interface WsMessageCreate {
  type: 'message.create'
  payload: {
    message_id: string
    session_id: string
    role: 'user' | 'agent' | 'system'
    content: string
    agent_id?: string
    model?: string
    created_at?: string
  }
}

export interface WsMessageUpdate {
  type: 'message.update'
  payload: {
    message_id: string
    session_id: string
    content: string
    /** true when this is the final update (streaming complete) */
    done?: boolean
  }
}

export interface WsTypingStart {
  type: 'typing.start'
  payload: {
    session_id?: string
    agent_id?: string
  }
}

export interface WsTypingStop {
  type: 'typing.stop'
  payload: {
    session_id?: string
  }
}

export interface WsError {
  type: 'error'
  payload: {
    code?: string
    message: string
  }
}

export interface WsPong {
  type: 'pong'
}

export interface WsSessionCreated {
  type: 'session.created'
  payload: {
    session_id: string
    agent_id: string
    name?: string
  }
}

export type WsInbound =
  | WsMessageCreate
  | WsMessageUpdate
  | WsTypingStart
  | WsTypingStop
  | WsError
  | WsPong
  | WsSessionCreated

// ---------------------------------------------------------------------------
// Attachments
// ---------------------------------------------------------------------------

export interface WsAttachment {
  name: string
  type: string
  /** Base64-encoded content or URL */
  data: string
}

// ---------------------------------------------------------------------------
// Event handler map
// ---------------------------------------------------------------------------

export type WsEventMap = {
  'message.create': WsMessageCreate
  'message.update': WsMessageUpdate
  'typing.start': WsTypingStart
  'typing.stop': WsTypingStop
  'error': WsError
  'pong': WsPong
  'session.created': WsSessionCreated
}

export type WsEventType = keyof WsEventMap
