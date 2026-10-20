import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { ActionAuthModal } from '@/components/landing/ActionAuthModal'
import { LandingNavbar, type LandingPanel } from '@/components/landing/LandingNavbar'
import { LandingHero } from '@/components/landing/LandingHero'
import { PublicSafetyMapSection } from '@/components/landing/PublicSafetyMapSection'
import { CaseLifecycleSection } from '@/components/landing/CaseLifecycleSection'
import { UnitsDirectorySection } from '@/components/landing/UnitsDirectorySection'
import { SosSimulatorSection } from '@/components/landing/SosSimulatorSection'
import { CommunityAlertsSection } from '@/components/landing/CommunityAlertsSection'
import { AiSafetySandboxSection } from '@/components/landing/AiSafetySandboxSection'
import { GovernanceSection } from '@/components/landing/GovernanceSection'
import { FinancialTransparencySection } from '@/components/landing/FinancialTransparencySection'
import type { ActionContext } from '@/components/landing/types'
import '@/components/landing/referenceLanding.css'

const HASH_TO_PANEL: Record<string, LandingPanel> = {
  map: 'map',
  'case-lifecycle': 'track',
  track: 'track',
  units: 'units',
  'sos-simulator': 'sos-simulator',
  'community-alerts': 'community-alerts',
  'ai-safety': 'ai-safety',
  governance: 'governance',
  transparency: 'transparency',
}
const PANEL_TO_HASH: Record<LandingPanel, string> = {
  map: 'map', track: 'case-lifecycle', units: 'units',
  'sos-simulator': 'sos-simulator', 'community-alerts': 'community-alerts',
  'ai-safety': 'ai-safety', governance: 'governance', transparency: 'transparency',
}
const PANEL_TITLES: Record<LandingPanel, string> = {
  map: 'Safety Map', track: 'Track a Case', units: 'Security Units',
  'sos-simulator': 'Emergency SOS Guidance', 'community-alerts': 'Community Alerts',
  'ai-safety': 'AI Safety Advisor', governance: 'Governance', transparency: 'Financial Transparency',
}

function panelFromHash() {
  if (typeof window === 'undefined') return null
  return HASH_TO_PANEL[decodeURIComponent(window.location.hash.slice(1))] ?? null
}

/**
 * This route intentionally renders only the composition supplied in the
 * reference. The former public tools are still reachable via the navigation
 * and old section hash links, but they open on demand rather than appearing
 * below the hero.
 */
export function LandingPage() {
  const [activeAction, setActiveAction] = useState<ActionContext | null>(null)
  const [panel, setPanel] = useState<LandingPanel | null>(panelFromHash)

  useEffect(() => {
    const onHashChange = () => setPanel(panelFromHash())
    window.addEventListener('hashchange', onHashChange)
    return () => window.removeEventListener('hashchange', onHashChange)
  }, [])

  useEffect(() => {
    if (!panel) return
    const previousOverflow = document.body.style.overflow
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    document.body.style.overflow = 'hidden'
    const closeWithEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') closePanel()
    }
    window.addEventListener('keydown', closeWithEscape)
    return () => {
      window.removeEventListener('keydown', closeWithEscape)
      document.body.style.overflow = previousOverflow
      previousFocus?.focus()
    }
  }, [panel])

  function openPanel(next: LandingPanel) {
    setPanel(next)
    window.history.replaceState(null, '', '#' + PANEL_TO_HASH[next])
  }

  function closePanel() {
    setPanel(null)
    if (panelFromHash()) {
      window.history.replaceState(null, '', window.location.pathname + window.location.search)
    }
  }

  function report() {
    closePanel()
    setActiveAction({
      type: 'report_incident',
      title: 'Report an Incident',
      description: 'Sign in or create an account to file an incident report.',
      returnUrl: '/report',
    })
  }

  function triggerAction(action: ActionContext) {
    closePanel()
    setActiveAction(action)
  }

  return (
    <div className="ng-ref-page">
      <div className="ng-ref-shell">
        <LandingNavbar onOpen={openPanel} onReport={report} />
        <LandingHero onActionTrigger={triggerAction} onOpen={openPanel} />
      </div>

      <ActionAuthModal open={Boolean(activeAction)} action={activeAction} onClose={() => setActiveAction(null)} />

      {panel && (
        <div className="ng-ref-modal-backdrop" onMouseDown={(event) => { if (event.currentTarget === event.target) closePanel() }}>
          <section className="ng-ref-modal-window" role="dialog" aria-modal="true" aria-label={PANEL_TITLES[panel]}>
            <div className="ng-ref-modal-heading">
              <h2>{PANEL_TITLES[panel]}</h2>
              <button type="button" onClick={closePanel} aria-label="Close panel"><X size={24} aria-hidden /></button>
            </div>
            <div className="ng-ref-modal-content">
              {panel === 'map' && <PublicSafetyMapSection onActionTrigger={triggerAction} />}
              {panel === 'track' && <CaseLifecycleSection onActionTrigger={triggerAction} />}
              {panel === 'units' && <UnitsDirectorySection onActionTrigger={triggerAction} />}
              {panel === 'sos-simulator' && <SosSimulatorSection onActionTrigger={triggerAction} />}
              {panel === 'community-alerts' && <CommunityAlertsSection onActionTrigger={triggerAction} />}
              {panel === 'ai-safety' && <AiSafetySandboxSection onActionTrigger={triggerAction} />}
              {panel === 'governance' && <GovernanceSection onActionTrigger={triggerAction} />}
              {panel === 'transparency' && <FinancialTransparencySection onActionTrigger={triggerAction} />}
            </div>
          </section>
        </div>
      )}
    </div>
  )
}
