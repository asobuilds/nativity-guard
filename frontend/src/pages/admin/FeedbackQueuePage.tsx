import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, MessageSquare, Send, X } from 'lucide-react'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useToast } from '@/components/ui/Toast'
import { api, ApiError } from '@/lib/apiClient'
import { cn } from '@/lib/cn'
import { relativeTime } from '@/lib/format'

interface FeedbackRow {
  id: string
  userId: string
  userEmail: string
  userFirstName: string
  userLastName: string
  category: string
  priority: string
  status: string
  subject: string
  body: string
  contactEmail?: string
  adminReply?: string
  repliedAt?: string
  closedAt?: string
  createdAt: string
}

type Filter = 'open' | 'in_review' | 'closed' | 'all'
type CategoryFilter = 'all' | 'bug' | 'feature' | 'complaint' | 'support' | 'other'

export function FeedbackQueuePage() {
  const queryClient = useQueryClient()
  const { notify } = useToast()
  const [statusFilter, setStatusFilter] = useState<Filter>('open')
  const [categoryFilter, setCategoryFilter] = useState<CategoryFilter>('all')
  const [replyingId, setReplyingId] = useState<string | null>(null)
  const [reply, setReply] = useState('')

  const query = useQuery({
    queryKey: ['admin-feedback', statusFilter, categoryFilter],
    queryFn: () =>
      api.get<{ feedback: FeedbackRow[] }>(
        `/admin/feedback?status=${statusFilter}&category=${categoryFilter}`,
      ),
    staleTime: 30_000,
  })

  const replyMutation = useMutation({
    mutationFn: ({ id, reply }: { id: string; reply: string }) =>
      api.post(`/admin/feedback/${id}/reply`, { reply }),
    onSuccess: () => {
      notify('Reply sent', 'success')
      setReplyingId(null)
      setReply('')
      void queryClient.invalidateQueries({ queryKey: ['admin-feedback'] })
    },
    onError: (err) => {
      notify(err instanceof ApiError ? err.message : 'Could not send reply', 'error')
    },
  })

  const closeMutation = useMutation({
    mutationFn: (id: string) => api.post(`/admin/feedback/${id}/close`, {}),
    onSuccess: () => {
      notify('Feedback closed', 'success')
      void queryClient.invalidateQueries({ queryKey: ['admin-feedback'] })
    },
    onError: (err) => {
      notify(err instanceof ApiError ? err.message : 'Could not close', 'error')
    },
  })

  const rows = query.data?.feedback ?? []

  return (
    <div className="mx-auto w-full max-w-4xl p-4 sm:p-6">
      <header className="mb-6">
        <h1 className="text-2xl font-bold text-ink">Feedback & support</h1>
        <p className="mt-1 text-sm text-ink-muted">
          Every user submission, sorted by priority and time. Reply once, then close.
        </p>
      </header>

      <div className="mb-4 flex flex-wrap gap-2">
        {(['open', 'in_review', 'closed', 'all'] as Filter[]).map((s) => (
          <button
            key={s}
            type="button"
            onClick={() => setStatusFilter(s)}
            className={cn(
              'rounded-lg px-3 py-1.5 text-xs font-medium transition-colors',
              statusFilter === s
                ? 'bg-signal/10 text-signal'
                : 'text-ink-muted hover:bg-surface-hi',
            )}
          >
            {s === 'in_review' ? 'In review' : s.charAt(0).toUpperCase() + s.slice(1)}
          </button>
        ))}
      </div>

      <div className="mb-4 flex flex-wrap gap-2">
        {(['all', 'bug', 'feature', 'complaint', 'support', 'other'] as CategoryFilter[]).map(
          (c) => (
            <button
              key={c}
              type="button"
              onClick={() => setCategoryFilter(c)}
              className={cn(
                'rounded-full px-3 py-1 text-[11px] font-medium transition-colors',
                categoryFilter === c
                  ? 'bg-signal/15 text-signal ring-1 ring-signal/30'
                  : 'bg-surface-hi text-ink-muted hover:text-ink',
              )}
            >
              {c === 'all' ? 'All categories' : c}
            </button>
          ),
        )}
      </div>

      <Card>
        <CardHeader title={`${rows.length} ${statusFilter === 'all' ? 'total' : statusFilter}`} />
        <CardBody>
          {query.isLoading ? (
            <div className="flex flex-col gap-3">
              {[0, 1, 2].map((i) => (
                <Skeleton key={i} className="h-32 w-full" />
              ))}
            </div>
          ) : query.isError ? (
            <ErrorState
              title="Could not load feedback"
              description="Try again in a moment."
              onRetry={() => void query.refetch()}
            />
          ) : rows.length === 0 ? (
            <EmptyState
              icon={<MessageSquare className="size-5" aria-hidden />}
              title="Nothing to review"
              description="No feedback matches these filters."
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
                      <div className="flex flex-wrap items-center gap-2">
                        <p className="text-sm font-semibold text-ink">{row.subject}</p>
                        <span
                          className={cn(
                            'rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase',
                            row.priority === 'high' && 'bg-emergency/15 text-emergency',
                            row.priority === 'normal' && 'bg-signal/15 text-signal',
                            row.priority === 'low' && 'bg-surface-hi text-ink-muted',
                          )}
                        >
                          {row.priority}
                        </span>
                        <span
                          className={cn(
                            'rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase',
                            row.status === 'open' && 'bg-signal/15 text-signal',
                            row.status === 'in_review' && 'bg-warn/15 text-warn',
                            row.status === 'resolved' && 'bg-ok/15 text-ok',
                            row.status === 'closed' && 'bg-surface-hi text-ink-muted',
                          )}
                        >
                          {row.status.replace('_', ' ')}
                        </span>
                      </div>

                      <p className="mt-1 text-xs text-ink-muted">
                        <strong className="text-ink">
                          {row.userFirstName} {row.userLastName}
                        </strong>{' '}
                        · {row.userEmail}
                        {row.contactEmail && row.contactEmail !== row.userEmail ? (
                          <>
                            {' '}
                            · reply-to <span className="text-signal">{row.contactEmail}</span>
                          </>
                        ) : null}
                        {' · '}submitted {relativeTime(row.createdAt)}
                      </p>

                      <p className="mt-2 whitespace-pre-wrap text-sm text-ink-muted">
                        {row.body}
                      </p>
                    </div>
                  </div>

                  {row.adminReply ? (
                    <div className="mt-3 rounded-lg border border-signal/30 bg-signal/5 p-3">
                      <p className="text-xs font-medium text-signal">Your reply</p>
                      <p className="mt-1 whitespace-pre-wrap text-sm text-ink">
                        {row.adminReply}
                      </p>
                    </div>
                  ) : null}

                  {replyingId === row.id ? (
                    <div className="mt-3 rounded-lg border border-border bg-surface p-3">
                      <label className="mb-1 block text-xs font-medium text-ink">
                        Reply to the user
                      </label>
                      <textarea
                        value={reply}
                        onChange={(e) => setReply(e.target.value)}
                        rows={3}
                        maxLength={5000}
                        className="w-full rounded-lg border border-border bg-surface-hi px-3 py-2 text-sm text-ink"
                        placeholder="Explain what changed, or ask for more detail."
                      />
                      <div className="mt-2 flex items-center justify-end gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setReplyingId(null)
                            setReply('')
                          }}
                        >
                          Cancel
                        </Button>
                        <Button
                          variant="primary"
                          size="sm"
                          disabled={!reply.trim() || replyMutation.isPending}
                          loading={replyMutation.isPending}
                          onClick={() =>
                            replyMutation.mutate({ id: row.id, reply: reply.trim() })
                          }
                        >
                          Send reply
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <div className="mt-3 flex flex-wrap gap-2">
                      {row.status !== 'closed' ? (
                        <>
                          <Button
                            variant="primary"
                            size="sm"
                            icon={<Send className="size-3.5" aria-hidden />}
                            onClick={() => {
                              setReplyingId(row.id)
                              setReply(row.adminReply ?? '')
                            }}
                          >
                            {row.adminReply ? 'Edit reply' : 'Reply'}
                          </Button>
                          {row.adminReply ? (
                            <Button
                              variant="secondary"
                              size="sm"
                              icon={<Check className="size-3.5" aria-hidden />}
                              loading={
                                closeMutation.isPending && closeMutation.variables === row.id
                              }
                              onClick={() => closeMutation.mutate(row.id)}
                            >
                              Close
                            </Button>
                          ) : null}
                          <Button
                            variant="ghost"
                            size="sm"
                            icon={<X className="size-3.5" aria-hidden />}
                            loading={
                              closeMutation.isPending && closeMutation.variables === row.id
                            }
                            onClick={() => closeMutation.mutate(row.id)}
                          >
                            Close without reply
                          </Button>
                        </>
                      ) : null}
                    </div>
                  )}
                </li>
              ))}
            </ul>
          )}
        </CardBody>
      </Card>
    </div>
  )
}