// ============================================================================
// Operator OS — Session Export Utilities
// Export conversation history as Markdown or JSON.
// ============================================================================

import { api } from '../services/api'
import type { Session, SessionMessage } from '../types/api'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function downloadBlob(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

function formatDate(iso: string): string {
  if (!iso) return ''
  return new Date(iso).toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function sanitizeFilename(name: string): string {
  return name.replace(/[^a-zA-Z0-9_\- ]/g, '').trim().replace(/\s+/g, '-') || 'session'
}

// ---------------------------------------------------------------------------
// Fetch all messages for a session (paginated)
// ---------------------------------------------------------------------------

async function fetchAllMessages(sessionId: string): Promise<SessionMessage[]> {
  const all: SessionMessage[] = []
  let page = 1
  const perPage = 100

  while (true) {
    const batch = await api.sessions.messages(sessionId, { page, per_page: perPage })
    all.push(...batch)
    if (batch.length < perPage) break
    page++
  }

  return all
}

// ---------------------------------------------------------------------------
// Export as Markdown
// ---------------------------------------------------------------------------

export async function exportAsMarkdown(session: Session): Promise<void> {
  const messages = await fetchAllMessages(session.id)
  const filename = `${sanitizeFilename(session.name)}.md`

  const lines: string[] = [
    `# ${session.name}`,
    '',
    `**Session ID:** ${session.id}`,
    `**Created:** ${formatDate(session.created_at)}`,
    `**Messages:** ${messages.length}`,
    '',
    '---',
    '',
  ]

  for (const msg of messages) {
    const role = msg.role === 'user' ? '👤 User' : msg.role === 'agent' ? '🤖 Agent' : '⚙️ System'
    const time = formatDate(msg.created_at)
    const model = msg.model ? ` (${msg.model})` : ''

    lines.push(`### ${role}${model}`)
    lines.push(`*${time}*`)
    lines.push('')
    lines.push(msg.content)
    lines.push('')
    lines.push('---')
    lines.push('')
  }

  downloadBlob(lines.join('\n'), filename, 'text/markdown')
}

// ---------------------------------------------------------------------------
// Export as JSON
// ---------------------------------------------------------------------------

export async function exportAsJSON(session: Session): Promise<void> {
  const messages = await fetchAllMessages(session.id)
  const filename = `${sanitizeFilename(session.name)}.json`

  const data = {
    session: {
      id: session.id,
      name: session.name,
      agent_id: session.agent_id,
      created_at: session.created_at,
      updated_at: session.updated_at,
      message_count: messages.length,
    },
    messages: messages.map((m) => ({
      id: m.id,
      role: m.role,
      content: m.content,
      model: m.model,
      agent_id: m.agent_id,
      created_at: m.created_at,
    })),
    exported_at: new Date().toISOString(),
  }

  downloadBlob(JSON.stringify(data, null, 2), filename, 'application/json')
}
