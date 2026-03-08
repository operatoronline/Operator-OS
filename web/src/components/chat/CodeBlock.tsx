// ============================================================================
// Operator OS — CodeBlock
// Fenced code block with language label + copy button.
// Ports styling from legacy index.html: code-surface bg, border, radius-sm,
// JetBrains Mono, 12.5px font size.
// ============================================================================

import { useState, useCallback, type ReactNode } from 'react'
import { Copy, Check } from '@phosphor-icons/react'

interface CodeBlockProps {
  language?: string
  children: ReactNode
}

export function CodeBlock({ language, children }: CodeBlockProps) {
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(() => {
    const text = extractText(children)
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }, [children])

  return (
    <div className="group relative my-3 rounded-[var(--radius-sm)] border border-[var(--border)] bg-[var(--code-surface)] overflow-hidden">
      {/* Header bar */}
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-[var(--border-subtle)] bg-[var(--surface-2)]">
        <span className="text-[11px] font-medium text-[var(--text-dim)] uppercase tracking-wider select-none">
          {language || 'text'}
        </span>
        <button
          onClick={handleCopy}
          className="flex items-center gap-1 text-[11px] text-[var(--text-dim)] hover:text-[var(--text-secondary)] transition-colors cursor-pointer"
          aria-label={copied ? 'Copied' : 'Copy code'}
        >
          {copied ? (
            <>
              <Check size={13} weight="bold" className="text-[var(--success)]" />
              <span className="text-[var(--success)]">Copied</span>
            </>
          ) : (
            <>
              <Copy size={13} />
              <span className="opacity-0 group-hover:opacity-100 transition-opacity">Copy</span>
            </>
          )}
        </button>
      </div>

      {/* Code body */}
      <div className="overflow-x-auto p-4">
        <pre className="!m-0 !p-0 !bg-transparent !border-none">
          {children}
        </pre>
      </div>
    </div>
  )
}

/** Recursively extract text content from ReactNode */
function extractText(node: ReactNode): string {
  if (node == null || typeof node === 'boolean') return ''
  if (typeof node === 'string' || typeof node === 'number') return String(node)
  if (Array.isArray(node)) return node.map(extractText).join('')
  if (typeof node === 'object' && 'props' in node) {
    return extractText((node as { props: { children?: ReactNode } }).props.children)
  }
  return ''
}
