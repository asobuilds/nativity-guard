import { useMemo } from 'react'
import { Wallet, TrendingUp, TrendingDown, Banknote } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Chips'
import { Skeleton, ErrorState, EmptyState } from '@/components/ui/States'
import { useAuth } from '@/auth/AuthContext'
import { useUnitLedgerBalance, useUnitPendingTransactions } from '@/hooks/useUnitLedger'
import { formatDateTime } from '@/lib/format'

function currency(n: number): string {
  return '₦' + (n ?? 0).toLocaleString(undefined, { maximumFractionDigits: 2 })
}

export function UnitFinancialLedgerPage() {
  const { user } = useAuth()
  const unitId = user?.unitId

  const balanceQuery = useUnitLedgerBalance(unitId)
  const pendingQuery = useUnitPendingTransactions(unitId)

  const pendingTransactions = useMemo(() => pendingQuery.data ?? [], [pendingQuery.data])

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
        <h1 className="text-xl font-semibold text-ink">Unit financial ledger</h1>
        <p className="mt-1 text-sm text-ink-muted">Live ledger balance and pending transactions for your unit.</p>
      </header>

      {balanceQuery.isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[0, 1, 2].map((i) => <Skeleton key={i} className="h-32 w-full" />)}
        </div>
      ) : balanceQuery.isError ? (
        <Card><ErrorState title="Could not load ledger balance" description="Try again." onRetry={() => void balanceQuery.refetch()} /></Card>
      ) : balanceQuery.data ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Card className="p-6 bg-surface-hi border-signal/20">
            <div className="flex items-center justify-between">
              <p className="text-xs uppercase tracking-widest text-ink-muted font-medium">Net Ledger Balance</p>
              <Wallet className="size-5 text-signal" aria-hidden />
            </div>
            <p className="mt-3 text-3xl font-bold text-ink tabular-nums">{currency(balanceQuery.data.balance)}</p>
            <p className="mt-1 text-xs text-ink-faint">Immutable ledger balance ({balanceQuery.data.currency})</p>
          </Card>
          <Card className="p-6">
            <div className="flex items-center justify-between">
              <p className="text-xs uppercase tracking-widest text-ink-muted font-medium">Total Inflow</p>
              <TrendingUp className="size-5 text-ok" aria-hidden />
            </div>
            <p className="mt-3 text-2xl font-semibold text-ink tabular-nums">{currency(balanceQuery.data.totalIn)}</p>
            <p className="mt-1 text-xs text-ink-faint">Total posted credits</p>
          </Card>
          <Card className="p-6">
            <div className="flex items-center justify-between">
              <p className="text-xs uppercase tracking-widest text-ink-muted font-medium">Total Outflow</p>
              <TrendingDown className="size-5 text-emergency" aria-hidden />
            </div>
            <p className="mt-3 text-2xl font-semibold text-ink tabular-nums">{currency(balanceQuery.data.totalOut)}</p>
            <p className="mt-1 text-xs text-ink-faint">Total posted debits</p>
          </Card>
        </div>
      ) : null}

      <Card className="overflow-hidden">
        <div className="border-b border-border px-4 py-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Banknote className="size-4 text-warn" />
            <h2 className="text-sm font-semibold text-ink">Pending Transactions</h2>
          </div>
          <Badge tone="warn">{pendingTransactions.length} pending</Badge>
        </div>
        {pendingQuery.isLoading ? (
          <Skeleton className="m-4 h-48 w-full" />
        ) : pendingQuery.isError ? (
          <div className="p-4"><ErrorState title="Could not load pending transactions" description="Try again." onRetry={() => void pendingQuery.refetch()} /></div>
        ) : pendingTransactions.length === 0 ? (
          <div className="p-8 text-center">
            <EmptyState title="No pending transactions" description="All unit transactions have been approved or processed." />
          </div>
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
                {pendingTransactions.map((t) => (
                  <tr key={t.id} className="border-b border-border last:border-0 hover:bg-surface-hi">
                    <td className="px-4 py-3 font-medium text-ink capitalize">{t.type}</td>
                    <td className="px-4 py-3 text-ink-muted">{t.description || '—'}</td>
                    <td className="px-4 py-3 text-right text-ink tabular-nums font-semibold">{currency(t.amount)}</td>
                    <td className="px-4 py-3"><Badge tone="warn">{t.status}</Badge></td>
                    <td className="px-4 py-3 text-ink-faint text-xs">{formatDateTime(t.createdAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  )
}
