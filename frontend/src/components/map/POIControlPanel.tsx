import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/cn'
import { POI_META, type POICategory } from '@/hooks/useMapPOIs'

interface POIControlPanelProps {
  active: Set<POICategory>
  onToggle: (category: POICategory) => void
  onClear: () => void
  count: number
  open: boolean
  onToggleOpen: () => void
}

const ORDER: POICategory[] = [
  'hospital',
  'police',
  'fire',
  'pharmacy',
  'bank',
  'school',
  'church',
  'mosque',
  'landmark',
]

/**
 * Compact panel that lets a user pick which POI categories to display.
 * Lives in the bottom-left of the map container, above the legend.
 */
export function POIControlPanel({
  active,
  onToggle,
  onClear,
  count,
  open,
  onToggleOpen,
}: POIControlPanelProps) {
  return (
    <div
      className={cn(
        'pointer-events-auto w-64 max-w-[80vw] overflow-hidden rounded-lg border border-border bg-base/95 shadow-lg backdrop-blur',
      )}
    >
      <button
        type="button"
        onClick={onToggleOpen}
        aria-expanded={open}
        className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left transition-colors hover:bg-surface-hi/60"
      >
        <span className="flex items-center gap-2 text-xs font-semibold text-ink">
          Nearby places
          <span className="rounded-full bg-signal/15 px-1.5 py-0.5 text-[10px] font-medium text-signal tabular-nums">
            {count}
          </span>
        </span>
        <ChevronDown
          className={cn('size-4 text-ink-muted transition-transform', open && 'rotate-180')}
          aria-hidden
        />
      </button>

      {open ? (
        <div className="border-t border-border p-2">
          <div className="grid grid-cols-1 gap-1">
            {ORDER.map((cat) => {
              const meta = POI_META[cat]
              const isOn = active.has(cat)
              return (
                <button
                  key={cat}
                  type="button"
                  onClick={() => onToggle(cat)}
                  aria-pressed={isOn}
                  className={cn(
                    'flex items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs transition-colors',
                    isOn
                      ? 'bg-surface-hi text-ink'
                      : 'text-ink-muted hover:bg-surface-hi/60',
                  )}
                >
                  <span
                    className="size-2.5 shrink-0 rounded-full"
                    style={{ backgroundColor: meta.color }}
                    aria-hidden
                  />
                  <span className="flex-1">{meta.label}</span>
                  {isOn ? (
                    <span className="size-1.5 rounded-full bg-signal" aria-hidden />
                  ) : null}
                </button>
              )
            })}
          </div>
          {active.size > 0 ? (
            <button
              type="button"
              onClick={onClear}
              className="mt-2 w-full rounded-md border border-border px-2 py-1 text-[11px] text-ink-muted transition-colors hover:bg-surface-hi"
            >
              Hide all
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}