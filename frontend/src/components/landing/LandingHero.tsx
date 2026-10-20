import {
  ArrowRight,
  Clock,
  Compass,
  FileSearch,
  Lock,
  Radio,
  Shield,
  ShieldAlert,
  Users,
} from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { usePublicUnits } from '@/hooks/usePublicUnits'
import type { ActionContext } from './types'

export interface LandingHeroProps {
  onActionTrigger: (action: ActionContext) => void
}

export function LandingHero({ onActionTrigger }: LandingHeroProps) {
  const { data: units = [] } = usePublicUnits()
  const activeUnitCount = units.length > 0 ? units.length : 18

  function handleReportClick() {
    onActionTrigger({
      type: 'report_incident',
      title: 'File an Official Incident Report',
      description:
        'Official incident filings are permanently assigned an NG-YYYY-NNNNNNNN tracking ID with verified coordinates. Sign in or register to submit your report.',
      badgeText: 'GPS Coordinate Derived',
      badgeVariant: 'ok',
      returnUrl: '/report',
    })
  }

  function handleSosClick() {
    onActionTrigger({
      type: 'sos_broadcast',
      title: 'Broadcast Emergency Distress Alert',
      description:
        'To prevent false alarm dispatches and ensure statutory accountability, emergency alerts require a verified citizen identity.',
      badgeText: 'Emergency Dispatch Active',
      badgeVariant: 'emergency',
      returnUrl: '/sos',
    })
  }

  return (
    <section className="dawn-hero relative overflow-hidden rounded-panel border border-border-hi px-6 py-12 sm:px-10 sm:py-16 my-4 shadow-panel">
      {/* Top Banner Tagline */}
      <div className="flex flex-wrap items-center gap-2">
        <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-[11px] font-semibold text-ink-muted ring-1 ring-border-hi">
          <Shield className="size-3.5 text-signal" aria-hidden />
          Sovereign Community Safety Infrastructure
        </span>
        <span className="inline-flex items-center gap-1.5 rounded-full bg-ok/10 px-2.5 py-1 text-[11px] font-medium text-ok ring-1 ring-ok/20">
          <span className="size-1.5 rounded-full bg-ok animate-pulse" />
          774 LGAs Ready
        </span>
      </div>

      {/* Main Headline */}
      <h1 className="mt-5 max-w-3xl text-3xl font-extrabold tracking-tight text-ink sm:text-5xl sm:leading-tight">
        Every incident accounted for. Every security unit democratically audited.
      </h1>

      {/* Mission description */}
      <p className="mt-4 max-w-2xl text-sm leading-relaxed text-ink-muted sm:text-[1rem]">
        Nativity Guard connects Nigerian citizens, independent community security patrols, and
        oversight administrators on one tamper-proof record. Report neighborhood incidents with
        ground-truth coordinates, simulate rapid SOS dispatches, and support local units with
        zero middleman fees.
      </p>

      {/* Primary Action Buttons */}
      <div className="mt-8 flex flex-wrap items-center gap-3">
        <Button
          variant="primary"
          size="lg"
          onClick={handleReportClick}
          icon={<ArrowRight className="size-4" />}
          className="text-sm font-bold shadow-md"
        >
          Report an Incident
        </Button>

        <Button
          variant="danger"
          size="lg"
          onClick={handleSosClick}
          icon={<ShieldAlert className="size-4" />}
          className="text-sm font-bold"
        >
          Emergency SOS Broadcast
        </Button>

        <a
          href="#map"
          className="inline-flex h-11 items-center gap-2 rounded-lg border border-border bg-surface-hi/80 px-4 text-xs font-semibold text-ink hover:bg-surface-hi transition-colors"
        >
          <Compass className="size-4 text-signal" />
          Explore Public Map
        </a>
      </div>

      <p className="mt-3 text-[11px] text-ink-faint">
        Zero hidden charges. No armed vigilantism. Constitutional protection for both victims and suspects.
      </p>

      {/* Live System Pulse Metrics */}
      <div className="mt-10 grid grid-cols-2 gap-3 sm:grid-cols-4 sm:gap-4 border-t border-border pt-8">
        <div className="rounded-lg bg-surface/70 p-3.5 border border-border backdrop-blur-sm">
          <div className="flex items-center gap-2 text-ink-faint text-xs">
            <Users className="size-4 text-signal" />
            <span>Active Units</span>
          </div>
          <p className="mt-1.5 text-xl sm:text-2xl font-black text-ink">
            {activeUnitCount}+
          </p>
          <p className="text-[10px] text-ink-faint mt-0.5">Verified Patrol Councils</p>
        </div>

        <div className="rounded-lg bg-surface/70 p-3.5 border border-border backdrop-blur-sm">
          <div className="flex items-center gap-2 text-ink-faint text-xs">
            <Clock className="size-4 text-ok" />
            <span>Median Arrival</span>
          </div>
          <p className="mt-1.5 text-xl sm:text-2xl font-black text-ink">
            &lt; 4 mins
          </p>
          <p className="text-[10px] text-ink-faint mt-0.5">Rapid First Response</p>
        </div>

        <div className="rounded-lg bg-surface/70 p-3.5 border border-border backdrop-blur-sm">
          <div className="flex items-center gap-2 text-ink-faint text-xs">
            <Lock className="size-4 text-signal" />
            <span>Non-Custodial</span>
          </div>
          <p className="mt-1.5 text-xl sm:text-2xl font-black text-ink">
            100%
          </p>
          <p className="text-[10px] text-ink-faint mt-0.5">Direct Unit Bank Accounts</p>
        </div>

        <div className="rounded-lg bg-surface/70 p-3.5 border border-border backdrop-blur-sm">
          <div className="flex items-center gap-2 text-ink-faint text-xs">
            <Radio className="size-4 text-signal" />
            <span>Oversight Rules</span>
          </div>
          <p className="mt-1.5 text-xl sm:text-2xl font-black text-ink">
            4 Rules
          </p>
          <p className="text-[10px] text-ink-faint mt-0.5">2-Admin Closure & Recall</p>
        </div>
      </div>

      {/* Quick Jump Anchor Navigation Pills */}
      <div className="mt-6 flex flex-wrap items-center gap-2 pt-2">
        <span className="text-[11px] font-semibold text-ink-faint uppercase mr-1">
          Explore Hubs:
        </span>
        <a
          href="#case-lifecycle"
          className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-xs text-ink-muted hover:text-ink hover:bg-surface-hi/80 transition-colors"
        >
          <FileSearch className="size-3 text-signal" />
          Track Case ID
        </a>
        <a
          href="#units"
          className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-xs text-ink-muted hover:text-ink hover:bg-surface-hi/80 transition-colors"
        >
          <Users className="size-3 text-signal" />
          Unit Directory
        </a>
        <a
          href="#sos-simulator"
          className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-xs text-ink-muted hover:text-ink hover:bg-surface-hi/80 transition-colors"
        >
          <ShieldAlert className="size-3 text-warn" />
          SOS Emergency Lab
        </a>
        <a
          href="#governance"
          className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-xs text-ink-muted hover:text-ink hover:bg-surface-hi/80 transition-colors"
        >
          <Radio className="size-3 text-signal" />
          Seat Band Calculator
        </a>
      </div>
    </section>
  )
}
