import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Banknote, TrendingDown, TrendingUp, Wallet } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Chips'
import { Skeleton, ErrorState, EmptyState } from '@/components/ui/States'
import { useAuth } from '@/auth/AuthContext'
import { api } from '@/lib/apiClient'
import { formatDateTime } from '@/lib/format'

interface Transaction {
  id: string
  unitId: string
  type: string
  amount: number
  status: string
  description?: string
  category?: string
  createdAt: string
}

interface FinanceSummary {
  totalIncome: number
  totalExpenses: number
  balance: number
  pendingCount: number
}

interface BankAccount {
  id: string
  bankName?: string
  accountName?: string
  accountNumber?: string
  isPublic?: boolean
}

function currency(n: number): string {
  return '₦' + (n ?? 0).toLocaleString(undefined, { maximumFractionDigits: 2 })
}

export function AdminFinancePage() {
  const { user } = useAuth()
  const unitId = user?.unitId

  const summary = useQuery({
    queryKey: ['finance', 'summary', unitId],
    queryFn: () => api.get<FinanceSummary>(`/finance/units/${unitId}/summary`),
    enabled: Boolean(unitId),
  })

  const txs = useQuery({
    queryKey: ['finance', 'transactions', unitId],
    queryFn: () => api.get<{ transactions: Transaction[] }>(`/finance/units/${unitId}/transactions`),
    select: (d) => d.transactions,
    enabled: Boolean(unitId),
  })

  const accounts = useQuery({
    queryKey: ['bank', 'accounts', unitId],
    queryFn: () => api.get<{ accounts: BankAccount[] }>(`/bank/${unitId}/accounts`),
    select: (d) => d.accounts,
    enabled: Boolean(unitId),
  })

  const recent = useMemo(() => (txs.data ?? []).slice(0, 20), [txs.data])

  if (!unitId) {
    return (
      <div className="mx-auto max-w-3xl space-y-4 p-4 sm:p-6">
        <Card className="p-5">
          <EmptyState title="No unit assigned" description="Your account is not yet linked to a security unit." />
        </Card>
      </div>
    )
  }

  return (
    <div className="mx-auto w-full max-w-6xl space-y-4 p-4 sm:p-6">
      <header>
        <h1 className="text-xl font-semibold text-ink">Unit finance</h1>
        <p className="mt-1 text-sm text-ink-muted">Ledger, transactions, and bank accounts for your unit.</p>
      </header>

      {summary.isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24 w-full" />)}
        </div>
      ) : summary.isError ? (
        <Card><ErrorState title="Could not load summary" description="Try again." onRetry={() => void summary.refetch()} /></Card>
      ) : summary.data ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Card>
            <div className="flex items-center justify-between">
              <p className="text-xs uppercase tracking-widest text-ink-muted">Income</p>
              <TrendingUp className="size-4 text-ok" aria-hidden />
            </div>
            <p className="mt-2 text-xl font-semibold text-ink tabular-nums">{currency(summary.data.totalIncome)}</p>
          </Card>
          <Card>
            <div className="flex items-center justify-between">
              <p className="text-xs uppercase tracking-widest text-ink-muted">Expenses</p>
              <TrendingDown className="size-4 text-emergency" aria-hidden />
            </div>
            <p className="mt-2 text-xl font-semibold text-ink tabular-nums">{currency(summary.data.totalExpenses)}</p>
          </Card>
          <Card>
            <div className="flex items-center justify-between">
              <p className="text-xs uppercase tracking-widest text-ink-muted">Balance</p>
              <Wallet className="size-4 text-signal" aria-hidden />
            </div>
            <p className="mt-2 text-xl font-semibold text-ink tabular-nums">{currency(summary.data.balance)}</p>
          </Card>
          <Card>
            <div className="flex items-center justify-between">
              <p className="text-xs uppercase tracking-widest text-ink-muted">Pending</p>
              <Banknote className="size-4 text-warn" aria-hidden />
            </div>
            <p className="mt-2 text-xl font-semibold text-ink tabular-nums">{summary.data.pendingCount}</p>
          </Card>
        </div>
      ) : null}

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2 overflow-hidden">
          <div className="border-b border-border px-4 py-3">
            <h2 className="text-sm font-semibold text-ink">Recent transactions</h2>
          </div>
          {txs.isLoading ? (
            <Skeleton className="m-4 h-40 w-full" />
          ) : txs.isError ? (
            <div className="p-4"><ErrorState title="Could not load transactions" description="Try again." onRetry={() => void txs.refetch()} /></div>
          ) : recent.length === 0 ? (
            <div className="p-4"><EmptyState title="No transactions" description="Transactions for your unit will appear here." /></div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border text-left text-xs uppercase tracking-wider text-ink-faint">
                    <th className="px-4 py-3">Type</th>
                    <th className="px-4 py-3">Description</th>
                    <th className="px-4 py-3 text-right">Amount</th>
                    <th className="px-4 py-3">Status</th>
                    <th className="px-4 py-3">Date</th>
                  </tr>
                </thead>
                <tbody>
                  {recent.map((t) => (
                    <tr key={t.id} className="border-b border-border last:border-0 hover:bg-surface-hi">
                      <td className="px-4 py-3 text-ink">{t.type}</td>
                      <td className="px-4 py-3 text-ink-muted">{t.description || '—'}</td>
                      <td className="px-4 py-3 text-right text-ink tabular-nums">{currency(t.amount)}</td>
                      <td className="px-4 py-3"><Badge tone={t.status === 'approved' ? 'ok' : t.status === 'rejected' ? 'warn' : undefined}>{t.status}</Badge></td>
                      <td className="px-4 py-3 text-ink-faint text-xs">{formatDateTime(t.createdAt)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>

        <Card>
          <h2 className="text-sm font-semibold text-ink">Bank accounts</h2>
          {accounts.isLoading ? (
            <Skeleton className="mt-3 h-32 w-full" />
          ) : accounts.isError ? (
            <p className="mt-3 text-sm text-ink-muted">Accounts unavailable.</p>
          ) : (accounts.data ?? []).length === 0 ? (
            <p className="mt-3 text-sm text-ink-muted">No accounts registered.</p>
          ) : (
            <ul className="mt-3 space-y-3">
              {(accounts.data ?? []).map((a) => (
                <li key={a.id} className="rounded-lg border border-border p-3">
                  <p className="text-sm font-medium text-ink">{a.bankName || 'Bank'}</p>
                  <p className="mt-0.5 text-xs text-ink-muted">{a.accountName || '—'}</p>
                  <p className="mt-0.5 text-xs text-ink-faint font-mono">{a.accountNumber || '—'}</p>
                  {a.isPublic ? <Badge tone="ok">Public</Badge> : null}
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>
    </div>
  )
}