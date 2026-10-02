import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, Clock, X } from 'lucide-react'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useToast } from '@/components/ui/Toast'
import { api, ApiError } from '@/lib/apiClient'

interface VerificationRow {
  id: string
  userId: string
  userEmail: string
  userFirstName: string
  userLastName: string
  documentType: string
  documentUrl: string
  status: 'pending' | 'verified' | 'rejected'
  submittedAt: string
  verifiedAt?: string
  rejectionReason?: string
}

type FilterStatus = 'pending' | 'verified' | 'rejected' | 'all'

export function IdentityVerificationQueuePage() {
  const queryClient = useQueryClient()
  const { notify } = useToast()
  const [filter, setFilter] = useState<FilterStatus>('pending')
  const [rejectingId, setRejectingId] = useState<string | null>(null)
  const [reason, setReason] = useState('')

  const query = useQuery({
    queryKey: ['identity-verifications', filter],
    queryFn: () =>
      api.get<{ verifications: VerificationRow[] }>(`/admin/identity?status=${filter}`),
    staleTime: 30_000,
  })

  const approveMutation = useMutation({
    mutationFn: (id: string) => api.post(`/admin/identity/${id}/approve`, {}),
    onSuccess: () => {
      notify('Identity approved', 'success')
      void queryClient.invalidateQueries({ queryKey: ['identity-verifications'] })
    },
    onError: (err) => {
      notify(err instanceof ApiError ? err.message : 'Could not approve', 'error')
    },
  })

  const rejectMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      api.post(`/admin/identity/${id}/reject`, { reason }),
    onSuccess: () => {
      notify('Identity rejected', 'success')
      setRejectingId(null)
      setReason('')
      void queryClient.invalidateQueries({ queryKey: ['identity-verifications'] })
    },
    onError: (err) => {
      notify(err instanceof ApiError ? err.message : 'Could not reject', 'error')
    },
  })

  const rows = query.data?.verifications ?? []

  return (
    <div className="mx-auto max-w-4xl p-4 sm:p-6">
      <header className="mb-6">
        <h1 className="text-2xl font-bold text-ink">Identity verification</h1>
        <p className="mt-1 text-sm text-ink-muted">
          Review government ID submissions. Approve only after comparing the submitted details to
          the document.
        </p>
      </header>

      <div className="mb-4 flex gap-2">
        {(['pending', 'verified', 'rejected', 'all'] as FilterStatus[]).map((s) => (
          <button
            key={s}
            type="button"
            onClick={() => setFilter(s)}
            className={`rounded-lg px-3 py-1.5 text-xs font-medium ${
              filter === s ? 'bg-signal/10 text-signal' : 'text-ink-muted hover:bg-surface-hi'
            }`}
          >
            {s.charAt(0).toUpperCase() + s.slice(1)}
          </button>
        ))}
      </div>

      <Card>
        <CardHeader title={`${rows.length} ${filter === 'all' ? 'total' : filter}`} />
        <CardBody>
          {query.isLoading ? (
            <div className="flex flex-col gap-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-24 w-full" />
              ))}
            </div>
          ) : query.isError ? (
            <ErrorState
              title="Could not load submissions"
              description="Try again in a moment."
              onRetry={() => void query.refetch()}
            />
          ) : rows.length === 0 ? (
            <EmptyState
              icon={<Clock className="size-5" aria-hidden />}
              title={`No ${filter === 'all' ? '' : filter} submissions`}
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
                      <p className="text-sm font-semibold text-ink">
                        {row.userFirstName} {row.userLastName}
                      </p>
                      <p className="text-xs text-ink-muted">{row.userEmail}</p>
                      <p className="mt-2 text-xs text-ink-muted">
                        <strong className="text-ink">Document:</strong>{' '}
                        {documentTypeLabel(row.documentType)} ·{' '}
                        <strong className="text-ink">Submitted:</strong>{' '}
                        {new Date(row.submittedAt).toLocaleString()}
                      </p>
                      {row.documentUrl ? (
                        <p className="mt-1 text-xs">
                          <a
                            href={row.documentUrl}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-signal hover:underline"
                          >
                            View document photo →
                          </a>
                        </p>
                      ) : (
                        <p className="mt-1 text-xs text-ink-faint">
                          No document photo available.
                        </p>
                      )}
                      {row.status === 'rejected' && row.rejectionReason ? (
                        <p className="mt-2 text-xs text-emergency">
                          <strong>Rejected:</strong> {row.rejectionReason}
                        </p>
                      ) : null}
                      {row.status === 'verified' && row.verifiedAt ? (
                        <p className="mt-2 text-xs text-ok">
                          Verified {new Date(row.verifiedAt).toLocaleString()}
                        </p>
                      ) : null}
                    </div>

                    {row.status === 'pending' && rejectingId !== row.id ? (
                      <div className="flex shrink-0 gap-2">
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
                        Reason for rejection (shown to the user)
                      </label>
                      <textarea
                        value={reason}
                        onChange={(e) => setReason(e.target.value)}
                        rows={2}
                        maxLength={500}
                        className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-sm text-ink"
                        placeholder="e.g. The name on the document does not match your account name."
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

function documentTypeLabel(type: string): string {
  const map: Record<string, string> = {
    nin: 'NIN',
    voters_card: "Voter's Card",
    drivers_license: "Driver's License",
    passport: 'Passport',
  }
  return map[type] ?? type
}