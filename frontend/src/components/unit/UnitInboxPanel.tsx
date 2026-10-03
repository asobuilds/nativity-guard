import { Link } from 'react-router-dom'
import { Check, FileText, ShieldAlert, UserPlus, X } from 'lucide-react'
import { cn } from '@/lib/cn'
import { initials, relativeTime } from '@/lib/format'
import { mediaURL } from '@/lib/apiClient'
import { PriorityChip, StatusChip } from '@/components/ui/Chips'
import type { UnitInbox } from '@/hooks/useUnitInbox'

interface UnitInboxPanelProps {
  inbox: UnitInbox | undefined
  isLoading: boolean
  onApproveApplication?: (membershipId: string) => void
  onRejectApplication?: (membershipId: string) => void
  actionBusy?: boolean
}

export function UnitInboxPanel({
  inbox,
  isLoading,
  onApproveApplication,
  onRejectApplication,
  actionBusy,
}: UnitInboxPanelProps) {
  if (isLoading) {
    return (
      <div className="space-y-4">
        {[0, 1, 2].map((i) => (
          <div key={i} className="h-32 animate-pulse rounded-panel bg-surface-hi/60" />
        ))}
      </div>
    )
  }

  if (!inbox) return null

  return (
    <div className="space-y-4">
      <Section
        icon={<FileText className="size-4 text-signal" aria-hidden />}
        title="Pending cases"
        count={inbox.pendingCaseCount}
      >
        {inbox.pendingCases.length === 0 ? (
          <Empty text="No cases awaiting action." />
        ) : (
          <ul className="divide-y divide-border">
            {inbox.pendingCases.map((c) => (
              <li key={c.id} className="flex items-center gap-3 p-3">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <PriorityChip level={c.priorityLevel} />
                    <StatusChip status={c.status} />
                  </div>
                  <Link
                    to={`/admin/cases/${c.id}`}
                    className="mt-1 block truncate text-sm font-medium text-ink hover:text-signal"
                  >
                    {c.title}
                  </Link>
                  <p className="text-xs text-ink-faint">
                    {c.trackingId} · {c.location || 'no location'} · {relativeTime(c.createdAt)}
                  </p>
                </div>
                <Link
                  to={`/admin/cases/${c.id}`}
                  className="shrink-0 text-xs font-medium text-signal hover:underline"
                >
                  Review →
                </Link>
              </li>
            ))}
          </ul>
        )}
      </Section>

      <Section
        icon={<ShieldAlert className="size-4 text-emergency" aria-hidden />}
        title="Unclaimed SOS alerts"
        count={inbox.sosAlertCount}
      >
        {inbox.sosAlerts.length === 0 ? (
          <Empty text="No SOS alerts waiting." />
        ) : (
          <ul className="divide-y divide-border">
            {inbox.sosAlerts.map((s) => (
              <li key={s.id} className="flex items-center gap-3 p-3">
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-ink">
                    {s.description || 'SOS alert — no description'}
                  </p>
                  <p className="text-xs text-ink-faint">
                    {relativeTime(s.createdAt)} · {s.latitude.toFixed(4)},{' '}
                    {s.longitude.toFixed(4)}
                  </p>
                </div>
                <Link
                  to={`/sos/${s.id}`}
                  className="shrink-0 text-xs font-medium text-signal hover:underline"
                >
                  Open →
                </Link>
              </li>
            ))}
          </ul>
        )}
      </Section>

      <Section
        icon={<UserPlus className="size-4 text-signal" aria-hidden />}
        title="Membership applications"
        count={inbox.applicationCount}
      >
        {inbox.applications.length === 0 ? (
          <Empty text="No pending applications." />
        ) : (
          <ul className="divide-y divide-border">
            {inbox.applications.map((a) => {
              const avatar = mediaURL(a.avatarPath) ?? null
              const fullName = [a.firstName, a.lastName].filter(Boolean).join(' ')
              return (
                <li key={a.membershipId} className="flex flex-wrap items-center gap-3 p-3">
                  {avatar ? (
                    <img
                      src={avatar}
                      alt=""
                      className="size-9 shrink-0 rounded-full border border-border object-cover"
                    />
                  ) : (
                    <div className="grid size-9 shrink-0 place-items-center rounded-full bg-surface-hi text-[10px] font-semibold text-ink">
                      {initials(a.firstName, a.lastName)}
                    </div>
                  )}
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium text-ink">
                      {fullName || 'Applicant'}
                    </p>
                    <p className="truncate text-xs text-ink-faint">
                      {a.email} · applied {relativeTime(a.createdAt)}
                    </p>
                  </div>
                  {onApproveApplication && onRejectApplication ? (
                    <div className="flex shrink-0 gap-1.5">
                      <button
                        type="button"
                        disabled={actionBusy}
                        onClick={() => onApproveApplication(a.membershipId)}
                        aria-label="Approve application"
                        className="inline-flex items-center gap-1 rounded-lg bg-signal px-2.5 py-1.5 text-[11px] font-medium text-signal-ink transition-colors hover:bg-signal/90 disabled:opacity-50"
                      >
                        <Check className="size-3" aria-hidden />
                        Approve
                      </button>
                      <button
                        type="button"
                        disabled={actionBusy}
                        onClick={() => onRejectApplication(a.membershipId)}
                        aria-label="Reject application"
                        className="inline-flex items-center gap-1 rounded-lg border border-border px-2.5 py-1.5 text-[11px] font-medium text-ink-muted transition-colors hover:bg-surface-hi disabled:opacity-50"
                      >
                        <X className="size-3" aria-hidden />
                        Reject
                      </button>
                    </div>
                  ) : null}
                </li>
              )
            })}
          </ul>
        )}
      </Section>
    </div>
  )
}

function Section({
  icon,
  title,
  count,
  children,
}: {
  icon: React.ReactNode
  title: string
  count: number
  children: React.ReactNode
}) {
  return (
    <div className="overflow-hidden rounded-panel border border-border bg-surface/70">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          {icon}
          <h2 className="text-sm font-semibold text-ink">{title}</h2>
        </div>
        <span
          className={cn(
            'rounded-full px-2 py-0.5 text-[11px] font-semibold tabular-nums',
            count > 0 ? 'bg-signal/15 text-signal' : 'bg-surface-hi text-ink-faint',
          )}
        >
          {count}
        </span>
      </div>
      {children}
    </div>
  )
}

function Empty({ text }: { text: string }) {
  return <p className="px-4 py-6 text-center text-xs text-ink-muted">{text}</p>
}
