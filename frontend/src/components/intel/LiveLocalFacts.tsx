import { ArrowBigUp, Radio } from 'lucide-react'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Skeleton } from '@/components/ui/States'
import { useLocalIntel, useUpvoteLocalIntel, type LocalIntelFact } from '@/hooks/useLocalIntel'
import { relativeTime } from '@/lib/format'
import { cn } from '@/lib/cn'

const CATEGORY_LABEL: Record<string, string> = {
  general: 'General',
  security: 'Security',
  road_block: 'Road block',
  traffic: 'Traffic',
  weather: 'Weather',
  power: 'Power',
  community: 'Community',
}

const CONFIDENCE_STYLE: Record<string, string> = {
  high: 'bg-signal/15 text-signal',
  medium: 'bg-warn/15 text-warn',
  low: 'bg-surface-hi text-ink-muted',
}

interface LocalFactsListProps {
  items: LocalIntelFact[]
  onUpvote: (id: string) => void
  /** Id currently being upvoted, if any. */
  pendingId?: string | null
}

/**
 * Presentational body of the widget, exported separately so rendering can be
 * tested without mounting react-query.
 */
export function LocalFactsList({ items, onUpvote, pendingId }: LocalFactsListProps) {
  if (items.length === 0) return null

  return (
    <ul className="flex flex-col divide-y divide-border">
      {items.map((item) => {
        const place = [item.lga, item.state].filter(Boolean).join(', ')
        const confidence = CONFIDENCE_STYLE[item.confidence] ?? CONFIDENCE_STYLE.medium
        return (
          <li key={item.id} className="flex items-start gap-3 py-3 first:pt-0 last:pb-0">
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-1.5">
                <span className="rounded-full bg-surface-hi px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-ink-muted">
                  {CATEGORY_LABEL[item.category] ?? item.category}
                </span>
                <span
                  className={cn(
                    'rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide',
                    confidence,
                  )}
                >
                  {item.confidence || 'medium'} confidence
                </span>
              </div>
              <p className="mt-1.5 text-sm leading-relaxed text-ink">{item.fact}</p>
              <p className="mt-1 text-[11px] text-ink-faint">
                {place ? `${place} · ` : ''}
                {item.expiresAt ? `expires ${relativeTime(item.expiresAt)}` : 'recently added'}
              </p>
            </div>
            <button
              type="button"
              aria-pressed={item.hasUpvoted}
              aria-label={
                item.hasUpvoted
                  ? `Already upvoted: ${item.fact}`
                  : `Upvote this fact: ${item.fact}`
              }
              disabled={item.hasUpvoted || pendingId === item.id}
              onClick={() => onUpvote(item.id)}
              className={cn(
                'flex shrink-0 flex-col items-center gap-0.5 rounded-lg border px-2.5 py-1.5 text-xs transition-colors',
                item.hasUpvoted
                  ? 'cursor-default border-signal/40 bg-signal/10 text-signal'
                  : 'border-border text-ink-muted hover:border-signal/40 hover:text-signal',
                pendingId === item.id && 'opacity-60',
              )}
            >
              <ArrowBigUp
                className={cn('size-4', item.hasUpvoted && 'fill-current')}
                aria-hidden
              />
              <span className="tabular-nums">{item.upvotes}</span>
            </button>
          </li>
        )
      })}
    </ul>
  )
}

/**
 * "Live Local Intelligence" — unexpired facts extracted from nearby reports,
 * SOS alerts and community posts. Renders nothing when there is nothing
 * fresh to show, rather than an empty card.
 */
export function LiveLocalFacts({ limit = 5 }: { limit?: number }) {
  const query = useLocalIntel(limit)
  const upvote = useUpvoteLocalIntel()

  if (query.isError) return null

  if (query.isLoading) {
    return (
      <Card>
        <CardHeader title="Live local intelligence" />
        <CardBody className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </CardBody>
      </Card>
    )
  }

  const items = query.data ?? []
  if (items.length === 0) return null

  return (
    <Card as="section">
      <CardHeader
        title="Live local intelligence"
        subtitle="Fresh facts from reports and alerts around you"
        actions={<Radio className="size-4 text-signal" aria-hidden />}
      />
      <CardBody>
        <LocalFactsList
          items={items}
          onUpvote={(id) => upvote.mutate(id)}
          pendingId={upvote.isPending ? upvote.variables : null}
        />
      </CardBody>
    </Card>
  )
}
