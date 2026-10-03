import { CheckCircle2, XCircle } from 'lucide-react'
import { cn } from '@/lib/cn'
import { initials } from '@/lib/format'
import { mediaURL } from '@/lib/apiClient'
import type { UnitCompliance } from '@/hooks/useUnitCompliance'

interface ComplianceMatrixProps {
  compliance: UnitCompliance | undefined
  isLoading: boolean
}

export function ComplianceMatrix({ compliance, isLoading }: ComplianceMatrixProps) {
  if (isLoading) {
    return <div className="h-40 animate-pulse rounded-panel bg-surface-hi/60" />
  }

  if (!compliance || compliance.officers.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border px-4 py-10 text-center text-sm text-ink-muted">
        No active officers in this unit yet.
      </div>
    )
  }

  const weeks = compliance.weeks

  return (
    <div className="space-y-3">
      <div className="overflow-x-auto rounded-panel border border-border bg-surface/60">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-surface-hi/40 text-left">
              <th className="px-3 py-2 font-medium text-ink-muted">Officer</th>
              {weeks.map((w) => (
                <th
                  key={w}
                  className="px-2 py-2 text-center text-[10px] font-medium uppercase tracking-wider text-ink-faint"
                >
                  {w.slice(5)}
                </th>
              ))}
              <th className="px-3 py-2 text-right font-medium text-ink-muted">Rate</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {compliance.officers.map((o) => {
              const avatar = mediaURL(o.avatarPath) ?? null
              const fullName = [o.firstName, o.lastName].filter(Boolean).join(' ')
              const ratePct = Math.round(o.complianceRate)
              return (
                <tr key={o.membershipId}>
                  <td className="px-3 py-2">
                    <div className="flex items-center gap-2">
                      {avatar ? (
                        <img
                          src={avatar}
                          alt=""
                          className="size-7 shrink-0 rounded-full border border-border object-cover"
                        />
                      ) : (
                        <div className="grid size-7 shrink-0 place-items-center rounded-full bg-surface-hi text-[9px] font-semibold text-ink">
                          {initials(o.firstName, o.lastName)}
                        </div>
                      )}
                      <div className="min-w-0">
                        <p className="truncate text-xs font-medium text-ink">
                          {fullName || 'Officer'}
                        </p>
                        <p className="truncate text-[10px] text-ink-faint">{o.email}</p>
                      </div>
                    </div>
                  </td>
                  {o.weekCounts.map((count, i) => (
                    <td key={i} className="px-2 py-2 text-center">
                      {count > 0 ? (
                        <CheckCircle2
                          className="mx-auto size-4 text-ok"
                          aria-label={`${count} filed`}
                        />
                      ) : (
                        <XCircle
                          className="mx-auto size-4 text-ink-faint/40"
                          aria-label="not filed"
                        />
                      )}
                    </td>
                  ))}
                  <td className="px-3 py-2 text-right">
                    <span
                      className={cn(
                        'inline-block rounded-full px-2 py-0.5 text-[11px] font-semibold tabular-nums',
                        ratePct >= 75 && 'bg-ok/15 text-ok',
                        ratePct >= 40 && ratePct < 75 && 'bg-warn/15 text-warn',
                        ratePct < 40 && 'bg-emergency/15 text-emergency',
                      )}
                    >
                      {ratePct}%
                    </span>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      <p className="text-[11px] text-ink-faint">
        Weeks run Monday to Sunday (UTC). A green tick means the officer filed at least one weekly
        update that week. Red means nothing filed.
      </p>
    </div>
  )
}
