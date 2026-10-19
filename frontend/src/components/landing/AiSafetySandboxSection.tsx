import { useState } from 'react'
import {
  Bot,
  CheckCircle2,
  Lock,
  MessageSquare,
  Send,
  Shield,
  Sparkles,
} from 'lucide-react'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Field'
import type { ActionContext } from './types'

export interface AiSafetySandboxSectionProps {
  onActionTrigger: (action: ActionContext) => void
}

interface PromptSample {
  prompt: string
  category: string
  factAnswer: string
  safetyProtocol: string[]
}

const SAFETY_PRESETS: PromptSample[] = [
  {
    category: 'Conflict De-escalation',
    prompt: 'What steps should our community take if cattle encroachment or boundary friction occurs?',
    factAnswer:
      'Nativity Guard coordinates strictly lawful response and prohibits armed citizen confrontation. Ground truth facts are captured first before any enforcement.',
    safetyProtocol: [
      'Document boundary coordinates and timestamps without entering disputed agricultural zones',
      'Notify the assigned Peacebuilding Committee mediator through your unit dashboard',
      'Dispatch verified unit officers strictly in a peacekeeping and observation capacity',
      'Do not mobilize informal armed groups or vigilantes; all actions remain on the auditable record',
    ],
  },
  {
    category: 'Travel Safety',
    prompt: 'How can market traders organize secure cooperative travel before dawn?',
    factAnswer:
      'Early morning transit across rural LGA feeder roads carries elevated risk when vehicles travel in isolation.',
    safetyProtocol: [
      'Coordinate group convoy departures at designated verified unit checkpoints',
      'Share convoy departure and ETA with the LGA channel',
      'Keep your Emergency SOS button primed on 1-tap standby with location sharing active',
      'Ensure at least one vehicle carries a verified unit contact radio or mobile terminal',
    ],
  },
  {
    category: 'Officer Accreditation',
    prompt: 'How do I verify if a security patrol officer is officially accredited?',
    factAnswer:
      'All authorized officers carry verifiable digital credentials issued by their registered unit with supervisory approval.',
    safetyProtocol: [
      'Request the officer’s 8-digit Nativity Guard badge ID (e.g. OFF-0482)',
      'Look up the officer on your community unit roster; check their active duty tier',
      'Accredited officers never demand cash spot-fines or confiscate property without a case tracking ID',
      'If an individual refuses identification, file an immediate confidential sighting report',
    ],
  },
  {
    category: 'Vigilante Protection',
    prompt: 'What should our street committee do if unauthorized levies or vigilante actions are demanded?',
    factAnswer:
      'Nativity Guard operates to dismantle fragmented vigilantism. No security unit is authorized to operate extortionate parallel justice.',
    safetyProtocol: [
      'Check if the group is listed in the official Nativity Guard Security Units Directory',
      'Legitimate units publish their verified bank details publicly and cannot force ad-hoc cash levies',
      'File an anonymous report under the Extortion & Malpractice category',
      'Unit leaders committing unlawful acts are subject to democratic revocation and statutory referral',
    ],
  },
]

