import { CheckCircle2, CircleDot, RotateCcw } from 'lucide-react'
import { cn } from '@/lib/cn'
import { statusMeta } from '@/lib/status'
import { formatDateTime, relativeTime } from '@/lib/format'
import type { Case, CaseStatus } from '@/types/api'

interface TimelineEntry {
  status?: string
  action?: string
  description?: string
  createdAt?: string
}

interface CaseTrackerProps {
  caseItem: Case
  timeline?: TimelineEntry[]
  compact?: boolean
  className?: string
}

/**
 * The case lifecycle, rendered the same way for everyone: the reporter on
 * their case page, the officer on their workspace, and the reviewer on the
 * closure page.
 *
 * The lifecycle is not a straight line. Five field states run one way, then
 * a review phase can loop back once if an administrator requests changes.
 * The tracker renders that loop-back explicitly rather than pretending it is
 * a sixth step on a rail.
 */
export function CaseTracker({ caseItem, timeline, compact = false, className }: CaseTrackerProps) {
  const steps: { key: CaseStatus; label: string; description: string; at?: string }[] = [
    {
      key: 'pending',
      label: statusMeta('pending').label,
      description: statusMeta('pending').description,
      at: caseItem.createdAt,
    },
    {
      key: 'assigned',
      label: statusMeta('assigned').label,
      description: statusMeta('assigned').description,
      at: timeline?.find((t) => t.status === 'assigned')?.createdAt,
    },
    {
      key: 'dispatched',
      label: statusMeta('dispatched').label,
      description: statusMeta('dispatched').description,
      at: timeline?.find((t) => t.status === 'dispatched')?.createdAt,
    },
    {
      key: 'on_scene',
      label: statusMeta('on_scene').label,
      description: statusMeta('on_scene').description,
      at: timeline?.find((t) => t.status === 'on_scene')?.createdAt,
    },
    {
      key: 'investigating',
      label: statusMeta('investigating').label,
      description: statusMeta('investigating').description,
      at: timeline?.find((t) => t.status === 'investigating')?.createdAt,
    },
    {
      key: 'pending_admin_review',
      label: statusMeta('pending_admin_review').label,
      description: statusMeta('pending_admin_review').description,
      at: timeline?.find((t) => t.status === 'pending_admin_review')?.createdAt,
    },
    {
      key: 'closed',
      label: statusMeta('closed').label,
      description: statusMeta('closed').description,
      at: timeline?.find((t) => t.status === 'closed')?.createdAt,
    },
  ]

  const currentStatus = caseItem.status
  const changesRequested = currentStatus === 'admin_changes_requested'
  const currentIdx = steps.findIndex((s) => s.key === currentStatus)
  const meta = statusMeta(currentStatus)

  // Completed stage count for the compact bar.
  const doneCount = changesRequested
    ? 5
    : currentIdx >= 0
      ? currentIdx
      : 0
  const totalSteps = steps.length
  const fraction = doneCount / totalSteps

  if (compact) {
    return (
      <div
        className={cn('h-1.5 w-full overflow-hidden rounded-full bg-surface-hi', className)}
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={totalSteps}
        aria-valuenow={doneCount}
        aria-label={`Case progress: ${doneCount} of ${totalSteps}`}
      >
        <div
          className={cn('h-full rounded-full transition-all', meta.dot)}
          style={{ width: `${Math.max(2, fraction * 100)}%` }}
        />
      </div>
    )
  }

  return (
    <div className={cn('space-y-3', className)} aria-label="Case progress">
      <ol className="space-y-3">
        {steps.map((step, idx) => {
          const done = idx < doneCount
          const current = !changesRequested && idx === currentIdx
          const isChangesBranch = changesRequested && step.key === 'pending_admin_review'
          const upcoming = !done && !current && !isChangesBranch

          return (
            <li key={step.key} className="flex items-start gap-3">
              <span
                aria-hidden
                className={cn(
                  'mt-0.5 grid size-6 shrink-0 place-items-center rounded-full text-[11px] font-semibold',
                  done && 'bg-signal text-signal-ink',
                  current && 'border-2 border-signal bg-signal/15 text-signal',
                  isChangesBranch && 'border-2 border-warn bg-warn/15 text-warn',
                  upcoming && 'border border-border-hi text-ink-faint',
                )}
              >
                {done ? (
                  <CheckCircle2 className="size-3.5" />
                ) : current ? (
                  <CircleDot className="size-3.5" />
                ) : null}
              </span>
              <div className="min-w-0 flex-1">
                <p
                  className={cn(
                    'text-sm font-medium',
                    done && 'text-ink-muted',
                    current && 'text-ink',
                    isChangesBranch && 'text-warn',
                    upcoming && 'text-ink-faint',
                  )}
                >
                  {step.label}
                </p>
                <p
                  className={cn(
                    'mt-0.5 text-xs',
                    upcoming ? 'text-ink-faint' : 'text-ink-muted',
                  )}
                >
                  {step.description}
                </p>
                {step.at ? (
                  <p className="mt-0.5 text-[11px] text-ink-faint tabular">
                    {formatDateTime(step.at)} · {relativeTime(step.at)}
                  </p>
                ) : null}
              </div>
            </li>
          )
        })}
      </ol>

      {changesRequested ? (
        <div className="flex items-start gap-2 rounded-lg border border-warn/30 bg-warn/5 p-3 text-xs text-warn">
          <RotateCcw className="mt-0.5 size-3.5 shrink-0" aria-hidden />
          <span>
            An administrator asked for changes. The officer is revising the report and will
            resubmit it for closure review.
          </span>
        </div>
      ) : null}
    </div>
  )
}