import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { AlertTriangle, FolderKanban, ShieldAlert, Users, Activity } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Skeleton, ErrorState, EmptyState } from '@/components/ui/States'
import { StatusChip } from '@/components/ui/Chips'
import { useAuth } from '@/auth/AuthContext'
import { useCases } from '@/hooks/useCases'
import { useUnits } from '@/hooks/useUnits'
import { useAdminAnalyticsOverview } from '@/hooks/useSuperAdmin'
import { relativeTime } from '@/lib/format'

export function AdminOverviewPage() {
  const { user } = useAuth()
  const isSuper = user?.role === 'super_admin'

  const superOverview = useAdminAnalyticsOverview()
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

  if (isSuper) {
    if (superOverview.isLoading) {
      return (
        <div className="mx-auto w-full max-w-6xl space-y-4 p-4 sm:p-6">
          <Skeleton className="h-8 w-64" />
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24 w-full" />)}
          </div>
        </div>
      )
    }
    if (superOverview.isError) {
      return (
        <div className="mx-auto w-full max-w-6xl p-4 sm:p-6">
          <Card><ErrorState title="Could not load command center analytics" description="Try again." onRetry={() => void superOverview.refetch()} /></Card>
        </div>
      )
    }
    const data = superOverview.data
    const maxDaily = data?.dailyActivity.reduce((max, d) => Math.max(max, d.count), 0) ?? 1
    const totalCasesSum = data?.casesByStatus.reduce((sum, s) => sum + s.count, 0) ?? 0

    return (
      <div className="mx-auto w-full max-w-6xl space-y-4 p-4 sm:p-6">
        <header>
          <h1 className="text-xl font-semibold text-ink">Super Admin Command Center</h1>
          <p className="mt-1 text-sm text-ink-muted">Real-time platform health, active SOS alerts, cases, and moderation queue.</p>
        </header>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Card>
            <div className="flex items-center justify-between">
              <p className="text-xs uppercase tracking-widest text-ink-muted">Active SOS Alerts</p>
              <AlertTriangle className="size-5 text-signal" />
            </div>
            <p className="mt-2 text-2xl font-semibold text-ink tabular-nums">{data?.activeSosCount ?? 0}</p>
          </Card>
          <Card>
            <div className="flex items-center justify-between">
              <p className="text-xs uppercase tracking-widest text-ink-muted">Total Cases</p>
              <FolderKanban className="size-5 text-signal" />
            </div>
            <p className="mt-2 text-2xl font-semibold text-ink tabular-nums">{totalCasesSum}</p>
          </Card>
          <Card>
            <div className="flex items-center justify-between">
              <p className="text-xs uppercase tracking-widest text-ink-muted">Verified Units</p>
              <ShieldAlert className="size-5 text-signal" />
            </div>
            <p className="mt-2 text-2xl font-semibold text-ink tabular-nums">{data?.verifiedUnitsCount ?? 0}</p>
          </Card>
          <Card>
            <div className="flex items-center justify-between">
              <p className="text-xs uppercase tracking-widest text-ink-muted">Total Officers</p>
              <Users className="size-5 text-signal" />
            </div>
            <p className="mt-2 text-2xl font-semibold text-ink tabular-nums">{data?.totalOfficers ?? 0}</p>
          </Card>
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <h2 className="text-sm font-semibold text-ink">Cases by status</h2>
            {data?.casesByStatus.length === 0 ? (
              <p className="mt-3 text-sm text-ink-muted">No cases recorded yet.</p>
            ) : (
              <div className="mt-3 space-y-3">
                {data?.casesByStatus.map((item) => {
                  const pct = totalCasesSum > 0 ? Math.round((item.count / totalCasesSum) * 100) : 0
                  return (
                    <div key={item.status} className="space-y-1">
                      <div className="flex justify-between text-xs">
                        <span className="capitalize text-ink">{item.status}</span>
                        <span className="text-ink-muted tabular-nums">{item.count} ({pct}%)</span>
                      </div>
                      <div className="h-2 overflow-hidden rounded-full bg-surface-hi">
                        <div className="h-full bg-signal rounded-full" style={{ width: `${pct}%` }} />
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </Card>

          <Card>
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold text-ink">Recent Moderation Reports</h2>
              <Link to="/super/analytics" className="text-xs text-signal hover:underline">View analytics →</Link>
            </div>
            {data?.recentReports.length === 0 ? (
              <p className="mt-3 text-sm text-ink-muted">No pending moderation reports.</p>
            ) : (
              <ul className="mt-3 space-y-2">
                {data?.recentReports.map((r) => (
                  <li key={r.id} className="rounded-lg border border-border p-2.5 text-sm">
                    <div className="flex items-center justify-between">
                      <span className="font-medium text-ink capitalize">{r.targetType} report</span>
                      <StatusChip status={r.status} />
                    </div>
                    <p className="mt-1 text-xs text-ink-muted line-clamp-1">{r.reason}</p>
                    <p className="mt-0.5 text-[10px] text-ink-faint">{relativeTime(r.createdAt)}</p>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>

        <Card>
          <div className="flex items-center gap-2">
            <Activity className="size-4 text-signal" />
            <h2 className="text-sm font-semibold text-ink">7-Day Case Volume Trend</h2>
          </div>
          {data?.dailyActivity.length === 0 ? (
            <p className="mt-3 text-sm text-ink-muted">No recent daily activity.</p>
          ) : (
            <div className="mt-4 flex items-end gap-3 h-32 pt-6 px-2 border-b border-border">
              {data?.dailyActivity.map((day) => {
                const heightPct = maxDaily > 0 ? Math.max(12, Math.round((day.count / maxDaily) * 100)) : 12
                return (
                  <div key={day.date} className="flex-1 flex flex-col items-center gap-1">
                    <span className="text-[10px] text-ink-muted tabular-nums">{day.count}</span>
                    <div className="w-full bg-signal/25 hover:bg-signal/35 rounded-t transition-all" style={{ height: `${heightPct}%` }}>
                      <div className="w-full bg-signal rounded-t h-full" />
                    </div>
                    <span className="text-[10px] text-ink-faint truncate max-w-[60px]">{day.date.slice(5)}</span>
                  </div>
                )
              })}
            </div>
          )}
        </Card>
      </div>
    )
  }

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
