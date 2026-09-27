import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Input, Textarea } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { api } from '@/lib/apiClient'
import { relativeTime, formatDateTime } from '@/lib/format'
import { USE_MOCKS } from '@/mocks/config'
import type { Announcement, CommunityEvent, CommunityPost } from '@/types/community'

type Tab = 'forum' | 'announcements' | 'events' | 'tips'

export function CommunityPage() {
  const [tab, setTab] = useState<Tab>('forum')
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [reply, setReply] = useState<Record<string, string>>({})
  const [openPost, setOpenPost] = useState<string | null>(null)
  const client = useQueryClient()
  const posts = useQuery({ queryKey: ['demo-posts'], queryFn: () => api.get<{ posts: CommunityPost[] }>('/community/posts'), enabled: USE_MOCKS && tab === 'forum' })
  const announcements = useQuery({ queryKey: ['demo-announcements'], queryFn: () => api.get<{ announcements: Announcement[] }>('/community/announcements'), enabled: USE_MOCKS && tab === 'announcements' })
  const events = useQuery({ queryKey: ['demo-events'], queryFn: () => api.get<{ events: CommunityEvent[] }>('/community/events'), enabled: USE_MOCKS && tab === 'events' })
  const create = useMutation({ mutationFn: () => api.post('/community/posts', { title: title.trim(), body: body.trim() }), onSuccess: () => { setTitle(''); setBody(''); void client.invalidateQueries({ queryKey: ['demo-posts'] }) } })
  const addReply = useMutation({ mutationFn: (postId: string) => api.post('/community/replies', { postId, body: reply[postId]?.trim() }), onSuccess: (_, postId) => { setReply((old) => ({ ...old, [postId]: '' })); void client.invalidateQueries({ queryKey: ['demo-posts'] }) } })
  const report = useMutation({ mutationFn: (postId: string) => api.post(`/community/posts/${postId}/report`, {}), onSuccess: () => void client.invalidateQueries({ queryKey: ['demo-posts'] }) })
  const rsvp = useMutation({ mutationFn: (eventId: string) => api.post(`/community/events/${eventId}/rsvp`, {}), onSuccess: () => void client.invalidateQueries({ queryKey: ['demo-events'] }) })

  return <div className="mx-auto max-w-3xl space-y-4 p-4 sm:p-6">
    <header><h1 className="text-xl font-semibold text-ink">Community</h1><p className="mt-1 text-sm text-ink-muted">Discuss local preparedness and share helpful information.</p></header>
    {!USE_MOCKS ? <Card className="p-5 text-sm text-ink-muted">Go has community routes, but this demo view expects different post and reply responses. Live moderation, including reporting content, needs a Go route before this view can be enabled.</Card> : <>
      <p className="rounded-lg border border-warn/40 bg-warn/10 p-3 text-xs text-ink">Demo community only. Posts, announcements and events are not public notices and reset when the page reloads. Reports of content are recorded in this demo only.</p>
      <nav aria-label="Community sections" className="flex flex-wrap gap-2">{(['forum', 'announcements', 'events', 'tips'] as const).map((item) => <Button key={item} size="sm" variant={tab === item ? 'primary' : 'secondary'} aria-pressed={tab === item} onClick={() => setTab(item)}>{item[0].toUpperCase() + item.slice(1)}</Button>)}</nav>

      {tab === 'forum' ? <section className="space-y-4" aria-label="Community forum">
        <Card className="space-y-3 p-4"><h2 className="text-sm font-semibold text-ink">Start a discussion</h2>
          <label className="block text-xs text-ink-muted">Title<Input maxLength={120} value={title} onChange={(e) => setTitle(e.target.value)} /></label>
          <label className="block text-xs text-ink-muted">Message<Textarea maxLength={2000} value={body} onChange={(e) => setBody(e.target.value)} /></label>
          <Button variant="primary" disabled={!title.trim() || !body.trim()} loading={create.isPending} onClick={() => create.mutate()}>Post to demo forum</Button>
          {create.isError ? <p role="alert" className="text-xs text-warn">Post failed. Your draft remains here.</p> : null}
        </Card>
        {posts.isLoading ? <Skeleton className="h-28 w-full" /> : posts.isError ? <Card><ErrorState title="Could not load posts" description="Try again." onRetry={() => void posts.refetch()} /></Card> : !posts.data?.posts.length ? <Card><EmptyState title="No discussions yet" description="Start the first demo discussion." /></Card> :
          <ul className="space-y-3">{posts.data.posts.map((post) => <li key={post.id}><Card className="p-4"><button type="button" aria-expanded={openPost === post.id} onClick={() => setOpenPost(openPost === post.id ? null : post.id)} className="w-full text-left"><h2 className="text-sm font-semibold text-ink">{post.title}</h2><p className="mt-1 text-xs text-ink-faint">{post.author} · {relativeTime(post.createdAt)} · {post.replies.length} replies</p></button><p className="mt-3 whitespace-pre-wrap text-sm text-ink-muted">{post.body}</p>
            {openPost === post.id ? <div className="mt-4 space-y-3 border-t border-border pt-3"><h3 className="text-xs font-semibold text-ink">Replies</h3>{post.replies.length ? <ul className="space-y-2">{post.replies.map((item) => <li key={item.id} className="rounded-lg bg-surface-hi p-3 text-xs text-ink"><p className="whitespace-pre-wrap">{item.body}</p><span className="text-ink-faint">{item.author} · {relativeTime(item.createdAt)}</span></li>)}</ul> : <p className="text-xs text-ink-muted">No replies yet.</p>}
              <label className="block text-xs text-ink-muted">Reply<Textarea maxLength={1000} value={reply[post.id] ?? ''} onChange={(e) => setReply((old) => ({ ...old, [post.id]: e.target.value }))} /></label><Button size="sm" disabled={!reply[post.id]?.trim()} loading={addReply.isPending} onClick={() => addReply.mutate(post.id)}>Reply</Button>{addReply.isError ? <p role="alert" className="text-xs text-warn">Reply failed.</p> : null}
            </div> : null}
            <div className="mt-3 border-t border-border pt-3"><Button size="sm" variant="ghost" disabled={post.reportedByMe || report.isPending} onClick={() => report.mutate(post.id)}>{post.reportedByMe ? 'Reported in demo' : 'Report content'}</Button>{report.isError ? <p role="alert" className="text-xs text-warn">Could not record the report.</p> : null}</div>
          </Card></li>)}</ul>}
      </section> : null}

      {tab === 'announcements' ? <section aria-label="Announcements">{announcements.isLoading ? <Skeleton className="h-24 w-full" /> : announcements.isError ? <Card><ErrorState title="Could not load announcements" description="Try again." onRetry={() => void announcements.refetch()} /></Card> : !announcements.data?.announcements.length ? <Card><EmptyState title="No announcements" description="Nothing has been posted." /></Card> : <ul className="space-y-3">{announcements.data.announcements.map((item) => <li key={item.id}><Card className="p-4"><h2 className="text-sm font-semibold text-ink">{item.title}</h2><p className="mt-2 text-sm text-ink-muted">{item.body}</p><p className="mt-2 text-xs text-ink-faint">{relativeTime(item.createdAt)}</p></Card></li>)}</ul>}</section> : null}

      {tab === 'events' ? <section aria-label="Events">{events.isLoading ? <Skeleton className="h-24 w-full" /> : events.isError ? <Card><ErrorState title="Could not load events" description="Try again." onRetry={() => void events.refetch()} /></Card> : !events.data?.events.length ? <Card><EmptyState title="No events" description="Nothing is scheduled in this demo." /></Card> : <ul className="space-y-3">{events.data.events.map((item) => <li key={item.id}><Card className="space-y-2 p-4"><h2 className="text-sm font-semibold text-ink">{item.title}</h2><p className="text-sm text-ink-muted">{item.description}</p><p className="text-xs text-ink-faint">{item.location} · {formatDateTime(item.startsAt)}</p><p className="text-xs text-ink-muted">{item.attendees} demo attendees</p><Button size="sm" loading={rsvp.isPending} onClick={() => rsvp.mutate(item.id)}>{item.attending ? 'Cancel demo RSVP' : 'RSVP in demo'}</Button>{rsvp.isError ? <p role="alert" className="text-xs text-warn">Could not update RSVP.</p> : null}</Card></li>)}</ul>}</section> : null}

      {tab === 'tips' ? <Card className="space-y-2 p-4"><h2 className="text-sm font-semibold text-ink">Preparedness tips</h2><p className="text-sm text-ink-muted">Keep a list of local emergency contacts where you can reach it. Agree on a meeting point with people you live with. This static demo copy is not AI generated or personalised.</p><p className="text-xs text-ink-faint">AI-assisted tips and caching are still to be built.</p></Card> : null}
      <p className="text-xs text-ink-faint">Need to report an incident? <Link to="/report" className="text-signal">Use the reporting flow</Link>.</p>
    </>}
  </div>
}
