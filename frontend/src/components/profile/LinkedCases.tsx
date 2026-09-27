import { useQuery } from '@tanstack/react-query'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { api } from '@/lib/apiClient'
import { formatDate } from '@/lib/format'

type LinkedCase = {
  caseId: string
  trackingId: string
  title: string
  status: string
  roleInCase: string
  openedAt: string
  closedAt?: string
}

type LinkedCasesResponse = { active: LinkedCase[]; history: LinkedCase[] }

function CaseList({ title, items }: { title: string; items: LinkedCase[] }) {
  return <section aria-label={title} className="space-y-3">
    <h2 className="text-lg font-semibold text-ink">{title} <span className="text-sm font-normal text-ink-muted">({items.length})</span></h2>
    {items.length ? <ul className="space-y-3">{items.map((item) => <li key={`${item.caseId}-${item.roleInCase}`}><Card className="space-y-2 p-4">
      <h3 className="font-medium text-ink">{item.title}</h3>
      <p className="text-sm text-ink-muted">You are linked to this case. This status does not establish wrongdoing.</p>
      <dl className="grid gap-2 text-sm sm:grid-cols-2">
        <div><dt className="text-ink-faint">Tracking ID</dt><dd className="break-all text-ink">{item.trackingId}</dd></div>
        <div><dt className="text-ink-faint">Status</dt><dd className="text-ink">{item.status.replaceAll('_', ' ')}</dd></div>
        <div><dt className="text-ink-faint">Linked as</dt><dd className="text-ink">{item.roleInCase.replaceAll('_', ' ')}</dd></div>
        <div><dt className="text-ink-faint">Opened</dt><dd className="text-ink">{formatDate(item.openedAt)}</dd></div>
        {item.closedAt ? <div><dt className="text-ink-faint">Closed</dt><dd className="text-ink">{formatDate(item.closedAt)}</dd></div> : null}
      </dl>
    </Card></li>)}</ul> : <Card><EmptyState title={`No ${title.toLowerCase()}`} description="Cases linked to your account will appear here." /></Card>}
  </section>
}

export function LinkedCases() {
  const linked = useQuery({ queryKey: ['my-linked-cases'], queryFn: () => api.get<LinkedCasesResponse>('/suspects/me/cases') })
  if (linked.isLoading) return <Skeleton className="h-48 w-full" />
  if (linked.isError) return <Card><ErrorState title="Could not load linked cases" description="Try again." onRetry={() => void linked.refetch()} /></Card>
  return <div className="space-y-6">
    <p className="text-sm text-ink-muted">These are cases that list a connection to your account. Only your own linked-case status is shown here.</p>
    <CaseList title="Active cases" items={linked.data?.active ?? []} />
    <CaseList title="History" items={linked.data?.history ?? []} />
  </div>
}
