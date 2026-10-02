import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { ChevronRight, Filter, MapPin, Search } from 'lucide-react'
import { cn } from '@/lib/cn'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { PriorityChip, StatusChip } from '@/components/ui/Chips'
import { Input } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { CaseTracker } from '@/components/cases/CaseTracker'
import { useCases } from '@/hooks/useCases'
import { CASE_STATUS_ORDER, isAwaitingDispatch, statusMeta } from '@/lib/status'
import { formatDate, relativeTime, truncate } from '@/lib/format'
import type { Case, CaseStatus } from '@/types/api'

type SortKey = 'priority' | 'newest' | 'oldest'

type StatusFilter = CaseStatus | 'all' | 'open'

const STATUS_FILTERS: StatusFilter[] = ['open', 'all', ...CASE_STATUS_ORDER]

const PRIORITY_RANK: Record<string, number> = { P1: 0, P2: 1, P3: 2 }

/**
 * The officer's queue.
 *
 * Ordered by urgency first (P1, then longest-waiting), because the thing an
 * officer needs at 3am is the next case to act on — not an alphabetical list.
 */
export function OfficerQueuePage() {
  const { data, isLoading, isError, refetch, isFetching } = useCases()

  const [query, setQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<CaseStatus | 'all' | 'open'>('open')
  const [sort, setSort] = useState<SortKey>('priority')
  const [showFilters, setShowFilters] = useState(false)

  const cases = useMemo(() => data ?? [], [data])

  const counts = useMemo(() => {
    return {
      open: cases.filter((c) => c.status !== 'closed').length,
      awaiting: cases.filter((c) => isAwaitingDispatch(c.status)).length,
      onScene: cases.filter((c) => c.status === 'on_scene').length,
      closed: cases.filter((c) => c.status === 'closed').length,
    }
  }, [cases])

  const visible = useMemo(() => {
    let list = cases

    if (statusFilter === 'open') list = list.filter((c) => c.status !== 'closed')
    else if (statusFilter !== 'all') list = list.filter((c) => c.status === statusFilter)

    const q = query.trim().toLowerCase()
    if (q) {
      list = list.filter((c) =>
        [c.title, c.description, c.trackingId, c.location]
          .filter(Boolean)
          .some((field) => field!.toLowerCase().includes(q)),
      )
    }

    const sorted = [...list]
    sorted.sort((a, b) => {
      if (sort === 'newest') {
        return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      }
      if (sort === 'oldest') {
        return new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
      }
      const rank = (PRIORITY_RANK[a.priorityLevel] ?? 3) - (PRIORITY_RANK[b.priorityLevel] ?? 3)
      if (rank !== 0) return rank
      return new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
    })
    return sorted
  }, [cases, statusFilter, query, sort])

  return (
    <div className="mx-auto w-full max-w-4xl p-4 sm:p-6">
      <header className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-ink">Case queue</h1>
          <p className="mt-1 text-sm text-ink-muted">
            {counts.open} open · {counts.awaiting} awaiting dispatch · {counts.onScene} on scene
            {isFetching ? ' · refreshing…' : ''}
          </p>
        </div>
        <Button
          size="sm"
          variant="secondary"
          icon={<Filter className="size-4" aria-hidden />}
          aria-expanded={showFilters}
          onClick={() => setShowFilters((v) => !v)}
        >
          Filters
        </Button>
      </header>

      <div className="mb-4 flex flex-col gap-3">
        <div className="relative">
          <Search
            className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint"
            aria-hidden
          />
          <Input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search title, tracking ID or location"
            aria-label="Search cases"
            className="pl-9"
          />
        </div>

        {showFilters ? (
          <div className="flex flex-col gap-3 rounded-panel border border-border bg-surface p-3">
            <div>
              <p className="mb-1.5 text-[11px] uppercase tracking-wide text-ink-faint">Status</p>
              <div className="flex flex-wrap gap-1.5">
                {STATUS_FILTERS.map((value) => (
                  <button
                    key={value}
                    type="button"
                    aria-pressed={statusFilter === value}
                    onClick={() => setStatusFilter(value)}
                    className={cn(
                      'rounded-full px-2.5 py-1 text-xs transition-colors',
                      statusFilter === value
                        ? 'bg-signal/15 text-signal ring-1 ring-signal/30'
                        : 'bg-surface-hi text-ink-muted hover:text-ink',
                    )}
                  >
                    {value === 'open' ? 'Open only' : value === 'all' ? 'All' : statusMeta(value).label}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <p className="mb-1.5 text-[11px] uppercase tracking-wide text-ink-faint">Sort</p>
              <div className="flex flex-wrap gap-1.5">
                {(
                  [
                    ['priority', 'Most urgent'],
                    ['oldest', 'Longest waiting'],
                    ['newest', 'Newest'],
                  ] as const
                ).map(([value, label]) => (
                  <button
                    key={value}
                    type="button"
                    aria-pressed={sort === value}
                    onClick={() => setSort(value)}
                    className={cn(
                      'rounded-full px-2.5 py-1 text-xs transition-colors',
                      sort === value
                        ? 'bg-signal/15 text-signal ring-1 ring-signal/30'
                        : 'bg-surface-hi text-ink-muted hover:text-ink',
                    )}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>
          </div>
        ) : null}
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-3">
          {[0, 1, 2].map((i) => (
            <Card key={i}>
              <div className="space-y-3 p-4">
                <Skeleton className="h-4 w-1/3" />
                <Skeleton className="h-3 w-2/3" />
                <Skeleton className="h-3 w-1/4" />
              </div>
            </Card>
          ))}
        </div>
      ) : isError ? (
        <Card>
          <ErrorState
            title="Could not load the queue"
            description="The case service did not respond. Your device may be offline."
            onRetry={() => void refetch()}
          />
        </Card>
      ) : visible.length === 0 ? (
        <Card>
          <EmptyState
            title={cases.length === 0 ? 'No cases assigned to you' : 'No cases match these filters'}
            description={
              cases.length === 0
                ? 'Cases routed to your unit will appear here. Dispatch is handled by your unit administrator.'
                : 'Try clearing the search or choosing a different status.'
            }
          />
        </Card>
      ) : (
        <ul className="flex flex-col gap-3">
          {visible.map((caseItem) => (
            <li key={caseItem.id}>
              <CaseRow caseItem={caseItem} />
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function CaseRow({ caseItem }: { caseItem: Case }) {
  const meta = statusMeta(caseItem.status)
  const overdue =
    isAwaitingDispatch(caseItem.status) &&
    Date.now() - new Date(caseItem.createdAt).getTime() > 24 * 3600_000

  return (
    <Link
      to={`/officer/cases/${caseItem.id}`}
      className="glass-panel--lift block rounded-panel focus:outline-none focus-visible:ring-2 focus-visible:ring-signal"
    >
      <div className="flex items-start gap-3 p-4">
        <span className={cn('mt-1.5 size-2 shrink-0 rounded-full', meta.dot)} aria-hidden />

        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <PriorityChip level={caseItem.priorityLevel} />
            <StatusChip status={caseItem.status} />
            {overdue ? (
              <span className="rounded-full bg-warn/10 px-2 py-0.5 text-[11px] text-warn ring-1 ring-warn/30">
                Awaiting dispatch &gt; 24h
              </span>
            ) : null}
          </div>

          <p className="mt-1.5 truncate text-sm font-medium text-ink">{caseItem.title}</p>
          <p className="mt-0.5 text-xs text-ink-muted">{truncate(caseItem.description, 140)}</p>

          <div className="mt-2">
            <CaseTracker caseItem={caseItem} compact />
          </div>

          <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-ink-faint">
            <span className="flex items-center gap-1">
              <MapPin className="size-3" aria-hidden />
              {caseItem.location || 'Location recorded'}
            </span>
            <span className="tabular-nums">{caseItem.trackingId}</span>
            <span title={formatDate(caseItem.createdAt)}>reported {relativeTime(caseItem.createdAt)}</span>
            {caseItem.assignedTo ? <span>assigned</span> : <span>unassigned</span>}
          </div>
        </div>

        <ChevronRight className="mt-1 size-4 shrink-0 text-ink-faint" aria-hidden />
      </div>
    </Link>
  )
}