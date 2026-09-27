import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '@/auth/AuthContext'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Input, Select, Textarea } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useUnits } from '@/hooks/useUnits'
import { api } from '@/lib/apiClient'
import { formatDateTime } from '@/lib/format'

interface Transfer {
  id: string
  targetId: string
  targetType: 'case' | 'suspect'
  fromUnitId: string
  toUnitId: string
  reason: string
  status: string
  approvalCount: number
  requiredApprovals: number
  createdAt: string
}
interface Approval { id: string; approverId: string; status: string; comment: string; createdAt: string }

export function TransfersPage() {
  const { user, role } = useAuth()
  const client = useQueryClient()
  const [caseId, setCaseId] = useState('')
  const [toUnitId, setToUnitId] = useState('')
  const [reason, setReason] = useState('')
  const [selected, setSelected] = useState<string | null>(null)
  const [comment, setComment] = useState('')
  const requests = useQuery({ queryKey: ['transfers'], queryFn: () => api.get<{ transferRequests: Transfer[] }>('/transfers') })
  const approvals = useQuery({ queryKey: ['transfer-approvals', selected], queryFn: () => api.get<{ approvals: Approval[] }>(`/transfers/${selected}/approvals`), enabled: Boolean(selected) })
  const units = useUnits()
  const create = useMutation({
    mutationFn: () => api.post('/transfers', { targetId: caseId.trim(), targetType: 'case', toUnitId, reason: reason.trim() }),
    onSuccess: () => { setCaseId(''); setToUnitId(''); setReason(''); void client.invalidateQueries({ queryKey: ['transfers'] }) },
  })
  const decide = useMutation({
    mutationFn: ({ id, action }: { id: string; action: 'approve' | 'reject' }) => api.post(`/transfers/${id}/${action}`, { comment: comment.trim() }),
    onSuccess: () => { setComment(''); setSelected(null); void client.invalidateQueries({ queryKey: ['transfers'] }); void client.invalidateQueries({ queryKey: ['transfer-approvals'] }) },
  })
  const unitName = (id: string) => units.data?.find((unit) => unit.id === id)?.name ?? id
  const ownUnit = user?.unitId
  const submit = (event: FormEvent) => { event.preventDefault(); if (caseId.trim() && toUnitId && reason.trim()) create.mutate() }

  return <main className="mx-auto max-w-5xl space-y-5 p-4 sm:p-6">
    <header><h1 className="text-2xl font-semibold text-ink">Unit transfers</h1><p className="mt-1 text-sm text-ink-muted">Request a case transfer and review approvals from the source unit.</p></header>
    {ownUnit ? <Card className="space-y-4 p-5"><h2 className="font-semibold text-ink">Request a case transfer</h2><p className="text-sm text-ink-muted">The case must belong to your unit. A transfer moves only after the required approvals.</p>
      <form className="grid gap-3 sm:grid-cols-2" onSubmit={submit}>
        <label className="text-sm text-ink-muted">Case ID<Input required value={caseId} onChange={(event) => setCaseId(event.target.value)} placeholder="Paste the case ID" /></label>
        <label className="text-sm text-ink-muted">Destination unit<Select required value={toUnitId} onChange={(event) => setToUnitId(event.target.value)}><option value="">Select a unit</option>{units.data?.filter((unit) => unit.id !== ownUnit).map((unit) => <option key={unit.id} value={unit.id}>{unit.name}</option>)}</Select></label>
        <label className="text-sm text-ink-muted sm:col-span-2">Reason<Textarea required maxLength={2000} value={reason} onChange={(event) => setReason(event.target.value)} /></label>
        <div className="sm:col-span-2"><Button type="submit" loading={create.isPending} disabled={!caseId.trim() || !toUnitId || !reason.trim()}>Request transfer</Button></div>
      </form>
      {units.isError ? <p role="alert" className="text-sm text-emergency">Could not load destination units.</p> : null}
      {create.isError ? <p role="alert" className="text-sm text-emergency">{create.error instanceof Error ? create.error.message : 'Transfer request failed.'}</p> : null}
      {create.isSuccess ? <p role="status" className="text-sm text-ink">Transfer request submitted.</p> : null}
    </Card> : role === 'unit_admin' ? <Card className="p-5 text-sm text-ink-muted">Your account needs a source unit before requesting transfers.</Card> : null}
    <section className="space-y-3" aria-label="Transfer requests"><h2 className="text-lg font-semibold text-ink">Transfer requests</h2>
      {requests.isLoading ? <Skeleton className="h-28 w-full" /> : requests.isError ? <Card><ErrorState title="Could not load transfers" description="Try again." onRetry={() => void requests.refetch()} /></Card> : !requests.data?.transferRequests.length ? <Card><EmptyState title="No transfer requests" description="Requests from your unit will appear here." /></Card> : <ul className="space-y-3">{requests.data.transferRequests.map((request) => <li key={request.id}><Card className="space-y-3 p-5">
        <div className="flex flex-wrap items-center justify-between gap-2"><h3 className="font-semibold text-ink">{request.targetType === 'case' ? 'Case' : 'Suspect'} transfer</h3><span className="rounded-full border border-border px-2 py-1 text-xs text-ink">{request.status}</span></div>
        <p className="break-all text-xs text-ink-faint">Record {request.targetId}</p><p className="text-sm text-ink-muted">{unitName(request.fromUnitId)} → {unitName(request.toUnitId)}</p>
        <p className="whitespace-pre-wrap text-sm text-ink">{request.reason}</p><p className="text-xs text-ink-faint">{request.approvalCount} of {request.requiredApprovals} approvals · {formatDateTime(request.createdAt)}</p>
        <Button size="sm" variant="secondary" onClick={() => { setSelected(selected === request.id ? null : request.id); setComment('') }}>{selected === request.id ? 'Close review' : 'Review approvals'}</Button>
        {selected === request.id ? <div className="space-y-3 border-t border-border pt-3">
          {approvals.isLoading ? <Skeleton className="h-16 w-full" /> : approvals.isError ? <ErrorState title="Could not load approvals" description="Try again." onRetry={() => void approvals.refetch()} /> : <ul className="space-y-1 text-xs text-ink-muted">{approvals.data?.approvals.length ? approvals.data.approvals.map((item) => <li key={item.id}>{item.status} · {formatDateTime(item.createdAt)} · {item.comment || 'No comment'}</li>) : <li>No decisions yet.</li>}</ul>}
          {request.status === 'pending' && request.fromUnitId === ownUnit ? <><label className="block text-sm text-ink-muted">Decision comment<Textarea maxLength={2000} value={comment} onChange={(event) => setComment(event.target.value)} /></label><div className="flex flex-wrap gap-2"><Button disabled={decide.isPending || approvals.isLoading || approvals.isError || approvals.data?.approvals.some((item) => item.approverId === user?.id)} onClick={() => decide.mutate({ id: request.id, action: 'approve' })}>Approve</Button><Button variant="secondary" disabled={!comment.trim() || decide.isPending || approvals.isLoading || approvals.isError || approvals.data?.approvals.some((item) => item.approverId === user?.id)} onClick={() => decide.mutate({ id: request.id, action: 'reject' })}>Reject with reason</Button></div>{decide.isError ? <p role="alert" className="text-sm text-emergency">{decide.error instanceof Error ? decide.error.message : 'Could not save decision.'}</p> : null}</> : null}
        </div> : null}
      </Card></li>)}</ul>}
    </section>
  </main>
}
