import { useEffect, useRef, useState, type FormEvent, type KeyboardEvent } from 'react'
import { Link } from 'react-router-dom'
import { MapPin, Send, ShieldAlert, Sparkles } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { ApiError, request } from '@/lib/apiClient'
import { cn } from '@/lib/cn'

type Exchange = { id: number; question: string; answer: string }

const SUGGESTIONS = [
  'How do I report an incident?',
  'What should I do if I see a suspicious person?',
  'How does the safety map work?',
  'How can I prepare an emergency contact plan?',
]

/**
 * `/assistant` — Safety Q&A with an AI responder.
 *
 * Chat-shaped layout: user questions on the right in the signal colour,
 * assistant replies on the left with an avatar. The conversation scrolls
 * itself and the composer grows with the text. Everything uses the theme
 * tokens, so the page renders correctly in every theme and text size.
 *
 * Deliberately plain text on the wire: the reply is rendered with
 * `whitespace-pre-wrap`, never as HTML. Markdown rendering can come later
 * behind a reviewed dependency.
 */
export function AiAssistantPage() {
  const [question, setQuestion] = useState('')
  const [exchanges, setExchanges] = useState<Exchange[]>([])
  const [tips, setTips] = useState('')
  const [location, setLocation] = useState('')
  const [pending, setPending] = useState<'chat' | 'tips' | null>(null)
  const [error, setError] = useState('')
  const [showTips, setShowTips] = useState(false)

  const scrollRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // Scroll to the newest message whenever the conversation or typing state changes.
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [exchanges.length, pending])

  // Auto-grow the composer up to ~8 lines, then scroll inside the textarea.
  useEffect(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = Math.min(el.scrollHeight, 160) + 'px'
  }, [question])

  async function submitQuestion(raw: string) {
    const submitted = raw.trim()
    if (!submitted || pending) return
    setPending('chat')
    setError('')

    const id = Date.now()
    setExchanges((items) => [...items, { id, question: submitted, answer: '' }])
    setQuestion('')

    try {
      const result = await request<{ response: string }>('/ai/chatbot', {
        method: 'POST',
        body: { question: submitted },
      })
      setExchanges((items) =>
        items.map((item) => (item.id === id ? { ...item, answer: result.response } : item)),
      )
    } catch (cause) {
      // The message did not get an answer: remove the placeholder, restore
      // the input, and show the exact backend error so it is never hidden.
      setExchanges((items) => items.filter((item) => item.id !== id))
      setQuestion(submitted)
      setError(
        cause instanceof ApiError
          ? `The assistant could not answer: ${cause.message}`
          : 'The assistant could not answer. Please try again.',
      )
    } finally {
      setPending(null)
    }
  }

  function onFormSubmit(e: FormEvent) {
    e.preventDefault()
    void submitQuestion(question)
  }

  function onKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      void submitQuestion(question)
    }
  }

  async function getTips() {
    if (pending) return
    setPending('tips')
    setError('')
    try {
      const result = await request<{ tips: string }>('/ai/smart-tips', {
        method: 'POST',
        body: { location: location.trim() },
      })
      setTips(result.tips)
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? `Safety tips are unavailable: ${cause.message}`
          : 'Safety tips are unavailable right now. Please try again later.',
      )
    } finally {
      setPending(null)
    }
  }

  return (
    <main className="mx-auto w-full max-w-3xl space-y-4 p-3 sm:p-6">
      <header>
        <h1 className="text-2xl font-semibold text-ink">Safety assistant</h1>
        <p className="mt-1 text-sm text-ink-muted">
          Practical safety guidance. Ask anything — replies come back in seconds.
        </p>
      </header>

      {/* Safety disclaimer */}
      <div className="rounded-2xl border border-border bg-surface/70 p-4 backdrop-blur-sm">
        <p className="text-[11px] font-semibold uppercase tracking-wider text-signal">
          Read before you ask
        </p>
        <p className="mt-1.5 text-sm leading-relaxed text-ink-muted">
          Advice may be wrong or out of date. Never share personal details, case evidence, or exact
          addresses here. For immediate danger, use{' '}
          <Link
            to="/sos"
            className="font-medium text-signal underline-offset-2 hover:underline"
          >
            Emergency SOS
          </Link>
          . Official reports and warnings are handled by people, not by AI.
        </p>
      </div>

      {/* Chat panel */}
      <section className="flex flex-col overflow-hidden rounded-3xl border border-border bg-surface/60 shadow-lg backdrop-blur-md">
        <div
          ref={scrollRef}
          className="flex-1 space-y-5 overflow-y-auto p-4 sm:p-6"
          style={{ maxHeight: '55vh', minHeight: '42vh' }}
          role="log"
          aria-label="Assistant conversation"
          aria-live="polite"
        >
          {exchanges.length === 0 && !pending ? (
            <EmptyState onPick={(text) => setQuestion(text)} />
          ) : (
            exchanges.map((item) => (
              <div key={item.id} className="space-y-3">
                <UserBubble text={item.question} />
                {item.answer ? <AssistantBubble text={item.answer} /> : <TypingBubble />}
              </div>
            ))
          )}
        </div>

        <form
          onSubmit={onFormSubmit}
          className="border-t border-border bg-base/50 p-3 backdrop-blur-sm sm:p-4"
        >
          <div className="flex items-end gap-2">
            <div className="relative flex-1">
              <textarea
                ref={textareaRef}
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                onKeyDown={onKeyDown}
                disabled={pending === 'chat'}
                maxLength={1000}
                rows={1}
                placeholder="Ask a safety question…"
                aria-label="Your question"
                className={cn(
                  'w-full resize-none rounded-2xl border border-border bg-surface px-4 py-3 pr-14 text-sm text-ink',
                  'placeholder:text-ink-faint',
                  'focus:border-signal focus:outline-none focus:ring-2 focus:ring-signal/30',
                  'transition-colors disabled:opacity-60',
                )}
              />
              <p className="pointer-events-none absolute bottom-1.5 right-3 text-[10px] tabular-nums text-ink-faint">
                {question.length}/1000
              </p>
            </div>
            <button
              type="submit"
              disabled={!question.trim() || pending !== null}
              aria-label="Send question"
              className={cn(
                'grid size-11 shrink-0 place-items-center rounded-2xl transition-all',
                'bg-signal text-signal-ink shadow-md',
                'hover:scale-[1.04] hover:shadow-lg',
                'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-signal',
                'disabled:cursor-not-allowed disabled:opacity-40 disabled:hover:scale-100',
              )}
            >
              {pending === 'chat' ? (
                <span
                  className="size-4 animate-spin rounded-full border-2 border-signal-ink/30 border-t-signal-ink"
                  aria-hidden
                />
              ) : (
                <Send className="size-4" aria-hidden />
              )}
            </button>
          </div>
          <p className="mt-2 text-center text-[11px] text-ink-faint">
            <kbd className="rounded border border-border bg-surface-hi px-1.5 py-0.5 text-[10px]">Enter</kbd> to send
            {' · '}
            <kbd className="rounded border border-border bg-surface-hi px-1.5 py-0.5 text-[10px]">Shift</kbd>
            {' + '}
            <kbd className="rounded border border-border bg-surface-hi px-1.5 py-0.5 text-[10px]">Enter</kbd> for a new line
          </p>
        </form>
      </section>

      {/* Safety tips panel — collapsed by default */}
      <section className="overflow-hidden rounded-2xl border border-border bg-surface/60 backdrop-blur-md">
        <button
          type="button"
          onClick={() => setShowTips((v) => !v)}
          aria-expanded={showTips}
          className="flex w-full items-center justify-between gap-3 p-4 text-left transition-colors hover:bg-surface-hi/40"
        >
          <span className="flex items-center gap-2">
            <Sparkles className="size-4 shrink-0 text-signal" aria-hidden />
            <span className="text-sm font-medium text-ink">Safety tips for your area</span>
          </span>
          <span
            aria-hidden
            className={cn(
              'text-xs text-ink-muted transition-transform duration-200',
              showTips && 'rotate-180',
            )}
          >
            ▾
          </span>
        </button>

        {showTips ? (
          <div className="space-y-3 border-t border-border p-4">
            <label htmlFor="tips-area" className="block text-xs text-ink-muted">
              Optional area — a city or district only, never a home address
            </label>
            <div className="flex items-center gap-2">
              <div className="relative flex-1">
                <MapPin
                  className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint"
                  aria-hidden
                />
                <input
                  id="tips-area"
                  value={location}
                  onChange={(e) => setLocation(e.target.value)}
                  maxLength={100}
                  placeholder="Lagos"
                  className={cn(
                    'w-full rounded-xl border border-border bg-surface py-2 pl-9 pr-3 text-sm text-ink',
                    'placeholder:text-ink-faint',
                    'focus:border-signal focus:outline-none focus:ring-2 focus:ring-signal/30',
                  )}
                />
              </div>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                disabled={pending !== null}
                loading={pending === 'tips'}
                onClick={() => void getTips()}
              >
                Get tips
              </Button>
            </div>
            {tips ? (
              <div className="rounded-xl border border-signal/30 bg-signal/5 p-3 text-sm leading-relaxed text-ink">
                <p className="whitespace-pre-wrap">{tips}</p>
              </div>
            ) : null}
          </div>
        ) : null}
      </section>

      {error ? (
        <p
          role="alert"
          className="rounded-xl border border-warn/30 bg-warn/10 px-3 py-2 text-sm text-warn"
        >
          {error}
        </p>
      ) : null}
    </main>
  )
}

