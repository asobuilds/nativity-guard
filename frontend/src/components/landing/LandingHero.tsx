import { useQuery } from '@tanstack/react-query'
import { ArrowRight, Clock3, FileText, MapPin, PhoneCall, Search, ShieldCheck, ShieldAlert, UsersRound } from 'lucide-react'
import { api } from '@/lib/apiClient'
import type { SecurityUnit } from '@/types/api'
import type { ActionContext } from './types'
import type { LandingPanel } from './LandingNavbar'

export interface LandingHeroProps {
  onActionTrigger: (action: ActionContext) => void
  onOpen: (panel: LandingPanel) => void
}

export function LandingHero({ onActionTrigger, onOpen }: LandingHeroProps) {
  // Do not count the sample records in usePublicUnits as live verified operations.
  const { data: verifiedCount } = useQuery({
    queryKey: ['public', 'verified-landing-units-count'],
    queryFn: async () => {
      const response = await api.get<{ units: SecurityUnit[] }>('/public/units')
      return (response.units ?? []).filter(unit => unit.status === 'active' && unit.isVerified).length
    },
    retry: false,
    staleTime: 5 * 60_000,
  })

  function report() {
    onActionTrigger({
      type: 'report_incident',
      title: 'Report an Incident',
      description: 'Sign in or create an account to file your incident and follow its progress.',
      returnUrl: '/report',
    })
  }

  function sos() {
    onActionTrigger({
      type: 'sos_broadcast',
      title: 'Emergency SOS',
      description: 'In-app SOS requires a verified account and does not guarantee a responder. For immediate emergency help, call 112.',
      badgeText: 'Sign-in required',
      badgeVariant: 'emergency',
      returnUrl: '/sos',
    })
  }

  const statistics = [
    { icon: UsersRound, number: verifiedCount === undefined ? '—' : String(verifiedCount), label: 'Verified patrol units' },
    { icon: MapPin, number: '36', label: 'Nigerian states' },
    { icon: ShieldCheck, number: '2', label: 'Case closure approvals' },
    { icon: Clock3, number: '112', label: 'National emergency line' },
  ]

  return (
    <main id="home" className="ng-ref-stage">
      <div className="ng-ref-photo" aria-hidden="true" />
      <div className="ng-ref-stage-shade" aria-hidden="true" />
      <div className="ng-ref-stage-content">
        <span className="ng-ref-eyebrow"><ShieldCheck size={15} strokeWidth={2.2} aria-hidden /> Community Safety & Case Tracking</span>
        <h1 className="ng-ref-headline">
          Your Community.<br />
          Your Safety. <span>Your Voice.</span>
        </h1>
        <p className="ng-ref-intro">Report security concerns, stay informed, and help build<br className="ng-ref-break" /> safer communities across Nigeria.</p>

        <div className="ng-ref-actions" aria-label="Main safety actions">
          <button type="button" className="ng-ref-action ng-ref-action-report" onClick={report}>
            <span className="ng-ref-action-icon"><FileText size={30} strokeWidth={2} aria-hidden /></span>
            <span className="ng-ref-action-text"><strong>Report an Incident</strong><small>Quick and easy reporting</small></span>
            <ArrowRight className="ng-ref-action-arrow" size={22} strokeWidth={2} aria-hidden />
          </button>
          <button type="button" className="ng-ref-action" onClick={() => onOpen('track')}>
            <span className="ng-ref-action-icon"><Search size={30} strokeWidth={2} aria-hidden /></span>
            <span className="ng-ref-action-text"><strong>Track My Report</strong><small>Check status and updates</small></span>
            <ArrowRight className="ng-ref-action-arrow" size={22} strokeWidth={2} aria-hidden />
          </button>
          <button type="button" className="ng-ref-action" onClick={() => onOpen('map')}>
            <span className="ng-ref-action-icon ng-ref-map-icon"><MapPin size={30} strokeWidth={2} aria-hidden /></span>
            <span className="ng-ref-action-text"><strong>Explore Safety Map</strong><small>View units and incidents</small></span>
            <ArrowRight className="ng-ref-action-arrow" size={22} strokeWidth={2} aria-hidden />
          </button>
        </div>

        <div className="ng-ref-sos-bar">
          <div className="ng-ref-sos-copy">
            <span className="ng-ref-sos-symbol"><ShieldAlert size={28} strokeWidth={2.1} aria-hidden /></span>
            <span><strong>Emergency SOS</strong><small>For immediate assistance, call 112. In-app SOS requires sign-in.</small></span>
          </div>
          <button type="button" className="ng-ref-sos-button" onClick={sos}><PhoneCall size={22} fill="currentColor" aria-hidden /> SOS</button>
        </div>

        <div className="ng-ref-stats" aria-label="Safety information">
          {statistics.map(({ icon: Icon, number, label }) => (
            <div className="ng-ref-stat" key={label}>
              <Icon size={29} fill="none" strokeWidth={2.4} aria-hidden />
              <div><strong>{number}</strong><small>{label}</small></div>
            </div>
          ))}
        </div>
      </div>
    </main>
  )
}
