import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Bell, Check, MapPin, Trash2 } from 'lucide-react'
import { Card, CardBody } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { useToast } from '@/components/ui/Toast'
import { api, ApiError } from '@/lib/apiClient'
import { useLocation, ipFallback } from '@/hooks/useLocation'
import { cn } from '@/lib/cn'

interface Subscription {
  id: string
  userId: string
  categories: string[]
  channels: string[]
  location: string
  latitude: number
  longitude: number
  radius: number
  isActive: boolean
  createdAt: string
  updatedAt: string
}

const CATEGORIES = [
  { value: 'security', label: 'Security incidents' },
  { value: 'community', label: 'Community notices' },
  { value: 'weather', label: 'Weather warnings' },
  { value: 'general', label: 'General updates' },
]

const CHANNELS: { value: string; label: string; available: boolean }[] = [
  { value: 'in_app', label: 'In-app notifications', available: true },
  { value: 'email', label: 'Email', available: false },
  { value: 'sms', label: 'SMS', available: false },
]

/**
 * `/subscriptions` — manage the caller's alert subscription.
 *
 * Coordinates come from the browser (via useLocation) with an IP-derived
 * fallback when permission is denied. If neither works, the subscription
 * still saves with no location — the user then receives every matching
 * alert nationwide (Option A).
 */
