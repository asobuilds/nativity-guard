import { Link } from 'react-router-dom'
import { Download, Database, FileText, Shield } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Skeleton, ErrorState } from '@/components/ui/States'
import { useSystemHealth, useSystemStats, useAllUsers } from '@/hooks/useSuperAdmin'
import { useUnits } from '@/hooks/useUnits'

export function SuperSettingsPage() {
  const stats = useSystemStats()
  const users = useAllUsers()
  const units = useUnits()
  const health = useSystemHealth()

  function exportData() {
    const payload = {
      exportedAt: new Date().toISOString(),
      stats: stats.data ?? null,
      userCount: users.data?.total ?? 0,
      unitCount: units.data?.length ?? 0,
      health: health.data ?? null,
    }
    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'nativityguard-export-' + Date.now() + '.json'
    a.click()
    URL.revokeObjectURL(url)
  }

  const healthOk = health.data
    ? health.data.databaseStatus === 'healthy' && health.data.serverStatus === 'healthy'
    : null

  return (
    <div className="mx-auto w-full max-w-4xl space-y-4 p-4 sm:p-6">
      <header>
        <h1 className="text-xl font-semibold text-ink">Platform settings</h1>
        <p className="mt-1 text-sm text-ink-muted">System configuration, health, and data exports.</p>
      </header>

      <Card className="p-5">
        <div className="flex items-center gap-2">
          <Shield className="size-4 text-signal" aria-hidden />
          <h2 className="text-sm font-semibold text-ink">System health</h2>
        </div>
        {health.isLoading ? (
          <Skeleton className="mt-3 h-24 w-full" />
        ) : health.isError ? (
          <ErrorState title="Health unavailable" description="Try again." onRetry={() => void health.refetch()} />
        ) : health.data ? (
          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            <div>
              <p className="text-xs text-ink-faint">Status</p>
              <div className="mt-1 flex items-center gap-2">
                <span className={'size-2 rounded-full ' + (healthOk ? 'bg-ok' : 'bg-warn')} aria-hidden />
                <span className="text-sm text-ink">{healthOk ? 'Healthy' : 'Degraded'}</span>
              </div>
            </div>
            <div>
              <p className="text-xs text-ink-faint">Database</p>
              <p className="mt-1 text-sm text-ink">{health.data.databaseStatus || '—'}</p>
            </div>
            <div>
              <p className="text-xs text-ink-faint">CPU</p>
              <p className="mt-1 text-sm text-ink tabular-nums">{health.data.cpuUsage.toFixed(1)}%</p>
            </div>
            <div>
              <p className="text-xs text-ink-faint">Memory</p>
              <p className="mt-1 text-sm text-ink tabular-nums">{health.data.memoryUsage.toFixed(1)}%</p>
            </div>
          </div>
        ) : null}
      </Card>

      <Card className="p-5">
        <div className="flex items-center gap-2">
          <Database className="size-4 text-signal" aria-hidden />
          <h2 className="text-sm font-semibold text-ink">Platform data</h2>
        </div>
        <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-3">
          <div>
            <dt className="text-xs text-ink-faint">Total users</dt>
            <dd className="mt-1 text-ink tabular-nums">{stats.data?.totalUsers ?? users.data?.total ?? 0}</dd>
          </div>
          <div>
            <dt className="text-xs text-ink-faint">Total units</dt>
            <dd className="mt-1 text-ink tabular-nums">{units.data?.length ?? 0}</dd>
          </div>
          <div>
            <dt className="text-xs text-ink-faint">Total cases</dt>
            <dd className="mt-1 text-ink tabular-nums">{stats.data?.totalCases ?? 0}</dd>
          </div>
        </dl>
        <Button className="mt-4" icon={<Download className="size-4" />} onClick={exportData}>
          Export platform snapshot
        </Button>
        <p className="mt-2 text-xs text-ink-faint">Downloads a JSON snapshot of current stats and health.</p>
      </Card>

      <Card className="p-5">
        <div className="flex items-center gap-2">
          <FileText className="size-4 text-signal" aria-hidden />
          <h2 className="text-sm font-semibold text-ink">Retention policy</h2>
        </div>
        <ul className="mt-4 space-y-2 text-sm text-ink-muted">
          <li><strong className="text-ink">Evidence:</strong> retained 5 years, then archived to cold storage.</li>
          <li><strong className="text-ink">Video evidence:</strong> retained 2 years hot, then archived.</li>
          <li><strong className="text-ink">Avatars & covers:</strong> retained while account exists + 30 days after deletion.</li>
          <li><strong className="text-ink">News & community media:</strong> retained indefinitely unless flagged.</li>
        </ul>
      </Card>

      <Card className="p-5">
        <h2 className="text-sm font-semibold text-ink">Related pages</h2>
        <ul className="mt-3 space-y-2 text-sm">
          <li><Link to="/super/audit" className="text-signal hover:underline">Audit log →</Link></li>
          <li><Link to="/super/users" className="text-signal hover:underline">User management →</Link></li>
          <li><Link to="/super/units" className="text-signal hover:underline">Unit registry →</Link></li>
          <li><Link to="/super/analytics" className="text-signal hover:underline">Analytics →</Link></li>
        </ul>
      </Card>
    </div>
  )
}