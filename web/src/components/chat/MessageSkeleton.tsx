// ============================================================================
// Operator OS — MessageSkeleton
// Loading placeholder for message history. Shows shimmer effect.
// ============================================================================

function SkeletonLine({ width }: { width: string }) {
  return (
    <div
      className="h-3 rounded bg-[var(--surface-3)] animate-pulse"
      style={{ width }}
    />
  )
}

function SkeletonBubble({ align, lines }: { align: 'left' | 'right'; lines: string[] }) {
  return (
    <div className={`flex ${align === 'right' ? 'justify-end' : 'justify-start'}`}>
      <div
        className={`rounded-2xl px-4 py-3 space-y-2 ${
          align === 'right'
            ? 'bg-[var(--user-bg)] border border-[var(--user-border)] rounded-br-[var(--radius-xs)]'
            : 'bg-[var(--surface-2)] rounded-bl-[var(--radius-xs)]'
        }`}
        style={{ maxWidth: 340 }}
      >
        {lines.map((w, i) => (
          <SkeletonLine key={i} width={w} />
        ))}
      </div>
    </div>
  )
}

export function MessageSkeleton({ count = 4 }: { count?: number }) {
  // Predefined patterns that look like a real conversation
  const patterns: { align: 'left' | 'right'; lines: string[] }[] = [
    { align: 'right', lines: ['180px'] },
    { align: 'left', lines: ['240px', '200px', '160px'] },
    { align: 'right', lines: ['120px', '80px'] },
    { align: 'left', lines: ['280px', '220px'] },
    { align: 'right', lines: ['200px'] },
    { align: 'left', lines: ['300px', '260px', '180px', '140px'] },
  ]

  return (
    <div className="space-y-5 animate-fade-in">
      {patterns.slice(0, count).map((p, i) => (
        <SkeletonBubble key={i} align={p.align} lines={p.lines} />
      ))}
    </div>
  )
}
