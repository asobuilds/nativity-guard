import { useMemo } from 'react'
import { Activity, FileText, ShieldAlert, Users } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Skeleton, ErrorState } from '@/components/ui/States'
import { useSystemStats, useAuditLogs } from '@/hooks/useSuperAdmin'
import { useUnits } from '@/hooks/useUnits'

function Bar({ label, value, max, tone = 'signal' }: { label: string; value: number; max: number; tone?: string }) {
  const pct = max > 0 ? Math.round((value / max) * 100) : 0
  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-xs">
        <span className="text-ink">{label}</span>
        <span className="text-ink-muted tabular-nums">{value}</span>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-surface-hi">
        <div
          className={'h-full rounded-full ' + (tone === 'signal' ? 'bg-signal' : tone === 'warn' ? 'bg-warn' : 'bg-ok')}
          style={{ width: pct + '%' }}
        />
      </div>
    </div>
  )
}

function Stat({ label, value, hint }: { label: string; value: string | number; hint?: string }) {
  return (
    <Card>
      <p className="text-xs uppercase tracking-widest text-ink-muted">{label}</p>
      <p className="mt-2 text-2xl font-semibold text-ink tabular-nums">{value}</p>
      {hint ? <p className="mt-1 text-xs text-ink-faint">{hint}</p> : null}
    </Card>
  )
}

export function SuperAnalyticsPage() {
  const stats = useSystemStats()
  const units = useUnits()
  const audit = useAuditLogs()

  const stateCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const u of units.data ?? []) {
      if (!u.state) continue
      counts[u.state] = (counts[u.state] ?? 0) + 1
    }
    return Object.entries(counts).sort((a, b) => b[1] - a[1]).slice(0, 8)
  }, [units.data])

  const actionCounts = useMemo(() => {
    const counts: Record<string, number> = {}
    for (const e of (audit.data ?? []).slice(0, 200)) {
      const a = e.action || 'other'
      counts[a] = (counts[a] ?? 0) + 1
    }
    return Object.entries(counts).sort((a, b) => b[1] - a[1]).slice(0, 8)
  }, [audit.data])

  const maxState = stateCounts.length ? stateCounts[0][1] : 0
  const maxAction = actionCounts.length ? actionCounts[0][1] : 0

  return (
    <div className="mx-auto w-full max-w-6xl space-y-4 p-4 sm:p-6">
      <header>
        <h1 className="text-xl font-semibold text-ink">Platform analytics</h1>
        <p className="mt-1 text-sm text-ink-muted">Aggregated view of platform activity.</p>
      </header>

      {stats.isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24 w-full" />)}
        </div>
      ) : stats.isError ? (
        <Card><ErrorState title="Could not load stats" description="Try again." onRetry={() => void stats.refetch()} /></Card>
      ) : stats.data ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Stat label="Users" value={stats.data.totalUsers} hint={stats.data.dailyActive + ' active today'} />
          <Stat label="Cases" value={stats.data.totalCases} hint={stats.data.pendingCases + ' pending · ' + stats.data.resolvedCases + ' resolved'} />
          <Stat label="Units" value={stats.data.totalUnits} hint={stats.data.totalOfficers + ' officers'} />
          <Stat label="SOS" value={stats.data.totalSOS} hint={stats.data.totalSuspects + ' suspects on file'} />
        </div>
      ) : null}

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <div className="flex items-center gap-2">
            <ShieldAlert className="size-4 text-signal" aria-hidden />
            <h2 className="text-sm font-semibold text-ink">Units by state</h2>
          </div>
          {units.isLoading ? (
            <Skeleton className="mt-3 h-40 w-full" />
          ) : units.isError ? (
            <p className="mt-3 text-sm text-ink-muted">Units unavailable.</p>
          ) : stateCounts.length === 0 ? (
            <p className="mt-3 text-sm text-ink-muted">No units registered yet.</p>
          ) : (
            <div className="mt-4 space-y-3">
              {stateCounts.map(([state, count]) => <Bar key={state} label={state} value={count} max={maxState} />)}
            </div>
          )}
        </Card>

        <Card>
          <div className="flex items-center gap-2">
            <Activity className="size-4 text-signal" aria-hidden />
            <h2 className="text-sm font-semibold text-ink">Recent actions</h2>
          </div>
          {audit.isLoading ? (
            <Skeleton className="mt-3 h-40 w-full" />
          ) : audit.isError ? (
            <p className="mt-3 text-sm text-ink-muted">Audit unavailable.</p>
          ) : actionCounts.length === 0 ? (
            <p className="mt-3 text-sm text-ink-muted">No activity yet.</p>
          ) : (
            <div className="mt-4 space-y-3">
              {actionCounts.map(([action, count]) => <Bar key={action} label={action} value={count} max={maxAction} tone="warn" />)}
            </div>
          )}
        </Card>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Card className="flex items-center gap-3 p-4">
          <FileText className="size-5 text-signal" aria-hidden />
          <div>
            <p className="text-xs text-ink-faint">Audit entries (recent)</p>
            <p className="text-lg text-ink tabular-nums">{(audit.data ?? []).length}</p>
          </div>
        </Card>
        <Card className="flex items-center gap-3 p-4">
          <Users className="size-5 text-signal" aria-hidden />
          <div>
            <p className="text-xs text-ink-faint">Officers</p>
            <p className="text-lg text-ink tabular-nums">{stats.data?.totalOfficers ?? 0}</p>
          </div>
        </Card>
        <Card className="flex items-center gap-3 p-4">
          <Activity className="size-5 text-signal" aria-hidden />
          <div>
            <p className="text-xs text-ink-faint">Daily active</p>
            <p className="text-lg text-ink tabular-nums">{stats.data?.dailyActive ?? 0}</p>
          </div>
        </Card>
      </div>
    </div>
  )
}