import { useState } from 'react'
import { Crown, Plus, Shield, Users, Vote, XCircle } from 'lucide-react'
import { cn } from '@/lib/cn'
import { initials } from '@/lib/format'
import { mediaURL } from '@/lib/apiClient'
import { useToast } from '@/components/ui/Toast'
import { useUnitGovernance, type Election } from '@/hooks/useUnitGovernance'
import { useUnitRoster } from '@/hooks/useUnitRoster'
import { useElectionActions } from '@/hooks/useElectionActions'
import { ApiError } from '@/lib/apiClient'
import { SeatGrid } from './SeatGrid'
import { ElectionCard } from './ElectionCard'
import { VotePanel } from './VotePanel'
import { OpenElectionDialog } from './OpenElectionDialog'

interface GovernancePanelProps {
  unitId: string
  isAdmin: boolean
}

export function GovernancePanel({ unitId, isAdmin }: GovernancePanelProps) {
  const { notify } = useToast()
  const governance = useUnitGovernance(unitId, true)
  const roster = useUnitRoster(unitId, true)
  const actions = useElectionActions(unitId)

  const [votingOn, setVotingOn] = useState<Election | null>(null)
  const [openingDialog, setOpeningDialog] = useState(false)
  const [myVotes, setMyVotes] = useState<Record<string, string>>({})

  const g = governance.data
  const verifiedCandidates = (roster.data?.members ?? []).filter(
    (m) => m.status === 'active' && m.verifiedAt,
  )

  function reportError(cause: unknown, fallback: string) {
    if (cause instanceof ApiError) notify(cause.message, 'error')
    else notify(fallback, 'error')
  }

  if (governance.isLoading) {
    return <div className="h-40 animate-pulse rounded-panel bg-surface-hi/60" />
  }

  if (governance.isError || !g) {
    return (
      <div className="rounded-lg border border-warn/30 bg-warn/5 p-4 text-sm text-warn">
        Could not load governance data. Try again in a moment.
      </div>
    )
  }

  const seatsFilled = g.seats.filter((s) => s.status === 'active').length
  const seatsTotal = g.seats.length

  return (
    <div className="space-y-4">
      {/* Header row */}
      <div className="grid grid-cols-3 gap-3">
        <StatCard
          icon={<Crown className="size-4 text-warn" aria-hidden />}
          label="Head admin"
          value={g.headAdminCount > 0 ? 'Elected' : 'None'}
        />
        <StatCard
          icon={<Shield className="size-4 text-signal" aria-hidden />}
          label="Admins"
          value={String(g.adminCount)}
        />
        <StatCard
          icon={<Users className="size-4 text-signal" aria-hidden />}
          label="Seats"
          value={seatsTotal > 0 ? `${seatsFilled}/${seatsTotal}` : '—'}
        />
      </div>

      {/* Open elections */}
      {g.openElections.length > 0 ? (
        <Section
          icon={<Vote className="size-4 text-signal" aria-hidden />}
          title="Open elections"
          count={g.openElections.length}
        >
          <div className="space-y-3 p-3">
            {g.openElections.map((e) => (
              <ElectionCard
                key={e.id}
                election={e}
                onOpenVote={(el) => setVotingOn(el)}
                onClose={
                  isAdmin
                    ? (el) => {
                        actions.closeElection.mutate(
                          { electionId: el.id },
                          {
                            onSuccess: () => notify('Election finalized', 'success'),
                            onError: (cause) =>
                              reportError(cause, 'Could not close election'),
                          },
                        )
                      }
                    : undefined
                }
                canManage={isAdmin}
                closing={
                  actions.closeElection.isPending &&
                  actions.closeElection.variables?.electionId === e.id
                }
              />
            ))}
          </div>
        </Section>
      ) : isAdmin ? (
        <button
          type="button"
          onClick={() => setOpeningDialog(true)}
          className="flex w-full items-center justify-center gap-2 rounded-panel border border-dashed border-border p-4 text-sm font-medium text-ink-muted transition-colors hover:border-signal/40 hover:text-ink"
        >
          <Plus className="size-4" aria-hidden />
          Open a new election
        </button>
      ) : null}

      {/* Seat grid */}
      {g.seats.length > 0 ? (
        <Section
          icon={<Shield className="size-4 text-signal" aria-hidden />}
          title="Admin seats"
          count={seatsTotal}
        >
          <div className="p-3">
            <SeatGrid seats={g.seats} admins={g.admins} />
          </div>
        </Section>
      ) : null}

      {/* Current admins list */}
      {g.admins.length > 0 ? (
        <Section
          icon={<Users className="size-4 text-signal" aria-hidden />}
          title="Current admins"
          count={g.admins.length}
        >
          <ul className="divide-y divide-border">
            {g.admins.map((a) => {
              const avatar = mediaURL(a.avatarPath) ?? null
              const fullName = [a.firstName, a.lastName].filter(Boolean).join(' ')
              return (
                <li key={a.membershipId} className="flex items-center gap-3 p-3">
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
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="truncate text-sm font-medium text-ink">
                        {fullName || 'Admin'}
                      </p>
                      {a.isHeadAdmin ? (
                        <span className="inline-flex items-center gap-1 rounded-full bg-warn/15 px-2 py-0.5 text-[10px] font-medium text-warn">
                          <Crown className="size-3" aria-hidden />
                          Head admin
                        </span>
                      ) : null}
                    </div>
                    <p className="truncate text-xs text-ink-faint">{a.email}</p>
                    {a.termEndAt ? (
                      <p className="text-[11px] text-ink-faint">
                        Term ends {new Date(a.termEndAt).toLocaleDateString()}
                        {a.consecutiveTerms > 1
                          ? ` · ${a.consecutiveTerms} terms served`
                          : ''}
                      </p>
                    ) : null}
                  </div>
                </li>
              )
            })}
          </ul>
        </Section>
      ) : null}

      {/* Recent finalized elections */}
      {g.recentElections.length > 0 ? (
        <Section
          icon={<Vote className="size-4 text-ink-muted" aria-hidden />}
          title="Recent elections"
          count={g.recentElections.length}
        >
          <div className="space-y-3 p-3">
            {g.recentElections.map((e) => (
              <ElectionCard key={e.id} election={e} />
            ))}
          </div>
        </Section>
      ) : null}

      {/* Open revocations — read-only summary for now */}
      {g.openRevocations.length > 0 ? (
        <Section
          icon={<XCircle className="size-4 text-emergency" aria-hidden />}
          title="Revocations in progress"
          count={g.openRevocations.length}
        >
          <ul className="divide-y divide-border">
            {g.openRevocations.map((r) => (
              <li key={r.id} className="p-3">
                <p className="text-sm text-ink">
                  {r.cycleType === 'head_admin' ? 'Head-admin' : 'Admin'} revocation ·{' '}
                  <span className="text-ink-muted">{r.targetRole}</span>
                </p>
                <p className="mt-1 text-xs text-ink-muted">{r.reason}</p>
                <p className="mt-1 text-[11px] text-ink-faint">
                  {r.forVotes} for · {r.againstVotes} against · {r.abstainVotes} abstain
                  {' · '}needs {r.requiredVotes} of {r.eligibleMemberCount}
                </p>
              </li>
            ))}
          </ul>
        </Section>
      ) : null}

      {/* Dialogs */}
      <OpenElectionDialog
        open={openingDialog}
        eligibleCount={verifiedCandidates.length}
        canOpen={verifiedCandidates.length >= 12}
        submitting={actions.openAdminElection.isPending}
        submittingHead={actions.openHeadAdminElection.isPending}
        onClose={() => setOpeningDialog(false)}
        onOpen={(g) => {
          actions.openAdminElection.mutate(
            { unitId, rotationGroup: g },
            {
              onSuccess: () => {
                notify('Election opened', 'success')
                setOpeningDialog(false)
              },
              onError: (cause) => reportError(cause, 'Could not open election'),
            },
          )
        }}
        onOpenHeadAdmin={() => {
          actions.openHeadAdminElection.mutate(
            { unitId },
            {
              onSuccess: () => {
                notify('Head-admin election opened', 'success')
                setOpeningDialog(false)
              },
              onError: (cause) => reportError(cause, 'Could not open head-admin election'),
            },
          )
        }}
      />

      <VotePanel
        open={Boolean(votingOn)}
        election={
          votingOn ?? {
            id: '',
            unitId: '',
            electionType: 'admin',
            cycleNumber: 0,
            rotationGroup: 'A',
            seatCount: 0,
            eligibleVoterCount: 0,
            quorumCount: 0,
            extendedOnce: false,
            quorumMet: false,
            termStart: '',
            termEnd: '',
            status: 'open',
            createdAt: '',
          }
        }
        candidates={verifiedCandidates}
        myVoteCandidateId={votingOn ? myVotes[votingOn.id] : undefined}
        submitting={actions.castVote.isPending}
        onClose={() => setVotingOn(null)}
        onVote={(candidateUserId) => {
          if (!votingOn) return
          actions.castVote.mutate(
            { electionId: votingOn.id, candidateId: candidateUserId },
            {
              onSuccess: () => {
                notify('Vote recorded', 'success')
                setMyVotes((prev) => ({ ...prev, [votingOn.id]: candidateUserId }))
                setVotingOn(null)
              },
              onError: (cause) => reportError(cause, 'Could not record vote'),
            },
          )
        }}
      />
    </div>
  )
}

function StatCard({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode
  label: string
  value: string
}) {
  return (
    <div className="rounded-lg border border-border bg-surface/70 p-3">
      <div className="flex items-center gap-1.5">
        {icon}
        <p className="text-[10px] uppercase tracking-wider text-ink-faint">{label}</p>
      </div>
      <p className="mt-1 text-lg font-semibold text-ink">{value}</p>
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
    <div className="overflow-hidden rounded-panel border border-border bg-surface/60">
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


