import { useEffect, useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Input, Select } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { api } from '@/lib/apiClient'
import { useUnits } from '@/hooks/useUnits'

type Membership = { unitId: string; role: string; status: string; isHeadAdmin: boolean; verifiedAt?: string | null }
type Policy = {
  regularAdminRemovalVotes: number
  headAdminRemovalMajority: number
  quorumPercent: number
  coolingOffMonths: number
  termMonths: number
  staggerMonths: number
  consecutiveTermLimit: number
  seatBands: Record<string, number>
}
type PolicyField = Exclude<keyof Policy, 'seatBands'>
const fields: { key: PolicyField; label: string; description: string; fraction?: boolean }[] = [
  { key: 'regularAdminRemovalVotes', label: 'Votes to remove an administrator', description: 'Minimum votes required.' },
  { key: 'headAdminRemovalMajority', label: 'Head administrator removal majority', description: 'Fraction between 0 and 1, for example 0.5.', fraction: true },
  { key: 'quorumPercent', label: 'Voting quorum', description: 'Fraction between 0 and 1, for example 0.5.', fraction: true },
  { key: 'coolingOffMonths', label: 'Cooling-off period (months)', description: 'Time before another term.' },
  { key: 'termMonths', label: 'Term length (months)', description: 'Length of an administrator term.' },
  { key: 'staggerMonths', label: 'Stagger period (months)', description: 'Spacing between elections.' },
  { key: 'consecutiveTermLimit', label: 'Consecutive term limit', description: 'Maximum consecutive terms.' },
]

export function UnitPolicyPage() {
  const client = useQueryClient()
  const units = useUnits()
  const memberships = useQuery({ queryKey: ['my-unit-memberships'], queryFn: () => api.get<{ memberships: Membership[] }>('/units/my-memberships') })
  const readable = (memberships.data?.memberships ?? []).filter((item) => item.status === 'active' && Boolean(item.verifiedAt))
  const [unitId, setUnitId] = useState('')
  const selectedId = readable.some((item) => item.unitId === unitId) ? unitId : readable[0]?.unitId ?? ''
  const membership = readable.find((item) => item.unitId === selectedId)
  const canEdit = Boolean(membership?.isHeadAdmin || membership?.role === 'unit_admin')
  const policy = useQuery({ queryKey: ['unit-policy', selectedId], queryFn: () => api.get<{ policy: Policy; version: string }>(`/units/${selectedId}/auth`), enabled: Boolean(selectedId) })
  const [draft, setDraft] = useState<Policy | null>(null)
  const [editing, setEditing] = useState(false)
  useEffect(() => { setDraft(policy.data?.policy ?? null); setEditing(false) }, [policy.data, selectedId])
  const save = useMutation({
    mutationFn: (value: Policy) => api.put<{ version: string }>(`/units/${selectedId}/auth`, value),
    onSuccess: () => { setEditing(false); void client.invalidateQueries({ queryKey: ['unit-policy', selectedId] }) },
  })
  function update(key: PolicyField, value: string) {
    setDraft((old) => old ? { ...old, [key]: value === '' ? Number.NaN : Number(value) } : old)
  }
  const valid = draft && fields.every(({ key, fraction }) => {
    const value = draft[key]
    return Number.isFinite(value) && (fraction ? value > 0 && value <= 1 : Number.isInteger(value) && value > 0)
  })
  function submit(event: FormEvent) {
    event.preventDefault()
    if (canEdit && valid && draft) save.mutate(draft)
  }

  return <main className="mx-auto max-w-3xl space-y-5 p-4 sm:p-6">
    <header><Link to="/admin/cases" className="text-sm text-signal hover:underline">← Case board</Link><h1 className="mt-3 text-2xl font-semibold text-ink">Unit governance policy</h1><p className="mt-1 text-sm text-ink-muted">View the rules for a unit where you are a verified member. Unit administrators can update them.</p></header>
    {memberships.isLoading || units.isLoading ? <Skeleton className="h-24 w-full" /> : memberships.isError || units.isError ? <Card><ErrorState title="Could not load your units" onRetry={() => { void memberships.refetch(); void units.refetch() }} /></Card> : !readable.length ? <Card><EmptyState title="No verified unit membership" description="Governance policy is available to verified members of their unit." /></Card> : <>
      <Card className="p-5"><label className="block text-sm font-medium text-ink" htmlFor="policy-unit">Unit</label><Select id="policy-unit" className="mt-2 w-full" value={selectedId} onChange={(event) => setUnitId(event.target.value)}>{readable.map((item) => <option value={item.unitId} key={item.unitId}>{units.data?.find((unit) => unit.id === item.unitId)?.name ?? item.unitId}</option>)}</Select></Card>
      {policy.isLoading ? <Skeleton className="h-96 w-full" /> : policy.isError ? <Card><ErrorState title="Could not load policy" description="Check your membership or try again." onRetry={() => void policy.refetch()} /></Card> : draft ? <Card className="space-y-5 p-5">
        <div className="flex flex-wrap items-center justify-between gap-2"><div><h2 className="font-semibold text-ink">Policy · {policy.data?.version}</h2><p className="text-xs text-ink-muted">Changes take effect for future governance actions.</p></div>{canEdit && !editing ? <Button size="sm" onClick={() => setEditing(true)}>Edit policy</Button> : null}</div>
        <form onSubmit={submit} className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">{fields.map(({ key, label, description, fraction }) => <label key={key} className="block text-sm text-ink-muted">{label}<Input className="mt-1 w-full" type="number" min={fraction ? '0.01' : '1'} max={fraction ? '1' : undefined} step={fraction ? '0.01' : '1'} value={Number.isNaN(draft[key]) ? '' : draft[key]} disabled={!editing} onChange={(event) => update(key, event.target.value)} /><span className="mt-1 block text-xs text-ink-faint">{description}</span></label>)}</div>
          <div><h3 className="text-sm font-medium text-ink">Seats by membership size</h3><ul className="mt-2 flex flex-wrap gap-2 text-sm text-ink-muted">{Object.entries(draft.seatBands ?? {}).map(([size, seats]) => <li className="rounded-lg border border-border px-3 py-2" key={size}>{size} members: {seats} seats</li>)}</ul><p className="mt-1 text-xs text-ink-faint">Seat bands are shown for reference and preserved when saving.</p></div>
          {editing ? <div className="flex flex-wrap gap-2"><Button type="submit" variant="primary" loading={save.isPending} disabled={!valid}>Save policy</Button><Button disabled={save.isPending} onClick={() => { setDraft(policy.data?.policy ?? null); setEditing(false); save.reset() }}>Cancel</Button></div> : null}
          {save.isError ? <p role="alert" className="text-sm text-emergency">{save.error instanceof Error ? save.error.message : 'Policy was not saved.'}</p> : null}
          {save.isSuccess && !editing ? <p role="status" className="text-sm text-ink">Policy saved. Latest version loaded above.</p> : null}
        </form>
      </Card> : null}
    </>}
  </main>
}
