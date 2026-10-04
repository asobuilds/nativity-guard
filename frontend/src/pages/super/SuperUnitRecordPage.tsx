import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  ArrowLeft,
  Crown,
  Gavel,
  ScrollText,
  Shield,
  ShieldCheck,
  ShieldX,
  Users,
  Vote,
} from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Skeleton, ErrorState } from '@/components/ui/States'
import { useUnitRecord } from '@/hooks/useUnitRecord'
import { relativeTime } from '@/lib/format'
import { mediaURL } from '@/lib/apiClient'
import { initials } from '@/lib/format'
import { cn } from '@/lib/cn'

type TabId = 'overview' | 'roster' | 'governance' | 'verifications' | 'audit'

/**
 * `/super/units/:id` — the platform's single-unit record.
 *
 * This is the super admin's inspection surface: everything about one unit
 * in one place. Read-only.
 */
export function SuperUnitRecordPage() {
  const { id } = useParams<{ id: string }>()
  const [tab, setTab] = useState<TabId>('overview')
  const query = useUnitRecord(id)

  if (query.isLoading) {
    return (
      <div className="mx-auto max-w-6xl space-y-4 p-4 sm:p-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full rounded-panel" />
      </div>
    )
  }

  if (query.isError || !query.data) {
    return (
      <div className="mx-auto max-w-6xl p-4 sm:p-6">
        <Link to="/super/units" className="text-sm text-signal hover:underline">
          ← Back to registry
        </Link>
        <Card className="mt-4">
          <ErrorState
            title="Unit record unavailable"
            description="Could not load this unit's record."
            onRetry={() => void query.refetch()}
          />
        </Card>
      </div>
    )
  }

  const record = query.data
  const { unit, counts } = record

  const tabs: { id: TabId; label: string; badge?: number }[] = [
    { id: 'overview', label: 'Overview' },
    { id: 'roster', label: 'Roster', badge: counts.active + counts.pending },
    { id: 'governance', label: 'Governance', badge: record.openElections.length },
    { id: 'verifications', label: 'Verification' },
    { id: 'audit', label: 'Audit', badge: record.auditLog.length },
  ]

  return (
    <div className="mx-auto max-w-6xl space-y-4 p-4 sm:p-6">
      <Link
        to="/super/units"
        className="inline-flex items-center gap-1.5 text-sm text-signal hover:underline"
      >
        <ArrowLeft className="size-3.5" aria-hidden />
        Back to registry
      </Link>

      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold text-ink">{unit.brandName || unit.name}</h1>
          <p className="mt-1 text-sm text-ink-muted">
            {unit.type}
            {[unit.city, unit.lga, unit.state].filter(Boolean).length > 0
              ? ` · ${[unit.city, unit.lga, unit.state].filter(Boolean).join(', ')}`
              : ''}
          </p>
        </div>
        {unit.isVerified ? (
          <span className="inline-flex items-center gap-1.5 rounded-full bg-ok/15 px-3 py-1 text-xs font-medium text-ok">
            <ShieldCheck className="size-3.5" aria-hidden />
            Verified
          </span>
        ) : (
          <span
            className={cn(
              'inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium',
              unit.verificationStatus === 'rejected'
                ? 'bg-emergency/15 text-emergency'
                : 'bg-warn/15 text-warn',
            )}
          >
            <ShieldX className="size-3.5" aria-hidden />
            {unit.verificationStatus.replace('_', ' ')}
          </span>
        )}
      </header>

      {/* Stat strip */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
        <StatCard label="Active members" value={counts.active} />
        <StatCard label="Pending" value={counts.pending} />
        <StatCard label="Admins" value={counts.admins} />
        <StatCard label="Open cases" value={record.cases.open} />
        <StatCard label="Total cases" value={record.cases.total} />
      </div>

      {/* Tabs */}
      <nav className="mobile-nav-scroll -mx-4 overflow-x-auto border-b border-border px-4 sm:mx-0 sm:px-0">
        <ul className="flex gap-1">
          {tabs.map((t) => {
            const active = t.id === tab
            return (
              <li key={t.id}>
                <button
                  type="button"
                  onClick={() => setTab(t.id)}
                  className={cn(
                    'relative whitespace-nowrap px-3 py-2.5 text-sm font-medium transition-colors border-b-2 -mb-px',
                    active
                      ? 'border-signal text-signal'
                      : 'border-transparent text-ink-muted hover:text-ink',
                  )}
                >
                  {t.label}
                  {typeof t.badge === 'number' && t.badge > 0 ? (
                    <span className="ml-1.5 rounded-full bg-signal/15 px-1.5 py-0.5 text-[10px] tabular-nums text-signal">
                      {t.badge}
                    </span>
                  ) : null}
                </button>
              </li>
            )
          })}
        </ul>
      </nav>

      {tab === 'overview' ? <OverviewTab record={record} /> : null}
      {tab === 'roster' ? <RosterTab record={record} /> : null}
      {tab === 'governance' ? <GovernanceTab record={record} /> : null}
      {tab === 'verifications' ? <VerificationTab record={record} /> : null}
      {tab === 'audit' ? <AuditTab record={record} /> : null}
    </div>
  )
}

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-panel border border-border bg-surface/60 p-3">
      <p className="text-[10px] uppercase tracking-wider text-ink-faint">{label}</p>
      <p className="mt-1 text-xl font-semibold tabular-nums text-ink">{value}</p>
    </div>
  )
}

