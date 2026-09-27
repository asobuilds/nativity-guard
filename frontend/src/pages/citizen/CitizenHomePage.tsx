import { Link, useNavigate } from 'react-router-dom'
import {
  FileText,
  MapPin,
  Map as MapIcon,
  Megaphone,
  Plus,
  ShieldAlert,
  Sparkles,
  Users,
} from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { StatusChip } from '@/components/ui/Chips'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useCases } from '@/hooks/useCases'
import { useNearbyUnits } from '@/hooks/useUnits'
import { useLocation } from '@/hooks/useLocation'
import { useAuth } from '@/auth/AuthContext'
import { relativeTime, initials } from '@/lib/format'
import { api } from '@/lib/apiClient'
import type { CommunityAlert, CommunityPost } from '@/types/community'

/**
 * Citizen home, rebuilt as a single dashboard rather than a bare cases list.
 *
 * The page composes four data sources — the signed-in profile, the reporter's
 * own cases, the units near the reporter, and the community feed — plus two
 * one-off inline queries for alerts. Every hook is called unconditionally at
 * the top: a conditional `useQuery` would move past the Rules-of-Hooks line
 * and re-create the React #301 bug we fixed (hooks shifting order between
 * renders), so the guards live in the render body, not before the hooks.
 *
 * The SOS FAB in `AppShell` is untouched here — this hero button is a
 * deliberate, confirmable secondary trigger, kept distinct from the always
 * present emergency floater.
 */
