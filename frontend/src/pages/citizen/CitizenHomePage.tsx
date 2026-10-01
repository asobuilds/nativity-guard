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
import { useProfile } from '@/hooks/useProfile'
import { useAuth } from '@/auth/AuthContext'
import { relativeTime, initials } from '@/lib/format'
import { api, mediaURL } from '@/lib/apiClient'
import { coverImageStyle, normalizeCoverAdjust } from '@/lib/coverPosition'
import type { User as UserType } from '@/types/api'
import type { CommunityAlert, CommunityPost } from '@/types/community'

interface ProfileWithImages extends UserType {
  avatarPath?: string
  coverPath?: string
  coverPositionX?: number
  coverPositionY?: number
  coverZoom?: number
}

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
  const profileQuery = useProfile()
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
    ? `${unitCount} security units active near you.`
    : 'Share your location to see units near you.'

  const profileWithImages = profileQuery.data as ProfileWithImages | undefined
  const avatarUrl =
    mediaURL(profileWithImages?.avatarPath ?? profileQuery.data?.photoUrl) ?? null
  const coverUrl = mediaURL(profileWithImages?.coverPath) ?? null
  const coverAdjust = normalizeCoverAdjust({
    coverPositionX: profileWithImages?.coverPositionX,
    coverPositionY: profileWithImages?.coverPositionY,
    coverZoom: profileWithImages?.coverZoom,
  })
  const initialsValue = initials(user?.firstName, user?.lastName)
  const fullName = [user?.firstName, user?.lastName].filter(Boolean).join(' ')

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
      <section className="mb-4 overflow-hidden rounded-panel border border-border bg-surface/50">
        <div className="relative">
          <div className="h-48 w-full overflow-hidden bg-gradient-to-r from-signal/25 via-signal/10 to-warn/20 sm:h-64">
            {coverUrl ? (
              <img
                src={coverUrl}
                alt=""
                draggable={false}
                className="select-none"
                style={coverImageStyle(coverAdjust)}
              />
            ) : null}
          </div>

          <Link
            to="/profile"
            aria-label="Open profile to change your photo"
            className="absolute left-1/2 -bottom-16 block -translate-x-1/2 rounded-full transition-transform hover:scale-[1.02] focus:outline-none focus-visible:ring-4 focus-visible:ring-signal/50 sm:-bottom-20"
          >
            {avatarUrl ? (
              <img
                src={avatarUrl}
                alt={fullName || 'Your profile photo'}
                className="size-32 rounded-full border-[6px] border-base bg-base object-cover shadow-panel sm:size-40"
                style={{ objectPosition: '50% 20%' }}
              />
            ) : (
              <div className="grid size-32 place-items-center rounded-full border-[6px] border-base bg-surface-hi text-4xl font-bold text-ink shadow-panel sm:size-40 sm:text-5xl">
                {initialsValue}
              </div>
            )}
          </Link>
        </div>

        <div className="px-6 pb-6 pt-24 text-center sm:pt-28">
          <h1 className="text-3xl font-bold tracking-tight text-ink">
            {firstName ? `Hi, ${firstName}` : 'Welcome back'}
          </h1>
          <p className="mt-2 text-base text-ink-muted">{heroSubtitle}</p>

          <div className="mt-5 flex justify-center">
            <Link
              to="/sos"
              aria-label="Open emergency SOS"
              className="inline-flex w-full max-w-xs items-center justify-center gap-2 rounded-lg bg-emergency px-6 py-3 font-semibold text-white hover:bg-emergency/90 min-h-[56px]"
            >
              <ShieldAlert className="size-5" />
              SOS
            </Link>
          </div>

          {!avatarUrl ? (
            <p className="mt-3 text-xs text-ink-faint">
              <Link to="/profile" className="text-signal hover:underline">
                Add a profile photo
              </Link>{' '}
              so officers can identify you.
            </p>
          ) : null}
        </div>
      </section>

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

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Card as="section">
          <CardHeader title="Your recent reports" />
          <CardBody className="flex flex-col gap-3">
            {casesQuery.isLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-16 w-full" />
              ))
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
              Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))
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
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => void nearby.refetch()}
                    >
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

      {postsQuery.isError || posts.length === 0 ? null : (
        <section className="mt-4">
          <Card>
            <CardHeader
              title="Latest from your community"
              subtitle="Real posts from residents in your area"
            />
            <CardBody className="flex flex-col gap-3">
              {postsQuery.isLoading ? (
                Array.from({ length: 2 }).map((_, i) => (
                  <Skeleton key={i} className="h-12 w-full" />
                ))
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
                        <p className="text-xs text-ink-faint">
                          {relativeTime(p.createdAt)}
                        </p>
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