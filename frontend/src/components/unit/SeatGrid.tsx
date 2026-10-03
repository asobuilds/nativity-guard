import { Crown, User } from 'lucide-react'
import { cn } from '@/lib/cn'
import { formatDate } from '@/lib/format'
import type { AdminSeat, GovernanceAdmin } from '@/hooks/useUnitGovernance'

interface SeatGridProps {
  seats: AdminSeat[]
  admins: GovernanceAdmin[]
}

/**
 * A compact grid of the unit's elected admin seats. Shows seat number,
 * current holder (or "vacant"), term end date, and a term-progress bar.
 */
export function SeatGrid({ seats, admins }: SeatGridProps) {
  if (seats.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border px-4 py-8 text-center text-xs text-ink-muted">
        No seats yet. An election creates the seat grid.
      </div>
    )
  }

  const holderByMemberId = new Map<string, GovernanceAdmin>()
  for (const a of admins) {
    holderByMemberId.set(a.userId, a)
  }

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {seats.map((seat) => {
        const holder = seat.memberId ? holderByMemberId.get(seat.memberId) : undefined
        const vacant = seat.status === 'vacant' || !holder
        const termStart = new Date(seat.termStart).getTime()
        const termEnd = new Date(seat.termEnd).getTime()
        const now = Date.now()
        const total = Math.max(1, termEnd - termStart)
        const done = Math.min(total, Math.max(0, now - termStart))
        const pct = Math.round((done / total) * 100)
        const daysLeft = Math.max(0, Math.round((termEnd - now) / 86_400_000))

        return (
          <div
            key={seat.id}
            className={cn(
              'rounded-lg border bg-surface/70 p-3 transition-colors',
              vacant ? 'border-dashed border-border' : 'border-border hover:border-border-hi',
            )}
          >
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wider text-ink-faint">
                {holder?.isHeadAdmin ? (
                  <Crown className="size-3.5 text-warn" aria-hidden />
                ) : (
                  <User className="size-3.5" aria-hidden />
                )}
                Seat {seat.seatNumber}
              </span>
              <span
                className={cn(
                  'rounded-full px-2 py-0.5 text-[10px] font-medium',
                  vacant ? 'bg-surface-hi text-ink-faint' : 'bg-signal/15 text-signal',
                )}
              >
                {vacant ? 'Vacant' : 'Active'}
              </span>
            </div>

            {vacant ? (
              <p className="mt-2 text-xs text-ink-muted">Awaiting vacancy fill</p>
            ) : (
              <>
                <p className="mt-2 truncate text-sm font-medium text-ink">
                  {[holder.firstName, holder.lastName].filter(Boolean).join(' ')}
                </p>
                <p className="truncate text-[11px] text-ink-faint">{holder.email}</p>
                <div className="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-surface-hi">
                  <div
                    className="h-full rounded-full bg-signal transition-all"
                    style={{ width: `${pct}%` }}
                  />
                </div>
                <p className="mt-1 text-[10px] text-ink-faint">
                  {daysLeft > 0 ? `${daysLeft} days left · ends ${formatDate(seat.termEnd)}` : `ended ${formatDate(seat.termEnd)}`}
                </p>
              </>
            )}
          </div>
        )
      })}
    </div>
  )
}
