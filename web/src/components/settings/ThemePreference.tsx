// ============================================================================
// Operator OS — ThemePreference
// Dark/light/system theme selector with live preview cards.
// ============================================================================

import { Sun, Moon, Desktop, CheckCircle } from '@phosphor-icons/react'
import { useUIStore } from '../../stores/uiStore'

type ThemeOption = 'light' | 'dark' | 'system'

const options: { value: ThemeOption; label: string; icon: typeof Sun; desc: string }[] = [
  { value: 'light', label: 'Light', icon: Sun, desc: 'Bright, clean interface' },
  { value: 'dark', label: 'Dark', icon: Moon, desc: 'Easy on the eyes' },
  { value: 'system', label: 'System', icon: Desktop, desc: 'Match your OS preference' },
]

export function ThemePreference() {
  const theme = useUIStore((s) => s.theme)
  const setTheme = useUIStore((s) => s.setTheme)
  const followSystem = useUIStore((s) => s.followSystem)

  const currentSelection: ThemeOption =
    useUIStore.getState().followingSystem ? 'system' : theme

  const handleSelect = (value: ThemeOption) => {
    if (value === 'system') {
      followSystem()
    } else {
      setTheme(value)
    }
  }

  return (
    <div>
      <div className="flex items-center gap-3 mb-5">
        <div className="w-9 h-9 rounded-xl bg-accent-subtle flex items-center justify-center">
          {theme === 'dark' ? (
            <Moon size={18} weight="duotone" className="text-accent-text" />
          ) : (
            <Sun size={18} weight="duotone" className="text-accent-text" />
          )}
        </div>
        <div>
          <h3 className="text-[15px] font-semibold text-[var(--text)]">Appearance</h3>
          <p className="text-xs text-[var(--text-dim)]">Choose your preferred theme</p>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        {options.map(({ value, label, icon: Icon, desc }) => {
          const selected = currentSelection === value
          return (
            <button
              key={value}
              onClick={() => handleSelect(value)}
              className={`
                relative flex flex-col items-center gap-2.5 p-5 rounded-[var(--radius)]
                border transition-all duration-200 cursor-pointer text-center
                ${selected
                  ? 'border-accent bg-accent-subtle shadow-[0_0_0_1px_var(--accent)]'
                  : 'border-[var(--border)] bg-[var(--surface-2)] hover:border-[var(--border-hover)] hover:bg-[var(--surface-3)]'
                }
              `}
            >
              {selected && (
                <CheckCircle
                  size={18}
                  weight="fill"
                  className="absolute top-2.5 right-2.5 text-accent"
                />
              )}
              <div
                className={`
                  w-10 h-10 rounded-xl flex items-center justify-center
                  ${selected ? 'bg-accent text-white' : 'bg-[var(--surface-3)] text-[var(--text-dim)]'}
                `}
              >
                <Icon size={22} weight={selected ? 'fill' : 'duotone'} />
              </div>
              <div>
                <div className={`text-sm font-semibold ${selected ? 'text-accent-text' : 'text-[var(--text)]'}`}>
                  {label}
                </div>
                <div className="text-[11px] text-[var(--text-dim)] mt-0.5">{desc}</div>
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}