export function SubscriptionsPage() {
  const queryClient = useQueryClient()
  const { notify } = useToast()
  const loc = useLocation()

  const query = useQuery({
    queryKey: ['alert-subscription'],
    queryFn: () =>
      api.get<{ subscription: Subscription | null }>('/alerts/subscriptions'),
  })

  const [categories, setCategories] = useState<string[]>(['security'])
  const [channels, setChannels] = useState<string[]>(['in_app'])
  const [coords, setCoords] = useState<{ lat: number; lng: number } | null>(null)
  const [radius, setRadius] = useState(10)
  const [locationLabel, setLocationLabel] = useState('')
  const requestedRef = useRef(false)

  // Once we have browser coordinates, keep them.
  useEffect(() => {
    if (loc.latitude !== null && loc.longitude !== null) {
      setCoords({ lat: loc.latitude, lng: loc.longitude })
    }
  }, [loc.latitude, loc.longitude])

  // Ask for location once on mount (if the browser hasn't decided yet). If
  // permission is already denied or unavailable, try the IP fallback.
  useEffect(() => {
    if (requestedRef.current) return
    if (coords) return
    if (loc.loading) return

    if (loc.permission === 'prompt' || loc.permission === 'unavailable') {
      requestedRef.current = true
      loc.request()
    } else if (loc.permission === 'denied') {
      requestedRef.current = true
      void ipFallback().then((fb) => {
        if (fb) setCoords({ lat: fb.latitude, lng: fb.longitude })
      })
    }
  }, [loc, coords])

  // Populate from server once loaded.
  useEffect(() => {
    const sub = query.data?.subscription
    if (sub) {
      setCategories(sub.categories)
      setChannels(sub.channels.length > 0 ? sub.channels : ['in_app'])
      setRadius(sub.radius || 10)
      setLocationLabel(sub.location)
      if (sub.latitude !== 0 || sub.longitude !== 0) {
        setCoords({ lat: sub.latitude, lng: sub.longitude })
      }
    }
  }, [query.data])

  const save = useMutation({
    mutationFn: () =>
      api.post<{ subscription: Subscription }>('/alerts/subscribe', {
        categories,
        channels,
        latitude: coords?.lat ?? 0,
        longitude: coords?.lng ?? 0,
        location: locationLabel,
        radius,
      }),
    onSuccess: () => {
      notify('Subscription saved', 'success')
      void queryClient.invalidateQueries({ queryKey: ['alert-subscription'] })
    },
    onError: (err) => {
      notify(err instanceof ApiError ? err.message : 'Could not save subscription', 'error')
    },
  })

  const remove = useMutation({
    mutationFn: (id: string) => api.delete(`/alerts/subscriptions/${id}`),
    onSuccess: () => {
      notify('Subscription removed', 'success')
      void queryClient.invalidateQueries({ queryKey: ['alert-subscription'] })
    },
    onError: (err) => {
      notify(err instanceof ApiError ? err.message : 'Could not remove subscription', 'error')
    },
  })

  const current = query.data?.subscription ?? null
  const busy = save.isPending || remove.isPending

  function toggleCategory(value: string) {
    setCategories((prev) =>
      prev.includes(value) ? prev.filter((c) => c !== value) : [...prev, value],
    )
  }

  function toggleChannel(value: string) {
    setChannels((prev) =>
      prev.includes(value) ? prev.filter((c) => c !== value) : [...prev, value],
    )
  }

  return (
    <div className="mx-auto max-w-2xl p-4 sm:p-6">
      <header className="mb-6">
        <h1 className="text-2xl font-bold text-ink">Alert subscriptions</h1>
        <p className="mt-1 text-sm text-ink-muted">
          Choose which alerts you want to receive and how far from your location
          they should reach. Your location is detected automatically.
        </p>
      </header>

      {query.isLoading ? (
        <Card>
          <CardBody>
            <p className="text-sm text-ink-muted">Loading…</p>
          </CardBody>
        </Card>
      ) : query.isError ? (
        <Card>
          <CardBody>
            <p className="text-sm text-emergency">Could not load your subscription.</p>
            <Button onClick={() => void query.refetch()} className="mt-3">
              Try again
            </Button>
          </CardBody>
        </Card>
      ) : (
        <>
          <Card className="space-y-5 p-5">
            {/* Location status */}
            <section>
              <h2 className="text-sm font-semibold text-ink">Your location</h2>
              <div className="mt-2 flex items-start gap-2 rounded-lg border border-border bg-surface-hi/30 p-3">
                <MapPin
                  className={cn(
                    'mt-0.5 size-4 shrink-0',
                    coords ? 'text-signal' : 'text-ink-faint',
                  )}
                  aria-hidden
                />
                <div className="text-xs text-ink-muted">
                  {coords ? (
                    <>
                      Using your location (
                      {coords.lat.toFixed(3)}, {coords.lng.toFixed(3)}). Alerts
                      within the radius below will reach you.
                    </>
                  ) : loc.loading ? (
                    <>Detecting your location…</>
                  ) : (
                    <>
                      No location available. Without it, you will receive every
                      alert in your chosen categories, nationwide.
                    </>
                  )}
                </div>
                {!coords && !loc.loading ? (
                  <button
                    type="button"
                    onClick={() => loc.request()}
                    className="ml-auto shrink-0 text-xs text-signal hover:underline"
                  >
                    Try again
                  </button>
                ) : null}
              </div>
            </section>

            <section>
              <h2 className="text-sm font-semibold text-ink">Alert categories</h2>
              <p className="mt-1 text-xs text-ink-muted">
                Pick at least one type of alert you want to receive.
              </p>
              <div className="mt-3 grid gap-2 sm:grid-cols-2">
                {CATEGORIES.map((c) => {
                  const checked = categories.includes(c.value)
                  return (
                    <button
                      key={c.value}
                      type="button"
                      onClick={() => toggleCategory(c.value)}
                      className={cn(
                        'flex items-center gap-2 rounded-lg border px-3 py-2 text-left text-sm transition-colors',
                        checked
                          ? 'border-signal bg-signal/10 text-ink'
                          : 'border-border bg-surface-hi/30 text-ink-muted hover:bg-surface-hi/60',
                      )}
                    >
                      <span
                        className={cn(
                          'grid size-4 shrink-0 place-items-center rounded border',
                          checked ? 'border-signal bg-signal' : 'border-border',
                        )}
                      >
                        {checked ? <Check className="size-3 text-signal-ink" /> : null}
                      </span>
                      {c.label}
                    </button>
                  )
                })}
              </div>
            </section>

            <section>
              <h2 className="text-sm font-semibold text-ink">Delivery channels</h2>
              <p className="mt-1 text-xs text-ink-muted">
                In-app notifications are always available. Email and SMS will arrive
                in a future release.
              </p>
              <div className="mt-3 grid gap-2 sm:grid-cols-3">
                {CHANNELS.map((ch) => {
                  const checked = channels.includes(ch.value)
                  return (
                    <button
                      key={ch.value}
                      type="button"
                      onClick={() => ch.available && toggleChannel(ch.value)}
                      disabled={!ch.available}
                      className={cn(
                        'flex items-center gap-2 rounded-lg border px-3 py-2 text-left text-sm transition-colors',
                        !ch.available && 'cursor-not-allowed opacity-50',
                        checked
                          ? 'border-signal bg-signal/10 text-ink'
                          : 'border-border bg-surface-hi/30 text-ink-muted hover:bg-surface-hi/60',
                      )}
                    >
                      <span
                        className={cn(
                          'grid size-4 shrink-0 place-items-center rounded border',
                          checked ? 'border-signal bg-signal' : 'border-border',
                        )}
                      >
                        {checked ? <Check className="size-3 text-signal-ink" /> : null}
                      </span>
                      <span className="flex-1">
                        {ch.label}
                        {!ch.available ? (
                          <span className="ml-1 text-[10px] uppercase text-ink-faint">
                            soon
                          </span>
                        ) : null}
                      </span>
                    </button>
                  )
                })}
              </div>
            </section>

            <section>
              <h2 className="text-sm font-semibold text-ink">
                Radius: {radius} km
              </h2>
              <p className="mt-1 text-xs text-ink-muted">
                Only alerts whose location is within this distance from you will
                reach your device.
              </p>
              <input
                type="range"
                min={1}
                max={50}
                step={1}
                value={radius}
                onChange={(e) => setRadius(Number(e.target.value))}
                className="mt-3 w-full accent-signal"
              />
            </section>
          </Card>

          <div className="mt-4 flex flex-wrap items-center gap-3">
            <Button
              variant="primary"
              onClick={() => save.mutate()}
              disabled={busy || categories.length === 0}
              loading={save.isPending}
            >
              {current ? 'Update subscription' : 'Subscribe'}
            </Button>
            {current ? (
              <Button
                variant="ghost"
                icon={<Trash2 className="size-4" aria-hidden />}
                onClick={() => remove.mutate(current.id)}
                disabled={busy}
              >
                Remove
              </Button>
            ) : null}
            {current ? (
              <span className="ml-auto flex items-center gap-1 text-xs text-ink-faint">
                <Bell className="size-3" aria-hidden />
                Active since {new Date(current.createdAt).toLocaleDateString()}
              </span>
            ) : null}
          </div>

          {!current ? (
            <p className="mt-3 text-xs text-ink-muted">
              You currently have no active subscription. Choose your preferences
              above and click Subscribe.
            </p>
          ) : null}
        </>
      )}
    </div>
  )
}