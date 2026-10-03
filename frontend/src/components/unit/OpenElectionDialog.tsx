import { useState } from 'react'
import { AlertTriangle, X } from 'lucide-react'
import { cn } from '@/lib/cn'

interface OpenElectionDialogProps {
  open: boolean
  eligibleCount: number
  canOpen: boolean
  onOpen: (rotationGroup: 'A' | 'B') => void
  onOpenHeadAdmin: () => void
  onClose: () => void
  submitting: boolean
  submittingHead: boolean
}

/**
 * Two ways to open an election: an admin election (seats candidates from the
 * whole membership), or a head-admin election (elects one head admin from
 * the current admins). Both need a minimum quorum the backend enforces; the
 * dialog shows the eligible count up front so the operator is not surprised.
 */
export function OpenElectionDialog({
  open,
  eligibleCount,
  canOpen,
  onOpen,
  onOpenHeadAdmin,
  onClose,
  submitting,
  submittingHead,
}: OpenElectionDialogProps) {
  const [rotationGroup, setRotationGroup] = useState<'A' | 'B'>('A')

  if (!open) return null

  const tooFewMembers = eligibleCount < 12

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Open an election"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="flex w-full max-w-md flex-col overflow-hidden rounded-3xl border border-border bg-base shadow-2xl">
        <header className="flex items-start justify-between border-b border-border p-4">
          <div>
            <h3 className="text-base font-semibold text-ink">Open an election</h3>
            <p className="mt-0.5 text-xs text-ink-muted">
              {eligibleCount} verified member{eligibleCount === 1 ? '' : 's'} eligible to vote.
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

        <div className="space-y-4 p-4">
          {tooFewMembers ? (
            <div className="flex items-start gap-2 rounded-lg border border-warn/30 bg-warn/5 p-3 text-xs text-warn">
              <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden />
              <span>
                A unit needs at least 12 verified members before an admin election can be
                opened. The backend will refuse otherwise.
              </span>
            </div>
          ) : null}

          <div>
            <p className="mb-2 text-xs font-medium uppercase tracking-wider text-ink-faint">
              Rotation group
            </p>
            <div className="grid grid-cols-2 gap-2">
              {(['A', 'B'] as const).map((g) => (
                <button
                  key={g}
                  type="button"
                  onClick={() => setRotationGroup(g)}
                  className={cn(
                    'rounded-lg border p-3 text-left transition-colors',
                    rotationGroup === g
                      ? 'border-signal bg-signal/10'
                      : 'border-border hover:border-border-hi',
                  )}
                >
                  <p className="text-sm font-semibold text-ink">Group {g}</p>
                  <p className="mt-0.5 text-[11px] text-ink-muted">
                    Staggered rotation to keep continuity
                  </p>
                </button>
              ))}
            </div>
          </div>

          <button
            type="button"
            disabled={!canOpen || submitting || tooFewMembers}
            onClick={() => onOpen(rotationGroup)}
            className="w-full rounded-lg bg-signal px-4 py-2.5 text-sm font-semibold text-signal-ink transition-colors hover:bg-signal/90 disabled:opacity-50"
          >
            {submitting ? 'Opening…' : 'Open admin election'}
          </button>

          <div className="border-t border-border pt-4">
            <p className="mb-2 text-xs font-medium uppercase tracking-wider text-ink-faint">
              Or start a head-admin election
            </p>
            <p className="mb-3 text-xs text-ink-muted">
              Existing admins vote for one of their own to lead the unit. Needs at
              least 3 admins in place.
            </p>
            <button
              type="button"
              disabled={!canOpen || submittingHead}
              onClick={onOpenHeadAdmin}
              className="w-full rounded-lg border border-border px-4 py-2 text-sm font-medium text-ink-muted transition-colors hover:bg-surface-hi disabled:opacity-50"
            >
              {submittingHead ? 'Opening…' : 'Open head-admin election'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
