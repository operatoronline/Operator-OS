// ============================================================================
// Operator OS — TypingIndicator
// Three-dot bounce animation shown when the agent is processing/typing.
// Ports the legacy typing-dots treatment.
// ============================================================================

export function TypingIndicator() {
  return (
    <div className="flex items-start animate-fade-slide">
      <div className="flex items-center gap-1 px-3 py-2.5 rounded-2xl bg-[var(--surface-2)] rounded-bl-[var(--radius-xs)]">
        <span className="w-1.5 h-1.5 rounded-full bg-[var(--text-dim)] animate-bounce [animation-delay:0ms] [animation-duration:1.2s]" />
        <span className="w-1.5 h-1.5 rounded-full bg-[var(--text-dim)] animate-bounce [animation-delay:200ms] [animation-duration:1.2s]" />
        <span className="w-1.5 h-1.5 rounded-full bg-[var(--text-dim)] animate-bounce [animation-delay:400ms] [animation-duration:1.2s]" />
      </div>
    </div>
  )
}
