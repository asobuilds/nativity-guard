import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { ActionAuthModal } from '@/components/landing/ActionAuthModal'
import { LandingNavbar, type LandingPanel } from '@/components/landing/LandingNavbar'
import { LandingToolContent } from '@/components/landing/LandingToolContent'
import { readLandingAppearance, saveLandingAppearance } from '@/components/landing/landingAppearance'
import { LandingHero } from '@/components/landing/LandingHero'
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
  const [appearance, setAppearance] = useState(readLandingAppearance)
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
    <div className="ng-ref-page" data-landing-palette={appearance.palette} data-landing-wallpaper={appearance.wallpaper}>
      <div className="ng-ref-shell">
        <LandingNavbar onOpen={openPanel} onReport={report} appearance={appearance} onAppearanceChange={value => setAppearance(saveLandingAppearance(value))} />
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
              <LandingToolContent panel={panel} onActionTrigger={triggerAction} />
            </div>
          </section>
        </div>
      )}
    </div>
  )
}
