import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/apiClient'

export interface LedgerBalance {
  unitId: string
  balance: number
  totalIn: number
  totalOut: number
  currency: string
}

export interface Transaction {
  id: string
  unitId: string
  type: string
  category?: string
  amount: number
  description: string
  status: string
  paymentMethod?: string
  referenceId?: string
  createdAt: string
  initiator?: {
    id: string
    firstName?: string
    lastName?: string
    email?: string
  }
}

export function useUnitLedgerBalance(unitId?: string | null) {
  return useQuery({
    queryKey: ['unit-ledger-balance', unitId],
    queryFn: () => api.get<LedgerBalance>(`/units/${unitId}/ledger/balance`),
    enabled: Boolean(unitId),
  })
}

export function useUnitPendingTransactions(unitId?: string | null) {
  return useQuery({
    queryKey: ['unit-pending-transactions', unitId],
    queryFn: () => api.get<{ transactions: Transaction[] }>(`/units/${unitId}/transactions?status=pending`),
    select: (d) => d.transactions,
    enabled: Boolean(unitId),
  })
}
