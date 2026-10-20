import { useEffect, useRef, useState } from 'react'
import { ArrowRight, CheckCircle2, ChevronDown, HeartHandshake, LockKeyhole, MapPinned } from 'lucide-react'
import { Link } from 'react-router-dom'
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

const MORE_SECTION_IDS = new Set([
  'units', 'sos-simulator', 'community-alerts', 'ai-safety', 'governance', 'transparency',
])

/** Public introduction. Existing map, case, emergency and governance tools remain functional. */
export function LandingPage() {
  const [activeAction, setActiveAction] = useState<ActionContext | null>(null)
  const moreToolsRef = useRef<HTMLDetailsElement>(null)

  // Footer links and bookmarked hashes must still work when advanced sections
  // have been grouped behind a disclosure. Never request location on page load.
  useEffect(() => {
    function revealHashDestination() {
      const id = decodeURIComponent(window.location.hash.slice(1))
      if (!MORE_SECTION_IDS.has(id) || !moreToolsRef.current) return
      moreToolsRef.current.open = true
      requestAnimationFrame(() => document.getElementById(id)?.scrollIntoView())
    }
    revealHashDestination()
    window.addEventListener('hashchange', revealHashDestination)
    return () => window.removeEventListener('hashchange', revealHashDestination)
  }, [])

  return (
    <div className="ng-public-scene ng-landing-scene min-h-screen bg-base text-ink selection:bg-signal selection:text-signal-ink">
      <ActionAuthModal
        open={Boolean(activeAction)}
        onClose={() => setActiveAction(null)}
        action={activeAction}
      />
      <LandingNavbar />

      <main className="relative z-10 mx-auto max-w-6xl space-y-10 px-4 pb-5 sm:space-y-14 sm:px-6 lg:px-8">
        <LandingHero onActionTrigger={setActiveAction} />

        <section id="how-it-works" aria-labelledby="how-heading" className="scroll-mt-24 pt-3">
          <div className="max-w-2xl">
            <p className="text-xs font-bold uppercase tracking-[.15em] text-signal">How it works</p>
            <h2 id="how-heading" className="mt-2 text-2xl font-bold tracking-tight sm:text-3xl">Getting started shouldn't be complicated.</h2>
            <p className="mt-3 text-sm leading-6 text-ink-muted">
              Know where to begin, what to expect, and how to follow up — without navigating a technical dashboard.
            </p>
          </div>
          <ol className="mt-6 grid gap-4 md:grid-cols-3">
            <li className="rounded-2xl border border-border bg-surface/80 p-5">
              <span className="grid size-10 place-items-center rounded-xl bg-signal/15 text-signal"><CheckCircle2 className="size-5" aria-hidden /></span>
              <h3 className="mt-4 text-base font-bold">Report a concern</h3>
              <p className="mt-2 text-sm leading-6 text-ink-muted">Share essential details through the reporting form after signing in.</p>
            </li>
            <li className="rounded-2xl border border-border bg-surface/80 p-5">
              <span className="grid size-10 place-items-center rounded-xl bg-signal/15 text-signal"><LockKeyhole className="size-5" aria-hidden /></span>
              <h3 className="mt-4 text-base font-bold">Keep your reference</h3>
              <p className="mt-2 text-sm leading-6 text-ink-muted">Use your report reference to find available public updates; sign in for private details.</p>
            </li>
            <li className="rounded-2xl border border-border bg-surface/80 p-5">
              <span className="grid size-10 place-items-center rounded-xl bg-signal/15 text-signal"><MapPinned className="size-5" aria-hidden /></span>
              <h3 className="mt-4 text-base font-bold">Stay connected</h3>
              <p className="mt-2 text-sm leading-6 text-ink-muted">Explore participating safety units and public incident information on the map.</p>
            </li>
          </ol>
          <p className="mt-3 text-xs text-ink-faint">
            A submitted report does not guarantee dispatch or a particular response time. Emergency help: call 112.
          </p>
        </section>

        <div id="case-lifecycle" className="scroll-mt-24">
          <CaseLifecycleSection onActionTrigger={setActiveAction} />
        </div>

        <div id="map" className="scroll-mt-24">
          <PublicSafetyMapSection onActionTrigger={setActiveAction} />
        </div>

        <details id="more-tools" ref={moreToolsRef} className="group scroll-mt-24 overflow-hidden rounded-2xl border border-border bg-surface/50">
          <summary className="flex cursor-pointer list-none items-center justify-between gap-4 p-5 marker:hidden hover:bg-surface-hi/40 sm:p-7 [&::-webkit-details-marker]:hidden">
            <span>
              <span className="block text-xs font-bold uppercase tracking-[.15em] text-signal">Go further</span>
              <span className="mt-1 block text-xl font-bold text-ink">Explore more community safety tools</span>
              <span className="mt-1 block text-sm leading-6 text-ink-muted">Security units, SOS guidance, alerts, AI assistance and public accountability.</span>
            </span>
            <ChevronDown className="size-6 shrink-0 text-signal transition-transform group-open:rotate-180" aria-hidden />
          </summary>
          <div className="space-y-10 border-t border-border px-4 pb-6 sm:px-7">
            <div id="units" className="scroll-mt-24"><UnitsDirectorySection onActionTrigger={setActiveAction} /></div>
            <div id="sos-simulator" className="scroll-mt-24"><SosSimulatorSection onActionTrigger={setActiveAction} /></div>
            <div id="community-alerts" className="scroll-mt-24"><CommunityAlertsSection onActionTrigger={setActiveAction} /></div>
            <div id="ai-safety" className="scroll-mt-24"><AiSafetySandboxSection onActionTrigger={setActiveAction} /></div>
            <div id="governance" className="scroll-mt-24"><GovernanceSection onActionTrigger={setActiveAction} /></div>
            <div id="transparency" className="scroll-mt-24"><FinancialTransparencySection onActionTrigger={setActiveAction} /></div>
          </div>
        </details>

        <section className="rounded-[1.75rem] border border-border-hi bg-surface p-6 sm:flex sm:items-center sm:justify-between sm:gap-8 sm:p-9">
          <div className="max-w-xl">
            <HeartHandshake className="size-7 text-signal" aria-hidden />
            <h2 className="mt-3 text-2xl font-bold">Safer communities start with us.</h2>
            <p className="mt-3 text-sm leading-6 text-ink-muted">
              Your observations matter. Join Nativity Guard to report concerns, follow updates, and participate in a more accountable community safety network.
            </p>
          </div>
          <Link to="/auth/signup" className="mt-6 inline-flex min-h-11 shrink-0 items-center justify-center gap-2 rounded-xl bg-signal px-6 text-sm font-bold text-signal-ink transition-opacity hover:opacity-90 sm:mt-0">
            Get Started <ArrowRight className="size-4" aria-hidden />
          </Link>
        </section>
      </main>

      <LandingFooter onActionTrigger={setActiveAction} />
    </div>
  )
}
