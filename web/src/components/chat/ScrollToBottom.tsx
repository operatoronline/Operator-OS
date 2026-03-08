// ============================================================================
// Operator OS — ScrollToBottom
// Floating button that appears when the user scrolls up from the bottom.
// Click to smoothly jump back to latest messages. Uses the glass treatment.
// ============================================================================

import { ArrowDown } from '@phosphor-icons/react'

interface ScrollToBottomProps {
  visible: boolean
  onClick: () => void
  /** Number of unread messages while scrolled up */
  unreadCount?: number
}

export function ScrollToBottom({ visible, onClick, unreadCount = 0 }: ScrollToBottomProps) {
  if (!visible) return null

  return (
    <button
      onClick={onClick}
      aria-label="Scroll to latest messages"
      className="absolute bottom-4 right-4 z-10
        w-9 h-9 rounded-full glass
        flex items-center justify-center
        text-[var(--text-secondary)] hover:text-[var(--text)]
        transition-all duration-200 ease-out
        hover:scale-105 active:scale-95
        animate-fade-slide cursor-pointer"
    >
      <ArrowDown size={16} weight="bold" />
      {unreadCount > 0 && (
        <span
          className="absolute -top-1.5 -right-1.5 min-w-[18px] h-[18px]
            rounded-full bg-[var(--accent)] text-white text-[10px] font-semibold
            flex items-center justify-center px-1 animate-scale-in"
        >
          {unreadCount > 99 ? '99+' : unreadCount}
        </span>
      )}
    </button>
  )
}
