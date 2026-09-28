import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '@/auth/AuthContext'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { api } from '@/lib/apiClient'
import { formatDateTime } from '@/lib/format'

type Session = {
  id: string
  jti: string
  device?: string
  userAgent?: string
  ipAddress?: string
  lastSeenAt: string
  expiresAt: string
}

export function SessionsPage() {
  const client = useQueryClient()
  const navigate = useNavigate()
  const { logout } = useAuth()
  const [confirmAll, setConfirmAll] = useState(false)
  const [confirmOne, setConfirmOne] = useState<string | null>(null)
  const sessions = useQuery({ queryKey: ['auth-sessions'], queryFn: () => api.get<{ sessions: Session[] }>('/auth/sessions') })
  const revoke = useMutation({
    mutationFn: (jti: string) => api.delete(`/auth/sessions/${encodeURIComponent(jti)}`),
    onSuccess: () => { setConfirmOne(null); void client.invalidateQueries({ queryKey: ['auth-sessions'] }) },
  })
  const revokeAll = useMutation({
    mutationFn: () => api.delete('/auth/sessions'),
    onSuccess: () => { logout(); navigate('/auth/login', { replace: true, state: { sessionsRevoked: true } }) },
  })

  return <main className="mx-auto max-w-3xl space-y-5 p-4 sm:p-6">
    <header>
      <Link to="/profile" className="text-sm text-signal hover:underline">← Profile</Link>
      <h1 className="mt-3 text-2xl font-semibold text-ink">Signed-in devices</h1>
      <p className="mt-1 text-sm text-ink-muted">Review your active sessions and end access you no longer recognise. The service does not identify which entry is this browser.</p>
    </header>

    {sessions.isLoading ? <Skeleton className="h-40 w-full" /> : sessions.isError ? <Card><ErrorState title="Could not load sessions" description="Your session list was not available." onRetry={() => void sessions.refetch()} /></Card> : !sessions.data?.sessions.length ? <Card><EmptyState title="No active sessions" description="Sign in again if you expected to see one." /></Card> : <ul className="space-y-3">
      {sessions.data.sessions.map((session) => <li key={session.id}><Card className="space-y-3 p-5">
        <div><h2 className="font-semibold text-ink">{session.device || 'Unidentified device'}</h2>
          <p className="break-words text-xs text-ink-muted">{session.userAgent || 'Browser details unavailable'}</p></div>
        <dl className="grid gap-2 text-sm sm:grid-cols-2">
          <div><dt className="text-ink-faint">Last active</dt><dd className="text-ink">{formatDateTime(session.lastSeenAt)}</dd></div>
          <div><dt className="text-ink-faint">Expires</dt><dd className="text-ink">{formatDateTime(session.expiresAt)}</dd></div>
          {session.ipAddress ? <div><dt className="text-ink-faint">IP address</dt><dd className="text-ink">{session.ipAddress}</dd></div> : null}
        </dl>
        {confirmOne === session.jti ? <div className="space-y-2 border-t border-border pt-3"><p className="text-sm text-ink-muted">End this session? If it is your current session, you may need to sign in again.</p><div className="flex gap-2"><Button size="sm" variant="danger" loading={revoke.isPending} onClick={() => revoke.mutate(session.jti)}>End session</Button><Button size="sm" disabled={revoke.isPending} onClick={() => setConfirmOne(null)}>Cancel</Button></div></div> : <Button size="sm" variant="secondary" onClick={() => { revoke.reset(); setConfirmOne(session.jti) }}>End this session</Button>}
        {revoke.isError && confirmOne === session.jti ? <p role="alert" className="text-sm text-emergency">{revoke.error instanceof Error ? revoke.error.message : 'Could not end session.'}</p> : null}
      </Card></li>)}
    </ul>}

    <Card className="space-y-3 p-5"><h2 className="font-semibold text-ink">End all sessions</h2><p className="text-sm text-ink-muted">This signs you out here and on other devices.</p>
      {confirmAll ? <div className="flex flex-wrap gap-2"><Button variant="danger" loading={revokeAll.isPending} onClick={() => revokeAll.mutate()}>Yes, sign out everywhere</Button><Button disabled={revokeAll.isPending} onClick={() => setConfirmAll(false)}>Cancel</Button></div> : <Button variant="secondary" onClick={() => setConfirmAll(true)}>Sign out everywhere</Button>}
      {revokeAll.isError ? <p role="alert" className="text-sm text-emergency">{revokeAll.error instanceof Error ? revokeAll.error.message : 'Could not end all sessions.'}</p> : null}
    </Card>
  </main>
}
