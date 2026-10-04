import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, Clock, Shield, X } from 'lucide-react'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useToast } from '@/components/ui/Toast'
import { api, ApiError } from '@/lib/apiClient'
import { unitKeys } from '@/hooks/useUnits'

interface UnitRow {
  id: string
  name: string
  type: string
  state: string
  lga: string
  ward?: string
  city: string
  registrationNumber: string
  contactPerson: string
  contactPhone: string
  contactEmail: string
  commanderName: string
  commanderNin: string
  commanderPhoneAlt?: string
  kindredHeadName?: string
  kindredHeadPhone?: string
  wardHeadName?: string
  wardHeadPhone?: string
  isVerified: boolean
  verificationStatus: 'pending' | 'under_review' | 'verified' | 'rejected'
  verificationNotes?: string
  verifiedAt?: string
  createdAt: string
}

type FilterStatus = 'pending' | 'under_review' | 'verified' | 'rejected' | 'all'

export function UnitVerificationQueuePage() {
  const queryClient = useQueryClient()
  const { notify } = useToast()
  const [filter, setFilter] = useState<FilterStatus>('pending')
  const [rejectingId, setRejectingId] = useState<string | null>(null)
  const [reason, setReason] = useState('')

  const query = useQuery({
    queryKey: ['unit-verifications', filter],
    queryFn: () =>
      api.get<{ units: UnitRow[] }>(`/admin/units/verifications?status=${filter}`),
    staleTime: 30_000,
  })

  const approveMutation = useMutation({
    mutationFn: (id: string) => api.post(`/admin/units/${id}/verify`, {}),
    onSuccess: () => {
      notify('Unit verified', 'success')
      void queryClient.invalidateQueries({ queryKey: ['unit-verifications'] })
      void queryClient.invalidateQueries({ queryKey: unitKeys.all })
      void queryClient.invalidateQueries({ queryKey: ['admin-unit-record'] })
    },
    onError: (err) => {
      notify(err instanceof ApiError ? err.message : 'Could not verify', 'error')
    },
  })

  const rejectMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      api.post(`/admin/units/${id}/reject`, { reason }),
    onSuccess: () => {
      notify('Unit rejected', 'success')
      setRejectingId(null)
      setReason('')
      void queryClient.invalidateQueries({ queryKey: ['unit-verifications'] })
      void queryClient.invalidateQueries({ queryKey: unitKeys.all })
      void queryClient.invalidateQueries({ queryKey: ['admin-unit-record'] })
    },
    onError: (err) => {
      notify(err instanceof ApiError ? err.message : 'Could not reject', 'error')
    },
  })

  const underReviewMutation = useMutation({
    mutationFn: (id: string) => api.post(`/admin/units/${id}/under-review`, {}),
    onSuccess: () => {
      notify('Marked under review', 'success')
      void queryClient.invalidateQueries({ queryKey: ['unit-verifications'] })
      void queryClient.invalidateQueries({ queryKey: unitKeys.all })
      void queryClient.invalidateQueries({ queryKey: ['admin-unit-record'] })
    },
    onError: (err) => {
      notify(err instanceof ApiError ? err.message : 'Could not update', 'error')
    },
  })

  const rows = query.data?.units ?? []

  return (
    <div className="mx-auto max-w-4xl p-4 sm:p-6">
      <header className="mb-6">
        <h1 className="text-2xl font-bold text-ink">Unit registration review</h1>
        <p className="mt-1 text-sm text-ink-muted">
          Approve a unit only after the registration details match the submitted
          documents and the traditional endorsement is verified.
        </p>
      </header>

      <div className="mb-4 flex flex-wrap gap-2">
        {(['pending', 'under_review', 'verified', 'rejected', 'all'] as FilterStatus[]).map(
          (s) => (
            <button
              key={s}
              type="button"
              onClick={() => setFilter(s)}
              className={`rounded-lg px-3 py-1.5 text-xs font-medium ${
                filter === s ? 'bg-signal/10 text-signal' : 'text-ink-muted hover:bg-surface-hi'
              }`}
            >
              {labelFor(s)}
            </button>
          ),
        )}
      </div>

      <Card>
        <CardHeader title={`${rows.length} ${filter === 'all' ? 'total' : labelFor(filter)}`} />
        <CardBody>
          {query.isLoading ? (
            <div className="flex flex-col gap-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-32 w-full" />
              ))}
            </div>
          ) : query.isError ? (
            <ErrorState
              title="Could not load units"
              description="Try again in a moment."
              onRetry={() => void query.refetch()}
            />
          ) : rows.length === 0 ? (
            <EmptyState
              icon={<Shield className="size-5" aria-hidden />}
              title={`No ${filter === 'all' ? '' : labelFor(filter).toLowerCase()} units`}
              description="Nothing to review right now."
            />
          ) : (
            <ul className="flex flex-col gap-3">
              {rows.map((row) => (
                <li
                  key={row.id}
                  className="rounded-lg border border-border bg-surface-hi/40 p-4"
                >
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <p className="text-sm font-semibold text-ink">{row.name}</p>
                        <StatusPill status={row.verificationStatus} />
                      </div>
                      <p className="mt-1 text-xs text-ink-muted">
                        <strong className="text-ink">Type:</strong> {row.type} ·{' '}
                        <strong className="text-ink">Reg #:</strong> {row.registrationNumber || '—'}
                      </p>
                      <p className="mt-1 text-xs text-ink-muted">
                        {row.city}, {row.lga}, {row.state}
                      </p>

                      <div className="mt-3 grid gap-1 text-xs text-ink-muted sm:grid-cols-2">
                        <p>
                          <strong className="text-ink">Contact:</strong> {row.contactPerson}
                        </p>
                        <p>
                          <strong className="text-ink">Phone:</strong> {row.contactPhone}
                        </p>
                        <p>
                          <strong className="text-ink">Email:</strong> {row.contactEmail}
                        </p>
                        <p>
                          <strong className="text-ink">Commander:</strong> {row.commanderName}
                        </p>
                        <p>
                          <strong className="text-ink">Commander NIN:</strong>{' '}
                          {row.commanderNin}
                        </p>
                        {row.commanderPhoneAlt ? (
                          <p>
                            <strong className="text-ink">Alt phone:</strong>{' '}
                            {row.commanderPhoneAlt}
                          </p>
                        ) : null}
                        {row.kindredHeadName ? (
                          <p>
                            <strong className="text-ink">Kindred head:</strong>{' '}
                            {row.kindredHeadName} ({row.kindredHeadPhone})
                          </p>
                        ) : null}
                        {row.wardHeadName ? (
                          <p>
                            <strong className="text-ink">Ward head:</strong>{' '}
                            {row.wardHeadName} ({row.wardHeadPhone})
                          </p>
                        ) : null}
                      </div>

                      <p className="mt-2 text-xs text-ink-faint">
                        Submitted {new Date(row.createdAt).toLocaleString()}
                      </p>
                      {row.verificationNotes ? (
                        <p className="mt-2 text-xs text-emergency">
                          <strong>Notes:</strong> {row.verificationNotes}
                        </p>
                      ) : null}
                    </div>

                    {(row.verificationStatus === 'pending' ||
                      row.verificationStatus === 'under_review') &&
                    rejectingId !== row.id ? (
                      <div className="flex shrink-0 flex-wrap gap-2">
                        {row.verificationStatus === 'pending' ? (
                          <Button
                            variant="secondary"
                            size="sm"
                            icon={<Clock className="size-3.5" aria-hidden />}
                            loading={
                              underReviewMutation.isPending &&
                              underReviewMutation.variables === row.id
                            }
                            onClick={() => underReviewMutation.mutate(row.id)}
                          >
                            Under review
                          </Button>
                        ) : null}
                        <Button
                          variant="primary"
                          size="sm"
                          icon={<Check className="size-3.5" aria-hidden />}
                          loading={
                            approveMutation.isPending && approveMutation.variables === row.id
                          }
                          onClick={() => approveMutation.mutate(row.id)}
                        >
                          Approve
                        </Button>
                        <Button
                          variant="secondary"
                          size="sm"
                          icon={<X className="size-3.5" aria-hidden />}
                          onClick={() => {
                            setRejectingId(row.id)
                            setReason('')
                          }}
                        >
                          Reject
                        </Button>
                      </div>
                    ) : null}
                  </div>

                  {rejectingId === row.id ? (
                    <div className="mt-3 rounded-lg border border-emergency/30 bg-emergency/5 p-3">
                      <label className="mb-1 block text-xs font-medium text-ink">
                        Reason for rejection (recorded on the unit)
                      </label>
                      <textarea
                        value={reason}
                        onChange={(e) => setReason(e.target.value)}
                        rows={3}
                        maxLength={1000}
                        className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-ink"
                        placeholder="e.g. The traditional endorsement from the Ward Head could not be confirmed."
                      />
                      <div className="mt-2 flex items-center justify-end gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setRejectingId(null)
                            setReason('')
                          }}
                        >
                          Cancel
                        </Button>
                        <Button
                          variant="primary"
                          size="sm"
                          disabled={!reason.trim() || rejectMutation.isPending}
                          loading={rejectMutation.isPending}
                          onClick={() =>
                            rejectMutation.mutate({ id: row.id, reason: reason.trim() })
                          }
                        >
                          Confirm rejection
                        </Button>
                      </div>
                    </div>
                  ) : null}
                </li>
              ))}
            </ul>
          )}
        </CardBody>
      </Card>
    </div>
  )
}

function labelFor(s: FilterStatus): string {
  switch (s) {
    case 'pending':
      return 'Pending'
    case 'under_review':
      return 'Under review'
    case 'verified':
      return 'Verified'
    case 'rejected':
      return 'Rejected'
    case 'all':
      return 'All'
  }
}

function StatusPill({ status }: { status: UnitRow['verificationStatus'] }) {
  const styles: Record<UnitRow['verificationStatus'], string> = {
    pending: 'bg-warn/15 text-warn',
    under_review: 'bg-signal/15 text-signal',
    verified: 'bg-ok/15 text-ok',
    rejected: 'bg-emergency/15 text-emergency',
  }
  return (
    <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium uppercase ${styles[status]}`}>
      {status.replace('_', ' ')}
    </span>
  )
}