import { useCallback, useEffect, useState } from 'react'
import { ActionAuthModal } from '@/components/landing/ActionAuthModal'
import { LandingNavbar, type LandingPanel } from '@/components/landing/LandingNavbar'
import { LandingQuickPanel } from '@/components/landing/LandingQuickPanel'
import { LandingTrustFooter } from '@/components/landing/LandingTrustFooter'
import { PANEL_TO_HASH, PANEL_TITLES, panelFromHash } from '@/components/landing/landingPanels'
import { LandingToolContent } from '@/components/landing/LandingToolContent'
import { readLandingAppearance, saveLandingAppearance } from '@/components/landing/landingAppearance'
import { LandingHero } from '@/components/landing/LandingHero'
import { LandingServiceNotice } from '@/components/landing/LandingServiceNotice'
import type { ActionContext } from '@/components/landing/types'
import '@/components/landing/referenceLanding.css'

/**
 * This route intentionally renders only the composition supplied in the
 * reference. The former public tools are still reachable via the navigation
 * and old section hash links, but they open on demand rather than appearing
 * below the hero.
 */
export function LandingPage() {
  const [activeAction, setActiveAction] = useState<ActionContext | null>(null)
  const [appearance, setAppearance] = useState(readLandingAppearance)
  const [panel, setPanel] = useState<LandingPanel | null>(() => typeof window === 'undefined' ? null : panelFromHash(window.location.hash))

  const closePanel = useCallback(() => {
    setPanel(null)
    if (typeof window !== 'undefined' && panelFromHash(window.location.hash)) {
      window.history.replaceState(null, '', window.location.pathname + window.location.search)
    }
  }, [])

  useEffect(() => {
    const syncPanelWithUrl = () => setPanel(panelFromHash(window.location.hash))
    window.addEventListener('hashchange', syncPanelWithUrl)
    window.addEventListener('popstate', syncPanelWithUrl)
    return () => {
      window.removeEventListener('hashchange', syncPanelWithUrl)
      window.removeEventListener('popstate', syncPanelWithUrl)
    }
  }, [])

  function openPanel(next: LandingPanel) {
    setPanel(next)
    if (typeof window !== 'undefined' && panelFromHash(window.location.hash) !== next) {
      window.history.pushState(null, '', '#' + PANEL_TO_HASH[next])
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
        <LandingServiceNotice />
        <LandingHero onActionTrigger={triggerAction} onOpen={openPanel} />
        <LandingTrustFooter />
      </div>

      <ActionAuthModal open={Boolean(activeAction)} action={activeAction} onClose={() => setActiveAction(null)} />

      {panel && (
        <LandingQuickPanel title={PANEL_TITLES[panel]} onClose={closePanel}>
          <LandingToolContent panel={panel} onActionTrigger={triggerAction} />
        </LandingQuickPanel>
      )}
    </div>
  )
}
