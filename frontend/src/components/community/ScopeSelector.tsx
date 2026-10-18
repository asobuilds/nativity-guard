import { cn } from '@/lib/cn'

/**
 * The community geographic scope ladder, most specific first (decision D-011).
 * A feed request widens how much of the caller's *own* area is returned; it can
 * never reach another place. The server resolves the caller's place — these are
 * audience choices, not location inputs.
 */
export const COMMUNITY_SCOPES = ['locality', 'lga', 'state', 'country', 'world'] as const

export type CommunityScope = (typeof COMMUNITY_SCOPES)[number]

const SCOPE_LABELS: Record<CommunityScope, string> = {
  locality: 'My village',
  lga: 'My LGA',
  state: 'My state',
  country: 'Nigeria',
  world: 'World',
}

export function scopeLabel(scope: CommunityScope): string {
  return SCOPE_LABELS[scope] ?? SCOPE_LABELS.world
}

interface ScopeSelectorProps {
  value: CommunityScope
  onChange: (scope: CommunityScope) => void
  label: string
  className?: string
}

/** A single-select pill group for choosing a feed or post audience. */
export function ScopeSelector({ value, onChange, label, className }: ScopeSelectorProps) {
  return (
    <div
      role="group"
      aria-label={label}
      className={cn('flex flex-wrap gap-1.5', className)}
    >
      {COMMUNITY_SCOPES.map((scope) => (
        <button
          key={scope}
          type="button"
          aria-pressed={value === scope}
          onClick={() => onChange(scope)}
          className={cn(
            'rounded-full px-3 py-1 text-xs font-medium ring-1 transition-colors',
            value === scope
              ? 'bg-signal/15 text-signal ring-signal/30'
              : 'bg-surface-hi text-ink-muted ring-border hover:text-ink',
          )}
        >
          {scopeLabel(scope)}
        </button>
      ))}
    </div>
  )
}
