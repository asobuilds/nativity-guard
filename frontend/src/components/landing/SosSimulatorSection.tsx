import { useState, type ReactNode } from 'react'
import {
  AlertTriangle,
  CheckCircle2,
  FileVideo,
  Mic,
  Navigation,
  Phone,
  Radio,
  ShieldAlert,
} from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import type { ActionContext } from './types'

export interface SosSimulatorSectionProps {
  onActionTrigger: (action: ActionContext) => void
}

interface SimStep {
  title: string
  subtitle: string
  description: string
  details: string[]
  icon: ReactNode
  statusColor: string
}

export function SosSimulatorSection({ onActionTrigger }: SosSimulatorSectionProps) {
  const [activeStep, setActiveStep] = useState(0)

  const steps: SimStep[] = [
    {
      title: '1. One-Tap Distress Broadcast',
      subtitle: 'Verified Coordinate Generation',
      description:
        'When danger strikes, you tap the Emergency SOS button. Your device samples high-precision GPS coordinates, and the server resolves your administrative locality using server-side reverse geocoding.',
      details: [
        'Place name derived on server: clients never send fake state/LGA strings',
        'Optional anonymous toggle: shields your profile from public feeds while preserving exact location for responders',
        'Immediate local panic broadcast to units within operational radius',
      ],
      icon: <Radio className="size-5 text-emergency" />,
      statusColor: 'text-emergency border-emergency/30 bg-emergency/10',
    },
    {
      title: '2. Voice & Video Evidence Preservation',
      subtitle: 'SHA-256 Content-Addressed Store',
      description:
        'Capture up to 60 seconds of voice audio or 30 seconds of short video from the scene. Bytes are presigned and verified before ingestion.',
      details: [
        'Voice Notes: Capped at 5 MB (Opus/AAC audio format with high compression)',
        'Incident Video: Capped at 20 MB (optimised 30s clips that transmit over 2G/3G)',
        'Content Deduplication: Identical evidence from multiple witnesses shares one storage key with refcounting',
      ],
      icon: <Mic className="size-5 text-warn" />,
      statusColor: 'text-warn border-warn/30 bg-warn/10',
    },
    {
      title: '3. Tactical Unit Dispatch & Assignment',
      subtitle: 'Two-Officer Command Protocol',
      description:
        'The alert sounds an alarm in the nearest verified unit operations queue. A unit administrator assigns a primary and paired officer to take the case.',
      details: [
        'Unit access is NOT case access: only assigned officers can view the sensitive distress details',
        'Officers acknowledge dispatch and mark their departure time in the audit log',
        'Direct officer self-closure is prevented: every closure requires 2 administrator sign-offs',
      ],
      icon: <ShieldAlert className="size-5 text-signal" />,
      statusColor: 'text-signal border-signal/30 bg-signal/10',
    },
    {
      title: '4. Live Responder Tracking',
      subtitle: 'Foreground Location Sharing',
      description:
        'Watch the responding officer approach on your map in real time. The officer shares foreground GPS breadcrumbs until arrival on scene.',
      details: [
        'Transparent ETA tracking so citizens are never left guessing',
        'Audit trail records exact arrival timestamp',
        'Case status transitions from dispatched → on_scene → investigating',
      ],
      icon: <Navigation className="size-5 text-ok" />,
      statusColor: 'text-ok border-ok/30 bg-ok/10',
    },
  ]

  function handleTriggerRealSOS() {
    onActionTrigger({
      type: 'sos_broadcast',
      title: 'Emergency SOS Broadcast',
      description:
        'Emergency SOS alerts real community security units and dispatches officers to your current GPS coordinates. False reports are subject to account termination.',
      badgeText: 'Emergency Dispatch Action',
      returnUrl: '/sos',
    })
  }

  return (
    <section id="sos-readiness" className="border-t border-border py-14">
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <span className="inline-flex items-center gap-1.5 rounded-full bg-emergency/10 px-3 py-1 text-[11px] font-medium text-emergency ring-1 ring-emergency/30">
            <AlertTriangle className="size-3.5" aria-hidden />
            Emergency Infrastructure
          </span>
          <h2 className="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
            How Emergency SOS Works in Practice
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
            Designed for low-bandwidth networks in rural and peri-urban Nigeria. Audio and video
            evidence upload seamlessly even on spotty 2G/3G connections.
          </p>
        </div>

        <div className="pt-2 md:pt-0">
          <Button
            variant="danger"
            size="md"
            onClick={handleTriggerRealSOS}
            icon={<ShieldAlert className="size-4" />}
            className="shadow-panel font-bold"
          >
            Launch Emergency SOS
          </Button>
        </div>
      </div>

      {/* Interactive Simulator Grid */}
      <div className="mt-8 grid gap-4 lg:grid-cols-12">
        {/* Step Selector Buttons */}
        <div className="space-y-2 lg:col-span-5">
          {steps.map((s, idx) => (
            <div
              key={s.title}
              onClick={() => setActiveStep(idx)}
              className={`cursor-pointer rounded-panel border p-4 transition-all ${
                activeStep === idx
                  ? 'border-signal bg-surface-hi shadow-panel'
                  : 'border-border bg-surface hover:border-border-hi'
              }`}
            >
              <div className="flex items-center gap-3">
                <span className="grid size-8 place-items-center rounded-lg bg-surface">
                  {s.icon}
                </span>
                <div>
                  <p className="text-xs font-semibold text-ink">{s.title}</p>
                  <p className="text-[11px] text-ink-muted">{s.subtitle}</p>
                </div>
              </div>
            </div>
          ))}

          {/* National Hotlines Box */}
          <Card className="p-4 border-warn/30 bg-warn/5 mt-4">
            <div className="flex items-start gap-2.5">
              <Phone className="size-4 text-warn mt-0.5 shrink-0" aria-hidden />
              <div>
                <p className="text-xs font-semibold text-ink">National Emergency Hotlines</p>
                <p className="text-[11px] text-ink-muted mt-0.5">
                  Always call statutory authorities for life-threatening emergencies:
                </p>
                <div className="mt-2 flex flex-wrap gap-2 text-xs">
                  <a
                    href="tel:112"
                    className="inline-flex items-center gap-1 rounded bg-surface px-2 py-1 text-ink font-mono font-bold hover:text-signal"
                  >
                    112 <span className="text-[10px] font-normal text-ink-faint">(National)</span>
                  </a>
                  <a
                    href="tel:199"
                    className="inline-flex items-center gap-1 rounded bg-surface px-2 py-1 text-ink font-mono font-bold hover:text-signal"
                  >
                    199 <span className="text-[10px] font-normal text-ink-faint">(Police)</span>
                  </a>
                  <a
                    href="tel:122"
                    className="inline-flex items-center gap-1 rounded bg-surface px-2 py-1 text-ink font-mono font-bold hover:text-signal"
                  >
                    122 <span className="text-[10px] font-normal text-ink-faint">(FRSC)</span>
                  </a>
                </div>
              </div>
            </div>
          </Card>
        </div>

        {/* Active Step Deep-Dive Canvas */}
        <div className="lg:col-span-7">
          <Card className="p-6 h-full flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between border-b border-border pb-3">
                <div className="flex items-center gap-2">
                  <span className="grid size-7 place-items-center rounded-full bg-signal/10 text-signal font-bold text-xs">
                    {activeStep + 1}
                  </span>
                  <span className="text-xs font-semibold text-ink">
                    Stage {activeStep + 1} of 4: {steps[activeStep].subtitle}
                  </span>
                </div>
                <span className="text-[11px] text-ink-faint">Auditable Protocol</span>
              </div>

              <h3 className="mt-4 text-lg font-bold text-ink">{steps[activeStep].title}</h3>
              <p className="mt-2 text-xs leading-relaxed text-ink-muted">
                {steps[activeStep].description}
              </p>

              {/* Technical Specifications for this step */}
              <div className="mt-6 space-y-2.5">
                <p className="text-xs font-semibold uppercase tracking-wider text-ink-muted">
                  Technical Architecture Rules
                </p>
                {steps[activeStep].details.map((detail) => (
                  <div key={detail} className="flex items-start gap-2.5 text-xs text-ink-muted">
                    <CheckCircle2 className="size-3.5 text-ok mt-0.5 shrink-0" aria-hidden />
                    <span>{detail}</span>
                  </div>
                ))}
              </div>

              {/* Simulation Visual Mockup */}
              {activeStep === 1 ? (
                <div className="mt-5 rounded-panel border border-border bg-surface-hi p-4">
                  <p className="text-[11px] font-semibold text-signal uppercase tracking-wider">
                    Bandwidth Optimization Benchmark
                  </p>
                  <div className="mt-2 grid grid-cols-2 gap-3 text-xs">
                    <div className="rounded bg-surface p-2.5 border border-border">
                      <div className="flex items-center gap-1.5 text-ink font-medium">
                        <Mic className="size-3.5 text-signal" />
                        Voice Note
                      </div>
                      <p className="text-[11px] text-ink-muted mt-1">60s Opus / AAC</p>
                      <p className="text-[10px] text-ok mt-0.5">~0.6 MB (Max 5 MB)</p>
                    </div>
                    <div className="rounded bg-surface p-2.5 border border-border">
                      <div className="flex items-center gap-1.5 text-ink font-medium">
                        <FileVideo className="size-3.5 text-warn" />
                        Incident Video
                      </div>
                      <p className="text-[11px] text-ink-muted mt-1">30s Mobile MP4</p>
                      <p className="text-[10px] text-ok mt-0.5">~18 MB (Max 20 MB)</p>
                    </div>
                  </div>
                </div>
              ) : null}
            </div>

            <div className="mt-8 border-t border-border pt-4 flex items-center justify-between">
              <span className="text-[11px] text-ink-faint">
                Step {activeStep + 1} of 4
              </span>
              <div className="flex gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={activeStep === 0}
                  onClick={() => setActiveStep((prev) => Math.max(0, prev - 1))}
                >
                  Previous
                </Button>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={() =>
                    activeStep === 3
                      ? handleTriggerRealSOS()
                      : setActiveStep((prev) => Math.min(3, prev + 1))
                  }
                >
                  {activeStep === 3 ? 'Trigger Official SOS' : 'Next Step'}
                </Button>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </section>
  )
}
