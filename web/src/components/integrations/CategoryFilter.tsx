// ============================================================================
// Operator OS — Category Filter
// Horizontal pill bar for filtering integrations by category.
// ============================================================================

import { memo } from 'react'

interface CategoryFilterProps {
  categories: string[]
  selected: string | null
  onSelect: (category: string | null) => void
  integrationCounts: Record<string, number>
}

export const CategoryFilter = memo(function CategoryFilter({
  categories,
  selected,
  onSelect,
  integrationCounts,
}: CategoryFilterProps) {
  if (categories.length === 0) return null

  const totalCount = Object.values(integrationCounts).reduce((a, b) => a + b, 0)

  return (
    <div className="flex items-center gap-2 overflow-x-auto pb-1 -mb-1 scrollbar-none">
      {/* All pill */}
      <button
        onClick={() => onSelect(null)}
        className={`
          shrink-0 px-3 py-1.5 rounded-full text-xs font-medium
          transition-all duration-150 cursor-pointer
          ${!selected
            ? 'bg-[var(--accent)] text-white shadow-[0_2px_8px_var(--glass-shadow)]'
            : 'bg-[var(--surface-2)] text-[var(--text-secondary)] hover:text-[var(--text)] hover:bg-[var(--surface-3)]'
          }
        `}
      >
        All
        <span className="ml-1.5 opacity-70">{totalCount}</span>
      </button>

      {categories.map((category) => {
        const count = integrationCounts[category] ?? 0
        const isActive = selected === category

        return (
          <button
            key={category}
            onClick={() => onSelect(isActive ? null : category)}
            className={`
              shrink-0 px-3 py-1.5 rounded-full text-xs font-medium
              transition-all duration-150 cursor-pointer capitalize
              ${isActive
                ? 'bg-[var(--accent)] text-white shadow-[0_2px_8px_var(--glass-shadow)]'
                : 'bg-[var(--surface-2)] text-[var(--text-secondary)] hover:text-[var(--text)] hover:bg-[var(--surface-3)]'
              }
            `}
          >
            {category}
            <span className="ml-1.5 opacity-70">{count}</span>
          </button>
        )
      })}
    </div>
  )
})