function OverviewTab({ record }: { record: ReturnType<typeof useUnitRecord>['data'] }) {
  if (!record) return null
  const u = record.unit
  return (
    <div className="space-y-4">
      <Card className="p-5">
        <h2 className="text-sm font-semibold text-ink">Registration</h2>
        <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-2">
          <Field label="Registration #" value={u.registrationNumber} />
          <Field label="Formation date" value={u.formationDate ? new Date(u.formationDate).toLocaleDateString() : '—'} />
          <Field label="Coverage radius" value={`${u.operationalRadius} km`} />
          <Field label="Coverage area" value={u.coverageArea} />
          <Field label="Contact person" value={u.contactPerson} />
          <Field label="Contact phone" value={u.contactPhone} />
          <Field label="Contact email" value={u.contactEmail} />
          <Field label="Commander" value={u.commanderName} />
          <Field label="Commander NIN" value={u.commanderNin} />
          <Field label="Commander alt phone" value={u.commanderPhoneAlt} />
          <Field label="Kindred head" value={u.kindredHeadName ? `${u.kindredHeadName} (${u.kindredHeadPhone})` : ''} />
          <Field label="Ward head" value={u.wardHeadName ? `${u.wardHeadName} (${u.wardHeadPhone})` : ''} />
        </dl>
      </Card>

      <Card className="p-5">
        <h2 className="text-sm font-semibold text-ink">Activity</h2>
        <div className="mt-3 grid gap-3 sm:grid-cols-3">
          <Field label="Open cases" value={String(record.cases.open)} />
          <Field label="Total cases" value={String(record.cases.total)} />
          <Field
            label="Verification"
            value={u.isVerified ? `Verified ${relativeTime(u.verifiedAt ?? '')}` : u.verificationStatus.replace('_', ' ')}
          />
        </div>
      </Card>
    </div>
  )
}

