import { useQuery } from '@tanstack/react-query'
import { useAuth } from '@/auth/AuthContext'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { api } from '@/lib/apiClient'

type Analytics = { totalCases: number; resolvedCases: number; pendingCases: number; monthlyStats: Array<{ Month?: string; month?: string; Count?: number; count?: number }> }

export function AdminAnalyticsPage() {
  const { user } = useAuth()
  const query = useQuery({
    queryKey: ['unit-analytics', user?.unitId],
    queryFn: () => api.get<Analytics>('/analytics/units/' + user!.unitId),
    enabled: !!user?.unitId,
  })
  if (!user?.unitId) return <div className="mx-auto max-w-3xl p-4 sm:p-6"><Card><EmptyState title="No unit assigned" description="Analytics become available after this administrator is linked to a security unit." /></Card></div>
  return <div className="mx-auto max-w-5xl space-y-4 p-4 sm:p-6">
    <header><h1 className="text-xl font-semibold text-ink">Unit analytics</h1><p className="mt-1 text-sm text-ink-muted">Live case performance for your security unit.</p></header>
    {query.isLoading ? <Skeleton className="h-32 w-full" /> : query.isError ? <Card><ErrorState title="Could not load unit analytics" description="Try again." onRetry={() => void query.refetch()} /></Card> : query.data && <>
      <div className="grid gap-4 sm:grid-cols-3">
        <Card><p className="text-xs uppercase tracking-widest text-ink-muted">Total cases</p><p className="mt-2 text-2xl font-semibold tabular-nums">{query.data.totalCases}</p></Card>
        <Card><p className="text-xs uppercase tracking-widest text-ink-muted">Resolved</p><p className="mt-2 text-2xl font-semibold tabular-nums">{query.data.resolvedCases}</p></Card>
        <Card><p className="text-xs uppercase tracking-widest text-ink-muted">Pending</p><p className="mt-2 text-2xl font-semibold tabular-nums">{query.data.pendingCases}</p></Card>
      </div>
      <Card><h2 className="text-sm font-semibold text-ink">Recent monthly volume</h2>{query.data.monthlyStats.length ? <ul className="mt-3 space-y-2">{query.data.monthlyStats.map((row, i) => <li key={(row.Month ?? row.month ?? '') + i} className="flex justify-between text-sm"><span>{row.Month ?? row.month}</span><span className="tabular-nums">{row.Count ?? row.count ?? 0} cases</span></li>)}</ul> : <p className="mt-3 text-sm text-ink-muted">No case history yet.</p>}</Card>
    </>}
  </div>
}
