import { useState } from 'react'
import { useAuth } from '@/auth/AuthContext'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { MessageCircle, Send, ShieldAlert, UserRoundX } from 'lucide-react'
import { useLocation } from '@/hooks/useLocation'
import { api } from '@/lib/apiClient'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Input, Select } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'

type Room = { id: string; scope: string; locality?: string; lga?: string; state?: string }
type Message = { id: string; roomId: string; senderId: string; sender: string; content: string; createdAt: string }
type ChatPage = { messages: Message[]; nextCursor: string; hasMore: boolean }

export function CommunityChatPage() {
  const location = useLocation({ auto: false })
  const { user, role } = useAuth()
  const canModerate = role === 'unit_admin' || role === 'super_admin'
  const queryClient = useQueryClient()
  const [scope, setScope] = useState('lga')
  const [room, setRoom] = useState<Room | null>(null)
  const [draft, setDraft] = useState('')
  const [reportId, setReportId] = useState('')
  const [reportReason, setReportReason] = useState('')
  const join = useMutation({
    mutationFn: () => api.post<{ room: Room }>('/community/chat/rooms/join', {
      scope,
      ...(location.latitude != null && location.longitude != null ? { latitude: location.latitude, longitude: location.longitude } : {}),
    }),
    onSuccess: data => { setRoom(data.room); void queryClient.invalidateQueries({ queryKey: ['citizen-chat', data.room.id] }) },
  })
  const messages = useQuery({
    queryKey: ['citizen-chat', room?.id],
    queryFn: () => api.get<ChatPage>(`/community/chat/rooms/${room?.id}/messages?limit=50`),
    enabled: Boolean(room?.id),
    refetchInterval: 5000,
  })
  const send = useMutation({
    mutationFn: () => api.post(`/community/chat/rooms/${room?.id}/messages`, { content: draft.trim() }),
    onSuccess: () => { setDraft(''); void queryClient.invalidateQueries({ queryKey: ['citizen-chat', room?.id] }) },
  })
  const report = useMutation({
    mutationFn: () => api.post(`/community/chat/messages/${reportId}/report`, { reason: reportReason.trim() }),
    onSuccess: () => { setReportId(''); setReportReason('') },
  })
  const block = useMutation({
    mutationFn: (userId: string) => api.post('/community/chat/blocks', { userId }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['citizen-chat', room?.id] }),
  })

  const moderation = useQuery({
    queryKey: ['citizen-chat', 'moderation'],
    queryFn: () => api.get<{ reports: Array<{ id: string; reason: string; createdAt: string; message: Message }> }>('/community/chat/reports?status=pending'),
    enabled: canModerate,
    refetchInterval: 15000,
  })
  const resolveReport = useMutation({
    mutationFn: (input: { id: string; status: 'reviewed' | 'dismissed'; action?: 'hide' }) =>
      api.post(`/community/chat/reports/${input.id}/resolve`, { status: input.status, action: input.action }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['citizen-chat', 'moderation'] }),
  })

  return (
    <main className="mx-auto w-full max-w-4xl space-y-4 p-4 sm:p-6">
      <header>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-ink"><MessageCircle className="size-6 text-signal" /> Community chat</h1>
        <p className="mt-1 text-sm text-ink-muted">Talk with people in your own locality, LGA or state. Your area is resolved by the server; changing the scope never moves the boundary to another place.</p>
      </header>
      {!room ? (
        <Card className="space-y-3 p-4">
          <label className="block text-xs text-ink-muted">Chat area
            <Select value={scope} onChange={event => setScope(event.target.value)}>
              <option value="locality">My locality</option>
              <option value="lga">My LGA</option>
              <option value="state">My state</option>
            </Select>
          </label>
          <Button loading={join.isPending} disabled={join.isPending} onClick={() => join.mutate()}>Join my local chat</Button>
          {join.isError ? (
            <div className="space-y-2">
              <p role="alert" className="text-sm text-danger">We could not resolve your area. Use your device location or make sure your verified unit has a current area.</p>
              <Button size="sm" variant="secondary" disabled={location.loading} onClick={() => location.request()}>
                {location.loading ? 'Getting location…' : 'Share my location'}
              </Button>
              {location.error ? <p className="text-xs text-ink-faint">{location.error}</p> : null}
              {location.latitude != null && location.longitude != null ? <p className="text-xs text-ok">Location ready. Select Join my local chat again.</p> : null}
            </div>
          ) : null}
        </Card>
      ) : (
        <Card className="flex min-h-[540px] flex-col p-3 sm:p-4">
          <div className="border-b border-border pb-3">
            <p className="font-semibold text-ink">{room.locality || room.lga || room.state} community</p>
            <p className="text-xs text-ink-faint">Visible to members resolved in this {room.scope} boundary</p>
          </div>
          <div className="flex-1 space-y-3 overflow-y-auto py-4">
            {messages.isLoading ? <Skeleton className="h-24 w-full" /> : messages.isError ? <ErrorState title="Could not load chat" description="Try again." onRetry={() => void messages.refetch()} /> : messages.data?.messages.length ? messages.data.messages.slice().reverse().map(message => (
              <div key={message.id} className="rounded-xl bg-surface-hi p-3">
                <div className="flex items-center justify-between gap-2">
                  <p className="text-xs font-semibold text-ink">{message.sender || 'Community member'}</p>
                  <time className="text-[10px] text-ink-faint">{new Date(message.createdAt).toLocaleString()}</time>
                </div>
                <p className="mt-2 whitespace-pre-wrap break-words text-sm text-ink">{message.content}</p>
                {message.senderId !== user?.id ? (
                  <div className="mt-2 flex flex-wrap gap-2">
                    <Button size="sm" variant="ghost" onClick={() => setReportId(message.id)}><ShieldAlert className="mr-1 size-3" /> Report</Button>
                    <Button size="sm" variant="ghost" onClick={() => block.mutate(message.senderId)}><UserRoundX className="mr-1 size-3" /> Block</Button>
                  </div>
                ) : null}
              </div>
            )) : <EmptyState title="A fresh local conversation" description="Start by sharing a helpful, respectful message with people in your area." />}
          </div>
          {reportId ? (
            <div className="space-y-2 border-t border-border py-3">
              <Input value={reportReason} onChange={event => setReportReason(event.target.value)} placeholder="Why should moderators review this message?" aria-label="Report reason" />
              <div className="flex gap-2">
                <Button size="sm" disabled={!reportReason.trim() || report.isPending} loading={report.isPending} onClick={() => report.mutate()}>Submit report</Button>
                <Button size="sm" variant="ghost" onClick={() => { setReportId(''); setReportReason('') }}>Cancel</Button>
              </div>
            </div>
          ) : null}
          <form className="flex items-center gap-2 border-t border-border pt-3" onSubmit={event => { event.preventDefault(); if (draft.trim() && !send.isPending) send.mutate() }}>
            <div className="min-w-0 flex-1"><Input value={draft} maxLength={4000} onChange={event => setDraft(event.target.value)} placeholder="Share something helpful…" aria-label="Community chat message" /></div>
            <Button type="submit" disabled={!draft.trim() || send.isPending} loading={send.isPending} icon={<Send className="size-4" />}>Send</Button>
          </form>
          {send.isError ? <p role="alert" className="text-xs text-danger">Message could not be sent.</p> : null}
          <Button className="mt-3 self-start" size="sm" variant="ghost" onClick={() => { setRoom(null); void queryClient.removeQueries({ queryKey: ['citizen-chat', room.id] }) }}>Change chat area</Button>
        </Card>
      )}
      {canModerate ? (
        <Card className="space-y-3 p-4">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h2 className="font-semibold text-ink">Community chat moderation</h2>
              <p className="text-xs text-ink-muted">Only reports within your authorised geographic area are shown.</p>
            </div>
            <span className="rounded-full bg-surface-hi px-2 py-1 text-xs text-ink-muted">{moderation.data?.reports.length ?? 0} pending</span>
          </div>
          {moderation.isLoading ? <Skeleton className="h-20 w-full" /> : moderation.isError ? (
            <ErrorState title="Could not load reports" description="Try again." onRetry={() => void moderation.refetch()} />
          ) : moderation.data?.reports.length ? moderation.data.reports.map(item => (
            <div key={item.id} className="space-y-2 rounded-xl border border-border p-3">
              <p className="text-xs font-semibold text-ink">Reported message · {item.message.sender || 'Community member'}</p>
              <p className="text-sm text-ink">{item.message.content}</p>
              <p className="text-xs text-ink-muted">Reason: {item.reason}</p>
              <div className="flex flex-wrap gap-2">
                <Button size="sm" variant="danger" disabled={resolveReport.isPending} loading={resolveReport.isPending} onClick={() => resolveReport.mutate({ id: item.id, status: 'reviewed', action: 'hide' })}>Hide message</Button>
                <Button size="sm" variant="secondary" disabled={resolveReport.isPending} onClick={() => resolveReport.mutate({ id: item.id, status: 'dismissed' })}>Dismiss report</Button>
              </div>
            </div>
          )) : <EmptyState title="No pending reports" description="New reports from your authorised area will appear here." />}
        </Card>
      ) : null}
    </main>
  )
}
