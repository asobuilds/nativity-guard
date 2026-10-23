import { useState, type ReactNode } from 'react'
import { AlertTriangle, Clock3, FileText, LockKeyhole, Phone, ShieldAlert } from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import type { ActionContext } from './types'

export interface SosSimulatorSectionProps {
  onActionTrigger: (action: ActionContext) => void
}

interface GuidanceStep {
  title: string
  summary: string
  details: string[]
  icon: ReactNode
}

/** Informational guidance. This is NOT a dispatch simulation or evidence of live response capacity. */
const GUIDANCE_STEPS: GuidanceStep[] = [
  {
    title: '1. Get urgent help',
    summary: 'In a real emergency, use the official emergency telephone number.',
    details: [
      'In Nigeria, call 112 for urgent assistance.',
      'Do not rely on an app submission to reach emergency services.',
    ],
    icon: <Phone className="size-5 text-emergency" aria-hidden />,
  },
  {
    title: '2. Understand the SOS feature',
    summary: 'An in-app SOS is an account feature, not an automatic emergency dispatch.',
    details: [
      'Signing in alone does not submit an SOS request.',
      'Availability of any receiving community unit depends on the unit and area.',
    ],
    icon: <ShieldAlert className="size-5 text-warn" aria-hidden />,
  },
  {
    title: '3. Protect private information',
    summary: 'Only share information that is necessary for your request.',
    details: [
      'Location and incident details can be sensitive personal information.',
      'Review the Privacy Notice before providing details.',
    ],
    icon: <LockKeyhole className="size-5 text-signal" aria-hidden />,
  },
  {
    title: '4. Follow available updates',
    summary: 'If a report is successfully accepted, updates may become available in your account.',
    details: [
      'A status update is not a confirmation that responders have been dispatched.',
      'No response time, officer arrival, or resolution is guaranteed.',
    ],
    icon: <Clock3 className="size-5 text-ok" aria-hidden />,
  },
]

export function SosSimulatorSection({ onActionTrigger }: SosSimulatorSectionProps) {
  const [activeStep, setActiveStep] = useState(0)
  const selected = GUIDANCE_STEPS[activeStep]

  function handleOpenSOSOptions() {
    onActionTrigger({
      type: 'sos_broadcast',
      title: 'Emergency SOS options',
      description: 'Sign in to review available SOS options. Signing in does not send an alert or guarantee a response. For immediate help in Nigeria, call 112.',
      badgeText: 'Sign-in required',
      returnUrl: '/sos',
    })
  }

  return (
    <section id="sos-readiness" className="border-t border-border py-14">
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <span className="inline-flex items-center gap-1.5 rounded-full bg-emergency/10 px-3 py-1 text-[11px] font-medium text-emergency ring-1 ring-emergency/30">
            <AlertTriangle className="size-3.5" aria-hidden />
            Emergency safety information
          </span>
          <h2 className="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
            Understand Emergency SOS
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
            This is guidance about the in-app SOS feature, not a live dispatch demonstration.
            Nativity Guard is not an emergency service or a police force.
          </p>
        </div>
        <Button variant="secondary" size="md" onClick={handleOpenSOSOptions} icon={<ShieldAlert className="size-4" aria-hidden />}>
          View SOS options
        </Button>
      </div>

      <div className="mt-6 rounded-panel border border-warn/40 bg-warn/10 p-4">
        <p className="text-sm font-semibold text-ink">Need immediate assistance?</p>
        <p className="mt-1 text-xs leading-relaxed text-ink-muted">
          Call the national emergency number <a href="tel:112" className="font-bold text-signal underline">112</a>.
          A report or SOS submitted through Nativity Guard may be delayed or not received.
        </p>
      </div>

      <div className="mt-8 grid gap-4 lg:grid-cols-12">
        <div className="space-y-2 lg:col-span-5">
          {GUIDANCE_STEPS.map((step, index) => (
            <button
              key={step.title}
              type="button"
              aria-pressed={activeStep === index}
              onClick={() => setActiveStep(index)}
              className={`w-full rounded-panel border p-4 text-left transition-colors ${activeStep === index
                ? 'border-signal bg-surface-hi shadow-panel'
                : 'border-border bg-surface hover:border-border-hi'}`}
            >
              <span className="flex items-center gap-3">
                <span className="grid size-8 shrink-0 place-items-center rounded-lg bg-surface">{step.icon}</span>
                <span className="text-xs font-semibold text-ink">{step.title}</span>
              </span>
            </button>
          ))}
        </div>

        <Card className="p-6 lg:col-span-7">
          <div className="flex items-center gap-2 border-b border-border pb-3 text-xs text-ink-muted">
            <FileText className="size-4" aria-hidden />
            Guidance {activeStep + 1} of {GUIDANCE_STEPS.length}
          </div>
          <h3 className="mt-4 text-lg font-bold text-ink">{selected.title}</h3>
          <p className="mt-2 text-sm leading-relaxed text-ink-muted">{selected.summary}</p>
          <ul className="mt-5 list-disc space-y-2 pl-5 text-xs leading-relaxed text-ink-muted">
            {selected.details.map((detail) => <li key={detail}>{detail}</li>)}
          </ul>
          <div className="mt-8 flex items-center justify-between border-t border-border pt-4">
            <Button variant="secondary" size="sm" disabled={activeStep === 0} onClick={() => setActiveStep((step) => Math.max(0, step - 1))}>
              Previous
            </Button>
            <Button variant="primary" size="sm" onClick={() => {
              if (activeStep === GUIDANCE_STEPS.length - 1) handleOpenSOSOptions()
              else setActiveStep((step) => Math.min(GUIDANCE_STEPS.length - 1, step + 1))
            }}>
              {activeStep === GUIDANCE_STEPS.length - 1 ? 'View SOS options' : 'Next'}
            </Button>
          </div>
        </Card>
      </div>
    </section>
  )
}
