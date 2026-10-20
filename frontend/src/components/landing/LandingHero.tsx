import {
  ArrowRight,
  BellRing,
  FileText,
  MapPinned,
  SearchCheck,
  ShieldCheck,
  Siren,
} from 'lucide-react'
import { Button } from '@/components/ui/Button'
import type { ActionContext } from './types'

export interface LandingHeroProps {
  onActionTrigger: (action: ActionContext) => void
}

export function LandingHero({ onActionTrigger }: LandingHeroProps) {
  function handleReportClick() {
    onActionTrigger({
      type: 'report_incident',
      title: 'Report a Community Safety Concern',
      description: 'Sign in or create an account to file a report and follow its progress.',
      returnUrl: '/report',
    })
  }

  function handleSosClick() {
    onActionTrigger({
      type: 'sos_broadcast',
      title: 'Open Emergency SOS',
      description: 'SOS broadcasts require a verified account. If you need immediate help, call the national emergency line on 112.',
      badgeText: 'Sign-in required for in-app SOS',
      badgeVariant: 'emergency',
      returnUrl: '/sos',
    })
  }

  return (
    <section
      aria-labelledby="landing-heading"
      className="relative isolate my-5 overflow-hidden rounded-[1.75rem] border border-border-hi/60 shadow-panel"
      style={{
        background: 'radial-gradient(ellipse at 90% 15%, rgba(33,132,105,.38), transparent 54%), linear-gradient(135deg, #071d19 0%, #0d3029 60%, #153c32 100%)',
      }}
    >
      <div className="pointer-events-none absolute -right-20 -top-20 size-80 rounded-full border border-white/10 sm:size-[31rem]" aria-hidden />
      <div className="pointer-events-none absolute -right-4 top-10 size-64 rounded-full border border-white/10 sm:size-[25rem]" aria-hidden />
      <div className="relative grid gap-10 px-5 pb-7 pt-10 sm:px-10 sm:pb-10 sm:pt-14 lg:grid-cols-[1.1fr_.9fr] lg:gap-12">
        <div>
          <span className="inline-flex items-center gap-2 rounded-full border border-[#88c0ac]/35 bg-white/10 px-3 py-1.5 text-[11px] font-semibold tracking-wide text-[#d5eadc]">
            <ShieldCheck className="size-3.5 text-[#f4cb78]" aria-hidden />
            Community safety, made simpler
          </span>
          <h1 id="landing-heading" className="mt-6 max-w-[40rem] text-4xl font-extrabold leading-[1.12] tracking-tight text-white sm:text-5xl lg:text-[3.35rem]">
            Your community.<br />
            <span className="text-[#f4cb78]">Your safety.</span><br />
            Your voice.
          </h1>
          <p className="mt-5 max-w-xl text-base leading-7 text-[#d7e8df]">
            Report safety concerns, follow case updates, and discover community safety resources across Nigeria — all in one place.
          </p>
          <div className="mt-7 flex flex-wrap gap-3">
            <Button variant="primary" size="lg" onClick={handleReportClick} icon={<ArrowRight className="size-4" />} className="font-bold">
              Report an Incident
            </Button>
            <a href="#case-lifecycle" className="inline-flex min-h-11 items-center gap-2 rounded-lg border border-white/30 bg-white/10 px-5 py-2 text-sm font-semibold text-white transition-colors hover:bg-white/20">
              <SearchCheck className="size-4 text-[#f4cb78]" aria-hidden />
              Track My Report
            </a>
          </div>
          <p className="mt-4 text-xs leading-5 text-[#c1d6cb]">
            New here? <a href="#how-it-works" className="font-semibold text-[#f4cb78] underline-offset-4 hover:underline">See how Nativity Guard works</a>.
          </p>
        </div>

        <div className="self-center rounded-[1.5rem] border border-white/15 bg-[#0c231f]/75 p-5 shadow-2xl backdrop-blur-sm sm:p-6">
          <div className="flex items-center justify-between border-b border-white/10 pb-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[.16em] text-[#9dc5b1]">Simple steps</p>
              <h2 className="mt-1 text-lg font-bold text-white">Know what happens next</h2>
            </div>
            <span className="grid size-10 place-items-center rounded-xl bg-[#f4cb78]/15 text-[#f4cb78]"><ShieldCheck className="size-5" aria-hidden /></span>
          </div>
          <ol className="mt-5 space-y-4">
            <li className="flex gap-3">
              <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-[#1f6952] text-white"><FileText className="size-5" aria-hidden /></span>
              <div><p className="text-sm font-bold text-white">1. Share what happened</p><p className="mt-0.5 text-xs leading-5 text-[#c8dfd3]">Submit your report through the app.</p></div>
            </li>
            <li className="flex gap-3">
              <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-[#1f6952] text-white"><BellRing className="size-5" aria-hidden /></span>
              <div><p className="text-sm font-bold text-white">2. Receive a reference</p><p className="mt-0.5 text-xs leading-5 text-[#c8dfd3]">Use your reference to check for updates.</p></div>
            </li>
            <li className="flex gap-3">
              <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-[#1f6952] text-white"><MapPinned className="size-5" aria-hidden /></span>
              <div><p className="text-sm font-bold text-white">3. Stay informed</p><p className="mt-0.5 text-xs leading-5 text-[#c8dfd3]">Explore public updates and nearby participating units.</p></div>
            </li>
          </ol>
          <a href="#map" className="mt-6 inline-flex w-full items-center justify-between rounded-xl border border-[#8eb5a0]/30 bg-white/10 px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/20">
            Explore the Safety Map <ArrowRight className="size-4 text-[#f4cb78]" aria-hidden />
          </a>
        </div>
      </div>

      <div className="relative mx-5 mb-5 rounded-xl border border-[#e99d99]/30 bg-[#5a201f]/75 p-4 sm:mx-10 sm:mb-8 sm:flex sm:items-center sm:justify-between sm:gap-5">
        <div className="flex items-start gap-3">
          <Siren className="mt-0.5 size-5 shrink-0 text-[#ffaba3]" aria-hidden />
          <div>
            <p className="text-sm font-bold text-white">Need help in an emergency?</p>
            <p className="mt-1 text-xs leading-5 text-[#ffe2df]">Call 112 for immediate assistance. In-app SOS requires sign-in and may depend on responder availability.</p>
          </div>
        </div>
        <button type="button" onClick={handleSosClick} className="mt-3 inline-flex min-h-11 w-full shrink-0 items-center justify-center gap-2 rounded-lg bg-[#ed4747] px-5 text-sm font-bold text-white transition-colors hover:bg-[#d83a3a] sm:mt-0 sm:w-auto">
          <Siren className="size-4" aria-hidden />
          Open SOS
        </button>
      </div>
    </section>
  )
}
