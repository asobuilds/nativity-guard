import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Card } from '@/components/ui/Card'
import { Select } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useUnits } from '@/hooks/useUnits'
import { api } from '@/lib/apiClient'
import { useAuth } from '@/auth/AuthContext'

type Membership = { unitId: string; role: string; status: string; isHeadAdmin: boolean }
type Audit = {
  unitId: string
  elections: { total: number; finalized: number; lowTurnout: number; totalVotes: number; recent: { id: string; status: string; seatCount: number; quorumCount: number; quorumMet: boolean; electionType: string; termStart: string; termEnd: string }[] }
  revocations: { total: number; completed: number; rejected: number }
  appeals: { total: number; upheld: number; overturned: number }
  currentAdmins: { adminCount: number; headAdminCount: number }
}

export function GovernanceAuditPage() {
  const { role } = useAuth()
  const units = useUnits()
  const memberships = useQuery({ queryKey: ['my-unit-memberships'], queryFn: () => api.get<{ memberships: Membership[] }>('/units/my-memberships') })
  const [selected, setSelected] = useState('')
  const allowed = (memberships.data?.memberships ?? []).filter((item) => item.status === 'active' && item.isHeadAdmin).map((item) => item.unitId)
  const available = role === 'super_admin' ? (units.data ?? []).map((unit) => unit.id) : allowed
  const unitId = available.includes(selected) ? selected : available[0] ?? ''
  const audit = useQuery({ queryKey: ['governance-audit', unitId], queryFn: () => api.get<Audit>(`/units/${unitId}/governance-audit`), enabled: Boolean(unitId) })
  return <main className="mx-auto max-w-4xl space-y-5 p-4 sm:p-6">
    <header><Link className="text-sm text-signal hover:underline" to="/admin/unit-policy">← Unit policy</Link><h1 className="mt-3 text-2xl font-semibold text-ink">Governance overview</h1><p className="mt-1 text-sm text-ink-muted">Election, revocation and appeal totals for a unit. Available to its head administrator or a super administrator.</p></header>
    {units.isLoading || memberships.isLoading ? <Skeleton className="h-20 w-full" /> : units.isError || memberships.isError ? <Card><ErrorState title="Could not load units" onRetry={() => { void units.refetch(); void memberships.refetch() }} /></Card> : !available.length ? <Card><EmptyState title="No governance overview available" description="Only a unit head administrator or super administrator can view this summary." /></Card> : <>
      <Card className="p-5"><label htmlFor="audit-unit" className="block text-sm font-medium text-ink">Unit</label><Select id="audit-unit" className="mt-2 w-full" value={unitId} onChange={(event) => setSelected(event.target.value)}>{available.map((id) => <option key={id} value={id}>{units.data?.find((unit) => unit.id === id)?.name ?? id}</option>)}</Select></Card>
      {audit.isLoading ? <Skeleton className="h-52 w-full" /> : audit.isError ? <Card><ErrorState title="Could not load governance summary" description="Check your permissions or try again." onRetry={() => void audit.refetch()} /></Card> : audit.data ? <>
        <section aria-label="Governance totals" className="grid gap-3 sm:grid-cols-2">
          <Card className="p-5"><h2 className="font-semibold text-ink">Elections</h2><p className="mt-2 text-sm text-ink-muted">{audit.data.elections.total} held · {audit.data.elections.finalized} finalized · {audit.data.elections.lowTurnout} with low turnout · {audit.data.elections.totalVotes} votes</p></Card>
          <Card className="p-5"><h2 className="font-semibold text-ink">Revocations</h2><p className="mt-2 text-sm text-ink-muted">{audit.data.revocations.total} cycles · {audit.data.revocations.completed} completed · {audit.data.revocations.rejected} rejected</p></Card>
          <Card className="p-5"><h2 className="font-semibold text-ink">Appeals</h2><p className="mt-2 text-sm text-ink-muted">{audit.data.appeals.total} filed · {audit.data.appeals.upheld} upheld · {audit.data.appeals.overturned} overturned</p></Card>
          <Card className="p-5"><h2 className="font-semibold text-ink">Current administrators</h2><p className="mt-2 text-sm text-ink-muted">{audit.data.currentAdmins.adminCount} unit admins · {audit.data.currentAdmins.headAdminCount} head admins</p></Card>
        </section>
        <Card className="p-5"><h2 className="font-semibold text-ink">Recent elections</h2>{audit.data.elections.recent.length ? <ul className="mt-3 space-y-3">{audit.data.elections.recent.map((item) => <li className="rounded-lg border border-border p-3 text-sm" key={item.id}><p className="font-medium text-ink">{item.electionType} · {item.status}</p><p className="text-ink-muted">{item.seatCount} seats · quorum {item.quorumMet ? 'met' : 'not met'} · term {item.termStart} to {item.termEnd}</p><p className="break-all text-xs text-ink-faint">Election ID: {item.id}</p></li>)}</ul> : <EmptyState title="No recent elections" />}</Card>
        <p className="text-sm text-ink-muted">This is an audit summary, not a voting ballot. The API exposes only the five most recent elections here; it has no member-facing election list.</p>
      </> : null}
    </>}
  </main>
}
