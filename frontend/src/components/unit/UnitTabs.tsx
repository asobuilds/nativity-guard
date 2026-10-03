import { cn } from '@/lib/cn'

export interface UnitTabDef {
  id: string
  label: string
  badge?: number
}

interface UnitTabsProps {
  tabs: UnitTabDef[]
  activeId: string
  onChange: (id: string) => void
}

/**
 * The member navigation bar. Scrollable on narrow screens — a unit admin
 * with all seven tabs open on a phone should still reach Compliance
 * without horizontal truncation.
 */
export function UnitTabs({ tabs, activeId, onChange }: UnitTabsProps) {
  return (
    <nav
      aria-label="Unit sections"
      className="mobile-nav-scroll -mx-4 overflow-x-auto border-b border-border px-4 sm:mx-0 sm:px-0"
      style={{ scrollbarWidth: 'none', msOverflowStyle: 'none' }}
    >
      <ul className="flex gap-1">
        {tabs.map((t) => {
          const active = t.id === activeId
          return (
            <li key={t.id}>
              <button
                type="button"
                onClick={() => onChange(t.id)}
                aria-current={active ? 'page' : undefined}
                className={cn(
                  'relative whitespace-nowrap px-3 py-2.5 text-sm font-medium transition-colors',
                  'border-b-2 -mb-px',
                  active
                    ? 'border-signal text-signal'
                    : 'border-transparent text-ink-muted hover:text-ink',
                )}
              >
                {t.label}
                {typeof t.badge === 'number' && t.badge > 0 ? (
                  <span className="ml-1.5 rounded-full bg-signal/15 px-1.5 py-0.5 text-[10px] tabular-nums text-signal">
                    {t.badge}
                  </span>
                ) : null}
              </button>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
