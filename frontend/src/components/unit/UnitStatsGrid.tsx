import { Activity, MapPin, ShieldCheck, Users } from 'lucide-react'
import type { UnitPublicSummary } from '@/hooks/useUnitPublicSummary'
import { cn } from '@/lib/cn'

interface UnitStatsGridProps {
  summary: UnitPublicSummary | undefined
  unit: { operationalRadius: number; totalMembers: number }
  isLoading?: boolean
}

export function UnitStatsGrid({ summary, unit, isLoading }: UnitStatsGridProps) {
  const stats = [
    {
      icon: <Activity className="size-4 text-signal" aria-hidden />,
      label: 'Open cases',
      value: summary ? String(summary.openCases) : '—',
    },
    {
      icon: <Users className="size-4 text-signal" aria-hidden />,
      label: 'Active members',
      value: summary
        ? String(summary.activeMembers || unit.totalMembers || 0)
        : String(unit.totalMembers || 0),
    },
    {
      icon: <MapPin className="size-4 text-signal" aria-hidden />,
      label: 'Coverage',
      value: `${unit.operationalRadius} km`,
    },
    {
      icon: <ShieldCheck className="size-4 text-signal" aria-hidden />,
      label: 'Verified',
      value:
        summary?.verifiedDays && summary.verifiedDays > 0
          ? `${summary.verifiedDays}d ago`
          : summary?.verified
            ? 'Yes'
            : 'Pending',
    },
  ]

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
      {stats.map((s) => (
        <div
          key={s.label}
          className="rounded-panel border border-border bg-surface/70 p-4 backdrop-blur-sm transition-colors hover:border-border-hi"
        >
          <div className="flex items-center gap-2">
            {s.icon}
            <p className="text-[11px] uppercase tracking-wider text-ink-faint">{s.label}</p>
          </div>
          <p
            className={cn(
              'mt-2 text-2xl font-semibold tabular-nums text-ink',
              isLoading && 'opacity-50',
            )}
          >
            {s.value}
          </p>
        </div>
      ))}
    </div>
  )
}
