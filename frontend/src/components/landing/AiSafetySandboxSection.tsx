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

/** Static UI examples, not live model responses or verified local facts. */
export const SAFETY_PRESETS: PromptSample[] = [
  {
    category: 'Check Information',
    prompt: 'How can I tell whether a public safety alert is trustworthy?',
    factAnswer:
      'An example response should encourage checking the publication date, location, issuing organisation, and trusted official information sources.',
    safetyProtocol: [
      'Check whether the alert identifies its source and date.',
      'Cross-check with reliable official announcements before forwarding.',
      'Treat urgent and unverified social messages with caution.',
    ],
  },
  {
    category: 'Report Responsibly',
    prompt: 'What information helps when reporting a non-urgent safety concern?',
    factAnswer:
      'Describe what you personally observed without presenting rumors as confirmed facts.',
    safetyProtocol: [
      'Provide the approximate place and time when appropriate.',
      'Distinguish what you saw from what other people told you.',
      'Do not confront anyone or publish private identifying details.',
    ],
  },
  {
    category: 'Privacy Basics',
    prompt: 'How do I protect personal details when asking for safety advice?',
    factAnswer:
      'Only share information needed for the question, and review the service privacy notice first.',
    safetyProtocol: [
      'Avoid putting passwords, account numbers, addresses, or ID documents in an example question.',
      'Read the published privacy policy before uploading media.',
      'Use authorised support channels for privacy requests.',
    ],
  },
  {
    category: 'Urgent Help',
    prompt: 'Can I rely on an app assistant in a real emergency?',
    factAnswer:
      'No. A chatbot or in-app report is not a substitute for official emergency services.',
    safetyProtocol: [
      'For urgent help in Nigeria, call 112.',
      'Follow advice from official emergency responders.',
      'Do not assume a chat message automatically dispatches assistance.',
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
        'Sign in to check whether the in-app assistant is available. Example answers on this page are static demonstrations, not verified real-time advice. For emergencies in Nigeria, call 112.',
      badgeText: 'Example question — not submitted yet',
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
            Safety information examples
          </span>
          <h2 className="mt-3 text-2xl font-bold tracking-tight text-ink sm:text-3xl">
            Explore Safety Questions
          </h2>
          <p className="mt-2 max-w-2xl text-sm leading-relaxed text-ink-muted">
            These are prewritten examples for learning how to ask safety questions. They are not
            live AI-generated answers, verified incident intelligence, professional advice, or evidence
            that an assistant is available in every area.
          </p>
        </div>

        <div className="pt-2 md:pt-0">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => handleAskCustom()}
            icon={<Bot className="size-3.5 text-signal" />}
          >
            Sign in to Check Assistant
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
            <button
              type="button"
              key={item.prompt}
              aria-pressed={selectedIdx === idx}
              onClick={() => setSelectedIdx(idx)}
              className={`w-full text-left cursor-pointer rounded-panel border p-3.5 transition-all ${
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
            </button>
          ))}

          {/* Quick Ask Input */}
          <form onSubmit={handleAskCustom} className="mt-4 pt-2">
            <div className="relative">
              <Input
                value={customQuestion}
                onChange={(e) => setCustomQuestion(e.target.value)}
                placeholder="Type a general safety question (no personal details)..."
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
              Your question is not sent by this example form. Sign-in is required to check the full assistant.
            </p>
          </form>
        </div>

        {/* Static explanatory sample, not a live model output */}
        <div className="lg:col-span-7">
          <Card className="p-6 h-full flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between border-b border-border pb-3">
                <div className="flex items-center gap-2">
                  <div className="grid size-7 place-items-center rounded-full bg-signal/15 text-signal">
                    <Shield className="size-4" />
                  </div>
                  <div>
                    <p className="text-xs font-semibold text-ink">Safety question example</p>
                    <p className="text-[10px] text-ink-faint">Static educational sample — not verified live advice</p>
                  </div>
                </div>
                <span className="rounded bg-surface-hi px-2 py-0.5 text-[10px] text-ok font-medium">
                  Example, not a live response
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
                  Example explanation
                </p>
                <p className="mt-1 text-xs leading-relaxed text-ink bg-surface p-3 rounded border border-border">
                  {activePreset.factAnswer}
                </p>
              </div>

              {/* Safety Protocols */}
              <div className="mt-5 space-y-2">
                <p className="text-xs font-semibold uppercase tracking-wider text-ink-muted">
                  General safety considerations
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
                Real-time service availability is not guaranteed
              </span>
              <Button
                variant="primary"
                size="sm"
                onClick={() => handleAskCustom()}
                icon={<Lock className="size-3.5" />}
              >
                Sign in to Explore Assistant
              </Button>
            </div>
          </Card>
        </div>
      </div>
    </section>
  )
}
