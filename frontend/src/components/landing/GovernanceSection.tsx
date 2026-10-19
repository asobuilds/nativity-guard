import { useState } from 'react'
import {
  AlertOctagon,
  CheckCircle2,
  FileCheck2,
  Scale,
  Shield,
  Sliders,
  UserCheck,
  Users,
  Vote,
} from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { usePublicBlueprint } from '@/hooks/usePublicBlueprint'
import type { ActionContext } from './types'

export interface GovernanceSectionProps {
  onActionTrigger: (action: ActionContext) => void
}

interface CardinalRule {
  id: string
  title: string
  subtitle: string
  description: string
  badge: string
  icon: typeof Shield
  safeguard: string
}

const CARDINAL_RULES: CardinalRule[] = [
  {
    id: 'compartmentalization',
    title: 'Unit Access ≠ Case Access',
    subtitle: 'Zero Unassigned Case Peeking',
    description:
      'Being a member of a security unit does not grant access to all neighborhood incidents. Officers only view cases explicitly assigned to their badge (primary, paired, or support).',
    badge: 'Privacy Shield',
    icon: Shield,
    safeguard: 'Protects victims against gossip, extortion, and unauthorized dossier snooping.',
  },
  {
    id: 'two_admin_closure',
    title: 'Two-Admin Case Closure Rule',
    subtitle: 'No Unilateral Investigations',
    description:
      'An investigating officer cannot mark their own case as closed. Closure requires formal investigative sign-off and approval from at least 2 independent unit administrators.',
    badge: 'Dual Oversight',
    icon: FileCheck2,
    safeguard: 'Prevents corrupt payoffs, premature dismissals, and single-point bias.',
  },
  {
    id: 'democratic_recall',
    title: 'Democratic Recall & Term Limits',
    subtitle: 'Removable by Verified Members',
    description:
      'Unit leaders serve 12-month staggered terms (max 2 consecutive terms). Any regular admin can be recalled by 10+ verified member votes + Head Admin sign-off. Head Admins are removable by simple majority.',
    badge: 'Auditable Recall',
    icon: Scale,
    safeguard: 'Eliminates permanent vigilante warlords; leadership answers to the community.',
  },
  {
    id: 'presumed_innocence',
    title: 'Presumed Innocence & Suspect Rights',
    subtitle: 'Fair Process by Code',
    description:
      'Citizens named as suspects have constitutional self-view rights. They view only public-safe status markers and verified timestamps—never officer identities, informant tips, or raw surveillance.',
    badge: 'Legal Protection',
    icon: UserCheck,
    safeguard: 'Guarantees constitutional fair hearing while protecting reporting citizens.',
  },
]

