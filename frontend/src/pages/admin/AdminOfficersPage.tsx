import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { Search, ShieldCheck, UserRound, Users } from 'lucide-react'
import { useAuth } from '@/auth/AuthContext'
import { Card } from '@/components/ui/Card'
import { Input } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useCases } from '@/hooks/useCases'
import { useUnitOfficers } from '@/hooks/useOfficers'
import { useUnits } from '@/hooks/useUnits'
import { ApiError, api } from '@/lib/apiClient'
import { matchesOfficerQuery, sortOfficers } from '@/lib/assignment'
import { USE_MOCKS } from '@/mocks/config'

type Membership = { unitId: string; status: string; role: string; isHeadAdmin: boolean }

/** The unit roster comes from Go. The demo roster remains on the other demo routes. */
export function AdminOfficersPage() {
  const { role } = useAuth()
  const { unitId: requestedUnitId } = useParams<{ unitId: string }>()
  const units = useUnits()
  const memberships = useQuery({
    queryKey: ['my-unit-memberships'],
    queryFn: () => api.get<{ memberships: Membership[] }>('/units/my-memberships'),
    enabled: role !== 'super_admin' && !USE_MOCKS,
  })
  const cases = useCases()
  const [query, setQuery] = useState('')

  const allowedIds = useMemo(() => new Set(
    USE_MOCKS
      ? (cases.data ?? []).map((item) => item.unitId).filter((id): id is string => Boolean(id))
      : (memberships.data?.memberships ?? [])
          .filter((item) => item.status === 'active' && (item.role === 'unit_admin' || item.isHeadAdmin))
          .map((item) => item.unitId),
  ), [cases.data, memberships.data])
  const availableUnits = useMemo(() =>
    (units.data ?? []).filter((unit) => role === 'super_admin' || allowedIds.has(unit.id)),
  [units.data, role, allowedIds])
  const unitId = availableUnits.some((unit) => unit.id === requestedUnitId)
    ? requestedUnitId
    : undefined
  const roster = useUnitOfficers(unitId)
  const officers = useMemo(
    () => sortOfficers(roster.data ?? []).filter((officer) => matchesOfficerQuery(officer, query)),
    [roster.data, query],
  )
  const unit = availableUnits.find((item) => item.id === unitId)

  return (
    <main className="mx-auto w-full max-w-6xl space-y-6 p-4 sm:p-6 lg:p-8">
      <header className="space-y-2">
        <p className="text-xs font-semibold uppercase tracking-widest text-signal">Unit operations</p>
        <h1 className="text-2xl font-bold tracking-tight text-ink sm:text-3xl">{requestedUnitId ? `${unit?.name ?? 'Unit'} officers` : 'Your units'}</h1>
        <p className="max-w-2xl text-sm text-ink-muted">
          {requestedUnitId ? 'See this unit’s officers and open assignments. Assign work from the case board.' : 'Open a unit to see its officers and assign work.'}
        </p>
      </header>

      {(units.isLoading || memberships.isLoading || (USE_MOCKS && cases.isLoading)) ? (
        <Skeleton className="h-28 rounded-panel" />
      ) : units.isError || memberships.isError ? (
        <ErrorState title="Could not load your units" description="Try again to view your officer roster."
          offline={ApiError.isNetwork(units.error) || ApiError.isNetwork(memberships.error)}
          onRetry={() => { void units.refetch(); void memberships.refetch() }} />
      ) : requestedUnitId && !unitId ? (
        <EmptyState icon={<Users className="size-6" />} title="Unit unavailable"
          description="You cannot view this unit’s roster, or it is no longer in the unit directory." />
      ) : availableUnits.length === 0 ? (
        <EmptyState icon={<Users className="size-6" />} title="No unit roster available"
          description="An active unit administrator membership is required to view a unit roster." />
      ) : !requestedUnitId ? (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {availableUnits.map((item) => (
            <Card key={item.id} as="article" className="p-5 transition-all duration-200 hover:-translate-y-1 hover:border-signal/60 hover:shadow-lg focus-within:border-signal">
              <span className="grid size-11 place-items-center rounded-xl bg-signal/10 text-signal"><Users className="size-5" aria-hidden /></span>
              <h2 className="mt-4 text-lg font-semibold text-ink">{item.name}</h2>
              <p className="mt-1 text-sm text-ink-muted">{item.city || item.state || 'Security unit'}</p>
              <Link className="mt-5 inline-block text-sm font-semibold text-signal underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-signal"
                to={`/admin/officers/${item.id}`}>Open unit and view officers</Link>
            </Card>
          ))}
        </div>
      ) : (
        <>
          <Card className="flex flex-col gap-4 p-4 sm:flex-row sm:items-end sm:justify-between">
            <div className="min-w-0 space-y-1">
              <Link to="/admin/officers" className="text-xs font-semibold uppercase tracking-wide text-signal hover:underline">← All units</Link>
              <p className="text-lg font-semibold text-ink">{unit?.name}</p>
              <p className="text-xs text-ink-muted">Only officers in this unit appear below.</p>
            </div>
            <Link to="/admin/cases" className="rounded-lg bg-signal px-4 py-2 text-center text-sm font-semibold text-signal-ink hover:opacity-90">
              Open case board
            </Link>
          </Card>

          {roster.isLoading ? (
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {[0, 1, 2].map((index) => <Skeleton key={index} className="h-48 rounded-panel" />)}
            </div>
          ) : roster.isError ? (
            <ErrorState title="Could not load officers" description="The unit roster is unavailable. Try again."
              offline={ApiError.isNetwork(roster.error)} onRetry={() => void roster.refetch()} />
          ) : (
            <>
              <div className="flex flex-wrap items-center justify-between gap-3">
                <p className="text-sm text-ink-muted">{roster.data?.length ?? 0} officers · {officers.length} shown</p>
                <div className="relative w-full sm:w-72">
                  <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-muted" aria-hidden />
                  <Input type="search" className="pl-9" aria-label="Search officers" placeholder="Search name, badge or rank"
                    value={query} onChange={(event) => setQuery(event.target.value)} />
                </div>
              </div>
              {officers.length === 0 ? (
                <EmptyState icon={<UserRound className="size-6" />} title="No officers found"
                  description={query ? 'Try another name, badge, rank or duty role.' : 'There are no officers on this unit’s roster yet.'} />
              ) : (
                <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                  {officers.map((officer) => {
                    const openCases = cases.data?.filter((item) => item.unitId === unitId && item.assignedTo === officer.id && item.status !== 'closed').length
                    return (
                      <Card key={officer.id} as="article" className="group p-5 transition-all duration-200 hover:-translate-y-1 hover:border-signal/60 hover:shadow-lg focus-within:border-signal">
                        <div className="flex items-start gap-3">
                          <span className="grid size-11 shrink-0 place-items-center rounded-xl bg-signal/10 text-signal"><ShieldCheck className="size-5" aria-hidden /></span>
                          <div className="min-w-0 flex-1">
                            <h2 className="truncate text-base font-semibold text-ink">{officer.name}</h2>
                            <p className="text-xs text-ink-muted">{officer.rank} · {officer.role}</p>
                          </div>
                          <span className={`rounded-full px-2 py-1 text-[11px] font-semibold ${officer.status === 'active' ? 'bg-ok/10 text-ok' : 'bg-warn/10 text-warn'}`}>
                            {officer.status}
                          </span>
                        </div>
                        <div className="mt-5 grid grid-cols-2 gap-3 border-t border-border pt-4 text-xs">
                          <div><p className="text-ink-muted">Badge</p><p className="mt-1 font-medium text-ink">{officer.badgeNumber}</p></div>
                          <div><p className="text-ink-muted">Open cases</p><p className="mt-1 font-medium text-ink">{openCases ?? '—'}</p></div>
                        </div>
                        <Link to="/admin/cases" className="mt-5 inline-flex rounded-md text-xs font-semibold text-signal underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-signal">
                          View case board to assign
                        </Link>
                      </Card>
                    )
                  })}
                </div>
              )}
            </>
          )}
        </>
      )}
    </main>
  )
}
