import { useMemo, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { FileText, ShieldAlert, Users, Activity } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Chips'
import { Skeleton, ErrorState, EmptyState } from '@/components/ui/States'
import { useSystemStats, useAuditLogs, useSystemHealth } from '@/hooks/useSuperAdmin'
import { useUnits } from '@/hooks/useUnits'
import { relativeTime } from '@/lib/format'

function KpiCard({ label, value, icon }: { label: string; value: number | string; icon: ReactNode }) {
  return (
    <Card>
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-xs uppercase tracking-widest text-ink-muted">{label}</p>
          <p className="mt-2 text-2xl font-semibold text-ink tabular-nums">{value}</p>
        </div>
        <span className="text-signal">{icon}</span>
      </div>
    </Card>
  )
}

export function SuperOverviewPage() {
  const stats = useSystemStats()
  const audit = useAuditLogs()
  const health = useSystemHealth()
  const units = useUnits()

  const topStates = useMemo(() => {
    const list = units.data ?? []
    const counts: Record<string, number> = {}
    for (const u of list) {
      if (!u.state) continue
      counts[u.state] = (counts[u.state] ?? 0) + 1
    }
    return Object.entries(counts)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 5)
  }, [units.data])

  const recentAudit = useMemo(() => (audit.data ?? []).slice(0, 5), [audit.data])

  const healthOk = health.data
    ? health.data.databaseStatus === 'healthy' && health.data.serverStatus === 'healthy'
    : null

  return (
    <div className="mx-auto w-full max-w-6xl space-y-4 p-4 sm:p-6">
      <header className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-ink">Platform overview</h1>
          <p className="mt-1 text-sm text-ink-muted">Live platform health and activity.</p>
        </div>
      </header>

      {stats.isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24 w-full" />)}
        </div>
      ) : stats.isError ? (
        <Card><ErrorState title="Could not load stats" description="Try again." onRetry={() => void stats.refetch()} /></Card>
      ) : stats.data ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <KpiCard label="Total users" value={stats.data.totalUsers} icon={<Users className="size-5" />} />
          <KpiCard label="Total cases" value={stats.data.totalCases} icon={<FileText className="size-5" />} />
          <KpiCard label="Active units" value={stats.data.totalUnits} icon={<ShieldAlert className="size-5" />} />
          <KpiCard label="Total SOS" value={stats.data.totalSOS} icon={<Activity className="size-5" />} />
        </div>
      ) : null}

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-1">
          <h2 className="text-sm font-semibold text-ink">Top states</h2>
          {units.isLoading ? (
            <Skeleton className="mt-3 h-24 w-full" />
          ) : units.isError ? (
            <p className="mt-3 text-sm text-ink-muted">Units unavailable.</p>
          ) : topStates.length === 0 ? (
            <p className="mt-3 text-sm text-ink-muted">No units registered yet.</p>
          ) : (
            <ul className="mt-3 space-y-2">
              {topStates.map(([state, count]) => (
                <li key={state} className="flex items-center justify-between text-sm">
                  <span className="text-ink">{state}</span>
                  <span className="text-ink-muted tabular-nums">{count}</span>
                </li>
              ))}
            </ul>
          )}
        </Card>

        <Card className="lg:col-span-1">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-ink">Recent activity</h2>
            <Link to="/super/audit" className="text-xs text-signal hover:underline">View all →</Link>
          </div>
          {audit.isLoading ? (
            <Skeleton className="mt-3 h-32 w-full" />
          ) : audit.isError ? (
            <p className="mt-3 text-sm text-ink-muted">Audit unavailable.</p>
          ) : recentAudit.length === 0 ? (
            <EmptyState title="No activity yet" description="Actions will appear here as they happen." />
          ) : (
            <ul className="mt-3 space-y-3">
              {recentAudit.map((entry) => (
                <li key={entry.id} className="text-sm">
                  <p className="text-ink">
                    <span className="font-medium">{entry.action}</span>{' '}
                    <span className="text-ink-muted">· {entry.entityType}</span>
                  </p>
                  <p className="mt-0.5 text-xs text-ink-faint">{relativeTime(entry.createdAt)}</p>
                </li>
              ))}
            </ul>
          )}
        </Card>

        <Card className="lg:col-span-1">
          <h2 className="text-sm font-semibold text-ink">System health</h2>
          {health.isLoading ? (
            <Skeleton className="mt-3 h-24 w-full" />
          ) : health.isError ? (
            <p className="mt-3 text-sm text-ink-muted">Health unavailable.</p>
          ) : health.data ? (
            <div className="mt-3 space-y-3">
              <div className="flex items-center gap-2">
                <span className={'size-2 rounded-full ' + (healthOk ? 'bg-ok' : 'bg-warn')} aria-hidden />
                <Badge tone={healthOk ? 'ok' : 'warn'}>{healthOk ? 'Healthy' : 'Degraded'}</Badge>
              </div>
              <dl className="grid grid-cols-2 gap-2 text-xs">
                <div>
                  <dt className="text-ink-faint">CPU</dt>
                  <dd className="text-ink tabular-nums">{health.data.cpuUsage.toFixed(1)}%</dd>
                </div>
                <div>
                  <dt className="text-ink-faint">Memory</dt>
                  <dd className="text-ink tabular-nums">{health.data.memoryUsage.toFixed(1)}%</dd>
                </div>
                <div>
                  <dt className="text-ink-faint">Active users</dt>
                  <dd className="text-ink tabular-nums">{health.data.activeUsers}</dd>
                </div>
                <div>
                  <dt className="text-ink-faint">Requests</dt>
                  <dd className="text-ink tabular-nums">{health.data.totalRequests}</dd>
                </div>
              </dl>
            </div>
          ) : null}
        </Card>
      </div>
    </div>
  )
}