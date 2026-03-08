// ============================================================================
// Operator OS — MarkdownRenderer
// Renders markdown content using react-markdown with GFM support and
// syntax highlighting. Sanitized via DOMPurify. Matches legacy index.html
// visual treatment (OKLCH tokens, typography, spacing).
// ============================================================================

import { memo, useMemo, type ComponentPropsWithoutRef } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import DOMPurify from 'dompurify'
import { CodeBlock } from './CodeBlock'

interface MarkdownRendererProps {
  content: string
  /** Whether the message is still streaming (disables some heavy processing) */
  streaming?: boolean
}

// Configure DOMPurify once
DOMPurify.addHook('afterSanitizeAttributes', (node) => {
  // Open links in new tab
  if (node.tagName === 'A') {
    node.setAttribute('target', '_blank')
    node.setAttribute('rel', 'noopener noreferrer')
  }
})

// Remark plugins (stable reference)
const remarkPlugins = [remarkGfm]
const rehypePlugins = [rehypeHighlight]

// Custom components for react-markdown
const mdComponents = {
  // ── Code blocks ──
  pre({ children }: ComponentPropsWithoutRef<'pre'>) {
    return <>{children}</>  // CodeBlock wraps the <pre> itself
  },

  code({ className, children, ...props }: ComponentPropsWithoutRef<'code'> & { node?: unknown }) {
    const match = /language-(\w+)/.exec(className || '')
    const isBlock = match || (typeof children === 'string' && children.includes('\n'))

    if (isBlock) {
      return (
        <CodeBlock language={match?.[1]}>
          <code className={className} {...props}>
            {children}
          </code>
        </CodeBlock>
      )
    }

    // Inline code
    return (
      <code
        className="bg-[var(--surface-2)] border border-[var(--border-subtle)] px-[var(--sp-1)] py-[1px] rounded-[var(--radius-xs)] font-mono text-[12.5px] text-[var(--accent-text)]"
        {...props}
      >
        {children}
      </code>
    )
  },

  // ── Block elements ──
  p({ children, ...props }: ComponentPropsWithoutRef<'p'>) {
    return <p className="my-2" {...props}>{children}</p>
  },

  h1({ children, ...props }: ComponentPropsWithoutRef<'h1'>) {
    return <h1 className="text-lg font-semibold mt-4 mb-2" {...props}>{children}</h1>
  },
  h2({ children, ...props }: ComponentPropsWithoutRef<'h2'>) {
    return <h2 className="text-base font-semibold mt-3 mb-2" {...props}>{children}</h2>
  },
  h3({ children, ...props }: ComponentPropsWithoutRef<'h3'>) {
    return <h3 className="text-sm font-semibold text-[var(--text-secondary)] mt-3 mb-1" {...props}>{children}</h3>
  },

  blockquote({ children, ...props }: ComponentPropsWithoutRef<'blockquote'>) {
    return (
      <blockquote
        className="border-l-[3px] border-[var(--accent)] pl-[var(--sp-4)] my-3 text-[var(--text-secondary)]"
        {...props}
      >
        {children}
      </blockquote>
    )
  },

  // ── Lists ──
  ul({ children, ...props }: ComponentPropsWithoutRef<'ul'>) {
    return <ul className="my-2 pl-5 list-disc" {...props}>{children}</ul>
  },
  ol({ children, ...props }: ComponentPropsWithoutRef<'ol'>) {
    return <ol className="my-2 pl-5 list-decimal" {...props}>{children}</ol>
  },
  li({ children, ...props }: ComponentPropsWithoutRef<'li'>) {
    return <li className="my-1" {...props}>{children}</li>
  },

  // ── Table ──
  table({ children, ...props }: ComponentPropsWithoutRef<'table'>) {
    return (
      <div className="overflow-x-auto my-3">
        <table className="w-full border-collapse text-[13px]" {...props}>{children}</table>
      </div>
    )
  },
  th({ children, ...props }: ComponentPropsWithoutRef<'th'>) {
    return (
      <th className="border border-[var(--border)] px-3 py-2 text-left font-semibold bg-[var(--surface-2)]" {...props}>
        {children}
      </th>
    )
  },
  td({ children, ...props }: ComponentPropsWithoutRef<'td'>) {
    return (
      <td className="border border-[var(--border)] px-3 py-2 text-left" {...props}>
        {children}
      </td>
    )
  },

  // ── Inline ──
  a({ children, href, ...props }: ComponentPropsWithoutRef<'a'>) {
    return (
      <a
        href={href}
        className="text-[var(--accent-text)] hover:underline"
        target="_blank"
        rel="noopener noreferrer"
        {...props}
      >
        {children}
      </a>
    )
  },

  strong({ children, ...props }: ComponentPropsWithoutRef<'strong'>) {
    return <strong className="font-semibold" {...props}>{children}</strong>
  },

  em({ children, ...props }: ComponentPropsWithoutRef<'em'>) {
    return <em className="text-[var(--text-secondary)]" {...props}>{children}</em>
  },

  hr(props: ComponentPropsWithoutRef<'hr'>) {
    return <hr className="border-none border-t border-[var(--border)] my-4" {...props} />
  },
}

function MarkdownRendererInner({ content, streaming = false }: MarkdownRendererProps) {
  // Sanitize the raw markdown (strip any embedded HTML attacks)
  const sanitized = useMemo(() => {
    return DOMPurify.sanitize(content, {
      ALLOWED_TAGS: [
        'p', 'br', 'strong', 'em', 'del', 'code', 'pre', 'blockquote',
        'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
        'ul', 'ol', 'li',
        'table', 'thead', 'tbody', 'tr', 'th', 'td',
        'a', 'img', 'hr',
        'details', 'summary',
        'sup', 'sub', 'mark',
      ],
      ALLOWED_ATTR: ['href', 'src', 'alt', 'title', 'open', 'class'],
      ALLOW_DATA_ATTR: false,
    })
  }, [content])

  return (
    <div className="markdown-body text-sm leading-[1.7] break-words">
      <ReactMarkdown
        remarkPlugins={remarkPlugins}
        rehypePlugins={streaming ? undefined : rehypePlugins}
        components={mdComponents}
      >
        {sanitized}
      </ReactMarkdown>
    </div>
  )
}

export const MarkdownRenderer = memo(MarkdownRendererInner)
