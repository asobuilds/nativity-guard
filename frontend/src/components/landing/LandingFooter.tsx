import { Link } from 'react-router-dom'
import {
  ExternalLink,
  Heart,
  PhoneCall,
  Shield,
  ShieldAlert,
} from 'lucide-react'
import type { ActionContext } from './types'

export interface LandingFooterProps {
  onActionTrigger?: (action: ActionContext) => void
}

export function LandingFooter({ onActionTrigger: _onActionTrigger }: LandingFooterProps) {
  return (
    <footer className="border-t border-border bg-surface/60 backdrop-blur-md mt-20">
      {/* Emergency Hotlines Strip */}
      <div className="border-b border-border bg-emergency/10 py-4 px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-6xl flex flex-col sm:flex-row items-center justify-between gap-3 text-center sm:text-left">
          <div className="flex items-center gap-2">
            <ShieldAlert className="size-4 text-emergency shrink-0" />
            <p className="text-xs font-semibold text-ink">
              Direct National Emergency Hotlines (Nigeria):
            </p>
          </div>
          <div className="flex flex-wrap items-center justify-center gap-2 sm:gap-4">
            <a
              href="tel:112"
              className="inline-flex items-center gap-1.5 rounded-full bg-surface px-3 py-1 text-xs font-bold text-emergency border border-emergency/30 hover:bg-surface-hi transition-colors"
            >
              <PhoneCall className="size-3" />
              112 (National Emergency)
            </a>
            <a
              href="tel:199"
              className="inline-flex items-center gap-1.5 rounded-full bg-surface px-3 py-1 text-xs font-bold text-emergency border border-emergency/30 hover:bg-surface-hi transition-colors"
            >
              <PhoneCall className="size-3" />
              199 (Police Dispatch)
            </a>
            <a
              href="tel:122"
              className="inline-flex items-center gap-1.5 rounded-full bg-surface px-3 py-1 text-xs font-bold text-emergency border border-emergency/30 hover:bg-surface-hi transition-colors"
            >
              <PhoneCall className="size-3" />
              122 (FRSC Road Safety)
            </a>
          </div>
        </div>
      </div>

      <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8 py-12">
        <div className="grid gap-8 sm:grid-cols-2 md:grid-cols-4">
          {/* Brand & Purpose */}
          <div className="sm:col-span-2">
            <div className="flex items-center gap-2">
              <div className="grid size-8 place-items-center rounded-lg bg-surface-hi text-signal ring-1 ring-border">
                <Shield className="size-4" aria-hidden />
              </div>
              <span className="text-sm font-bold tracking-wider text-ink uppercase">
                Nativity Guard
              </span>
            </div>

            <p className="mt-3 max-w-md text-xs leading-relaxed text-ink-muted">
              Sovereign community safety and democratic governance infrastructure for Nigerian
              neighborhoods. Every report answered, every case accounted for, and every unit
              democratically overseen.
            </p>

            <div className="mt-4 flex items-center gap-2 text-[11px] text-ink-faint">
              <span className="size-2 rounded-full bg-ok" />
              <span>System Operational across 774 Local Government Areas</span>
            </div>
          </div>

          {/* Platform Navigation */}
          <div>
            <h4 className="text-xs font-bold uppercase tracking-wider text-ink">Civic Hubs</h4>
            <ul className="mt-3 space-y-2 text-xs text-ink-muted">
              <li>
                <a href="#map" className="hover:text-ink transition-colors">
                  Public Safety Map
                </a>
              </li>
              <li>
                <a href="#case-lifecycle" className="hover:text-ink transition-colors">
                  8-Stage Case Tracker
                </a>
              </li>
              <li>
                <a href="#units" className="hover:text-ink transition-colors">
                  Security Units Directory
                </a>
              </li>
              <li>
                <a href="#sos-simulator" className="hover:text-ink transition-colors">
                  Emergency SOS Protocol
                </a>
              </li>
              <li>
                <a href="#community-alerts" className="hover:text-ink transition-colors">
                  Verified Community Alerts
                </a>
              </li>
              <li>
                <a href="#ai-safety" className="hover:text-ink transition-colors">
                  AI Safety Advisor
                </a>
              </li>
            </ul>
          </div>

          {/* Constitution & Transparency */}
          <div>
            <h4 className="text-xs font-bold uppercase tracking-wider text-ink">
              Governance & Legal
            </h4>
            <ul className="mt-3 space-y-2 text-xs text-ink-muted">
              <li>
                <a href="#governance" className="hover:text-ink transition-colors">
                  The 4 Cardinal Rules
                </a>
              </li>
              <li>
                <a href="#governance" className="hover:text-ink transition-colors">
                  Seat Band Calculator
                </a>
              </li>
              <li>
                <a href="#transparency" className="hover:text-ink transition-colors">
                  Financial Transparency
                </a>
              </li>
              <li>
                <Link to="/about" className="inline-flex items-center gap-1 hover:text-ink transition-colors">
                  Architecture Blueprint
                  <ExternalLink className="size-3 text-signal" />
                </Link>
              </li>
              <li>
                <Link to="/terms" className="hover:text-ink transition-colors">
                  Terms of Service & Rules
                </Link>
              </li>
              <li>
                <a href="mailto:support@nativityguard.app" className="hover:text-ink transition-colors">
                  Inquiries & Support
                </a>
              </li>
            </ul>
          </div>
        </div>

        {/* Legal & Non-Vigilante Disclaimers */}
        <div className="mt-10 border-t border-border pt-6 text-[11px] leading-relaxed text-ink-faint space-y-2">
          <p>
            <strong>Strict Anti-Vigilante Policy:</strong> Nativity Guard coordinates strictly lawful
            community protection and emergency dispatch. It does not arm, direct, or encourage any
            citizen to confront another. Extortionate ad-hoc levies, informal armed patrols, and
            extrajudicial punishments are strictly prohibited under platform rules.
          </p>
          <p>
            <strong>Non-Custodial Transparency:</strong> Nativity Guard does not accept or hold
            operational funds for local patrol units. Donations made to local units are transferred
            directly from citizens to verified commercial bank accounts owned by each registered unit.
          </p>
        </div>

        {/* Bottom Copyright & Open Source notice */}
        <div className="mt-8 flex flex-col sm:flex-row items-center justify-between gap-4 border-t border-border pt-6 text-xs text-ink-faint">
          <p>© {new Date().getFullYear()} Nativity Guard. Open civic safety infrastructure.</p>
          <p className="flex items-center gap-1.5">
            Built for Nigerian communities with
            <Heart className="size-3 text-signal fill-signal" />
            and open accountability.
          </p>
        </div>
      </div>
    </footer>
  )
}
