import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { FolderKanban, ShieldAlert, Users } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Skeleton, ErrorState, EmptyState } from '@/components/ui/States'
import { StatusChip } from '@/components/ui/Chips'
import { useAuth } from '@/auth/AuthContext'
import { useCases } from '@/hooks/useCases'
import { useUnits } from '@/hooks/useUnits'
import { relativeTime } from '@/lib/format'

export function AdminOverviewPage() {
  const { user } = useAuth()
  const cases = useCases()
  const units = useUnits()

  const myUnit = useMemo(() => {
    if (!user?.unitId) return null
    return (units.data ?? []).find((u) => u.id === user.unitId) ?? null
  }, [user?.unitId, units.data])

  const unitCases = useMemo(() => {
    const list = cases.data ?? []
    if (!user?.unitId) return []
    return list.filter((c) => c.unitId === user.unitId)
  }, [cases.data, user?.unitId])

  const counts = useMemo(() => {
    const by = { pending: 0, assigned: 0, dispatched: 0, closed: 0, other: 0 }
    for (const c of unitCases) {
      const s = c.status as keyof typeof by
      if (s in by) by[s]++
      else by.other++
    }
    return by
  }, [unitCases])

  const recent = useMemo(
    () => [...unitCases].sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt)).slice(0, 5),
    [unitCases],
  )

  if (!user?.unitId) {
    return (
      <div className="mx-auto max-w-3xl space-y-4 p-4 sm:p-6">
        <Card className="p-5">
          <EmptyState
            title="No unit assigned"
            description="Your account is not yet linked to a security unit. Ask a super admin to assign you."
          />
        </Card>
      </div>
    )
  }

  return (
    <div className="mx-auto w-full max-w-6xl space-y-4 p-4 sm:p-6">
      <header>
        <h1 className="text-xl font-semibold text-ink">Unit operations</h1>
        <p className="mt-1 text-sm text-ink-muted">{myUnit ? myUnit.name + ' · ' + [myUnit.city, myUnit.state].filter(Boolean).join(', ') : 'Your unit'}</p>
      </header>

      {cases.isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24 w-full" />)}
        </div>
      ) : cases.isError ? (
        <Card><ErrorState title="Could not load cases" description="Try again." onRetry={() => void cases.refetch()} /></Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Card><p className="text-xs uppercase tracking-widest text-ink-muted">Pending</p><p className="mt-2 text-2xl font-semibold text-ink tabular-nums">{counts.pending}</p></Card>
          <Card><p className="text-xs uppercase tracking-widest text-ink-muted">Assigned</p><p className="mt-2 text-2xl font-semibold text-ink tabular-nums">{counts.assigned}</p></Card>
          <Card><p className="text-xs uppercase tracking-widest text-ink-muted">Dispatched</p><p className="mt-2 text-2xl font-semibold text-ink tabular-nums">{counts.dispatched}</p></Card>
          <Card><p className="text-xs uppercase tracking-widest text-ink-muted">Closed</p><p className="mt-2 text-2xl font-semibold text-ink tabular-nums">{counts.closed}</p></Card>
        </div>
      )}

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-ink">Recent cases</h2>
            <Link to="/admin/cases" className="text-xs text-signal hover:underline">Open case board →</Link>
          </div>
          {cases.isLoading ? (
            <Skeleton className="mt-3 h-40 w-full" />
          ) : recent.length === 0 ? (
            <EmptyState title="No cases yet" description="Cases assigned to your unit will appear here." />
          ) : (
            <ul className="mt-3 space-y-2">
              {recent.map((c) => (
                <li key={c.id}>
                  <Link to={`/admin/cases/${c.id}`} className="block rounded-lg border border-border p-3 hover:bg-surface-hi">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium text-ink">{c.title}</p>
                        <p className="mt-0.5 text-xs text-ink-faint">{c.trackingId} · {relativeTime(c.createdAt)}</p>
                      </div>
                      <StatusChip status={c.status} />
                    </div>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </Card>

        <Card>
          <h2 className="text-sm font-semibold text-ink">Quick actions</h2>
          <ul className="mt-3 space-y-2 text-sm">
            <li><Link to="/admin/cases" className="flex items-center gap-2 text-signal hover:underline"><FolderKanban className="size-4" /> Case board</Link></li>
            <li><Link to="/admin/officers" className="flex items-center gap-2 text-signal hover:underline"><Users className="size-4" /> Unit officers</Link></li>
            <li><Link to="/admin/transfers" className="flex items-center gap-2 text-signal hover:underline"><ShieldAlert className="size-4" /> Transfers</Link></li>
            <li><Link to="/admin/finance" className="flex items-center gap-2 text-signal hover:underline"><FolderKanban className="size-4" /> Finance</Link></li>
          </ul>
        </Card>
      </div>
    </div>
  )
}