export function GovernanceSection({ onActionTrigger }: GovernanceSectionProps) {
  const { data: blueprint } = usePublicBlueprint()
  const [memberCount, setMemberCount] = useState<number>(35)

  // Seat band calculation based on Nativity Guard constitutional rules
  function getSeatsForMembers(count: number): number {
    if (count <= 20) return 5
    if (count <= 40) return 7
    if (count <= 70) return 9
    return 10
  }

  const seats = getSeatsForMembers(memberCount)
  const quorum = Math.ceil(memberCount * 0.5)

  function handleTriggerVote(actionType: 'vote_election' | 'vote_revocation') {
    onActionTrigger({
      type: actionType,
      title: actionType === 'vote_election' ? 'Cast Your Unit Election Ballot' : 'Initiate Democratic Recall',
      description:
        'Nativity Guard governance gives verified members one vote per election cycle. Authenticate your verified identity to cast your ballot.',
      badgeText: `50% Quorum (${quorum} Votes Required)`,
      badgeVariant: 'signal',
      returnUrl: '/governance',
      payload: { memberCount, requiredSeats: seats, quorum },
    })
  }

  return (
    <section id="governance" className="border-t border-border py-14">
      {/* Section Header */}
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-[11px] font-medium text-ink-muted ring-1 ring-border-hi">
            <Scale className="size-3.5 text-signal" aria-hidden />
            Constitutional Architecture
          </span>
          <h2 className="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
            Elected Leadership & Anti-Vigilante Governance
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
            Nativity Guard rejects vigilante violence and warlordism through strictly enforced democratic
            rules. Unit administrators are elected, rotating, and immediately subject to recall.
          </p>
        </div>

        <div className="pt-2 md:pt-0">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => handleTriggerVote('vote_election')}
            icon={<Vote className="size-3.5 text-signal" />}
          >
            Unit Voting Portal
          </Button>
        </div>
      </div>

      {/* The 4 Cardinal Rules Grid */}
      <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {CARDINAL_RULES.map((rule) => {
          const IconComponent = rule.icon
          return (
            <Card
              key={rule.id}
              className="flex flex-col justify-between p-5 border-border hover:border-border-hi transition-all"
            >
              <div>
                <div className="flex items-center justify-between">
                  <div className="grid size-9 place-items-center rounded-lg bg-surface-hi text-signal ring-1 ring-border">
                    <IconComponent className="size-4" />
                  </div>
                  <span className="rounded-full bg-surface-hi px-2 py-0.5 text-[10px] font-semibold text-ink-muted">
                    {rule.badge}
                  </span>
                </div>

                <h3 className="mt-4 text-sm font-bold text-ink">{rule.title}</h3>
                <p className="text-[11px] font-medium text-signal mt-0.5">{rule.subtitle}</p>
                <p className="mt-2.5 text-xs text-ink-muted leading-relaxed">{rule.description}</p>
              </div>

              <div className="mt-4 rounded bg-surface-hi p-2.5 border border-border">
                <div className="flex items-start gap-1.5">
                  <CheckCircle2 className="size-3.5 text-ok shrink-0 mt-0.5" />
                  <p className="text-[10px] text-ink-faint leading-normal font-medium">
                    {rule.safeguard}
                  </p>
                </div>
              </div>
            </Card>
          )
        })}
      </div>

      {/* Interactive Representation & Quorum Calculator */}
      <div className="mt-8 rounded-panel border border-border bg-surface p-6">
        <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between border-b border-border pb-4">
          <div>
            <div className="flex items-center gap-2">
              <Sliders className="size-4 text-signal" />
              <h3 className="text-base font-bold text-ink">
                Unit Electoral Seat Band & Quorum Calculator
              </h3>
            </div>
            <p className="text-xs text-ink-muted mt-1">
              Explore how administrative seats and voting quorum dynamically scale with verified active
              membership according to the Nativity Guard constitution.
            </p>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-xs text-ink-muted">Term Length:</span>
            <span className="rounded bg-surface-hi px-2.5 py-1 text-xs font-semibold text-ink">
              {blueprint?.governance.elections.term_months ?? 12} Months
            </span>
          </div>
        </div>

        <div className="mt-6 grid gap-6 lg:grid-cols-12 items-center">
          {/* Slider & Input Controls */}
          <div className="lg:col-span-6 space-y-4">
            <div>
              <div className="flex items-center justify-between text-xs font-medium text-ink mb-1.5">
                <span>Active Verified Unit Members:</span>
                <span className="text-sm font-bold text-signal">{memberCount} Members</span>
              </div>
              <input
                type="range"
                min="12"
                max="100"
                value={memberCount}
                onChange={(e) => setMemberCount(parseInt(e.target.value, 10))}
                className="w-full accent-signal cursor-pointer"
                aria-label="Member count slider"
              />
              <div className="flex justify-between text-[10px] text-ink-faint mt-1">
                <span>12 (Minimum Quorum)</span>
                <span>40</span>
                <span>70</span>
                <span>100 (Capped Band)</span>
              </div>
            </div>

            {/* Constitution Details Banner */}
            <div className="space-y-2 rounded-lg bg-surface-hi p-3.5 border border-border text-xs text-ink-muted">
              <div className="flex items-center justify-between">
                <span>Staggered Election Cycle:</span>
                <span className="font-semibold text-ink">
                  Every {blueprint?.governance.elections.stagger_months ?? 6} Months
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span>Term Limits:</span>
                <span className="font-semibold text-ink">
                  Max {blueprint?.governance.elections.term_limit ?? 2} Consecutive Terms
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span>Mandatory Cooling-off:</span>
                <span className="font-semibold text-ink">
                  {blueprint?.governance.elections.cooling_off_months ?? 6} Months
                </span>
              </div>
            </div>
          </div>

          {/* Real-time Computed Values */}
          <div className="lg:col-span-6 grid grid-cols-2 gap-3 sm:gap-4">
            <div className="rounded-lg border border-border bg-surface-hi p-4 text-center">
              <Users className="size-5 text-signal mx-auto" />
              <p className="mt-2 text-2xl font-black text-ink">{seats}</p>
              <p className="text-xs font-medium text-ink-muted">Admin Council Seats</p>
              <span className="mt-1 block text-[10px] text-ink-faint">
                {memberCount <= 20
                  ? 'Tier 1 (12–20 Members)'
                  : memberCount <= 40
                  ? 'Tier 2 (21–40 Members)'
                  : memberCount <= 70
                  ? 'Tier 3 (41–70 Members)'
                  : 'Tier 4 (71–100 Members)'}
              </span>
            </div>

            <div className="rounded-lg border border-border bg-surface-hi p-4 text-center">
              <Vote className="size-5 text-ok mx-auto" />
              <p className="mt-2 text-2xl font-black text-ink">{quorum}</p>
              <p className="text-xs font-medium text-ink-muted">Quorum Votes (50%)</p>
              <span className="mt-1 block text-[10px] text-ink-faint">
                Required to pass resolutions
              </span>
            </div>

            <div className="col-span-2 rounded-lg border border-border bg-surface p-3 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <AlertOctagon className="size-4 text-warn shrink-0" />
                <div>
                  <p className="text-xs font-semibold text-ink">Democratic Recall Rule</p>
                  <p className="text-[10px] text-ink-faint">
                    Regular Admin: 10+ member votes + Head Admin sign-off | Head Admin: 50%+1 member vote
                  </p>
                </div>
              </div>

              <Button
                variant="ghost"
                size="sm"
                onClick={() => handleTriggerVote('vote_revocation')}
                className="text-xs"
              >
                Recall Petition
              </Button>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
