import { Crown, Shield, User as UserIcon } from 'lucide-react'
import { cn } from '@/lib/cn'
import { formatDate, initials, relativeTime } from '@/lib/format'
import { mediaURL } from '@/lib/apiClient'
import type { RosterMember, UnitRoster } from '@/hooks/useUnitRoster'

interface UnitMemberRosterProps {
  roster: UnitRoster | undefined
  isLoading: boolean
}

const ROLE_META: Record<string, { label: string; color: string; ring: string }> = {
  unit_admin: { label: 'Admin', color: 'text-signal', ring: 'ring-signal/30 bg-signal/10' },
  officer: { label: 'Officer', color: 'text-ink-muted', ring: 'ring-border bg-surface-hi' },
}

function roleStyle(member: RosterMember) {
  if (member.isHeadAdmin) {
    return { label: 'Head admin', color: 'text-warn', ring: 'ring-warn/40 bg-warn/10' }
  }
  return ROLE_META[member.role] ?? ROLE_META.officer
}

function sortByHierarchy(members: RosterMember[]): RosterMember[] {
  const rank = (m: RosterMember) => {
    if (m.isHeadAdmin && m.status === 'active') return 0
    if (m.role === 'unit_admin' && m.status === 'active') return 1
    if (m.role === 'officer' && m.status === 'active') return 2
    if (m.status === 'pending') return 3
    return 4
  }
  return [...members].sort((a, b) => {
    const ra = rank(a)
    const rb = rank(b)
    if (ra !== rb) return ra - rb
    return (a.firstName + a.lastName).localeCompare(b.firstName + b.lastName)
  })
}

export function UnitMemberRoster({ roster, isLoading }: UnitMemberRosterProps) {
  if (isLoading) {
    return (
      <div className="space-y-2">
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="h-16 animate-pulse rounded-lg bg-surface-hi/60" />
        ))}
      </div>
    )
  }

  if (!roster || roster.members.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-border px-4 py-10 text-center text-sm text-ink-muted">
        This unit has no members yet.
      </div>
    )
  }

  const sorted = sortByHierarchy(roster.members)

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-3 gap-3">
        <Stat label="Admins" value={roster.counts.admins} />
        <Stat label="Officers" value={roster.counts.officers} />
        <Stat label="Pending" value={roster.counts.pending} />
      </div>

      <ul className="divide-y divide-border rounded-lg border border-border bg-surface/60">
        {sorted.map((m) => (
          <RosterRow key={m.membershipId} member={m} />
        ))}
      </ul>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg border border-border bg-surface/60 p-3 text-center">
      <p className="text-[11px] uppercase tracking-wider text-ink-faint">{label}</p>
      <p className="mt-1 text-xl font-semibold tabular-nums text-ink">{value}</p>
    </div>
  )
}

function RosterRow({ member }: { member: RosterMember }) {
  const meta = roleStyle(member)
  const avatar = mediaURL(member.avatarPath) ?? null
  const fullName = [member.firstName, member.lastName].filter(Boolean).join(' ')

  const Icon = member.isHeadAdmin
    ? Crown
    : member.role === 'unit_admin'
      ? Shield
      : UserIcon

  return (
    <li className="flex items-center gap-3 p-3">
      <div className="relative shrink-0">
        {avatar ? (
          <img
            src={avatar}
            alt=""
            className="size-10 rounded-full border border-border object-cover"
          />
        ) : (
          <div className="grid size-10 place-items-center rounded-full bg-surface-hi text-xs font-semibold text-ink">
            {initials(member.firstName, member.lastName)}
          </div>
        )}
        {member.status === 'pending' ? (
          <span className="absolute -right-1 -top-1 grid size-4 place-items-center rounded-full bg-warn text-[8px] font-bold text-white">
            P
          </span>
        ) : null}
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <p className="truncate text-sm font-medium text-ink">{fullName || 'Unnamed member'}</p>
          <span
            className={cn(
              'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ring-1',
              meta.ring,
              meta.color,
            )}
          >
            <Icon className="size-3" aria-hidden />
            {meta.label}
          </span>
          {member.status === 'pending' ? (
            <span className="rounded-full bg-warn/15 px-2 py-0.5 text-[10px] font-medium text-warn">
              Pending
            </span>
          ) : null}
        </div>
        <p className="mt-0.5 truncate text-xs text-ink-muted">{member.email}</p>
        {member.termEndAt ? (
          <p className="mt-0.5 text-[11px] text-ink-faint">
            Term ends {formatDate(member.termEndAt)}
            {member.consecutiveTerms > 1 ? ` · ${member.consecutiveTerms} terms` : ''}
          </p>
        ) : (
          <p className="mt-0.5 text-[11px] text-ink-faint">
            Joined {relativeTime(member.acceptedAt || member.createdAt)}
          </p>
        )}
      </div>
    </li>
  )
}
