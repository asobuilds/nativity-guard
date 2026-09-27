import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '@/auth/AuthContext'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Input, Textarea } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { api } from '@/lib/apiClient'
import { formatDateTime } from '@/lib/format'

type Appeal = {
  id: string
  revocationCycleId: string
  appellantUserId: string
  reason: string
  status: 'pending' | 'upheld' | 'overturned'
  filedAt: string
  decisionReason: string
  decidedAt: string | null
}

export function AppealsPage() {
  const { role } = useAuth()
  const admin = role === 'super_admin'
  const client = useQueryClient()
  const [cycleId, setCycleId] = useState('')
  const [reason, setReason] = useState('')
  const [filedId, setFiledId] = useState<string | null>(null)
  const [lookupId, setLookupId] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [decisionReason, setDecisionReason] = useState('')
  const list = useQuery({ queryKey: ['appeals'], queryFn: () => api.get<{ appeals: Appeal[] }>('/appeals'), enabled: admin })
  const own = useQuery({ queryKey: ['appeal', selectedId], queryFn: () => api.get<Appeal>(`/appeals/${selectedId}`), enabled: !admin && Boolean(selectedId) })
  const file = useMutation({
    mutationFn: () => api.post<{ id: string }>('/appeals', { revocationCycleId: cycleId.trim(), reason: reason.trim() }),
    onSuccess: (created) => { setFiledId(created.id); setSelectedId(created.id); setCycleId(''); setReason('') },
  })
  const decide = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: 'uphold' | 'overturn' }) => api.put(`/appeals/${id}/decision`, { decision, reason: decisionReason.trim() }),
    onSuccess: () => { setDecisionReason(''); setSelectedId(null); void client.invalidateQueries({ queryKey: ['appeals'] }) },
  })
  const submit = (event: FormEvent) => { event.preventDefault(); if (cycleId.trim()) file.mutate() }
  const appeals = list.data?.appeals ?? []
  return <main className="mx-auto max-w-4xl space-y-5 p-4 sm:p-6">
    <header><h1 className="text-2xl font-semibold text-ink">Appeals</h1><p className="mt-1 text-sm text-ink-muted">Review removal decisions through the Go appeals service.</p></header>
    {!admin ? <>
      <Card className="space-y-4 p-5"><h2 className="font-semibold text-ink">Appeal a completed revocation</h2>
        <p className="text-sm text-ink-muted">Only the removed member may file, within 14 days of the completed cycle. One appeal is allowed per cycle.</p>
        <form className="space-y-3" onSubmit={submit}>
          <label className="block text-sm text-ink-muted">Revocation cycle ID<Input required value={cycleId} onChange={(e) => setCycleId(e.target.value)} placeholder="Paste the ID of your completed revocation cycle" /></label>
          <label className="block text-sm text-ink-muted">Your reason<Textarea maxLength={2000} value={reason} onChange={(e) => setReason(e.target.value)} /></label>
          <Button type="submit" loading={file.isPending} disabled={!cycleId.trim()}>File appeal</Button>
        </form>
        {file.isError ? <p role="alert" className="text-sm text-emergency">{file.error instanceof Error ? file.error.message : 'Could not file appeal.'}</p> : null}
        {filedId ? <p role="status" className="text-sm text-ink">Appeal filed. Keep this ID to check its status: <strong className="break-all">{filedId}</strong></p> : null}
      </Card>
      <Card className="space-y-3 p-5"><h2 className="font-semibold text-ink">Check your appeal</h2><p className="text-sm text-ink-muted">Enter your appeal ID. Only you and a super admin can view it.</p>
        <form className="flex flex-wrap gap-2" onSubmit={(event) => { event.preventDefault(); setSelectedId(lookupId.trim()) }}><Input aria-label="Appeal ID" className="min-w-0 flex-1" value={lookupId} onChange={(event) => setLookupId(event.target.value)} /><Button type="submit" disabled={!lookupId.trim()}>Check status</Button></form>
        {own.isLoading ? <Skeleton className="h-20 w-full" /> : own.isError ? <ErrorState title="Could not load appeal" description={own.error instanceof Error ? own.error.message : 'Check the ID and try again.'} /> : own.data ? <div className="text-sm text-ink"><p>Status: <strong>{own.data.status}</strong></p><p>Filed: {formatDateTime(own.data.filedAt)}</p>{own.data.decisionReason ? <p>Decision reason: {own.data.decisionReason}</p> : null}</div> : null}
      </Card>
    </> : <section className="space-y-4" aria-label="Appeal review">
      {list.isLoading ? <Skeleton className="h-28 w-full" /> : list.isError ? <Card><ErrorState title="Could not load appeals" description="Try again." onRetry={() => void list.refetch()} /></Card> : appeals.length === 0 ? <Card><EmptyState title="No appeals" description="Filed appeals will appear here." /></Card> : <ul className="space-y-4">{appeals.map((appeal) => <li key={appeal.id}><Card className="space-y-3 p-5">
        <div className="flex flex-wrap items-start justify-between gap-2"><h2 className="font-semibold text-ink">Appeal {appeal.id}</h2><span className="rounded-full border border-border px-2 py-1 text-xs text-ink">{appeal.status}</span></div>
        <p className="break-all text-xs text-ink-faint">Cycle: {appeal.revocationCycleId} · Appellant: {appeal.appellantUserId}</p>
        <p className="text-xs text-ink-faint">Filed {formatDateTime(appeal.filedAt)}</p>
        <p className="whitespace-pre-wrap text-sm text-ink">Reason: {appeal.reason || 'No reason given'}</p>
        {appeal.decisionReason ? <p className="whitespace-pre-wrap text-sm text-ink-muted">Decision: {appeal.decisionReason}</p> : null}
        {appeal.status === 'pending' ? <div className="space-y-3"><Button size="sm" variant="secondary" onClick={() => { setSelectedId(selectedId === appeal.id ? null : appeal.id); setDecisionReason('') }}>{selectedId === appeal.id ? 'Cancel decision' : 'Review decision'}</Button>
          {selectedId === appeal.id ? <div className="space-y-3 border-t border-border pt-3"><label className="block text-sm text-ink-muted">Decision reason<Textarea maxLength={2000} value={decisionReason} onChange={(event) => setDecisionReason(event.target.value)} /></label><div className="flex gap-2"><Button disabled={!decisionReason.trim() || decide.isPending} onClick={() => decide.mutate({ id: appeal.id, decision: 'uphold' })}>Uphold removal</Button><Button variant="secondary" disabled={!decisionReason.trim() || decide.isPending} onClick={() => decide.mutate({ id: appeal.id, decision: 'overturn' })}>Overturn removal</Button></div>{decide.isError ? <p role="alert" className="text-sm text-emergency">{decide.error instanceof Error ? decide.error.message : 'Decision failed.'}</p> : null}</div> : null}
        </div> : null}
      </Card></li>)}</ul>}
    </section>}
  </main>
}
