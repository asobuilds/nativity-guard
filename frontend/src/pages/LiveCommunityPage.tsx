import { useState, type FormEvent } from 'react'
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '@/auth/AuthContext'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Field, Input, Textarea } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { ScopeSelector, scopeLabel, type CommunityScope } from '@/components/community/ScopeSelector'
import { api, ApiError } from '@/lib/apiClient'
import { formatDateTime, relativeTime } from '@/lib/format'
import { useLocation } from '@/hooks/useLocation'

type Tab = 'forum' | 'announcements' | 'events' | 'moderation'
type Place = { scope?: string; locality?: string; lga?: string; state?: string; country?: string }
type Post = {
  id: string
  title: string
  content: string
  createdAt: string
  author: { name: string }
  replyCount: number
  isLocked?: boolean
  scope?: string
  place?: Place
  /** The caller filed a report that is still live. */
  reportedByMe?: boolean
  /** The caller wrote this post, so reporting it is meaningless. */
  isMine?: boolean
  replies?: Reply[]
}
type Reply = { id: string; content: string; author: { name: string }; createdAt: string }
type Announcement = { id: string; title: string; content: string; publishedAt: string }
type Event = { id: string; title: string; description: string; location: string; eventDate: string; attendeeCount: number }
type FeedPage = { posts: Post[]; nextCursor?: string }
type ModerationReport = { id: string; reason: string; status: string; createdAt: string; post: Post }

const errorText = (error: unknown) => error instanceof ApiError ? error.message : 'Could not save. Please check your connection and retry.'

/** A short, human label for where a post is addressed, e.g. "Ikeja LGA". */
function placeLabel(post: Post): string {
  const place = post.place
  const scope = (post.scope ?? place?.scope ?? 'world') as CommunityScope
  switch (scope) {
    case 'locality':
      return place?.locality ? `${place.locality} village` : scopeLabel('locality')
    case 'lga':
      return place?.lga ? `${place.lga} LGA` : scopeLabel('lga')
    case 'state':
      return place?.state || scopeLabel('state')
    case 'country':
      return place?.country || scopeLabel('country')
    default:
      return scopeLabel('world')
  }
}

/**
 * The report affordance for a single post.
 *
 * Reports are only accepted for posts you can see (enforced server-side by the
 * same scope rule as the feed), and reporting is idempotent, so the settled state
 * is a plain label rather than a disabled button that invites a retry.
 */
function PostReportAction({ post, pending, onReport }: { post: Post; pending: boolean; onReport: (id: string) => void }) {
  if (post.isMine) return null
  if (post.reportedByMe) return <span className="text-xs text-ink-faint">Reported — awaiting review</span>
  return (
    <Button size="sm" variant="ghost" disabled={pending} onClick={() => onReport(post.id)}>
      Report content
    </Button>
  )
}

/**
 * The community feed.
 *
 * Geography is the boundary: the server scopes every read to the caller's own
 * place and refuses replies and reports outside it, so this page only chooses how
 * wide within that area to look (`scope`). Coordinates are attached when the
 * browser has them, so a citizen — who has no unit — still resolves a place.
 * Writes are live; there is no build flag gating them (decision D-016).
 *
 * The feed pages with a keyset cursor (`nextCursor`), because offset paging
 * skips and repeats posts while newer ones are being written.
 */
