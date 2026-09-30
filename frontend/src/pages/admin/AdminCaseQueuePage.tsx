import { useMemo, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import {
  ChevronRight,
  Clock,
  FileCheck,
  Filter,
  MapPin,
  Search,
  ShieldQuestion,
  TriangleAlert,
  UserPlus,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { Button } from '@/components/ui/Button'
import { useAuth } from '@/auth/AuthContext'
import { Card } from '@/components/ui/Card'
import { PriorityChip, StatusChip } from '@/components/ui/Chips'
import { Input } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useToast } from '@/components/ui/Toast'
import { AssignOfficerDialog } from '@/components/admin/AssignOfficerDialog'
import { useCases } from '@/hooks/useCases'
import { useClaimCase } from '@/hooks/useCases'
import { useAssignOfficer, useUnitOfficers } from '@/hooks/useOfficers'
import { useUnits } from '@/hooks/useUnits'
import { ApiError } from '@/lib/apiClient'
import { inferAdminUnitId } from '@/lib/assignment'
import { CASE_STATUS_ORDER, statusMeta } from '@/lib/status'
import { relativeTime, truncate } from '@/lib/format'
import type { Case, CaseStatus } from '@/types/api'

type SortKey = 'priority' | 'waiting' | 'newest'

/** The things a unit administrator actually acts on. */
type Focus = 'none' | 'unassigned' | 'awaiting' | 'unverified' | 'to_decide'

const PRIORITY_RANK: Record<string, number> = { P1: 0, P2: 1, P3: 2 }

const STATUS_FILTERS: (CaseStatus | 'all' | 'open')[] = ['open', 'all', ...CASE_STATUS_ORDER]

const AWAITING_MS = 24 * 3_600_000

/**
 * The unit administrator's review queue.
 *
 * Triage-first: the questions an admin opens this screen with are "what does
 * nobody have?" and "what is waiting on *me*?" — so those are the counters, and
 * each one is a filter rather than a decoration.
 *
 * A case in `pending_admin_review` leads the counters because it is the only kind
 * of case no one else can move: the officer has submitted it and is blocked until
 * an administrator decides. Leaving one sitting unnoticed is the failure mode this
 * screen exists to prevent.
 *
 * Assignment is the only *transition* offered here; the approve / request-changes
 * decision lives on the case review screen, where the final report it is judging
 * is readable in full.
 */
export function AdminCaseQueuePage() {
  const { notify } = useToast()
  const casesQuery = useCases()
  const units = useUnits()

  const cases = useMemo(() => casesQuery.data ?? [], [casesQuery.data])

  /**
   * A unit admin's own unit. `GET /auth/profile` carries no `unitId`, so it is
   * inferred from the cases the API already scoped to this user; when the list
   * spans units (a super admin) no roster is assumed.
   */
  const unitId = useMemo(() => inferAdminUnitId(cases), [cases])
  const directory = useUnitOfficers(unitId)
  // `?? []` inline would mint a fresh array every render and churn the memo
  // below; deriving it once keeps the identity stable while the roster loads.
  const officers = useMemo(() => directory.data ?? [], [directory.data])

  const unitLabel = useMemo(() => {
    const map = new Map((units.data ?? []).map((u) => [u.id, u.name]))
    return (id: string | undefined) => (id ? map.get(id) ?? shortId(id) : 'Unit')
  }, [units.data])

  const officerNameById = useMemo(
    () => new Map(officers.map((o) => [o.id, `${o.rank} ${o.name}`])),
    [officers],
  )

  const [focus, setFocus] = useState<Focus>('none')
  const [statusFilter, setStatusFilter] = useState<CaseStatus | 'all' | 'open'>('open')
  const [query, setQuery] = useState('')
  const [sort, setSort] = useState<SortKey>('priority')
  const [showFilters, setShowFilters] = useState(false)
  const [assignTarget, setAssignTarget] = useState<Case | null>(null)

  const assign = useAssignOfficer(assignTarget?.id)

  // Evidence only appears on `GET /cases` if the backend preloads it. If no case
  // carries the array, hide the counter rather than report a confident zero.
  const evidenceAvailable = useMemo(
    () => cases.some((c) => Array.isArray(c.evidence)),
    [cases],
  )

  const attention = useMemo(() => {
    const unverified = evidenceAvailable
      ? cases.reduce(
          (total, c) => total + (c.evidence ?? []).filter((e) => !e.isVerified).length,
          0,
        )
      : 0
    return {
      toDecide: cases.filter((c) => c.status === 'pending_admin_review').length,
      unassigned: cases.filter((c) => c.status === 'pending').length,
      awaiting: cases.filter((c) => c.status === 'assigned').length,
      stale: cases.filter(
        (c) =>
          c.status === 'assigned' &&
          Date.now() - new Date(c.assignedAt ?? c.createdAt).getTime() > AWAITING_MS,
      ).length,
      unverified,
    }
  }, [cases, evidenceAvailable])

  const visible = useMemo(() => {
    let list = cases

    if (focus === 'to_decide') list = list.filter((c) => c.status === 'pending_admin_review')
    else if (focus === 'unassigned') list = list.filter((c) => c.status === 'pending')
    else if (focus === 'awaiting') list = list.filter((c) => c.status === 'assigned')
    else if (focus === 'unverified')
      list = list.filter((c) => (c.evidence ?? []).some((e) => !e.isVerified))
    else if (statusFilter === 'open') list = list.filter((c) => c.status !== 'closed')
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
      if (sort === 'waiting') {
        return new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
      }
      const rank = (PRIORITY_RANK[a.priorityLevel] ?? 3) - (PRIORITY_RANK[b.priorityLevel] ?? 3)
      if (rank !== 0) return rank
      return new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()
    })
    return sorted
  }, [cases, focus, statusFilter, query, sort])

  function toggleFocus(next: Exclude<Focus, 'none'>) {
    setFocus((current) => (current === next ? 'none' : next))
    setStatusFilter('open')
  }

  return (
    <div className="mx-auto w-full max-w-5xl p-4 sm:p-6">
      <header className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-ink">Case review</h1>
          <p className="mt-1 text-sm text-ink-muted">
            {unitId ? `${unitLabel(unitId)} · ` : ''}
            {cases.length} case{cases.length === 1 ? '' : 's'} in view ·{' '}
            {attention.unassigned} awaiting assignment
            {casesQuery.isFetching ? ' · refreshing…' : ''}
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

      {/* What needs a decision today */}
      <div
        className={cn(
          'mb-4 grid gap-3 sm:grid-cols-2',
          evidenceAvailable ? 'lg:grid-cols-4' : 'lg:grid-cols-3',
        )}
      >
        <AttentionCard
          label="Awaiting your decision"
          value={attention.toDecide}
          hint={
            attention.toDecide > 0
              ? 'Submitted for closure — only an admin can move these'
              : 'Nothing is blocked on you'
          }
          warn={attention.toDecide > 0}
          icon={<FileCheck className="size-4" aria-hidden />}
          active={focus === 'to_decide'}
          onClick={() => toggleFocus('to_decide')}
        />
        <AttentionCard
          label="Awaiting assignment"
          value={attention.unassigned}
          hint="Nobody has picked these up yet"
          warn={attention.unassigned > 0}
          icon={<UserPlus className="size-4" aria-hidden />}
          active={focus === 'unassigned'}
          onClick={() => toggleFocus('unassigned')}
        />
        <AttentionCard
          label="Assigned, not dispatched"
          value={attention.awaiting}
          hint={attention.stale > 0 ? `${attention.stale} waiting over 24 hours` : 'All under 24 hours'}
          warn={attention.stale > 0}
          icon={<Clock className="size-4" aria-hidden />}
          active={focus === 'awaiting'}
          onClick={() => toggleFocus('awaiting')}
        />
        {evidenceAvailable ? (
          <AttentionCard
            label="Evidence unverified"
            value={attention.unverified}
            hint="Items attached but not yet checked"
            warn={attention.unverified > 0}
            icon={<ShieldQuestion className="size-4" aria-hidden />}
            active={focus === 'unverified'}
            onClick={() => toggleFocus('unverified')}
          />
        ) : null}
      </div>

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
                    aria-pressed={focus === 'none' && statusFilter === value}
                    onClick={() => {
                      setFocus('none')
                      setStatusFilter(value)
                    }}
                    className={cn(
                      'rounded-full px-2.5 py-1 text-xs transition-colors',
                      focus === 'none' && statusFilter === value
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
                    ['waiting', 'Longest waiting'],
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

      {casesQuery.isLoading ? (
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
      ) : casesQuery.isError ? (
        <Card>
          <ErrorState
            title="Could not load cases"
            description="The case service did not respond. Your device may be offline."
            offline={ApiError.isNetwork(casesQuery.error)}
            onRetry={() => void casesQuery.refetch()}
          />
        </Card>
      ) : visible.length === 0 ? (
        <Card>
          <EmptyState
            title={cases.length === 0 ? 'No cases in your unit' : 'No cases match these filters'}
            description={
              cases.length === 0
                ? 'Cases reported to your unit will appear here for triage and assignment.'
                : 'Clear the search, or turn off the current focus, to see more.'
            }
          />
        </Card>
      ) : (
        <ul className="flex flex-col gap-3">
          {visible.map((caseItem) => (
            <li key={caseItem.id}>
              <ReviewRow
                caseItem={caseItem}
                officerName={
                  caseItem.assignedTo ? officerNameById.get(caseItem.assignedTo) : undefined
                }
                unitName={unitLabel(caseItem.unitId)}
                onAssign={() => setAssignTarget(caseItem)}
              />
            </li>
          ))}
        </ul>
      )}

      {assignTarget ? (
        <AssignOfficerDialog
          key={assignTarget.id}
          caseItem={assignTarget}
          assignments={
            // The queue endpoint does not return assignment rows; the primary
            // officer on the case is still enough to pre-select sensibly.
            assignTarget.assignedTo
              ? [
                  {
                    id: 'current',
                    caseId: assignTarget.id,
                    officerId: assignTarget.assignedTo,
                    role: 'primary',
                  },
                ]
              : []
          }
          submitting={assign.isPending}
          onClose={() => setAssignTarget(null)}
          onSubmit={({ officerId, role }) =>
            assign.mutate(
              { officerId, role },
              {
                onSuccess: () => {
                  setAssignTarget(null)
                  notify('Officer assigned.', 'success')
                },
                onError: (cause) =>
                  notify(
                    ApiError.isNetwork(cause)
                      ? 'No connection — the assignment was not saved.'
                      : cause instanceof ApiError
                        ? cause.message
                        : 'Could not assign that officer.',
                    'error',
                  ),
              },
            )
          }
        />
      ) : null}
    </div>
  )
}

function AttentionCard({
  label,
  value,
  hint,
  warn,
  icon,
  active,
  onClick,
}: {
  label: string
  value: number
  hint: string
  warn: boolean
  icon: ReactNode
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={cn(
        'flex flex-col gap-1 rounded-panel border p-3 text-left transition-colors',
        active ? 'border-signal/50 bg-signal/10' : 'border-border bg-surface hover:border-border-hi',
      )}
    >
      <span className="flex items-center gap-1.5 text-[11px] uppercase tracking-wide text-ink-faint">
        <span aria-hidden>{icon}</span>
        {label}
      </span>
      <span className={cn('text-2xl font-semibold tabular-nums', warn ? 'text-warn' : 'text-ink')}>
        {value}
      </span>
      <span className="text-[11px] text-ink-muted">{hint}</span>
    </button>
  )
}

function ReviewRow({
  caseItem,
  officerName,
  unitName,
  onAssign,
}: {
  caseItem: Case
  officerName?: string
  unitName: string
  onAssign: () => void
}) {
  const meta = statusMeta(caseItem.status)
  const unassigned = !caseItem.assignedTo && caseItem.status !== 'closed'
  const stale =
    caseItem.status === 'assigned' &&
    Date.now() - new Date(caseItem.assignedAt ?? caseItem.createdAt).getTime() > AWAITING_MS

  return (
    <div
      className={cn(
        'flex items-start gap-3 rounded-panel border bg-surface p-4 shadow-panel transition-colors',
        unassigned ? 'border-warn/40' : 'border-border hover:border-border-hi',
      )}
    >
      <span className={cn('mt-1.5 size-2 shrink-0 rounded-full', meta.dot)} aria-hidden />

      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <PriorityChip level={caseItem.priorityLevel} />
          <StatusChip status={caseItem.status} />
          {unassigned ? (
            <span className="flex items-center gap-1 rounded-full bg-warn/10 px-2 py-0.5 text-[11px] text-warn ring-1 ring-warn/30">
              <TriangleAlert className="size-3" aria-hidden />
              No officer assigned
            </span>
          ) : null}
          {stale ? (
            <span className="rounded-full bg-warn/10 px-2 py-0.5 text-[11px] text-warn ring-1 ring-warn/30">
              Assigned &gt; 24h, not dispatched
            </span>
          ) : null}
        </div>

        <Link
          to={`/admin/cases/${caseItem.id}`}
          className="mt-1.5 block truncate text-sm font-medium text-ink hover:text-signal focus:outline-none focus-visible:ring-2 focus-visible:ring-signal"
        >
          {caseItem.title}
        </Link>
        <p className="mt-0.5 text-xs text-ink-muted">{truncate(caseItem.description, 140)}</p>

        <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-ink-faint">
          <span className="flex items-center gap-1">
            <MapPin className="size-3" aria-hidden />
            {caseItem.location || 'Location recorded'}
          </span>
          <span className="tabular-nums">{caseItem.trackingId}</span>
          <span>{unitName}</span>
          <span>{officerName ?? (unassigned ? 'unassigned' : 'assigned')}</span>
          <span>reported {relativeTime(caseItem.createdAt)}</span>
        </div>
      </div>

      <div className="flex shrink-0 flex-col items-end gap-2">
        <ClaimButton caseItem={caseItem} />
        <Button
          size="sm"
          variant={unassigned ? 'primary' : 'secondary'}
          icon={<UserPlus className="size-4" aria-hidden />}
          onClick={onAssign}
        >
          {unassigned ? 'Assign' : 'Reassign'}
        </Button>
        <Link
          to={`/admin/cases/${caseItem.id}`}
          className="flex items-center gap-1 text-[11px] text-ink-muted transition-colors hover:text-ink"
        >
          Review
          <ChevronRight className="size-3" aria-hidden />
        </Link>
      </div>
    </div>
  )
}

function shortId(id: string | undefined | null): string {
  if (!id) return '—'
  return id.length > 10 ? `${id.slice(0, 8)}…` : id
}


/**
 * Self-contained "Take this case" button.
 *
 * Renders only when the case is:
 *   - owned by a different unit than the caller's,
 *   - not yet assigned to an officer (no work in progress), and
 *   - not closed.
 *
 * Placing the mutation here keeps the parent row clean — no props to
 * thread through, no hooks added to the existing component.
 */
function ClaimButton({ caseItem }: { caseItem: Case }) {
  const { user } = useAuth()
  const claimCase = useClaimCase()
  const canClaim =
    caseItem.unitId !== user?.unitId &&
    !caseItem.assignedTo &&
    caseItem.status !== 'closed'
  if (!canClaim) return null
  return (
    <Button
      size="sm"
      variant="ghost"
      loading={claimCase.isPending}
      onClick={() => claimCase.mutate(caseItem.id)}
    >
      Take this case
    </Button>
  )
}