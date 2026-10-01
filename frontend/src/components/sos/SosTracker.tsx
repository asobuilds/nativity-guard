import { CheckCircle2, CircleDot, TriangleAlert } from 'lucide-react'
import { cn } from '@/lib/cn'
import { formatDateTime, relativeTime } from '@/lib/format'

interface SosResponderLike {
  id: string
  unitId: string
  role: 'primary' | 'support'
  acceptedAt: string
  unit?: { id: string; name: string; type?: string }
}

interface SosLike {
  id: string
  createdAt: string
  status: string
  dispatchState?: string
  acceptedAt?: string
  acceptedByUnitId?: string
  responders?: SosResponderLike[]
}

interface SosTrackerProps {
  alert: SosLike
  compact?: boolean
  className?: string
}

/**
 * Reporter-facing SOS tracker.
 *
 * Four stages, matching the states that actually exist on the SOS model:
 * filed, acknowledged, responding, resolved. Escalation is shown as a
 * warning on the current stage rather than as its own step.
 */
export function SosTracker({ alert, compact = false, className }: SosTrackerProps) {
  const primaryResponder =
    alert.responders?.find((r) => r.role === 'primary') ?? alert.responders?.[0]
  const acknowledgedAt = alert.acceptedAt ?? primaryResponder?.acceptedAt
  const acknowledgedByName = primaryResponder?.unit?.name

  const resolved = alert.status === 'resolved'
  const dispatched = alert.status === 'dispatched' || resolved
  const acknowledged = Boolean(acknowledgedAt) || dispatched
  const escalated = alert.status === 'escalated'

  const steps = [
    {
      key: 'filed',
      label: 'Filed',
      description: 'Your SOS reached the platform.',
      at: alert.createdAt as string | undefined,
      done: true,
    },
    {
      key: 'acknowledged',
      label: 'Acknowledged',
      description: acknowledgedByName
        ? `Accepted by ${acknowledgedByName}.`
        : 'A unit has accepted your request and is preparing to respond.',
      at: acknowledgedAt,
      done: acknowledged,
    },
    {
      key: 'dispatched',
      label: 'Responding',
      description: 'The unit is on the way to your location.',
      at: undefined as string | undefined,
      done: dispatched,
    },
    {
      key: 'resolved',
      label: 'Resolved',
      description: 'The unit has cleared the emergency.',
      at: undefined as string | undefined,
      done: resolved,
    },
  ]

  const currentIdx = steps.findIndex((s) => !s.done)
  const completed = steps.filter((s) => s.done).length
  const totalSteps = steps.length
  const fraction = completed / totalSteps

  if (compact) {
    const label =
      currentIdx < 0 ? steps[totalSteps - 1].label : steps[currentIdx].label

    return (
      <div
        className={cn('w-full', className)}
        aria-label={`SOS progress: ${completed} of ${totalSteps}`}
      >
        <div className="flex items-center justify-between gap-3 text-[11px]">
          <span className="font-medium text-ink">{label}</span>
          <span className="tabular text-ink-faint">
            {completed} / {totalSteps}
          </span>
        </div>
        <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-surface-hi">
          <div
            className={cn(
              'h-full rounded-full transition-all',
              resolved ? 'bg-ok' : 'bg-emergency',
            )}
            style={{ width: `${Math.max(4, fraction * 100)}%` }}
          />
        </div>
      </div>
    )
  }

  return (
    <div className={cn('space-y-3', className)} aria-label="SOS progress">
      <ol className="space-y-3">
        {steps.map((step, idx) => {
          const done = step.done
          const current = idx === currentIdx
          const upcoming = !done && !current

          return (
            <li key={step.key} className="flex items-start gap-3">
              <span
                aria-hidden
                className={cn(
                  'mt-0.5 grid size-6 shrink-0 place-items-center rounded-full text-[11px] font-semibold',
                  done && 'bg-ok text-signal-ink',
                  current && 'border-2 border-emergency bg-emergency/15 text-emergency',
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

      {escalated ? (
        <div className="flex items-start gap-2 rounded-lg border border-warn/30 bg-warn/5 p-3 text-xs text-warn">
          <TriangleAlert className="mt-0.5 size-3.5 shrink-0" aria-hidden />
          <span>
            This SOS has been escalated to a higher authority because no unit resolved it in
            time.
          </span>
        </div>
      ) : null}
    </div>
  )
}