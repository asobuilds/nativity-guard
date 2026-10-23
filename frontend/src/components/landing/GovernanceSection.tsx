import { useState } from 'react'
import {
  AlertCircle,
  FileCheck2,
  Scale,
  Shield,
  Sliders,
  UserCheck,
  Users,
  Vote,
} from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { usePublicBlueprint } from '@/hooks/usePublicBlueprint'
import type { ActionContext } from './types'

export interface GovernanceSectionProps {
  onActionTrigger: (action: ActionContext) => void
}

const GOVERNANCE_CONCEPTS = [
  {
    title: 'Controlled case access',
    label: 'Access rules',
    description: 'Membership in a unit should not, by itself, provide access to every report. Actual permissions depend on backend authorization rules.',
    icon: Shield,
  },
  {
    title: 'Administrative review',
    label: 'Review workflow',
    description: 'A case may progress through officer and administrative review stages. The published workflow is not proof that any individual case received independent review.',
    icon: FileCheck2,
  },
  {
    title: 'Unit leadership',
    label: 'Policy configuration',
    description: 'The platform describes elections, terms and recall rules. Each unit must satisfy its active policy and eligibility requirements.',
    icon: Scale,
  },
  {
    title: 'Privacy and fair treatment',
    label: 'Design principle',
    description: 'The interface aims to limit exposure of sensitive reports and personal details. Legal protections come from applicable law, not an app badge.',
    icon: UserCheck,
  },
] as const

/** This example does not cast votes, enroll members or certify a real election. */
export function GovernanceSection(_props: GovernanceSectionProps) {
  const { data: blueprint, isPending, isError } = usePublicBlueprint()
  const [memberCount, setMemberCount] = useState(35)

  const elections = blueprint?.governance?.elections
  const band = memberCount <= 20 ? '12-20' : memberCount <= 40 ? '21-40' : memberCount <= 70 ? '41-70' : '71-100'
  const seats = elections?.seat_bands?.[band]
  const quorumPercent = elections?.quorum_percent
  const quorum = typeof quorumPercent === 'number' && quorumPercent >= 0 && quorumPercent <= 100
    ? Math.ceil(memberCount * quorumPercent / 100)
    : null

  return (
    <section id="governance" className="border-t border-border py-14">
      <div>
        <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-[11px] font-medium text-ink-muted ring-1 ring-border-hi">
          <Scale className="size-3.5 text-signal" aria-hidden />
          Governance overview
        </span>
        <h2 className="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
          Unit Governance &amp; Accountability
        </h2>
        <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
          Explore how Nativity Guard describes permissions, reviews and unit governance.
          Platform policy descriptions do not confer public authority, legal accreditation or guarantee fair outcomes.
        </p>
      </div>

      <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {GOVERNANCE_CONCEPTS.map((concept) => {
          const Icon = concept.icon
          return (
            <Card key={concept.title} className="flex flex-col p-5 border-border hover:border-border-hi transition-all">
              <div className="flex items-center justify-between gap-2">
                <span className="grid size-9 place-items-center rounded-lg bg-surface-hi text-signal ring-1 ring-border">
                  <Icon className="size-4" aria-hidden />
                </span>
                <span className="rounded-full bg-surface-hi px-2 py-0.5 text-[10px] text-ink-muted">{concept.label}</span>
              </div>
              <h3 className="mt-4 text-sm font-bold text-ink">{concept.title}</h3>
              <p className="mt-2.5 text-xs leading-relaxed text-ink-muted">{concept.description}</p>
            </Card>
          )
        })}
      </div>

      <div className="mt-8 rounded-panel border border-border bg-surface p-6">
        <div className="flex items-start gap-3 border-b border-border pb-4">
          <Sliders className="mt-1 size-5 shrink-0 text-signal" aria-hidden />
          <div>
            <h3 className="text-base font-bold text-ink">Illustrative Council &amp; Quorum Calculator</h3>
            <p className="mt-1 text-xs leading-relaxed text-ink-muted">
              This is a hypothetical example, not the membership count or election result of any real unit.
              Figures only use the governance configuration published by the service.
            </p>
          </div>
        </div>

        {isError ? (
          <p role="alert" className="mt-5 flex items-center gap-2 text-sm text-warn">
            <AlertCircle className="size-4" aria-hidden />
            Published governance rules could not be loaded. No substitute policy or figures are shown.
          </p>
        ) : isPending ? (
          <p role="status" className="mt-5 text-sm text-ink-muted">Loading published governance configuration...</p>
        ) : typeof seats !== 'number' || quorum === null ? (
          <p role="alert" className="mt-5 text-sm text-warn">
            The service has not supplied a complete, usable policy configuration.
          </p>
        ) : (
          <div className="mt-6 grid gap-6 lg:grid-cols-12">
            <div className="lg:col-span-6 space-y-4">
              <label htmlFor="governance-example-members" className="flex items-center justify-between gap-2 text-xs font-medium text-ink">
                Example eligible members
                <strong className="text-sm text-signal">{memberCount}</strong>
              </label>
              <input
                id="governance-example-members"
                type="range"
                min="12"
                max="100"
                value={memberCount}
                onChange={(event) => setMemberCount(Number(event.target.value))}
                className="w-full accent-signal"
              />
              <div className="flex justify-between text-[10px] text-ink-faint">
                <span>12</span><span>40</span><span>70</span><span>100</span>
              </div>
              <p className="text-xs leading-relaxed text-ink-muted">
                Published term length: {elections?.term_months} months.
                Figures shown here do not verify voter eligibility, active quorum rules for a specific unit,
                or that an election is currently open.
              </p>
            </div>
            <div className="lg:col-span-6 grid grid-cols-2 gap-3">
              <div className="rounded-lg border border-border bg-surface-hi p-4 text-center">
                <Users className="mx-auto size-5 text-signal" aria-hidden />
                <p className="mt-2 text-2xl font-black text-ink">{seats}</p>
                <p className="text-xs text-ink-muted">Example council seats</p>
              </div>
              <div className="rounded-lg border border-border bg-surface-hi p-4 text-center">
                <Vote className="mx-auto size-5 text-signal" aria-hidden />
                <p className="mt-2 text-2xl font-black text-ink">{quorum}</p>
                <p className="text-xs text-ink-muted">Example quorum at {quorumPercent}%</p>
              </div>
            </div>
          </div>
        )}
        <p className="mt-5 text-xs text-ink-faint">
          Viewing this calculator does not start a petition, submit a vote, or give a unit permission to operate.
        </p>
      </div>
    </section>
  )
}
