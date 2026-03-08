import { ChatCircle } from '@phosphor-icons/react'

export function ChatPage() {
  return (
    <div className="h-full flex flex-col items-center justify-center text-text-dim">
      <ChatCircle size={48} weight="thin" className="mb-4 text-accent-text" />
      <h2 className="text-lg font-semibold text-text mb-1">Operator OS</h2>
      <p className="text-sm">Chat interface coming in Phase 2</p>
    </div>
  )
}
