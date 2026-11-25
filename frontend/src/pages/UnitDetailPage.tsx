import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  ArrowLeft,
  FileText,
  Mail,
  MapPin,
  Navigation,
  Phone,
  Shield,
  ShieldAlert,
  Users,
  X,
} from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { PriorityChip, StatusChip } from '@/components/ui/Chips'
import { Skeleton, ErrorState } from '@/components/ui/States'
import { useToast } from '@/components/ui/Toast'
import { MapView } from '@/components/map/MapView'
import { UnitHero } from '@/components/unit/UnitHero'
import { UnitStatsGrid } from '@/components/unit/UnitStatsGrid'
import { UnitTabs, type UnitTabDef } from '@/components/unit/UnitTabs'
import { UnitMemberRoster } from '@/components/unit/UnitMemberRoster'
import { ComplianceMatrix } from '@/components/unit/ComplianceMatrix'
import { GovernancePanel } from '@/components/unit/GovernancePanel'
import { InviteManager } from '@/components/unit/InviteManager'
import { api, ApiError } from '@/lib/apiClient'
import { useLocation } from '@/hooks/useLocation'
import { useMapPOIs, type POICategory } from '@/hooks/useMapPOIs'
import { useDirections } from '@/hooks/useDirections'
import { useUnitPublicSummary } from '@/hooks/useUnitPublicSummary'
import { useUnitAccess } from '@/hooks/useUnitAccess'
import { useUnitRoster } from '@/hooks/useUnitRoster'
import { useUnitInbox } from '@/hooks/useUnitInbox'
import { useUnitCompliance } from '@/hooks/useUnitCompliance'
import { useMembershipActions } from '@/hooks/useMembershipActions'
import { formatDate, relativeTime } from '@/lib/format'
import type { SecurityUnit } from '@/types/api'

const MAP_HEIGHT = '56vh'

type TabId =
  | 'overview'
  | 'roster'
  | 'cases'
  | 'sos'
  | 'invites'
  | 'governance'
  | 'compliance'

