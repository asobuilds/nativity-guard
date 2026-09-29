import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Input } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { api } from '@/lib/apiClient'
import { relativeTime } from '@/lib/format'
import { USE_MOCKS } from '@/mocks/config'
import type { CommunityAlert, NewsItem, Subscription } from '@/types/community'

const severityClass: Record<string, string> = {
  info: 'text-signal',
  caution: 'text-warn',
  urgent: 'text-emergency',
}

function DemoBanner({ text }: { text: string }) {
  if (!USE_MOCKS) return null
  return <p className="rounded-lg border border-warn/40 bg-warn/10 p-3 text-xs text-ink">{text}</p>
}

interface RawAlert {
  id?: string
  title?: string
  description?: string
  content?: string
  severity?: string
  location?: string
  createdAt?: string
  confirmations?: number
  confirmCount?: number
  confirmedByMe?: boolean
}

function normaliseAlert(raw: RawAlert): CommunityAlert {
  return {
    id: String(raw.id ?? ''),
    title: String(raw.title ?? ''),
    description: String(raw.description ?? raw.content ?? ''),
    severity: String(raw.severity ?? 'info').toLowerCase() as CommunityAlert['severity'],
    location: String(raw.location ?? ''),
    createdAt: String(raw.createdAt ?? ''),
    confirmations: Number(raw.confirmations ?? raw.confirmCount ?? 0),
    confirmedByMe: Boolean(raw.confirmedByMe ?? false),
  } as CommunityAlert
}