function UserBubble({ text }: { text: string }) {
  return (
    <div className="flex justify-end">
      <div className="max-w-[85%] rounded-2xl rounded-tr-sm bg-signal px-4 py-2.5 text-sm leading-relaxed text-signal-ink shadow-sm">
        <p className="whitespace-pre-wrap break-words">{text}</p>
      </div>
    </div>
  )
}

function AssistantBubble({ text }: { text: string }) {
  return (
    <div className="flex items-start gap-3">
      <div className="grid size-8 shrink-0 place-items-center rounded-full bg-signal/15 ring-1 ring-signal/30">
        <Sparkles className="size-4 text-signal" aria-hidden />
      </div>
      <div className="max-w-[85%] rounded-2xl rounded-tl-sm border border-border bg-surface-hi/60 px-4 py-3 text-sm leading-relaxed text-ink shadow-sm">
        <p className="whitespace-pre-wrap break-words">{text}</p>
      </div>
    </div>
  )
}

function TypingBubble() {
  return (
    <div className="flex items-start gap-3">
      <div className="grid size-8 shrink-0 place-items-center rounded-full bg-signal/15 ring-1 ring-signal/30">
        <Sparkles className="size-4 text-signal" aria-hidden />
      </div>
      <div className="flex items-center gap-1.5 rounded-2xl rounded-tl-sm border border-border bg-surface-hi/60 px-4 py-3">
        <span
          className="size-1.5 animate-pulse rounded-full bg-ink-faint"
          style={{ animationDelay: '0ms' }}
        />
        <span
          className="size-1.5 animate-pulse rounded-full bg-ink-faint"
          style={{ animationDelay: '150ms' }}
        />
        <span
          className="size-1.5 animate-pulse rounded-full bg-ink-faint"
          style={{ animationDelay: '300ms' }}
        />
        <span className="sr-only">Assistant is typing…</span>
      </div>
    </div>
  )
}

function EmptyState({ onPick }: { onPick: (text: string) => void }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-5 py-6 text-center">
      <div className="grid size-14 place-items-center rounded-2xl bg-signal/15 ring-1 ring-signal/30">
        <ShieldAlert className="size-6 text-signal" aria-hidden />
      </div>
      <div className="max-w-sm">
        <p className="text-sm font-medium text-ink">Ask anything about staying safe</p>
        <p className="mt-1 text-xs text-ink-muted">
          Start with one of these, or type your own question below.
        </p>
      </div>
      <div className="flex flex-wrap justify-center gap-2">
        {SUGGESTIONS.map((s) => (
          <button
            key={s}
            type="button"
            onClick={() => onPick(s)}
            className={cn(
              'rounded-full border border-border bg-surface-hi/40 px-3 py-1.5 text-xs text-ink-muted',
              'transition-all hover:border-signal/40 hover:bg-signal/10 hover:text-ink',
            )}
          >
            {s}
          </button>
        ))}
      </div>
    </div>
  )
}