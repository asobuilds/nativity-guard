import { Fragment, useEffect, useMemo, useState } from 'react'
import { Search } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Input } from '@/components/ui/Field'
import { Tabs } from '@/components/ui/Tabs'
import { Skeleton, ErrorState, EmptyState } from '@/components/ui/States'
import { useAuditLogs, useActivityLogs, type AuditLogEntry, type ActivityLogEntry } from '@/hooks/useSuperAdmin'
import { formatDateTime, relativeTime } from '@/lib/format'

const PAGE_SIZE = 50

function prettyJson(raw: string): string {
  if (!raw) return ''
  try { return JSON.stringify(JSON.parse(raw), null, 2) } catch { return raw }
}

export function SuperAuditPage() {
  const [tab, setTab] = useState<'audit' | 'activity'>('audit')
  const audit = useAuditLogs()
  const activity = useActivityLogs()

  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [expanded, setExpanded] = useState<string | null>(null)

  useEffect(() => { setPage(1); setExpanded(null) }, [tab, search])

  const auditRows = useMemo(() => {
    const q = search.trim().toLowerCase()
    const list = audit.data ?? []
    if (!q) return list
    return list.filter((e) => [e.action, e.entityType, e.entityId, e.user?.email ?? ''].join(' ').toLowerCase().includes(q))
  }, [audit.data, search])

  const activityRows = useMemo(() => {
    const q = search.trim().toLowerCase()
    const list = activity.data ?? []
    if (!q) return list
    return list.filter((e) => [e.activityType, e.description, e.location, e.user?.email ?? ''].join(' ').toLowerCase().includes(q))
  }, [activity.data, search])

  const rows = tab === 'audit' ? auditRows : activityRows
  const pageCount = Math.max(1, Math.ceil(rows.length / PAGE_SIZE))
  const paginated = rows.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)
  const query = tab === 'audit' ? audit : activity

  return (
    <div className="mx-auto w-full max-w-6xl space-y-4 p-4 sm:p-6">
      <header>
        <h1 className="text-xl font-semibold text-ink">Audit log</h1>
        <p className="mt-1 text-sm text-ink-muted">Every action taken on the platform, newest first.</p>
      </header>

      <Card>
        <div className="flex flex-wrap items-center gap-3">
          <Tabs value={tab} onChange={(v) => setTab(v as typeof tab)} options={[
            { value: 'audit', label: 'Audit (' + auditRows.length + ')' },
            { value: 'activity', label: 'Activity (' + activityRows.length + ')' },
          ]} />
          <div className="relative min-w-56 flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint" aria-hidden />
            <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Filter by action, entity, or user" className="pl-9" />
          </div>
        </div>
      </Card>

      {query.isLoading ? (
        <Card><Skeleton className="h-64 w-full" /></Card>
      ) : query.isError ? (
        <Card><ErrorState title="Could not load log" description="Try again." onRetry={() => void query.refetch()} /></Card>
      ) : rows.length === 0 ? (
        <Card><EmptyState title="No entries" description="Nothing matches the current filter." /></Card>
      ) : (
        <>
          <Card className="overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border text-left text-xs uppercase tracking-wider text-ink-faint">
                    <th className="px-4 py-3">Time</th>
                    <th className="px-4 py-3">User</th>
                    <th className="px-4 py-3">{tab === 'audit' ? 'Action' : 'Activity'}</th>
                    <th className="px-4 py-3">{tab === 'audit' ? 'Entity' : 'Description'}</th>
                    <th className="px-4 py-3">IP</th>
                  </tr>
                </thead>
                <tbody>
                  {paginated.map((row) => {
                    const id = row.id
                    const isExp = expanded === id
                    const email = row.user?.email ?? '—'
                    const cell1 = tab === 'audit' ? (row as AuditLogEntry).action : (row as ActivityLogEntry).activityType
                    const cell2 = tab === 'audit'
                      ? ((row as AuditLogEntry).entityType + ' · ' + ((row as AuditLogEntry).entityId || '—'))
                      : ((row as ActivityLogEntry).description || '—')
                    const hasDetail = tab === 'audit'
                      ? Boolean((row as AuditLogEntry).oldValue || (row as AuditLogEntry).newValue)
                      : Boolean((row as ActivityLogEntry).device || (row as ActivityLogEntry).location)
                    return (
                      <Fragment key={id}>
                        <tr
                          className="cursor-pointer border-b border-border last:border-0 hover:bg-surface-hi"
                          onClick={() => hasDetail && setExpanded(isExp ? null : id)}
                        >
                          <td className="px-4 py-3 text-ink-muted whitespace-nowrap">{relativeTime(tab === 'audit' ? (row as AuditLogEntry).createdAt : (row as ActivityLogEntry).createdAt)}</td>
                          <td className="px-4 py-3 text-ink truncate max-w-xs">{email}</td>
                          <td className="px-4 py-3 text-ink">{cell1}</td>
                          <td className="px-4 py-3 text-ink-muted truncate max-w-md">{cell2}</td>
                          <td className="px-4 py-3 text-ink-faint font-mono text-xs">{row.ipAddress || '—'}</td>
                        </tr>
                        {isExp && tab === 'audit' && (
                          <tr className="bg-void/60">
                            <td colSpan={5} className="px-4 py-4">
                              <div className="grid gap-3 md:grid-cols-2">
                                <div>
                                  <p className="mb-1 text-xs uppercase tracking-widest text-ink-faint">Old value</p>
                                  <pre className="max-h-64 overflow-auto rounded border border-border bg-void p-3 text-xs text-ink-muted">{prettyJson((row as AuditLogEntry).oldValue) || '—'}</pre>
                                </div>
                                <div>
                                  <p className="mb-1 text-xs uppercase tracking-widest text-ink-faint">New value</p>
                                  <pre className="max-h-64 overflow-auto rounded border border-border bg-void p-3 text-xs text-ink-muted">{prettyJson((row as AuditLogEntry).newValue) || '—'}</pre>
                                </div>
                              </div>
                              <p className="mt-3 text-xs text-ink-faint">Full timestamp: {formatDateTime((row as AuditLogEntry).createdAt)}</p>
                            </td>
                          </tr>
                        )}
                        {isExp && tab === 'activity' && (
                          <tr className="bg-void/60">
                            <td colSpan={5} className="px-4 py-4">
                              <dl className="grid gap-3 text-xs sm:grid-cols-3">
                                <div><dt className="text-ink-faint">Device</dt><dd className="text-ink">{(row as ActivityLogEntry).device || '—'}</dd></div>
                                <div><dt className="text-ink-faint">Location</dt><dd className="text-ink">{(row as ActivityLogEntry).location || '—'}</dd></div>
                                <div><dt className="text-ink-faint">Duration</dt><dd className="text-ink tabular-nums">{(row as ActivityLogEntry).duration || 0}s</dd></div>
                              </dl>
                            </td>
                          </tr>
                        )}
                      </Fragment>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </Card>

          <div className="flex items-center justify-between text-sm">
            <span className="text-ink-muted">Page {page} of {pageCount} · {rows.length} entries</span>
            <div className="flex gap-2">
              <Button size="sm" variant="ghost" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>Previous</Button>
              <Button size="sm" variant="ghost" onClick={() => setPage((p) => Math.min(pageCount, p + 1))} disabled={page >= pageCount}>Next</Button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}