export function AlertsPage() {
  const result = useQuery({
    queryKey: ['alerts'],
    queryFn: async () => {
      const r = await api.get<{ alerts: RawAlert[] }>('/alerts')
      return { alerts: (r.alerts ?? []).map(normaliseAlert) }
    },
  })

  return (
    <div className="mx-auto max-w-3xl space-y-4 p-4 sm:p-6">
      <h1 className="text-xl font-semibold text-ink">Community alerts</h1>
      <DemoBanner text="Demo data only. Nothing here is a real alert or emergency instruction." />
      <div className="flex gap-3 text-sm">
        <Link className="text-signal" to="/news">News</Link>
        <Link className="text-signal" to="/subscriptions">Subscriptions</Link>
      </div>
      {result.isLoading ? (
        <Skeleton className="h-28 w-full" />
      ) : result.isError ? (
        <Card><ErrorState title="Could not load alerts" description="Try again." onRetry={() => void result.refetch()} /></Card>
      ) : !result.data?.alerts.length ? (
        <Card><EmptyState title="No alerts" description="There are no community alerts right now." /></Card>
      ) : (
        <ul className="space-y-3">
          {result.data.alerts.map((alert) => (
            <li key={alert.id}>
              <Link to={`/alerts/${alert.id}`} className="block rounded-panel focus-visible:outline-2 focus-visible:outline-signal">
                <Card className="p-4">
                  <p className={`text-xs font-semibold uppercase ${severityClass[alert.severity] ?? 'text-signal'}`}>{alert.severity}</p>
                  <h2 className="mt-1 text-sm font-semibold text-ink">{alert.title}</h2>
                  <p className="mt-1 text-xs text-ink-muted">{alert.location} · {relativeTime(alert.createdAt)}</p>
                </Card>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export function AlertDetailPage() {
  const { id } = useParams<{ id: string }>()
  const client = useQueryClient()
  const [shareError, setShareError] = useState('')

  const result = useQuery({
    queryKey: ['alert', id],
    queryFn: async () => {
      const r = await api.get<{ alert: RawAlert }>(`/alerts/${id}`)
      return { alert: normaliseAlert(r.alert) }
    },
    enabled: Boolean(id),
  })

  const confirm = useMutation({
    mutationFn: () => api.post(`/alerts/${id}/confirm`, {}),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['alert', id] })
      void client.invalidateQueries({ queryKey: ['alerts'] })
    },
  })

  async function share() {
    try {
      const url = window.location.href
      if (navigator.share) await navigator.share({ title: result.data?.alert.title, url })
      else if (navigator.clipboard) { await navigator.clipboard.writeText(url); setShareError('Link copied.') }
      else setShareError('Sharing is unavailable in this browser.')
    } catch {
      setShareError('Sharing was cancelled or unavailable.')
    }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-4 p-4 sm:p-6">
      <Link to="/alerts" className="text-sm text-signal">Back to alerts</Link>
      <DemoBanner text="Demo data only. Nothing here is a real alert or emergency instruction." />
      {result.isLoading ? (
        <Skeleton className="h-48 w-full" />
      ) : result.isError || !result.data ? (
        <Card><ErrorState title="Alert unavailable" description="This item could not be found." onRetry={() => void result.refetch()} /></Card>
      ) : (
        <Card className="space-y-3 p-5">
          <p className={`text-xs font-semibold uppercase ${severityClass[result.data.alert.severity] ?? 'text-signal'}`}>{result.data.alert.severity}</p>
          <h1 className="text-lg font-semibold text-ink">{result.data.alert.title}</h1>
          <p className="text-sm text-ink-muted">{result.data.alert.description}</p>
          <p className="text-xs text-ink-faint">{result.data.alert.location} · {relativeTime(result.data.alert.createdAt)}</p>
          <p className="text-xs text-ink-muted">
            {result.data.alert.confirmations} {result.data.alert.confirmations === 1 ? 'confirmation' : 'confirmations'}.
          </p>
          <div className="flex flex-wrap gap-2">
            <Button disabled={result.data.alert.confirmedByMe} loading={confirm.isPending} onClick={() => confirm.mutate()}>
              {result.data.alert.confirmedByMe ? 'Confirmed' : 'Confirm I saw this'}
            </Button>
            <Button onClick={() => void share()}>Share</Button>
          </div>
          {confirm.isError ? <p role="alert" className="text-xs text-warn">Confirmation failed. Try again.</p> : null}
          {shareError ? <p role="status" className="text-xs text-ink-muted">{shareError}</p> : null}
        </Card>
      )}
    </div>
  )
}

export function NewsPage() {
  const result = useQuery({
    queryKey: ['news', USE_MOCKS ? 'demo' : 'go'],
    queryFn: async (): Promise<{ news: NewsItem[] }> => {
      if (USE_MOCKS) return api.get<{ news: NewsItem[] }>('/news')
      const response = await api.get<{ news: { id: string; title: string; content: string; category: string; publishedAt: string }[] }>('/news')
      return {
        news: response.news.map((item) => ({
          id: item.id,
          title: item.title,
          summary: item.content,
          kind: 'news',
          publishedAt: item.publishedAt,
        })),
      }
    },
  })

  return (
    <div className="mx-auto max-w-3xl space-y-4 p-4 sm:p-6">
      <h1 className="text-xl font-semibold text-ink">News and updates</h1>
      <DemoBanner text="Demo news only. Nothing here is a live announcement." />
      <Link to="/alerts" className="text-sm text-signal">View alerts</Link>
      {result.isLoading ? (
        <Skeleton className="h-28 w-full" />
      ) : result.isError ? (
        <Card><ErrorState title="Could not load news" description="Try again." onRetry={() => void result.refetch()} /></Card>
      ) : !result.data?.news.length ? (
        <Card><EmptyState title="No news yet" description={USE_MOCKS ? 'Nothing has been published in this demo.' : 'No published news is available.'} /></Card>
      ) : (
        <ul className="space-y-3">
          {result.data.news.map((item) => (
            <li key={item.id}>
              <Card className="p-4">
                <span className="text-xs uppercase text-signal">{item.kind}</span>
                <h2 className="mt-1 text-sm font-semibold text-ink">{item.title}</h2>
                <p className="mt-2 whitespace-pre-wrap text-sm text-ink-muted">{item.summary}</p>
                <p className="mt-2 text-xs text-ink-faint">{relativeTime(item.publishedAt)}</p>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export function SubscriptionsPage() {
  const client = useQueryClient()
  const result = useQuery({
    queryKey: ['subscriptions'],
    queryFn: () => api.get<{ subscription: Subscription }>('/alerts/subscriptions'),
  })
  const [areas, setAreas] = useState<string | null>(null)
  const [categories, setCategories] = useState<string[] | null>(null)
  const [channels, setChannels] = useState<string[] | null>(null)
  const [permission, setPermission] = useState('')

  const save = useMutation({
    mutationFn: (subscription: Pick<Subscription, 'areas' | 'categories' | 'channels'>) => api.post('/alerts/subscribe', subscription),
    onSuccess: () => { void client.invalidateQueries({ queryKey: ['subscriptions'] }) },
  })

  const device = useMutation({
    mutationFn: (enabled: boolean) => enabled ? api.post('/notifications/register', {}) : api.delete('/notifications/unregister'),
    onSuccess: () => { void client.invalidateQueries({ queryKey: ['subscriptions'] }) },
  })

  const current = result.data?.subscription
  const selectedCategories = categories ?? current?.categories ?? []
  const selectedChannels = channels ?? current?.channels ?? ['in-app']

  async function optIn() {
    if (!('Notification' in window)) { setPermission('This browser does not support notifications.'); return }
    const answer = await Notification.requestPermission()
    if (answer !== 'granted') { setPermission('Permission was not granted; nothing was registered.'); return }
    device.mutate(true)
  }

  return (
    <div className="mx-auto max-w-3xl space-y-4 p-4 sm:p-6">
      <h1 className="text-xl font-semibold text-ink">Alert subscriptions</h1>
      <DemoBanner text="Demo data only. Nothing here is a real subscription." />
      {result.isLoading ? (
        <Skeleton className="h-40 w-full" />
      ) : result.isError || !current ? (
        <Card><ErrorState title="Subscriptions unavailable" description="Try again." onRetry={() => void result.refetch()} /></Card>
      ) : (
        <Card className="space-y-4 p-5">
          <p className="text-sm text-ink-muted">Choose what you would like to follow.</p>
          <label className="block text-xs text-ink-muted">
            Areas (comma separated)
            <Input value={areas ?? current.areas.join(', ')} onChange={(e) => setAreas(e.target.value)} placeholder="Your area" />
          </label>
          <fieldset>
            <legend className="text-xs text-ink-muted">Categories</legend>
            <div className="mt-2 flex flex-wrap gap-3">
              {['safety', 'community', 'weather'].map((category) => (
                <label key={category} className="text-sm text-ink">
                  <input
                    type="checkbox"
                    checked={selectedCategories.includes(category)}
                    onChange={() =>
                      setCategories(
                        selectedCategories.includes(category)
                          ? selectedCategories.filter((v) => v !== category)
                          : [...selectedCategories, category],
                      )
                    }
                  /> {category}
                </label>
              ))}
            </div>
          </fieldset>
          <fieldset>
            <legend className="text-xs text-ink-muted">Channels</legend>
            <div className="mt-2 flex flex-wrap gap-3">
              {['in-app', 'email', 'sms'].map((channel) => (
                <label key={channel} className="text-sm text-ink">
                  <input
                    type="checkbox"
                    checked={selectedChannels.includes(channel)}
                    onChange={() =>
                      setChannels(
                        selectedChannels.includes(channel)
                          ? selectedChannels.filter((v) => v !== channel)
                          : [...selectedChannels, channel],
                      )
                    }
                  /> {channel}
                </label>
              ))}
            </div>
          </fieldset>
          <Button
            variant="primary"
            loading={save.isPending}
            onClick={() =>
              save.mutate({
                areas: (areas ?? current.areas.join(',')).split(',').map((v) => v.trim()).filter(Boolean),
                categories: selectedCategories,
                channels: selectedChannels,
              })
            }
          >
            Save preferences
          </Button>
          {save.isSuccess ? <p role="status" className="text-xs text-ok">Saved.</p> : null}
          {save.isError ? <p role="alert" className="text-xs text-warn">Could not save preferences.</p> : null}
          <div className="border-t border-border pt-4">
            <p className="text-sm font-medium text-ink">Browser notification permission</p>
            <p className="mt-1 text-xs text-ink-muted">Optional. Register this device to receive push notifications.</p>
            <Button
              className="mt-3"
              loading={device.isPending}
              onClick={() => void (current.deviceRegistered ? device.mutate(false) : optIn())}
            >
              {current.deviceRegistered ? 'Remove registration' : 'Allow notification registration'}
            </Button>
            {permission ? <p role="status" className="mt-2 text-xs text-ink-muted">{permission}</p> : null}
            {device.isError ? <p role="alert" className="mt-2 text-xs text-warn">Registration failed.</p> : null}
          </div>
        </Card>
      )}
    </div>
  )
}