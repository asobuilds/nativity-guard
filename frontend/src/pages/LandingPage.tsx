import { useState } from 'react'
import { ActionAuthModal } from '@/components/landing/ActionAuthModal'
import { LandingNavbar } from '@/components/landing/LandingNavbar'
import { LandingHero } from '@/components/landing/LandingHero'
import { PublicSafetyMapSection } from '@/components/landing/PublicSafetyMapSection'
import { CaseLifecycleSection } from '@/components/landing/CaseLifecycleSection'
import { UnitsDirectorySection } from '@/components/landing/UnitsDirectorySection'
import { SosSimulatorSection } from '@/components/landing/SosSimulatorSection'
import { CommunityAlertsSection } from '@/components/landing/CommunityAlertsSection'
import { AiSafetySandboxSection } from '@/components/landing/AiSafetySandboxSection'
import { GovernanceSection } from '@/components/landing/GovernanceSection'
import { FinancialTransparencySection } from '@/components/landing/FinancialTransparencySection'
import { LandingFooter } from '@/components/landing/LandingFooter'
import type { ActionContext } from '@/components/landing/types'

/**
 * Nativity Guard Public Landing Hub — `/`
 *
 * Provides a comprehensive, interactive exploration of the entire system for
 * citizens and guests before signing in:
 * - Public Safety Map with State/LGA filters and unit patrol zones
 * - 8-Stage Case Lifecycle visualizer and live incident lookup
 * - Accredited Security Units Directory with 1-click verified bank details
 * - Emergency SOS Simulator and direct Nigerian distress hotlines
 * - Verified Community Alerts ticker and weather warnings
 * - Grounded Living Local Intelligence AI safety advisor sandbox
 * - Democratic Governance visualizer and administrative seat band calculator
 * - Non-custodial Financial Transparency and public supporter wall
 *
 * Official operations (SOS broadcast, filing incidents, enlisting in units,
 * voting in elections) seamlessly invoke the Action-Aware Authentication modal.
 */
export function LandingPage() {
  const [activeAction, setActiveAction] = useState<ActionContext | null>(null)

  function handleActionTrigger(action: ActionContext) {
    setActiveAction(action)
  }

  return (
    <div className="ng-public-scene min-h-screen bg-base text-ink selection:bg-signal selection:text-signal-ink">
      {/* Action-Aware Authentication Modal */}
      <ActionAuthModal
        open={Boolean(activeAction)}
        onClose={() => setActiveAction(null)}
        action={activeAction}
      />

      {/* Top Glass Navbar with Theme Switcher & Mobile Drawer */}
      <LandingNavbar />

      {/* Main Content Hubs */}
      <main className="relative z-10 mx-auto max-w-6xl px-4 sm:px-6 lg:px-8 space-y-12 sm:space-y-16">
        {/* Hub 1: Hero & Real-time Pulse */}
        <LandingHero onActionTrigger={handleActionTrigger} />

        {/* Hub 2: Interactive Public Safety Map */}
        <div id="map" className="scroll-mt-20">
          <PublicSafetyMapSection onActionTrigger={handleActionTrigger} />
        </div>

        {/* Hub 3: 8-Stage Case Lifecycle & Public Feed */}
        <div id="case-lifecycle" className="scroll-mt-20">
          <CaseLifecycleSection onActionTrigger={handleActionTrigger} />
        </div>

        {/* Hub 4: Accredited Security Units Directory */}
        <div id="units" className="scroll-mt-20">
          <UnitsDirectorySection onActionTrigger={handleActionTrigger} />
        </div>

        {/* Hub 5: Emergency SOS Simulator & Readiness Lab */}
        <div id="sos-simulator" className="scroll-mt-20">
          <SosSimulatorSection onActionTrigger={handleActionTrigger} />
        </div>

        {/* Hub 6: Community Safety Bulletins & Alerts Ticker */}
        <div id="community-alerts" className="scroll-mt-20">
          <CommunityAlertsSection onActionTrigger={handleActionTrigger} />
        </div>

        {/* Hub 7: AI Safety Advisor Sandbox */}
        <div id="ai-safety" className="scroll-mt-20">
          <AiSafetySandboxSection onActionTrigger={handleActionTrigger} />
        </div>

        {/* Hub 8: Democratic Governance & Seat Band Calculator */}
        <div id="governance" className="scroll-mt-20">
          <GovernanceSection onActionTrigger={handleActionTrigger} />
        </div>

        {/* Hub 9: Financial Transparency & Supporter Ledger */}
        <div id="transparency" className="scroll-mt-20">
          <FinancialTransparencySection onActionTrigger={handleActionTrigger} />
        </div>
      </main>

      {/* Footer & Emergency Hotlines */}
      <LandingFooter onActionTrigger={handleActionTrigger} />
    </div>
  )
}
