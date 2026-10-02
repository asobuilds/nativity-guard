import { useState, type FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, MessageSquare, Send } from 'lucide-react'
import { Card, CardBody, CardHeader } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useToast } from '@/components/ui/Toast'
import { api, ApiError } from '@/lib/apiClient'
import { cn } from '@/lib/cn'
import { formatDateTime, relativeTime } from '@/lib/format'

interface Feedback {
  id: string
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

const CATEGORIES = [
  { value: 'bug', label: 'Report a bug' },
  { value: 'feature', label: 'Suggest a feature' },
  { value: 'complaint', label: 'File a complaint' },
  { value: 'support', label: 'Ask for help' },
  { value: 'other', label: 'Something else' },
]

const STATUS_STYLE: Record<string, string> = {
  open: 'bg-signal/15 text-signal',
  in_review: 'bg-warn/15 text-warn',
  resolved: 'bg-ok/15 text-ok',
  closed: 'bg-surface-hi text-ink-muted',
}

export function FeedbackPage() {
  const queryClient = useQueryClient()
  const { notify } = useToast()

  const [category, setCategory] = useState('bug')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [contactEmail, setContactEmail] = useState('')

  const query = useQuery({
    queryKey: ['my-feedback'],
    queryFn: () => api.get<{ feedback: Feedback[] }>('/feedback/my'),
    staleTime: 30_000,
  })

  const submit = useMutation({
    mutationFn: () =>
      api.post<{ feedback: Feedback }>('/feedback', {
        category,
        subject: subject.trim(),
        body: body.trim(),
        contactEmail: contactEmail.trim() || undefined,
      }),
    onSuccess: () => {
      notify('Thank you — your feedback was received.', 'success')
      setSubject('')
      setBody('')
      setContactEmail('')
      void queryClient.invalidateQueries({ queryKey: ['my-feedback'] })
    },
    onError: (err) => {
      notify(err instanceof ApiError ? err.message : 'Could not send feedback', 'error')
    },
  })

  function onSubmit(e: FormEvent) {
    e.preventDefault()
    if (!subject.trim() || !body.trim()) return
    submit.mutate()
  }

  const rows = query.data?.feedback ?? []

  return (
    <div className="mx-auto w-full max-w-3xl space-y-5 p-4 sm:p-6">
      <header>
        <h1 className="text-2xl font-bold text-ink">Feedback & support</h1>
        <p className="mt-1 text-sm text-ink-muted">
          Report a bug, ask for help, request a feature, or tell us what is not working. Every
          submission reaches the platform team and gets a written reply.
        </p>
      </header>

      <Card>
        <CardHeader title="Send us a message" />
        <CardBody>
          <form className="flex flex-col gap-4" onSubmit={onSubmit} noValidate>
            <Field label="What is this about?" required>
              {({ id, ...aria }) => (
                <Select
                  id={id}
                  {...aria}
                  value={category}
                  onChange={(e) => setCategory(e.target.value)}
                >
                  {CATEGORIES.map((c) => (
                    <option key={c.value} value={c.value}>
                      {c.label}
                    </option>
                  ))}
                </Select>
              )}
            </Field>

            <Field
              label="Subject"
              required
              hint="One line that names the issue — this is what we sort by."
            >
              {({ id, ...aria }) => (
                <Input
                  id={id}
                  {...aria}
                  value={subject}
                  maxLength={200}
                  onChange={(e) => setSubject(e.target.value)}
                  placeholder="e.g. Map does not load on the report page"
                />
              )}
            </Field>

            <Field
              label="Details"
              required
              hint="What you were doing, what you saw, and what you expected."
            >
              {({ id, ...aria }) => (
                <Textarea
                  id={id}
                  {...aria}
                  rows={6}
                  maxLength={5000}
                  value={body}
                  onChange={(e) => setBody(e.target.value)}
                  placeholder="Steps to reproduce, screenshots described, or the full story…"
                />
              )}
            </Field>

            <Field
              label="Contact email (optional)"
              hint="Only if you want a reply to a different address. We will message you here in the app either way."
            >
              {({ id, ...aria }) => (
                <Input
                  id={id}
                  {...aria}
                  type="email"
                  value={contactEmail}
                  maxLength={200}
                  onChange={(e) => setContactEmail(e.target.value)}
                  placeholder="you@example.com"
                />
              )}
            </Field>

            <div className="flex justify-end">
              <Button
                type="submit"
                variant="primary"
                icon={<Send className="size-4" aria-hidden />}
                loading={submit.isPending}
                disabled={submit.isPending || !subject.trim() || !body.trim()}
              >
                Send
              </Button>
            </div>
          </form>
        </CardBody>
      </Card>

      <Card>
        <CardHeader title="Your submissions" subtitle="Newest first" />
        <CardBody>
          {query.isLoading ? (
            <div className="flex flex-col gap-3">
              {[0, 1, 2].map((i) => (
                <Skeleton key={i} className="h-20 w-full" />
              ))}
            </div>
          ) : query.isError ? (
            <ErrorState
              title="Could not load your submissions"
              description="Try again in a moment."
              onRetry={() => void query.refetch()}
            />
          ) : rows.length === 0 ? (
            <EmptyState
              icon={<MessageSquare className="size-5" aria-hidden />}
              title="Nothing submitted yet"
              description="Your messages to the team will appear here with their replies."
            />
          ) : (
            <ul className="flex flex-col gap-3">
              {rows.map((item) => (
                <li
                  key={item.id}
                  className="rounded-lg border border-border bg-surface-hi/30 p-4"
                >
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-semibold text-ink">{item.subject}</p>
                      <p className="mt-0.5 text-xs text-ink-muted">
                        {labelForCategory(item.category)} · submitted{' '}
                        {formatDateTime(item.createdAt)} ({relativeTime(item.createdAt)})
                      </p>
                      <p className="mt-2 whitespace-pre-wrap text-sm text-ink-muted">{item.body}</p>
                    </div>
                    <span
                      className={cn(
                        'rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase',
                        STATUS_STYLE[item.status] ?? STATUS_STYLE.closed,
                      )}
                    >
                      {item.status.replace('_', ' ')}
                    </span>
                  </div>

                  {item.adminReply ? (
                    <div className="mt-3 rounded-lg border border-signal/30 bg-signal/5 p-3">
                      <p className="flex items-center gap-1.5 text-xs font-medium text-signal">
                        <CheckCircle2 className="size-3.5" aria-hidden />
                        Reply from the team
                        {item.repliedAt ? (
                          <span className="text-ink-faint">
                            {' '}
                            · {relativeTime(item.repliedAt)}
                          </span>
                        ) : null}
                      </p>
                      <p className="mt-1.5 whitespace-pre-wrap text-sm text-ink">
                        {item.adminReply}
                      </p>
                    </div>
                  ) : (
                    <p className="mt-3 text-xs text-ink-faint">
                      Awaiting a reply from the team.
                    </p>
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

function labelForCategory(value: string): string {
  const map: Record<string, string> = {
    bug: 'Bug report',
    feature: 'Feature request',
    complaint: 'Complaint',
    support: 'Support',
    other: 'Other',
  }
  return map[value] ?? value
}