export function AiSafetySandboxSection({ onActionTrigger }: AiSafetySandboxSectionProps) {
  const [selectedIdx, setSelectedIdx] = useState(0)
  const [customQuestion, setCustomQuestion] = useState('')

  const activePreset = SAFETY_PRESETS[selectedIdx]

  function handleAskCustom(e?: React.FormEvent) {
    if (e) e.preventDefault()
    const q = customQuestion.trim() || activePreset.prompt
    onActionTrigger({
      type: 'ai_consultation',
      title: 'Consult AI Safety Advisor',
      description:
        'Nativity Guard’s Living Local Intelligence provides personalized risk evaluations, case drafting assistance, and emergency safety checklists.',
      badgeText: `Question: "${q.slice(0, 42)}..."`,
      returnUrl: '/assistant',
      payload: { initialQuestion: q },
    })
  }

  return (
    <section id="ai-safety" className="border-t border-border py-14">
      <div className="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <span className="inline-flex items-center gap-1.5 rounded-full bg-surface-hi px-3 py-1 text-[11px] font-medium text-ink-muted ring-1 ring-border-hi">
            <Sparkles className="size-3.5 text-signal" aria-hidden />
            Living Local Intelligence
          </span>
          <h2 className="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
            Interactive AI Safety Advisor Sandbox
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
            Nativity Guard learns <strong>facts before it learns models</strong>. Test our localized
            safety advisory engine trained on peacebuilding protocols, de-escalation guidelines, and
            rural Nigerian safety realities.
          </p>
        </div>

        <div className="pt-2 md:pt-0">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => handleAskCustom()}
            icon={<Bot className="size-3.5 text-signal" />}
          >
            Open Full Safety Assistant
          </Button>
        </div>
      </div>

      {/* Preset Category Chips */}
      <div className="mt-6 flex flex-wrap items-center gap-2">
        {SAFETY_PRESETS.map((preset, idx) => (
          <button
            key={preset.category}
            type="button"
            onClick={() => setSelectedIdx(idx)}
            className={`rounded-full px-3 py-1.5 text-xs font-medium transition-all ${
              selectedIdx === idx
                ? 'bg-signal text-signal-ink font-semibold shadow-panel'
                : 'bg-surface-hi text-ink-muted hover:text-ink'
            }`}
          >
            {preset.category}
          </button>
        ))}
      </div>

      {/* Interactive Sandbox Layout */}
      <div className="mt-6 grid gap-4 lg:grid-cols-12">
        {/* Preset Prompt Selection Cards */}
        <div className="space-y-2.5 lg:col-span-5">
          {SAFETY_PRESETS.map((item, idx) => (
            <div
              key={item.prompt}
              onClick={() => setSelectedIdx(idx)}
              className={`cursor-pointer rounded-panel border p-3.5 transition-all ${
                selectedIdx === idx
                  ? 'border-signal bg-surface-hi shadow-panel'
                  : 'border-border bg-surface hover:border-border-hi'
              }`}
            >
              <div className="flex items-start gap-2.5">
                <MessageSquare className="size-4 text-signal mt-0.5 shrink-0" aria-hidden />
                <div>
                  <p className="text-xs font-medium text-ink line-clamp-2 leading-relaxed">
                    "{item.prompt}"
                  </p>
                  <span className="text-[10px] text-ink-faint mt-1 block">
                    Category: {item.category}
                  </span>
                </div>
              </div>
            </div>
          ))}

          {/* Quick Ask Input */}
          <form onSubmit={handleAskCustom} className="mt-4 pt-2">
            <div className="relative">
              <Input
                value={customQuestion}
                onChange={(e) => setCustomQuestion(e.target.value)}
                placeholder="Ask any neighborhood safety question..."
                className="pr-10 text-xs"
              />
              <button
                type="submit"
                aria-label="Submit question to safety assistant"
                className="absolute right-2 top-1/2 -translate-y-1/2 grid size-7 place-items-center rounded bg-signal text-signal-ink hover:bg-signal/90"
              >
                <Send className="size-3.5" />
              </button>
            </div>
            <p className="mt-1.5 text-[10px] text-ink-faint">
              Press enter to consult with your authenticated account.
            </p>
          </form>
        </div>

        {/* Live Grounded Response Display */}
        <div className="lg:col-span-7">
          <Card className="p-6 h-full flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between border-b border-border pb-3">
                <div className="flex items-center gap-2">
                  <div className="grid size-7 place-items-center rounded-full bg-signal/15 text-signal">
                    <Shield className="size-4" />
                  </div>
                  <div>
                    <p className="text-xs font-semibold text-ink">Nativity Guard Safety Intelligence</p>
                    <p className="text-[10px] text-ink-faint">Verified Community Safety Protocol</p>
                  </div>
                </div>
                <span className="rounded bg-surface-hi px-2 py-0.5 text-[10px] text-ok font-medium">
                  Facts Over Inferences
                </span>
              </div>

              {/* Question Banner */}
              <div className="mt-4 rounded-lg bg-surface-hi p-3 border border-border">
                <p className="text-xs font-semibold text-signal">User Query:</p>
                <p className="text-xs text-ink mt-0.5 font-medium leading-relaxed">
                  "{activePreset.prompt}"
                </p>
              </div>

              {/* Grounded Fact Statement */}
              <div className="mt-4">
                <p className="text-xs font-semibold uppercase tracking-wider text-ink-muted">
                  Primary Grounding Fact
                </p>
                <p className="mt-1 text-xs leading-relaxed text-ink bg-surface p-3 rounded border border-border">
                  {activePreset.factAnswer}
                </p>
              </div>

              {/* Safety Protocols */}
              <div className="mt-5 space-y-2">
                <p className="text-xs font-semibold uppercase tracking-wider text-ink-muted">
                  Recommended Community Actions
                </p>
                {activePreset.safetyProtocol.map((rule) => (
                  <div key={rule} className="flex items-start gap-2.5 text-xs text-ink-muted">
                    <CheckCircle2 className="size-3.5 text-ok mt-0.5 shrink-0" aria-hidden />
                    <span className="leading-relaxed">{rule}</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="mt-6 border-t border-border pt-4 flex items-center justify-between">
              <span className="text-[11px] text-ink-faint">
                Live Advisor Available 24/7
              </span>
              <Button
                variant="primary"
                size="sm"
                onClick={() => handleAskCustom()}
                icon={<Lock className="size-3.5" />}
              >
                Sign In to Start Full Personalized Session
              </Button>
            </div>
          </Card>
        </div>
      </div>
    </section>
  )
}