export function CitizenHomePage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const casesQuery = useCases()
  const nearby = useNearbyUnits(
    location.latitude ?? undefined,
    location.longitude ?? undefined,
    10,
  )
  // One-off inline queries — no new global hooks created.
  const alertsQuery = useQuery({
    queryKey: ['dashboard-alerts'],
    queryFn: () => api.get<{ alerts: CommunityAlert[] }>('/alerts'),
  })
  const postsQuery = useQuery({
    queryKey: ['dashboard-posts'],
    queryFn: () => api.get<{ posts: CommunityPost[] }>('/community/posts'),
  })

  const firstName = user?.firstName ?? ''
  const hasLocation =
    typeof location.latitude === 'number' && typeof location.longitude === 'number'
  const unitCount = nearby.data?.length ?? 0
  const heroSubtitle = hasLocation
    ? `You're safe. ${unitCount} units active near you.`
    : "You're safe. Share your location to see units near you."

  const recentCases = (casesQuery.data ?? []).slice(0, 3)
  const nearbyList = (nearby.data ?? [])
    .slice()
    .sort((a, b) => a.distance - b.distance)
    .slice(0, 3)

  const alerts = alertsQuery.data?.alerts ?? []
  const weekAgoMs = Date.now() - 7 * 24 * 60 * 60 * 1000
  const recentAlertCount = alerts.filter((a) => {
    const t = new Date(a.createdAt).getTime()
    return Number.isFinite(t) && t >= weekAgoMs
  }).length

  const posts = (postsQuery.data?.posts ?? []).slice(0, 2)

  return (
    <div className="mx-auto w-full max-w-3xl p-4 sm:p-6">
      {/* HERO BAND */}
      <section className="mb-4 rounded-panel border border-border bg-surface/50 p-6">
        <div className="flex flex-col items-start justify-between gap-4 md:flex-row md:items-center">
          <div>
            <h1 className="text-2xl font-bold text-ink">Welcome back, {firstName || 'there'}</h1>
            <p className="mt-1 text-ink-muted">{heroSubtitle}</p>
          </div>
          <Link
            to="/sos"
            aria-label="Open emergency SOS"
            className="inline-flex w-full items-center justify-center gap-2 rounded-lg bg-emergency px-5 py-3 font-semibold text-white hover:bg-emergency/90 min-h-[56px] md:w-auto md:min-w-[140px]"
          >
            <ShieldAlert className="size-5" />
            SOS
          </Link>
        </div>
      </section>

      {/* QUICK ACTIONS */}
      <nav aria-label="Quick actions" className="mb-4">
        <h2 className="mb-2 text-xs font-semibold uppercase tracking-wide text-ink-muted">
          Quick actions
        </h2>
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {QUICK_ACTIONS.map((a) => (
            <Link key={a.to} to={a.to} className="block">
              <Card>
                <CardBody className="flex flex-col items-center justify-center gap-2 min-h-[88px]">
                  {a.icon}
                  <span className="text-xs text-ink">{a.label}</span>
                </CardBody>
              </Card>
            </Link>
          ))}
        </div>
      </nav>

      {/* TWO-COLUMN: recent cases | nearby units */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Card as="section">
          <CardHeader title="Your recent reports" />
          <CardBody className="flex flex-col gap-3">
            {casesQuery.isLoading ? (
              Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-16 w-full" />)
            ) : casesQuery.isError ? (
              <ErrorState
                title="Could not load your reports"
                description="Check your connection and try again."
                onRetry={() => void casesQuery.refetch()}
              />
            ) : recentCases.length === 0 ? (
              <EmptyState
                icon={<Plus className="size-5" aria-hidden />}
                title="No reports yet — your first one takes 2 minutes."
                description="When you report an incident it appears here with live status from the responding unit."
                action={
                  <Button variant="primary" size="sm" onClick={() => navigate('/report')}>
                    Start a report
                  </Button>
                }
              />
            ) : (
              <ul className="flex flex-col gap-3">
                {recentCases.map((c) => (
                  <li key={c.id}>
                    <Link
                      to={`/cases/${c.id}`}
                      aria-label={`Open report ${c.trackingId}: ${c.title}`}
                      className="block rounded-panel transition-colors hover:border-border-hi focus:outline-none focus-visible:ring-2 focus-visible:ring-signal"
                    >
                      <Card as="article">
                        <CardHeader
                          title={c.title}
                          subtitle={
                            <span className="tabular text-[11px] text-ink-faint">
                              {c.trackingId} · reported {relativeTime(c.createdAt)}
                            </span>
                          }
                          actions={<StatusChip status={c.status} />}
                        />
                      </Card>
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </CardBody>
        </Card>

        <Card as="section">
          <CardHeader title="Nearby units" />
          <CardBody className="flex flex-col gap-3">
            {!hasLocation ? (
              <EmptyState
                icon={<MapPin className="size-5" aria-hidden />}
                title="Enable location to see units near you."
                description="Nativity Guard needs your location to show the units patrolling your area."
                action={
                  <Button variant="secondary" size="sm" onClick={() => location.request()}>
                    Use my location
                  </Button>
                }
              />
            ) : nearby.isLoading ? (
              Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} className="h-12 w-full" />)
            ) : nearby.isError || nearbyList.length === 0 ? (
              <EmptyState
                icon={<MapPin className="size-5" aria-hidden />}
                title={nearby.isError ? 'Could not load nearby units' : 'No units within 10 km'}
                description={
                  nearby.isError
                    ? 'Try again in a moment.'
                    : 'There are no registered units within range right now.'
                }
                action={
                  nearby.isError ? (
                    <Button variant="secondary" size="sm" onClick={() => void nearby.refetch()}>
                      Retry
                    </Button>
                  ) : undefined
                }
              />
            ) : (
              <ul className="flex flex-col gap-3">
                {nearbyList.map((u) => (
                  <li key={u.id}>
                    <Link
                      to="/map"
                      aria-label={`View ${u.name} on the safety map`}
                      className="block rounded-panel transition-colors hover:border-border-hi focus:outline-none focus-visible:ring-2 focus-visible:ring-signal"
                    >
                      <Card as="article">
                        <CardBody className="p-3">
                          <p className="text-sm font-medium text-ink">{u.name}</p>
                          <p className="text-xs text-ink-muted">
                            {u.distance.toFixed(1)} km away ·{' '}
                            {[u.state, u.lga].filter(Boolean).join(', ')}
                          </p>
                        </CardBody>
                      </Card>
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </CardBody>
        </Card>
      </div>

      {/* COMMUNITY PULSE — hidden silently when it errors or is empty */}
      {postsQuery.isError || posts.length === 0 ? null : (
        <section className="mt-4">
          <Card>
            <CardHeader
              title="Latest from your community"
              subtitle="Real posts from residents in your area"
            />
            <CardBody className="flex flex-col gap-3">
              {postsQuery.isLoading ? (
                Array.from({ length: 2 }).map((_, i) => <Skeleton key={i} className="h-12 w-full" />)
              ) : (
                <ul className="flex flex-col gap-3">
                  {posts.map((p) => (
                    <li key={p.id} className="flex items-start gap-3">
                      <span
                        className="grid size-8 shrink-0 place-items-center rounded-full bg-surface-hi text-[10px] font-semibold text-ink"
                        aria-label={`Posted by ${p.author}`}
                      >
                        {authorInitials(p.author)}
                      </span>
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-medium text-ink">{p.title}</p>
                        <p className="text-xs text-ink-faint">{relativeTime(p.createdAt)}</p>
                      </div>
                      <Link to="/community" className="shrink-0 text-xs font-medium text-signal">
                        View
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
            </CardBody>
          </Card>
        </section>
      )}

      {/* AWARENESS STRIP */}
      <section className="mt-4">
        <Card className="border border-warn/30">
          <CardBody className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <Sparkles className="size-4 text-warn" aria-hidden />
              {alertsQuery.isLoading ? (
                <Skeleton className="h-5 w-48" />
              ) : (
                <span className="text-sm text-ink">
                  Stay aware — {recentAlertCount} alerts in your area this week.
                </span>
              )}
            </div>
            <Link to="/alerts" className="shrink-0 text-xs font-medium text-signal">
              View alerts
            </Link>
          </CardBody>
        </Card>
      </section>
    </div>
  )
}

const QUICK_ACTIONS = [
  { to: '/report', label: 'Report', icon: <FileText className="size-6 text-signal" /> },
  { to: '/map', label: 'Safety map', icon: <MapIcon className="size-6 text-signal" /> },
  { to: '/community', label: 'Community', icon: <Users className="size-6 text-signal" /> },
  { to: '/alerts', label: 'Alerts', icon: <Megaphone className="size-6 text-signal" /> },
]

function authorInitials(author: string | undefined): string {
  if (!author || typeof author !== 'string') return '?'
  const parts = author.trim().split(/\s+/).filter(Boolean)
  const [first, ...rest] = parts
  return initials(first, rest.join(' '))
}
