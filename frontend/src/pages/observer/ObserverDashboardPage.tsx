import type { ReactNode } from 'react'
import {
  Activity,
  FolderKanban,
  MapPin,
  RefreshCw,
  Shield,
  ShieldAlert,
} from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Chips'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useObserverIncidents, useObserverUnits } from '@/hooks/useObserverDashboard'
import {
  observerCategoryLabel,
  observerLocationLabel,
  observerStatusLabel,
  observerStatusTone,
} from '@/lib/observer'
import { relativeTime } from '@/lib/format'

function Metric({
  label,
  value,
  icon,
}: {
  label: string
  value: number | string
  icon: ReactNode
}) {
  return (
    <Card>
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-xs uppercase tracking-widest text-ink-muted">{label}</p>
          <p className="mt-2 text-2xl font-semibold text-ink tabular-nums">{value}</p>
        </div>
        <span className="text-signal">{icon}</span>
      </div>
    </Card>
  )
}

function IncidentSkeleton() {
  return (
    <div className="flex flex-col gap-2">
      {Array.from({ length: 4 }).map((_, index) => (
        <Skeleton key={index} className="h-28 w-full" />
      ))}
    </div>
  )
}

function UnitSkeleton() {
  return (
    <div className="flex flex-col gap-2">
      {Array.from({ length: 4 }).map((_, index) => (
        <Skeleton key={index} className="h-24 w-full" />
      ))}
    </div>
  )
}

export function ObserverDashboardPage() {
  const incidents = useObserverIncidents()
  const units = useObserverUnits()

  const incidentList = incidents.data?.incidents ?? []
  const unitList = units.data?.units ?? []
  const activeSOS = unitList.reduce((total, unit) => total + unit.activeSOS, 0)
  const openCases = unitList.reduce((total, unit) => total + unit.openCases, 0)

  function refreshAll() {
    void incidents.refetch()
    void units.refetch()
  }

  return (
    <div className="mx-auto w-full max-w-6xl space-y-4 p-4 sm:p-6">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-ink">Agency oversight</h1>
          <p className="mt-1 text-sm text-ink-muted">
            Active incidents and responding-unit activity inside your agency area.
          </p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          icon={<RefreshCw className="size-4" aria-hidden />}
          loading={incidents.isFetching || units.isFetching}
          onClick={refreshAll}
        >
          Refresh
        </Button>
      </header>

      <div className="grid gap-4 sm:grid-cols-3">
        <Metric
          label="Incidents"
          value={incidents.data?.total ?? '—'}
          icon={<ShieldAlert className="size-5" aria-hidden />}
        />
        <Metric
          label="Active SOS"
          value={units.data ? activeSOS : '—'}
          icon={<Activity className="size-5" aria-hidden />}
        />
        <Metric
          label="Open cases"
          value={units.data ? openCases : '—'}
          icon={<FolderKanban className="size-5" aria-hidden />}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader
            title="Incident feed"
            subtitle={
              incidents.data
                ? `${incidentList.length} shown of ${incidents.data.total} in your area`
                : 'Active SOS alerts and open cases in your area'
            }
          />
          <CardBody>
            {incidents.isLoading ? (
              <IncidentSkeleton />
            ) : incidents.isError ? (
              <ErrorState
                title="Could not load incidents"
                description="The incident feed is not reachable right now."
                onRetry={() => void incidents.refetch()}
              />
            ) : incidentList.length === 0 ? (
              <EmptyState
                icon={<ShieldAlert className="size-5" aria-hidden />}
                title="No active incidents"
                description="New SOS alerts and open cases in your area will appear here."
              />
            ) : (
              <ul className="flex flex-col gap-2">
                {incidentList.map((incident, index) => (
                  <li
                    key={`${incident.time}-${incident.respondingUnit}-${index}`}
                    className="rounded-xl border border-border p-3"
                  >
                    <div className="flex flex-wrap items-start justify-between gap-2">
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-1.5">
                          <Badge tone="signal">{observerCategoryLabel(incident.type)}</Badge>
                          <Badge tone={observerStatusTone(incident.status)}>
                            {observerStatusLabel(incident.status)}
                          </Badge>
                        </div>
                        <p className="mt-1.5 text-sm font-semibold text-ink">
                          {incident.respondingUnit}
                        </p>
                        <p className="flex items-center gap-1 text-xs text-ink-muted">
                          <MapPin className="size-3 shrink-0" aria-hidden />
                          {observerLocationLabel(incident.location)}
                        </p>
                      </div>
                      <span className="shrink-0 text-[11px] text-ink-faint">
                        {relativeTime(incident.time)}
                      </span>
                    </div>
                    {incident.description ? (
                      <p className="mt-2 border-t border-border pt-2 text-sm leading-relaxed text-ink-muted">
                        {incident.description}
                      </p>
                    ) : null}
                  </li>
                ))}
              </ul>
            )}
          </CardBody>
        </Card>

        <Card>
          <CardHeader
            title="Responding units"
            subtitle={
              units.data
                ? `${unitList.length} ${unitList.length === 1 ? 'unit' : 'units'} active in your area`
                : 'Activity-derived unit summary'
            }
          />
          <CardBody>
            {units.isLoading ? (
              <UnitSkeleton />
            ) : units.isError ? (
              <ErrorState
                title="Could not load units"
                description="The unit summary is not reachable right now."
                onRetry={() => void units.refetch()}
              />
            ) : unitList.length === 0 ? (
              <EmptyState
                icon={<Shield className="size-5" aria-hidden />}
                title="No unit activity"
                description="Units responding to incidents in your area will appear here."
              />
            ) : (
              <ul className="flex flex-col gap-2">
                {unitList.map((unit, index) => (
                  <li
                    key={`${unit.respondingUnit}-${unit.time}-${index}`}
                    className="rounded-xl border border-border p-3"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="truncate text-sm font-semibold text-ink">
                          {unit.respondingUnit}
                        </p>
                        <p className="text-xs text-ink-muted">
                          {observerStatusLabel(unit.type)} · {observerLocationLabel(unit.location)}
                        </p>
                      </div>
                      <Badge tone={observerStatusTone(unit.status)}>
                        {observerStatusLabel(unit.status)}
                      </Badge>
                    </div>
                    <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-ink-muted">
                      <span>
                        Active SOS{' '}
                        <span className="font-semibold text-ink tabular-nums">{unit.activeSOS}</span>
                      </span>
                      <span>
                        Open cases{' '}
                        <span className="font-semibold text-ink tabular-nums">{unit.openCases}</span>
                      </span>
                      <span className="text-ink-faint">{relativeTime(unit.time)}</span>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </CardBody>
        </Card>
      </div>
    </div>
  )
}