export function LiveCommunityPage() {
  const { role } = useAuth()
  const client = useQueryClient()
  const geo = useLocation()
  const [tab, setTab] = useState<Tab>('forum')
  const [scope, setScope] = useState<CommunityScope>('lga')
  const [composeScope, setComposeScope] = useState<CommunityScope>('lga')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [reply, setReply] = useState('')
  const [eventDescription, setEventDescription] = useState('')
  const [eventLocation, setEventLocation] = useState('')
  const [eventDate, setEventDate] = useState('')
  const [eventEndDate, setEventEndDate] = useState('')
  const [rsvped, setRsvped] = useState<string[]>([])

  const hasFix = geo.latitude != null && geo.longitude != null
  const coordQuery = hasFix ? `lat=${geo.latitude}&lng=${geo.longitude}` : ''
  const coordBody = hasFix ? { latitude: geo.latitude as number, longitude: geo.longitude as number } : {}
  const canPublish = role === 'unit_admin' || role === 'super_admin'
  const canModerate = role === 'unit_admin' || role === 'super_admin'

  const posts = useInfiniteQuery({
    queryKey: ['community', 'posts', scope, coordQuery],
    initialPageParam: '',
    queryFn: ({ pageParam }) => {
      const params = new URLSearchParams({ scope, limit: '20' })
      if (pageParam) params.set('cursor', pageParam)
      if (coordQuery) {
        params.set('lat', String(geo.latitude))
        params.set('lng', String(geo.longitude))
      }
      return api.get<FeedPage>(`/community/posts?${params.toString()}`)
    },
    getNextPageParam: (lastPage) => lastPage.nextCursor || undefined,
    enabled: tab === 'forum',
  })
  const detail = useQuery({
    queryKey: ['community', 'post', selectedId, coordQuery],
    queryFn: () => api.get<{ post: Post }>(`/community/posts/${selectedId}${coordQuery ? `?${coordQuery}` : ''}`),
    enabled: tab === 'forum' && !!selectedId,
  })
  const announcements = useQuery({ queryKey: ['community', 'announcements'], queryFn: () => api.get<{ announcements: Announcement[] }>('/community/announcements'), enabled: tab === 'announcements' })
  const events = useQuery({ queryKey: ['community', 'events'], queryFn: () => api.get<{ events: Event[] }>('/community/events'), enabled: tab === 'events' })
  const reports = useQuery({
    queryKey: ['community', 'reports'],
    queryFn: () => api.get<{ reports: ModerationReport[]; count: number }>('/community/reports?status=pending'),
    enabled: tab === 'moderation' && canModerate,
  })

  const createPost = useMutation({ mutationFn: () => api.post('/community/posts', { title: title.trim(), content: content.trim(), scope: composeScope, ...coordBody }), onSuccess: () => { setTitle(''); setContent(''); void client.invalidateQueries({ queryKey: ['community', 'posts'] }) } })
  const createReply = useMutation({ mutationFn: () => api.post('/community/replies', { postId: selectedId, content: reply.trim(), ...coordBody }), onSuccess: () => { setReply(''); void client.invalidateQueries({ queryKey: ['community', 'post', selectedId] }); void client.invalidateQueries({ queryKey: ['community', 'posts'] }) } })
  const createAnnouncement = useMutation({ mutationFn: () => api.post('/community/announcements', { title: title.trim(), content: content.trim() }), onSuccess: () => { setTitle(''); setContent(''); void client.invalidateQueries({ queryKey: ['community', 'announcements'] }) } })
  const createEvent = useMutation({ mutationFn: () => api.post('/community/events', { title: title.trim(), description: eventDescription.trim(), location: eventLocation.trim(), eventDate: new Date(eventDate).toISOString(), endDate: new Date(eventEndDate).toISOString(), type: 'meeting' }), onSuccess: () => { setTitle(''); setEventDescription(''); setEventLocation(''); setEventDate(''); setEventEndDate(''); void client.invalidateQueries({ queryKey: ['community', 'events'] }) } })
  const rsvp = useMutation({ mutationFn: (id: string) => api.post(`/community/events/${id}/rsvp`, {}), onSuccess: (_, id) => { setRsvped(current => [...current, id]); void client.invalidateQueries({ queryKey: ['community', 'events'] }) } })
  const reportPost = useMutation({
    mutationFn: (postId: string) => api.post(`/community/posts/${postId}/report`, {}),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['community', 'posts'] })
      void client.invalidateQueries({ queryKey: ['community', 'post'] })
      void client.invalidateQueries({ queryKey: ['community', 'reports'] })
    },
  })
  const resolveReport = useMutation({
    mutationFn: (input: { id: string; status: 'reviewed' | 'dismissed' }) => api.post(`/community/reports/${input.id}/resolve`, { status: input.status }),
    onSuccess: () => void client.invalidateQueries({ queryKey: ['community', 'reports'] }),
  })

  function submit(event: FormEvent, mutation: { mutate: () => void }) { event.preventDefault(); mutation.mutate() }

  const tabs: Tab[] = canModerate ? ['forum', 'announcements', 'events', 'moderation'] : ['forum', 'announcements', 'events']
  const feed = posts.data?.pages.flatMap(page => page.posts) ?? []

  return <div className="mx-auto max-w-3xl space-y-4 p-4 sm:p-6">
    <header>
      <h1 className="text-xl font-semibold text-ink">Community</h1>
      <p className="mt-1 text-sm text-ink-muted">Local preparedness, unit announcements and events — scoped to where you are.</p>
    </header>

    <nav aria-label="Community sections" className="flex flex-wrap gap-2">{tabs.map(item => <Button key={item} size="sm" variant={tab === item ? 'primary' : 'secondary'} aria-pressed={tab === item} onClick={() => { setTab(item); setSelectedId(null); setTitle(''); setContent('') }}>{item === 'forum' ? 'Discussions' : item[0].toUpperCase() + item.slice(1)}</Button>)}</nav>

    {tab === 'forum' && <section aria-label="Community discussions" className="space-y-4">
      <Card className="space-y-2 p-4">
        <p className="text-xs font-medium text-ink-muted">Show discussions from</p>
        <ScopeSelector value={scope} onChange={setScope} label="Feed area" />
        <p className="text-xs text-ink-faint">{hasFix ? 'Using your current location.' : 'Enable location so posts near you can be matched; without it you only see national and world discussions.'}</p>
      </Card>

      <Card className="p-4">
        <form className="space-y-3" onSubmit={event => submit(event, createPost)}>
          <h2 className="font-semibold text-ink">Start a discussion</h2>
          <Field label="Title" required>{props => <Input {...props} maxLength={120} required value={title} onChange={e => setTitle(e.target.value)} />}</Field>
          <Field label="Message" required>{props => <Textarea {...props} maxLength={2000} required value={content} onChange={e => setContent(e.target.value)} />}</Field>
          <div className="space-y-2">
            <p className="text-xs font-medium text-ink-muted">Who can see this</p>
            <ScopeSelector value={composeScope} onChange={setComposeScope} label="Post audience" />
            <p className="text-xs text-ink-faint">Only people in the chosen area can read, reply and report. Posting to an area needs a location — share yours if publishing fails.</p>
          </div>
          <Button variant="primary" type="submit" disabled={!title.trim() || !content.trim()} loading={createPost.isPending}>Publish post</Button>
          {createPost.isError && <p role="alert" className="text-sm text-emergency">{errorText(createPost.error)}</p>}
        </form>
      </Card>

      {selectedId ? <>
        <Button size="sm" onClick={() => setSelectedId(null)}>Back to discussions</Button>
        {detail.isLoading
          ? <Skeleton className="h-36 w-full" />
          : detail.isError
            ? <Card><ErrorState title="Could not load discussion" description="It may be outside your area, or the connection dropped." onRetry={() => void detail.refetch()} /></Card>
            : detail.data && <Card className="space-y-3 p-4">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="font-semibold text-ink">{detail.data.post.title}</h2>
                <span className="rounded-full bg-surface-hi px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-ink-muted ring-1 ring-border">{placeLabel(detail.data.post)}</span>
              </div>
              <p className="text-xs text-ink-faint">{detail.data.post.author?.name ?? 'Community member'} · {relativeTime(detail.data.post.createdAt)}</p>
              <p className="whitespace-pre-wrap text-sm text-ink-muted">{detail.data.post.content}</p>
              <PostReportAction post={detail.data.post} pending={reportPost.isPending} onReport={id => reportPost.mutate(id)} />
              {reportPost.isError && <p role="alert" className="text-sm text-emergency">{errorText(reportPost.error)}</p>}
              <h3 className="border-t border-border pt-3 text-sm font-semibold text-ink">Replies</h3>
              {detail.data.post.replies?.length
                ? <ul className="space-y-3">{detail.data.post.replies.map(item => <li key={item.id} className="rounded-lg bg-surface-hi p-3"><p className="whitespace-pre-wrap text-sm text-ink">{item.content}</p><p className="text-xs text-ink-faint">{item.author?.name ?? 'Community member'} · {relativeTime(item.createdAt)}</p></li>)}</ul>
                : <p className="text-sm text-ink-muted">No replies yet.</p>}
              {!detail.data.post.isLocked
                ? <form className="space-y-3" onSubmit={event => submit(event, createReply)}>
                  <Field label="Your reply" required>{props => <Textarea {...props} maxLength={1000} required value={reply} onChange={e => setReply(e.target.value)} />}</Field>
                  <Button type="submit" variant="primary" disabled={!reply.trim()} loading={createReply.isPending}>Reply</Button>
                  {createReply.isError && <p role="alert" className="text-sm text-emergency">{errorText(createReply.error)}</p>}
                </form>
                : <p className="text-sm text-ink-muted">This discussion is locked.</p>}
            </Card>}
      </> : posts.isLoading
        ? <Skeleton className="h-28 w-full" />
        : posts.isError
          ? <Card><ErrorState title="Could not load discussions" description="Try again." onRetry={() => void posts.refetch()} /></Card>
          : feed.length
            ? <div className="space-y-3">
              <ul className="space-y-3">{feed.map(post => <li key={post.id}><Card className="p-4"><button className="w-full text-left" onClick={() => setSelectedId(post.id)}><div className="flex flex-wrap items-center gap-2"><h2 className="font-semibold text-ink">{post.title}</h2><span className="rounded-full bg-surface-hi px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-ink-muted ring-1 ring-border">{placeLabel(post)}</span></div><p className="text-xs text-ink-faint">{post.author?.name ?? 'Community member'} · {relativeTime(post.createdAt)} · {post.replyCount} replies</p><p className="mt-2 whitespace-pre-wrap text-sm text-ink-muted">{post.content}</p><span className="mt-2 inline-block text-sm text-signal">Read discussion</span></button>{!post.isMine && <div className="mt-3 border-t border-border pt-2"><PostReportAction post={post} pending={reportPost.isPending} onReport={id => reportPost.mutate(id)} /></div>}</Card></li>)}</ul>
              {posts.hasNextPage && <Button variant="secondary" loading={posts.isFetchingNextPage} onClick={() => void posts.fetchNextPage()}>Load older discussions</Button>}
              {reportPost.isError && <p role="alert" className="text-sm text-emergency">{errorText(reportPost.error)}</p>}
            </div>
            : <Card><EmptyState title="No discussions in this area yet" description="Try widening the area above, or start the first one." /></Card>}
    </section>}

    {tab === 'announcements' && <section aria-label="Announcements" className="space-y-4">{canPublish && <Card className="p-4"><form className="space-y-3" onSubmit={event => submit(event, createAnnouncement)}><h2 className="font-semibold text-ink">Publish announcement</h2><Field label="Title" required>{props => <Input {...props} required maxLength={120} value={title} onChange={e => setTitle(e.target.value)} />}</Field><Field label="Message" required>{props => <Textarea {...props} required maxLength={2000} value={content} onChange={e => setContent(e.target.value)} />}</Field><Button type="submit" variant="primary" disabled={!title.trim() || !content.trim()} loading={createAnnouncement.isPending}>Publish</Button>{createAnnouncement.isError && <p role="alert" className="text-sm text-emergency">{errorText(createAnnouncement.error)}</p>}</form></Card>}{announcements.isLoading ? <Skeleton className="h-24 w-full" /> : announcements.isError ? <Card><ErrorState title="Could not load announcements" description="Try again." onRetry={() => void announcements.refetch()} /></Card> : announcements.data?.announcements.length ? <ul className="space-y-3">{announcements.data.announcements.map(item => <li key={item.id}><Card className="p-4"><h2 className="font-semibold text-ink">{item.title}</h2><p className="mt-2 whitespace-pre-wrap text-sm text-ink-muted">{item.content}</p><p className="text-xs text-ink-faint">{relativeTime(item.publishedAt)}</p></Card></li>)}</ul> : <Card><EmptyState title="No announcements" description="Nothing has been posted." /></Card>}</section>}

    {tab === 'events' && <section aria-label="Events" className="space-y-4">{canPublish && <Card className="p-4"><form className="space-y-3" onSubmit={event => submit(event, createEvent)}><h2 className="font-semibold text-ink">Create event</h2><Field label="Title" required>{props => <Input {...props} required value={title} onChange={e => setTitle(e.target.value)} />}</Field><Field label="Description" required>{props => <Textarea {...props} required value={eventDescription} onChange={e => setEventDescription(e.target.value)} />}</Field><Field label="Location" required>{props => <Input {...props} required value={eventLocation} onChange={e => setEventLocation(e.target.value)} />}</Field><Field label="Start time" required>{props => <Input {...props} required type="datetime-local" value={eventDate} onChange={e => setEventDate(e.target.value)} />}</Field><Field label="End time" required error={eventEndDate && eventDate && new Date(eventEndDate) <= new Date(eventDate) ? 'End time must be after the start.' : undefined}>{props => <Input {...props} required type="datetime-local" min={eventDate || undefined} value={eventEndDate} onChange={e => setEventEndDate(e.target.value)} />}</Field><Button type="submit" variant="primary" disabled={!title.trim() || !eventDescription.trim() || !eventLocation.trim() || !eventDate || !eventEndDate || new Date(eventEndDate) <= new Date(eventDate)} loading={createEvent.isPending}>Create event</Button>{createEvent.isError && <p role="alert" className="text-sm text-emergency">{errorText(createEvent.error)}</p>}</form></Card>}{events.isLoading ? <Skeleton className="h-24 w-full" /> : events.isError ? <Card><ErrorState title="Could not load events" description="Try again." onRetry={() => void events.refetch()} /></Card> : events.data?.events.length ? <ul className="space-y-3">{events.data.events.map(item => <li key={item.id}><Card className="space-y-2 p-4"><h2 className="font-semibold text-ink">{item.title}</h2><p className="text-sm text-ink-muted">{item.description}</p><p className="text-xs text-ink-faint">{item.location} · {formatDateTime(item.eventDate)} · {item.attendeeCount} attending</p><Button size="sm" disabled={rsvped.includes(item.id)} loading={rsvp.isPending && rsvp.variables === item.id} onClick={() => rsvp.mutate(item.id)}>{rsvped.includes(item.id) ? 'RSVP recorded' : 'RSVP'}</Button>{rsvp.isError && rsvp.variables === item.id && <p role="alert" className="text-sm text-emergency">{errorText(rsvp.error)}</p>}</Card></li>)}</ul> : <Card><EmptyState title="No upcoming events" description="Check back later." /></Card>}</section>}

    {tab === 'moderation' && <section aria-label="Moderation queue" className="space-y-4">
      <Card className="p-4">
        <h2 className="font-semibold text-ink">Reported content</h2>
        <p className="mt-1 text-xs text-ink-muted">Oldest first. You see reports for content you are responsible for: the whole platform as a super admin, or your own area as a unit admin. Resolving a report records the decision; it does not remove the post.</p>
      </Card>
      {reports.isLoading ? <Skeleton className="h-24 w-full" />
        : reports.isError ? <Card><ErrorState title="Could not load reports" description="Try again." onRetry={() => void reports.refetch()} /></Card>
        : reports.data?.reports.length
          ? <ul className="space-y-3">{reports.data.reports.map(report => <li key={report.id}><Card className="space-y-2 p-4">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="font-semibold text-ink">{report.post.title}</h3>
              <span className="rounded-full bg-surface-hi px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-ink-muted ring-1 ring-border">{placeLabel(report.post)}</span>
            </div>
            <p className="text-xs text-ink-faint">{report.post.author?.name ?? 'Community member'} · reported {relativeTime(report.createdAt)}</p>
            <p className="whitespace-pre-wrap text-sm text-ink-muted">{report.post.content}</p>
            <p className="text-xs text-ink-muted"><span className="font-semibold">Reason:</span> {report.reason}</p>
            <div className="flex flex-wrap gap-2">
              <Button size="sm" variant="secondary" disabled={resolveReport.isPending} onClick={() => resolveReport.mutate({ id: report.id, status: 'dismissed' })}>Dismiss</Button>
              <Button size="sm" variant="primary" disabled={resolveReport.isPending} onClick={() => resolveReport.mutate({ id: report.id, status: 'reviewed' })}>Mark reviewed</Button>
            </div>
          </Card></li>)}</ul>
          : <Card><EmptyState title="No pending reports" description="Nothing needs moderation right now." /></Card>}
      {resolveReport.isError && <p role="alert" className="text-sm text-emergency">{errorText(resolveReport.error)}</p>}
    </section>}
  </div>
}
