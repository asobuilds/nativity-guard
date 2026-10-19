import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, ShieldAlert, XCircle } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorState, Skeleton } from '@/components/ui/States'
import { useToast } from '@/components/ui/Toast'
import { api, API_BASE, ApiError } from '@/lib/apiClient'
import { relativeTime } from '@/lib/format'

interface ReportPost {
  id: string
  title: string
  content: string
  createdAt: string
  author?: { name: string }
  scope?: string
  isLocked?: boolean
  /** Present when the API exposes the report's attachments on the post. */
  voiceNoteKey?: string
  videoKey?: string
}

interface ModerationReport {
  id: string
  reason: string
  status: string
  createdAt: string
  post: ReportPost
  /** Present when the API exposes the report's attachments directly. */
  voiceNoteKey?: string
  videoKey?: string
}

interface ReportsResponse {
  reports: ModerationReport[]
  count: number
}

type ResolveStatus = 'dismissed' | 'reviewed'

/**
 * Build the private media URL the file handler accepts.
 *
 * Stored keys are `<category>/<yyyy>/<mm>/<hash>.<ext>`, while
 * `GET /files/:category/:hash` resolves the yyyy/mm levels itself and enforces
 * the report access rule via `?reportId=`. The route therefore takes the bare
 * hash, not the full key.
 */
function reportMediaSrc(key: string, reportId: string): string {
  const parts = key.split('/').filter(Boolean)
  const category = parts[0] ?? 'report_media'
  const filename = parts[parts.length - 1] ?? ''
  const hash = filename.replace(/\.[^.]+$/, '')
  return `${API_BASE}/files/${encodeURIComponent(category)}/${encodeURIComponent(hash)}?reportId=${encodeURIComponent(reportId)}`
}

function ReportMedia({ report }: { report: ModerationReport }) {
  const voiceKey = report.voiceNoteKey ?? report.post.voiceNoteKey
  const videoKey = report.videoKey ?? report.post.videoKey
  if (!voiceKey && !videoKey) return null

  return (
    <div className="space-y-2">
      <p className="text-xs font-medium text-ink-muted">Attached media</p>
      {voiceKey ? (
        <audio
          controls
          preload="metadata"
          src={reportMediaSrc(voiceKey, report.id)}
          className="w-full"
        >
          Your browser does not support audio playback.
        </audio>
      ) : null}
      {videoKey ? (
        <video
          controls
          preload="metadata"
          playsInline
          src={reportMediaSrc(videoKey, report.id)}
          className="max-h-72 w-full rounded-lg border border-border bg-black"
        >
          Your browser does not support video playback.
        </video>
      ) : null}
    </div>
  )
}

/**
 * Admin moderation queue.
 *
 * `GET /community/reports` already scopes the rows to the areas the caller may
 * moderate, so this page renders what the server returns rather than filtering
 * client-side. Resolving a report is terminal and deliberately does not remove
 * the post — takedown is a separate capability (RES-015).
 */
export function AdminReportsPage() {
  const client = useQueryClient()
  const { notify } = useToast()

  const reports = useQuery({
    queryKey: ['community', 'reports', 'pending'],
    queryFn: () => api.get<ReportsResponse>('/community/reports?status=pending'),
  })

  const resolve = useMutation({
    mutationFn: (input: { id: string; status: ResolveStatus }) =>
      api.post(`/community/reports/${input.id}/resolve`, { status: input.status }),
    onSuccess: (_data, input) => {
      notify(
        input.status === 'reviewed' ? 'Report marked reviewed' : 'Report dismissed',
        'success',
      )
      void client.invalidateQueries({ queryKey: ['community', 'reports'] })
    },
    onError: (error) => {
      notify(
        error instanceof ApiError ? error.message : 'Could not resolve the report',
        'error',
      )
    },
  })

  const rows = reports.data?.reports ?? []

  return (
    <div className="mx-auto w-full max-w-3xl space-y-4 p-4 sm:p-6">
      <header className="flex items-start gap-3">
        <span className="grid size-11 shrink-0 place-items-center rounded-full border border-signal/30 bg-signal/10 text-signal">
          <ShieldAlert className="size-5" aria-hidden />
        </span>
        <div>
          <h1 className="text-xl font-semibold text-ink">Moderation</h1>
          <p className="mt-1 text-sm text-ink-muted">
            Reports for posts in the areas you moderate, oldest first.
          </p>
        </div>
      </header>

      <Card className="p-4">
        <p className="text-xs text-ink-muted">
          Resolving a report records the decision; it does not remove the post. Dismiss
          when the content does not break the rules, or mark it reviewed once you have
          acted on it.
        </p>
      </Card>

      {reports.isLoading ? (
        <div className="space-y-3">
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-40 w-full" />
        </div>
      ) : reports.isError ? (
        <Card>
          <ErrorState
            title="Could not load reports"
            description="Reconnect and try again."
            onRetry={() => void reports.refetch()}
          />
        </Card>
      ) : !rows.length ? (
        <Card>
          <EmptyState
            title="No pending reports"
            description="Nothing needs moderation right now."
          />
        </Card>
      ) : (
        <ul className="space-y-3">
          {rows.map((report) => {
            const active = resolve.isPending && resolve.variables?.id === report.id
            return (
              <li key={report.id}>
                <Card as="article" className="space-y-3 p-4">
                  <div className="flex flex-wrap items-center gap-2">
                    <h2 className="font-semibold text-ink">{report.post.title}</h2>
                    <span className="rounded-full bg-surface-hi px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-ink-muted ring-1 ring-border">
                      {report.status}
                    </span>
                  </div>
                  <p className="text-xs text-ink-faint">
                    {report.post.author?.name ?? 'Community member'} · reported{' '}
                    {relativeTime(report.createdAt)}
                  </p>

                  <p className="whitespace-pre-wrap text-sm text-ink-muted">
                    {report.post.content}
                  </p>

                  <div className="rounded-lg border border-warn/30 bg-warn/5 p-3">
                    <p className="text-xs font-semibold uppercase tracking-wide text-warn">
                      Reason
                    </p>
                    <p className="mt-1 whitespace-pre-wrap text-sm text-ink">
                      {report.reason || 'No reason given.'}
                    </p>
                  </div>

                  <ReportMedia report={report} />

                  {report.post.isLocked ? (
                    <p className="text-xs text-ink-muted">This discussion is locked.</p>
                  ) : null}

                  <div className="flex flex-wrap gap-2 border-t border-border pt-3">
                    <Button
                      size="sm"
                      variant="secondary"
                      disabled={resolve.isPending}
                      loading={active && resolve.variables?.status === 'dismissed'}
                      icon={<XCircle className="size-4" />}
                      onClick={() => resolve.mutate({ id: report.id, status: 'dismissed' })}
                    >
                      Dismiss
                    </Button>
                    <Button
                      size="sm"
                      variant="primary"
                      disabled={resolve.isPending}
                      loading={active && resolve.variables?.status === 'reviewed'}
                      icon={<CheckCircle2 className="size-4" />}
                      onClick={() => resolve.mutate({ id: report.id, status: 'reviewed' })}
                    >
                      Mark reviewed
                    </Button>
                  </div>
                </Card>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
