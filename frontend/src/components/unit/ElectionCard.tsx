import { AlertTriangle, CalendarClock, Users } from 'lucide-react'
import { cn } from '@/lib/cn'
import { formatDate } from '@/lib/format'
import type { Election } from '@/hooks/useUnitGovernance'

interface ElectionCardProps {
  election: Election
  onOpenVote?: (election: Election) => void
  onClose?: (election: Election) => void
  canManage?: boolean
  closing?: boolean
}

/**
 * One election, whether open or finalized. Open elections show days left to
 * vote and an action button. Finalized elections show the outcome.
 */
export function ElectionCard({
  election,
  onOpenVote,
  onClose,
  canManage,
  closing,
}: ElectionCardProps) {
  const isOpen = election.status === 'open'
  const lowTurnout = election.status === 'finalized_low_turnout'
  const finalized = election.status === 'finalized' || lowTurnout
  const votingEnds = election.votingEndsAt ? new Date(election.votingEndsAt) : null
  const daysLeft = votingEnds
    ? Math.max(0, Math.round((votingEnds.getTime() - Date.now()) / 86_400_000))
    : null

  return (
    <div
      className={cn(
        'rounded-lg border p-4 transition-colors',
        isOpen ? 'border-signal/40 bg-signal/5' : 'border-border bg-surface/60',
      )}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="rounded-full bg-signal/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-signal">
              {election.electionType === 'head_admin' ? 'Head-admin election' : 'Admin election'}
            </span>
            <span
              className={cn(
                'rounded-full px-2 py-0.5 text-[10px] font-medium uppercase tracking-wider',
                isOpen && 'bg-signal/15 text-signal',
                finalized && !lowTurnout && 'bg-ok/15 text-ok',
                lowTurnout && 'bg-warn/15 text-warn',
                !isOpen && !finalized && 'bg-surface-hi text-ink-muted',
              )}
            >
              {isOpen ? 'Open' : lowTurnout ? 'Finalized · low turnout' : 'Finalized'}
            </span>
            {election.extendedOnce && isOpen ? (
              <span className="inline-flex items-center gap-1 rounded-full bg-warn/15 px-2 py-0.5 text-[10px] font-medium text-warn">
                <AlertTriangle className="size-3" aria-hidden />
                Extended once
              </span>
            ) : null}
          </div>

          <div className="mt-2 flex flex-wrap gap-3 text-xs text-ink-muted">
            <span className="flex items-center gap-1.5">
              <Users className="size-3.5" aria-hidden />
              {election.seatCount} seat{election.seatCount === 1 ? '' : 's'} ·{' '}
              {election.eligibleVoterCount} eligible
            </span>
            <span className="flex items-center gap-1.5">
              <CalendarClock className="size-3.5" aria-hidden />
              Quorum {election.quorumCount}
            </span>
          </div>

          {isOpen && votingEnds ? (
            <p className="mt-2 text-xs text-ink-faint">
              {daysLeft === 0
                ? 'Voting window closes today'
                : `Voting closes in ${daysLeft} day${daysLeft === 1 ? '' : 's'} · ${formatDate(election.votingEndsAt!)}`}
            </p>
          ) : null}

          {finalized ? (
            <p className="mt-2 text-xs text-ink-faint">
              Term {formatDate(election.termStart)} – {formatDate(election.termEnd)}
              {election.quorumMet ? ' · quorum met' : ' · quorum not met'}
            </p>
          ) : null}
        </div>

        {isOpen ? (
          <div className="flex shrink-0 flex-wrap gap-2">
            {onOpenVote ? (
              <button
                type="button"
                onClick={() => onOpenVote(election)}
                className="rounded-lg bg-signal px-3 py-1.5 text-xs font-semibold text-signal-ink transition-colors hover:bg-signal/90"
              >
                Vote
              </button>
            ) : null}
            {canManage && onClose ? (
              <button
                type="button"
                onClick={() => onClose(election)}
                disabled={closing}
                className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-ink-muted transition-colors hover:bg-surface-hi disabled:opacity-50"
              >
                Close election
              </button>
            ) : null}
          </div>
        ) : null}
      </div>
    </div>
  )
}