function RosterTab({ record }: { record: ReturnType<typeof useUnitRecord>['data'] }) {
  if (!record) return null
  if (record.members.length === 0) {
    return <Card className="p-6 text-center text-sm text-ink-muted">No members yet.</Card>
  }
  return (
    <Card className="overflow-hidden">
      <ul className="divide-y divide-border">
        {record.members.map((m) => {
          const avatar = mediaURL(m.avatarPath) ?? null
          const fullName = [m.firstName, m.lastName].filter(Boolean).join(' ')
          const Icon = m.isHeadAdmin ? Crown : m.role === 'unit_admin' ? Shield : Users
          return (
            <li key={m.membershipId} className="flex items-center gap-3 p-3">
              {avatar ? (
                <img src={avatar} alt="" className="size-9 rounded-full border border-border object-cover" />
              ) : (
                <div className="grid size-9 place-items-center rounded-full bg-surface-hi text-[10px] font-semibold text-ink">
                  {initials(m.firstName, m.lastName)}
                </div>
              )}
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <p className="truncate text-sm font-medium text-ink">{fullName}</p>
                  <span className="inline-flex items-center gap-1 rounded-full bg-surface-hi px-2 py-0.5 text-[10px] text-ink-muted">
                    <Icon className="size-3" aria-hidden />
                    {m.isHeadAdmin ? 'Head admin' : m.role.replace('_', ' ')}
                  </span>
                  {m.status !== 'active' ? (
                    <span className="rounded-full bg-warn/15 px-2 py-0.5 text-[10px] font-medium text-warn">
                      {m.status}
                    </span>
                  ) : null}
                </div>
                <p className="truncate text-xs text-ink-faint">{m.email}</p>
              </div>
              <span className="shrink-0 text-[11px] text-ink-faint">
                joined {relativeTime(m.acceptedAt || m.createdAt)}
              </span>
            </li>
          )
        })}
      </ul>
    </Card>
  )
}

function GovernanceTab({ record }: { record: ReturnType<typeof useUnitRecord>['data'] }) {
  if (!record) return null
  return (
    <div className="space-y-4">
      {record.openElections.length > 0 ? (
        <Card className="p-5">
          <h2 className="flex items-center gap-2 text-sm font-semibold text-ink">
            <Vote className="size-4 text-signal" aria-hidden />
            Open elections ({record.openElections.length})
          </h2>
          <ul className="mt-3 space-y-3">
            {record.openElections.map((e) => (
              <li key={e.id} className="rounded-lg border border-signal/40 bg-signal/5 p-3">
                <p className="text-sm font-medium text-ink">
                  {e.electionType === 'head_admin' ? 'Head-admin' : 'Admin'} election ·{' '}
                  {e.seatCount} seat{e.seatCount === 1 ? '' : 's'}
                </p>
                <p className="mt-1 text-xs text-ink-muted">
                  {e.eligibleVoterCount} eligible · quorum {e.quorumCount}
                  {e.votingEndsAt ? ` · closes ${new Date(e.votingEndsAt).toLocaleDateString()}` : ''}
                </p>
              </li>
            ))}
          </ul>
        </Card>
      ) : null}

      {record.seats.length > 0 ? (
        <Card className="p-5">
          <h2 className="text-sm font-semibold text-ink">Seats ({record.seats.length})</h2>
          <ul className="mt-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {record.seats.map((s) => {
              const holder = record.members.find((m) => m.userId === s.memberId)
              return (
                <li key={s.id} className="rounded-lg border border-border bg-surface/60 p-3">
                  <p className="text-[11px] font-semibold uppercase tracking-wider text-ink-faint">
                    Seat {s.seatNumber} · {s.rotationGroup}
                  </p>
                  <p className="mt-1 text-sm text-ink">
                    {holder ? `${holder.firstName} ${holder.lastName}` : 'Vacant'}
                  </p>
                  <p className="text-[11px] text-ink-faint">
                    {s.status} · ends {new Date(s.termEnd).toLocaleDateString()}
                  </p>
                </li>
              )
            })}
          </ul>
        </Card>
      ) : null}

      {record.recentElections.length > 0 ? (
        <Card className="p-5">
          <h2 className="flex items-center gap-2 text-sm font-semibold text-ink">
            <Vote className="size-4 text-ink-muted" aria-hidden />
            Recent elections
          </h2>
          <ul className="mt-3 space-y-2">
            {record.recentElections.map((e) => (
              <li key={e.id} className="rounded-lg border border-border bg-surface/60 p-3">
                <p className="text-sm text-ink">
                  {e.electionType === 'head_admin' ? 'Head-admin' : 'Admin'} · {e.status.replace('_', ' ')}
                </p>
                <p className="text-[11px] text-ink-faint">
                  {new Date(e.termStart).toLocaleDateString()} → {new Date(e.termEnd).toLocaleDateString()}
                  {e.quorumMet ? ' · quorum met' : ' · no quorum'}
                </p>
              </li>
            ))}
          </ul>
        </Card>
      ) : null}

      {record.revocations.length > 0 ? (
        <Card className="p-5">
          <h2 className="flex items-center gap-2 text-sm font-semibold text-ink">
            <Gavel className="size-4 text-emergency" aria-hidden />
            Revocations ({record.revocations.length})
          </h2>
          <ul className="mt-3 space-y-2">
            {record.revocations.map((r) => (
              <li key={r.id} className="rounded-lg border border-border bg-surface/60 p-3">
                <p className="text-sm text-ink">
                  {r.cycleType === 'head_admin' ? 'Head-admin' : 'Admin'} · {r.status}
                </p>
                <p className="mt-1 text-xs text-ink-muted">{r.reason}</p>
                <p className="mt-1 text-[11px] text-ink-faint">
                  {r.forVotes} for · {r.againstVotes} against · {r.abstainVotes} abstain
                </p>
              </li>
            ))}
          </ul>
        </Card>
      ) : null}
    </div>
  )
}