export function UnitDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { latitude: userLat, longitude: userLng } = useLocation()
  const { notify } = useToast()
  const [directionsActive, setDirectionsActive] = useState(false)
  const [activeTab, setActiveTab] = useState<TabId>('overview')

  const [poiCategories] = useState<Set<POICategory>>(
    () => new Set<POICategory>(['hospital', 'police', 'bank', 'school']),
  )

  const query = useQuery({
    queryKey: ['unit', id],
    queryFn: () => api.get<{ unit: SecurityUnit }>(`/units/${id}`),
    select: (d) => d.unit,
    enabled: Boolean(id),
  })

  const summaryQuery = useUnitPublicSummary(id)
  const accessQuery = useUnitAccess(id)

  const unit = query.data
  const isMember = accessQuery.data?.isMember === true
  const isAdmin =
    accessQuery.data?.isHeadAdmin === true || accessQuery.data?.role === 'unit_admin'

  const rosterQuery = useUnitRoster(id, isMember)
  const inboxQuery = useUnitInbox(isAdmin ? id : undefined)
  const complianceQuery = useUnitCompliance(
    isAdmin && activeTab === 'compliance' ? id : undefined,
  )
  const membership = useMembershipActions(id)

  const hasCoords =
    !!unit &&
    Number.isFinite(unit.latitude) &&
    Number.isFinite(unit.longitude) &&
    unit.latitude !== 0 &&
    unit.longitude !== 0

  const pois = useMapPOIs(
    hasCoords ? unit!.latitude : null,
    hasCoords ? unit!.longitude : null,
    2000,
    [...poiCategories],
  )

  const directionsFrom =
    userLat != null && userLng != null ? { lat: userLat, lng: userLng } : null
  const directionsTo =
    hasCoords && directionsActive ? { lat: unit!.latitude, lng: unit!.longitude } : null
  const directionsRoute = useDirections(directionsFrom, directionsTo)

  function reportError(cause: unknown, fallback: string) {
    if (cause instanceof ApiError) notify(cause.message, 'error')
    else notify(fallback, 'error')
  }

  if (query.isLoading) {
    return (
      <div className="mx-auto max-w-5xl space-y-4 p-4 sm:p-6">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-64 w-full rounded-panel" />
        <Skeleton className="h-24 w-full rounded-panel" />
      </div>
    )
  }

  if (query.isError || !unit) {
    return (
      <div className="mx-auto max-w-5xl p-4 sm:p-6">
        <Link to="/units" className="text-sm text-signal hover:underline">
          ← Back to units
        </Link>
        <Card className="mt-4">
          <ErrorState
            title="Unit unavailable"
            description="This unit could not be found."
            onRetry={() => void query.refetch()}
          />
        </Card>
      </div>
    )
  }

  const canGetDirections = hasCoords && directionsFrom != null
  const isRoute = directionsActive && directionsTo != null
  const displayName = unit.brandName || unit.name
  const location = [unit.city, unit.lga, unit.state, unit.ward].filter(Boolean).join(', ')
  const rosterCounts = rosterQuery.data?.counts

  const membershipBusy =
    membership.approve.isPending ||
    membership.reject.isPending ||
    membership.revoke.isPending ||
    membership.promote.isPending

  const tabs: UnitTabDef[] = isMember
    ? [
        { id: 'overview', label: 'Overview' },
        { id: 'roster', label: 'Roster', badge: rosterCounts?.total },
        ...(isAdmin
          ? [
              { id: 'cases', label: 'Cases', badge: inboxQuery.data?.pendingCaseCount },
              { id: 'sos', label: 'SOS', badge: inboxQuery.data?.sosAlertCount },
            ]
          : []),
        { id: 'invites', label: 'Invites' },
        { id: 'governance', label: 'Governance' },
        ...(isAdmin ? [{ id: 'compliance', label: 'Compliance' }] : []),
      ]
    : []

  return (
    <div className="mx-auto max-w-5xl space-y-4 p-4 sm:p-6">
      <Link
        to="/units"
        className="inline-flex items-center gap-1.5 text-sm text-signal hover:underline"
      >
        <ArrowLeft className="size-3.5" aria-hidden />
        Back to units
      </Link>

      <UnitHero unit={unit} />

      <UnitStatsGrid
        summary={summaryQuery.data}
        unit={{
          operationalRadius: unit.operationalRadius ?? 10,
          totalMembers: unit.totalMembers ?? unit.memberCount ?? 0,
        }}
        isLoading={summaryQuery.isLoading}
      />

      {isMember ? (
        <UnitTabs tabs={tabs} activeId={activeTab} onChange={(id) => setActiveTab(id as TabId)} />
      ) : null}

      {(activeTab === 'overview' || !isMember) ? (
        <>
          {hasCoords ? (
            <>
              <Card className="overflow-hidden">
                <MapView
                  mode={isRoute ? 'route' : 'view'}
                  route={isRoute ? directionsRoute.data ?? null : null}
                  cases={[]}
                  units={isRoute ? [] : [unit]}
                  pois={isRoute ? [] : pois.data?.items ?? []}
                  hiddenPOICategories={new Set()}
                  showUnitCoverage={!isRoute}
                  showHotspots={false}
                  userLocation={
                    userLat != null && userLng != null
                      ? { latitude: userLat, longitude: userLng }
                      : null
                  }
                  allowLocate
                  height={MAP_HEIGHT}
                />
              </Card>

              {isRoute ? (
                <Card className="border-signal/40">
                  <div className="flex flex-col gap-3 p-5">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="flex items-center gap-2 text-base font-semibold text-ink">
                          <Navigation className="size-5 text-signal" aria-hidden />
                          Directions to {displayName}
                        </p>
                        {directionsRoute.data ? (
                          <p className="mt-1 text-sm font-medium text-signal">
                            {directionsRoute.data.summary}
                          </p>
                        ) : directionsRoute.isLoading ? (
                          <p className="mt-1 text-sm text-ink-faint">Calculating route…</p>
                        ) : directionsRoute.isError ? (
                          <p className="mt-1 text-sm text-warn">
                            Could not calculate the route. Try again.
                          </p>
                        ) : null}
                      </div>
                      <Button
                        size="sm"
                        variant="ghost"
                        icon={<X className="size-4" aria-hidden />}
                        onClick={() => setDirectionsActive(false)}
                      >
                        Close
                      </Button>
                    </div>

                    {directionsRoute.data && directionsRoute.data.steps.length > 0 ? (
                      <ol className="flex flex-col gap-2 rounded-lg border border-border bg-surface-hi/30 p-3">
                        {directionsRoute.data.steps.slice(0, 10).map((step, i) => (
                          <li key={i} className="flex items-start gap-2 text-sm">
                            <span className="tabular-nums text-ink-faint">{i + 1}.</span>
                            <span className="flex-1 text-ink-muted">
                              {step.instruction}
                              <span className="ml-1 text-ink-faint">
                                ({Math.round(step.distanceMeters)} m)
                              </span>
                            </span>
                          </li>
                        ))}
                      </ol>
                    ) : null}
                  </div>
                </Card>
              ) : canGetDirections ? (
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="secondary"
                    icon={<Navigation className="size-4" aria-hidden />}
                    onClick={() => setDirectionsActive(true)}
                  >
                    Get directions to this unit
                  </Button>
                </div>
              ) : (
                <p className="text-xs text-ink-muted">
                  Enable your location to get directions to this unit.
                </p>
              )}
            </>
          ) : null}

          <div className="grid gap-3 sm:grid-cols-2">
            {unit.contactPhone ? (
              <Card className="flex items-center gap-3 p-4">
                <Phone className="size-4 text-signal" aria-hidden />
                <div>
                  <p className="text-xs text-ink-faint">Phone</p>
                  <a
                    href={`tel:${unit.contactPhone}`}
                    className="text-sm text-ink hover:underline"
                  >
                    {unit.contactPhone}
                  </a>
                </div>
              </Card>
            ) : null}
            {unit.contactEmail ? (
              <Card className="flex items-center gap-3 p-4">
                <Mail className="size-4 text-signal" aria-hidden />
                <div>
                  <p className="text-xs text-ink-faint">Email</p>
                  <a
                    href={`mailto:${unit.contactEmail}`}
                    className="text-sm text-ink hover:underline"
                  >
                    {unit.contactEmail}
                  </a>
                </div>
              </Card>
            ) : null}
            {location ? (
              <Card className="flex items-center gap-3 p-4">
                <MapPin className="size-4 text-signal" aria-hidden />
                <div>
                  <p className="text-xs text-ink-faint">Location</p>
                  <p className="text-sm text-ink">{location}</p>
                </div>
              </Card>
            ) : null}
            {unit.contactPerson ? (
              <Card className="flex items-center gap-3 p-4">
                <Users className="size-4 text-signal" aria-hidden />
                <div>
                  <p className="text-xs text-ink-faint">Contact person</p>
                  <p className="text-sm text-ink">{unit.contactPerson}</p>
                </div>
              </Card>
            ) : null}
          </div>

          {unit.coverageArea ? (
            <Card className="p-5">
              <h2 className="text-sm font-semibold text-ink">Coverage area</h2>
              <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-ink-muted">
                {unit.coverageArea}
              </p>
            </Card>
          ) : null}

          {unit.commanderName ? (
            <Card className="p-5">
              <h2 className="flex items-center gap-2 text-sm font-semibold text-ink">
                <Shield className="size-4 text-signal" aria-hidden />
                Commander
              </h2>
              <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-2">
                <div>
                  <dt className="text-xs text-ink-faint">Name</dt>
                  <dd className="text-ink">{unit.commanderName}</dd>
                </div>
                {unit.commanderPhoneAlt ? (
                  <div>
                    <dt className="text-xs text-ink-faint">Alt phone</dt>
                    <dd className="text-ink">{unit.commanderPhoneAlt}</dd>
                  </div>
                ) : null}
                {unit.commanderOccupation ? (
                  <div>
                    <dt className="text-xs text-ink-faint">Occupation</dt>
                    <dd className="text-ink">{unit.commanderOccupation}</dd>
                  </div>
                ) : null}
                {unit.commanderPriorExperience ? (
                  <div className="sm:col-span-2">
                    <dt className="text-xs text-ink-faint">Prior experience</dt>
                    <dd className="whitespace-pre-wrap text-ink-muted">
                      {unit.commanderPriorExperience}
                    </dd>
                  </div>
                ) : null}
              </dl>
            </Card>
          ) : null}

          {unit.formationDate ? (
            <p className="text-center text-xs text-ink-faint">
              Operating since {formatDate(unit.formationDate)}
            </p>
          ) : null}

          <div className="flex flex-wrap gap-2 pt-2">
            <Link to="/report">
              <Button variant="primary">Report near this unit</Button>
            </Link>
            <Link to="/map">
              <Button variant="secondary" icon={<MapPin className="size-4" aria-hidden />}>
                View on map
              </Button>
            </Link>
          </div>
        </>
      ) : null}

      {isMember && activeTab === 'roster' ? (
        <Card className="p-5">
          <h2 className="text-base font-semibold text-ink">Members</h2>
          <p className="mt-1 text-xs text-ink-muted">
            Everyone serving in this unit. Head admin, admins and officers.
          </p>
          <div className="mt-4">
            <UnitMemberRoster
              roster={rosterQuery.data}
              isLoading={rosterQuery.isLoading}
              isAdmin={isAdmin}
              currentUserId={accessQuery.data?.membershipId ? undefined : undefined}
              onApprove={(membershipId) =>
                membership.approve.mutate(
                  { unitId: unit.id, membershipId },
                  {
                    onSuccess: () => notify('Member approved', 'success'),
                    onError: (cause) => reportError(cause, 'Could not approve'),
                  },
                )
              }
              onReject={(membershipId, reason) =>
                membership.reject.mutate(
                  { unitId: unit.id, membershipId, reason },
                  {
                    onSuccess: () => notify('Application rejected', 'success'),
                    onError: (cause) => reportError(cause, 'Could not reject'),
                  },
                )
              }
              onPromote={(membershipId) =>
                membership.promote.mutate(
                  { unitId: unit.id, membershipId },
                  {
                    onSuccess: () => notify('Member promoted', 'success'),
                    onError: (cause) => reportError(cause, 'Could not promote'),
                  },
                )
              }
              onRevoke={(membershipId, reason) =>
                membership.revoke.mutate(
                  { unitId: unit.id, membershipId, reason },
                  {
                    onSuccess: () => notify('Member revoked', 'success'),
                    onError: (cause) => reportError(cause, 'Could not revoke'),
                  },
                )
              }
              busy={membershipBusy}
            />
          </div>
        </Card>
      ) : null}

      {isAdmin && activeTab === 'cases' ? (
        <Card className="p-5">
          <div className="flex items-center justify-between gap-2">
            <div>
              <h2 className="flex items-center gap-2 text-base font-semibold text-ink">
                <FileText className="size-4 text-signal" aria-hidden />
                Pending unit cases
              </h2>
              <p className="mt-1 text-xs text-ink-muted">
                Cases awaiting assignment or dispatch in this unit.
              </p>
            </div>
            <Link to="/unit/dashboard" className="text-xs font-medium text-signal hover:underline">
              Unit dashboard →
            </Link>
          </div>
          <div className="mt-4">
            {inboxQuery.isLoading ? (
              <div className="h-32 animate-pulse rounded-panel bg-surface-hi/60" />
            ) : inboxQuery.isError ? (
              <ErrorState
                title="Could not load unit cases"
                description="You may not have unit admin access to view this queue."
                onRetry={() => void inboxQuery.refetch()}
              />
            ) : !inboxQuery.data || inboxQuery.data.pendingCases.length === 0 ? (
              <p className="rounded-lg border border-dashed border-border px-4 py-8 text-center text-sm text-ink-muted">
                No cases awaiting action in this unit.
              </p>
            ) : (
              <ul className="divide-y divide-border rounded-panel border border-border bg-surface/60">
                {inboxQuery.data.pendingCases.map((c) => (
                  <li key={c.id} className="flex items-center gap-3 p-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <PriorityChip level={c.priorityLevel} />
                        <StatusChip status={c.status} />
                      </div>
                      <Link
                        to={`/admin/cases/${c.id}`}
                        className="mt-1 block truncate text-sm font-medium text-ink hover:text-signal"
                      >
                        {c.title}
                      </Link>
                      <p className="text-xs text-ink-faint">
                        {c.trackingId} · {c.location || 'no location'} · {relativeTime(c.createdAt)}
                      </p>
                    </div>
                    <Link
                      to={`/admin/cases/${c.id}`}
                      className="shrink-0 text-xs font-medium text-signal hover:underline"
                    >
                      Review →
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </Card>
      ) : null}

      {isAdmin && activeTab === 'sos' ? (
        <Card className="p-5">
          <div className="flex items-center justify-between gap-2">
            <div>
              <h2 className="flex items-center gap-2 text-base font-semibold text-ink">
                <ShieldAlert className="size-4 text-emergency" aria-hidden />
                Unclaimed SOS alerts
              </h2>
              <p className="mt-1 text-xs text-ink-muted">
                Pending SOS alerts routed to this unit.
              </p>
            </div>
            <Link to="/unit/dashboard" className="text-xs font-medium text-signal hover:underline">
              Unit dashboard →
            </Link>
          </div>
          <div className="mt-4">
            {inboxQuery.isLoading ? (
              <div className="h-32 animate-pulse rounded-panel bg-surface-hi/60" />
            ) : inboxQuery.isError ? (
              <ErrorState
                title="Could not load unit SOS alerts"
                description="You may not have unit admin access to view this queue."
                onRetry={() => void inboxQuery.refetch()}
              />
            ) : !inboxQuery.data || inboxQuery.data.sosAlerts.length === 0 ? (
              <p className="rounded-lg border border-dashed border-border px-4 py-8 text-center text-sm text-ink-muted">
                No SOS alerts waiting for this unit.
              </p>
            ) : (
              <ul className="divide-y divide-border rounded-panel border border-border bg-surface/60">
                {inboxQuery.data.sosAlerts.map((s) => (
                  <li key={s.id} className="flex items-center gap-3 p-3">
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium text-ink">
                        {s.description || 'SOS alert — no description'}
                      </p>
                      <p className="text-xs text-ink-faint">
                        {relativeTime(s.createdAt)} · {s.latitude.toFixed(4)},{' '}
                        {s.longitude.toFixed(4)}
                      </p>
                    </div>
                    <Link
                      to={`/sos/${s.id}`}
                      className="shrink-0 text-xs font-medium text-signal hover:underline"
                    >
                      Open →
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </Card>
      ) : null}

      {isMember && activeTab === 'invites' ? (
        <Card className="p-5">
          <h2 className="text-base font-semibold text-ink">Invites</h2>
          <p className="mt-1 text-xs text-ink-muted">
            {isAdmin
              ? 'Generate signup links that let a new member apply to this unit.'
              : 'Only unit admins can create invites.'}
          </p>
          <div className="mt-4">
            {isAdmin ? (
              <InviteManager unitId={unit.id} />
            ) : (
              <p className="rounded-lg border border-dashed border-border px-4 py-8 text-center text-sm text-ink-muted">
                You do not have permission to manage invites.
              </p>
            )}
          </div>
        </Card>
      ) : null}

      {isMember && activeTab === 'governance' ? (
        <GovernancePanel unitId={unit.id} isAdmin={isAdmin} />
      ) : null}

      {isAdmin && activeTab === 'compliance' ? (
        <Card className="p-5">
          <h2 className="text-base font-semibold text-ink">Weekly compliance</h2>
          <p className="mt-1 text-xs text-ink-muted">
            Weekly update filings per active officer across the last 8 weeks.
          </p>
          <div className="mt-4">
            {complianceQuery.isError ? (
              <ErrorState
                title="Could not load compliance matrix"
                description="Weekly compliance is available to unit admins."
                onRetry={() => void complianceQuery.refetch()}
              />
            ) : (
              <ComplianceMatrix
                compliance={complianceQuery.data}
                isLoading={complianceQuery.isLoading}
              />
            )}
          </div>
        </Card>
      ) : null}
    </div>
  )
}
