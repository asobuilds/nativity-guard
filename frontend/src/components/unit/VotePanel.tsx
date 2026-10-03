import { useState } from 'react'
import { CheckCircle2, X } from 'lucide-react'
import { cn } from '@/lib/cn'
import { initials } from '@/lib/format'
import { mediaURL } from '@/lib/apiClient'
import type { Election } from '@/hooks/useUnitGovernance'
import type { RosterMember } from '@/hooks/useUnitRoster'

interface VotePanelProps {
  open: boolean
  election: Election
  candidates: RosterMember[]
  onVote: (candidateUserId: string) => void
  onClose: () => void
  submitting: boolean
  myVoteCandidateId?: string
}

/**
 * Modal that lists every verified candidate in the unit and lets the caller
 * pick one. Once a vote is registered, the chosen candidate is highlighted
 * and the rest are disabled — the backend already refuses a second vote,
 * this just makes the state visible.
 */
export function VotePanel({
  open,
  election,
  candidates,
  onVote,
  onClose,
  submitting,
  myVoteCandidateId,
}: VotePanelProps) {
  const [selected, setSelected] = useState<string | null>(myVoteCandidateId ?? null)

  if (!open) return null

  const hasVoted = Boolean(myVoteCandidateId)

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Cast your vote"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="flex max-h-[85vh] w-full max-w-lg flex-col overflow-hidden rounded-3xl border border-border bg-base shadow-2xl">
        <header className="flex items-start justify-between border-b border-border p-4">
          <div>
            <h3 className="text-base font-semibold text-ink">
              {election.electionType === 'head_admin' ? 'Head-admin election' : 'Admin election'}
            </h3>
            <p className="mt-0.5 text-xs text-ink-muted">
              {hasVoted
                ? 'Your vote is recorded. Thank you.'
                : `Choose one candidate. ${election.seatCount} seat${
                    election.seatCount === 1 ? '' : 's'
                  } will be filled.`}
            </p>
          </div>
          <button
            type="button"
            aria-label="Close"
            onClick={onClose}
            className="grid size-8 place-items-center rounded-full text-ink-muted hover:bg-surface-hi"
          >
            <X className="size-4" aria-hidden />
          </button>
        </header>

        <div className="overflow-y-auto p-3">
          {candidates.length === 0 ? (
            <p className="p-6 text-center text-sm text-ink-muted">
              No verified candidates available to vote for.
            </p>
          ) : (
            <ul className="space-y-2">
              {candidates.map((c) => {
                const avatar = mediaURL(c.avatarPath) ?? null
                const fullName = [c.firstName, c.lastName].filter(Boolean).join(' ')
                const isMine = myVoteCandidateId === c.userId
                const isChosen = selected === c.userId
                return (
                  <li key={c.membershipId}>
                    <button
                      type="button"
                      disabled={hasVoted || submitting}
                      onClick={() => setSelected(c.userId)}
                      className={cn(
                        'flex w-full items-center gap-3 rounded-xl border p-3 text-left transition-all',
                        isMine && 'border-ok/50 bg-ok/5',
                        !isMine && isChosen && 'border-signal bg-signal/5',
                        !isMine && !isChosen && 'border-border hover:border-border-hi',
                        hasVoted && !isMine && 'opacity-50',
                      )}
                    >
                      {avatar ? (
                        <img
                          src={avatar}
                          alt=""
                          className="size-10 shrink-0 rounded-full border border-border object-cover"
                        />
                      ) : (
                        <div className="grid size-10 shrink-0 place-items-center rounded-full bg-surface-hi text-xs font-semibold text-ink">
                          {initials(c.firstName, c.lastName)}
                        </div>
                      )}
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium text-ink">
                          {fullName || 'Member'}
                        </p>
                        <p className="truncate text-xs text-ink-faint">{c.email}</p>
                      </div>
                      {isMine ? (
                        <CheckCircle2 className="size-5 shrink-0 text-ok" aria-hidden />
                      ) : null}
                    </button>
                  </li>
                )
              })}
            </ul>
          )}
        </div>

        <footer className="flex items-center justify-end gap-2 border-t border-border p-4">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-border px-3 py-1.5 text-sm text-ink-muted transition-colors hover:bg-surface-hi"
          >
            Close
          </button>
          <button
            type="button"
            disabled={!selected || hasVoted || submitting}
            onClick={() => selected && onVote(selected)}
            className="rounded-lg bg-signal px-4 py-1.5 text-sm font-semibold text-signal-ink transition-colors hover:bg-signal/90 disabled:opacity-50"
          >
            {hasVoted ? 'Vote recorded' : submitting ? 'Casting…' : 'Cast vote'}
          </button>
        </footer>
      </div>
    </div>
  )
}