function VerificationTab({ record }: { record: ReturnType<typeof useUnitRecord>['data'] }) {
  if (!record) return null
  const u = record.unit
  return (
    <Card className="p-5">
      <h2 className="text-sm font-semibold text-ink">Verification</h2>
      <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-2">
        <Field label="Status" value={u.verificationStatus.replace('_', ' ')} />
        <Field label="Verified" value={u.isVerified ? 'Yes' : 'No'} />
        <Field label="Submitted" value={u.verificationSubmittedAt ? new Date(u.verificationSubmittedAt).toLocaleString() : '—'} />
        <Field label="Verified at" value={u.verifiedAt ? new Date(u.verifiedAt).toLocaleString() : '—'} />
        <Field label="Verified by" value={u.verifiedBy ?? '—'} />
        <Field label="Notes" value={u.verificationNotes ?? '—'} />
      </dl>
    </Card>
  )
}

function AuditTab({ record }: { record: ReturnType<typeof useUnitRecord>['data'] }) {
  if (!record) return null
  if (record.auditLog.length === 0) {
    return <Card className="p-6 text-center text-sm text-ink-muted">No audit entries for this unit yet.</Card>
  }
  return (
    <Card className="overflow-hidden">
      <ul className="divide-y divide-border">
        {record.auditLog.map((entry) => (
          <li key={entry.id} className="flex items-start gap-3 p-3">
            <span className="mt-0.5 grid size-8 shrink-0 place-items-center rounded-full bg-surface-hi">
              <ScrollText className="size-3.5 text-ink-muted" aria-hidden />
            </span>
            <div className="min-w-0 flex-1">
              <p className="text-sm text-ink">{entry.action}</p>
              <p className="text-[11px] text-ink-faint">
                {entry.user ? `${entry.user.firstName} ${entry.user.lastName} · ` : ''}
                {relativeTime(entry.timestamp)}
                {entry.ipAddress ? ` · from ${entry.ipAddress}` : ''}
              </p>
              {entry.newValue ? (
                <details className="mt-1">
                  <summary className="cursor-pointer text-[11px] text-signal">Details</summary>
                  <pre className="mt-1 max-h-40 overflow-auto whitespace-pre-wrap rounded bg-surface-hi p-2 text-[10px] text-ink-muted">
                    {entry.newValue}
                  </pre>
                </details>
              ) : null}
            </div>
          </li>
        ))}
      </ul>
    </Card>
  )
}

function Field({ label, value }: { label: string; value?: string | null }) {
  return (
    <div className="min-w-0">
      <dt className="text-[11px] uppercase tracking-wider text-ink-faint">{label}</dt>
      <dd className="mt-0.5 truncate text-sm text-ink">{value || '—'}</dd>
    </div>
  )
}

// Unused import guards

