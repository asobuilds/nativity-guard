import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { ApiError, request } from '@/lib/apiClient'

type Exchange = { question: string; answer: string }

/**
 * `/assistant` — general safety Q&A and context-aware tips.
 *
 * The backend was previously gated behind a build-time env flag
 * (`VITE_ENABLE_AI_ASSISTANT`), which showed a "coming soon" placeholder
 * whenever the flag was not set. That flag is gone: the AI service is live,
 * so the form renders for every signed-in user.
 *
 * Errors are surfaced verbatim from the API so a misconfiguration (a missing
 * provider key, a rate limit) is visible instead of hidden behind a vague
 * "try again later".
 */
export function AiAssistantPage() {
  const [question, setQuestion] = useState('')
  const [exchanges, setExchanges] = useState<Exchange[]>([])
  const [tips, setTips] = useState('')
  const [location, setLocation] = useState('')
  const [pending, setPending] = useState<'chat' | 'tips' | null>(null)
  const [error, setError] = useState('')

  async function ask(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const submitted = question.trim()
    if (!submitted || pending) return
    setPending('chat')
    setError('')
    try {
      const result = await request<{ response: string }>('/ai/chatbot', {
        method: 'POST',
        body: { question: submitted },
      })
      setExchanges((items) => [...items, { question: submitted, answer: result.response }])
      setQuestion('')
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? `The assistant could not answer: ${cause.message}`
          : 'The assistant could not answer. Your question is still here; try again later.',
      )
    } finally {
      setPending(null)
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
    <main className="mx-auto w-full max-w-3xl space-y-5 p-4 sm:p-6">
      <header>
        <h1 className="text-2xl font-semibold text-ink">Safety assistant</h1>
        <p className="mt-2 text-sm text-ink-muted">General safety information and practical tips.</p>
      </header>

      <Card className="space-y-2 p-4">
        <p className="text-sm font-semibold text-ink">AI-generated advice may be wrong or out of date.</p>
        <p className="text-sm text-ink-muted">
          Do not share personal details, case evidence or exact locations here. For immediate danger,
          use <Link className="text-signal underline" to="/sos">Emergency SOS</Link>. Reports and
          official warnings are handled by people.
        </p>
      </Card>

      <Card className="space-y-4 p-4">
        <h2 className="font-semibold text-ink">Ask a safety question</h2>
        <div role="log" aria-label="Assistant conversation" className="space-y-3">
          {exchanges.map((item, index) => (
            <div key={index} className="space-y-2 border-b border-border pb-3 text-sm">
              <p className="whitespace-pre-wrap text-ink">
                <strong>You:</strong> {item.question}
              </p>
              <p className="whitespace-pre-wrap text-ink-muted">
                <strong>AI suggestion:</strong> {item.answer}
              </p>
            </div>
          ))}
        </div>
        <form onSubmit={(event) => void ask(event)} className="space-y-3">
          <label htmlFor="ai-question" className="block text-sm text-ink">
            Your question
          </label>
          <textarea
            id="ai-question"
            value={question}
            onChange={(event) => setQuestion(event.target.value)}
            maxLength={1000}
            rows={3}
            className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-ink"
            placeholder="How can I prepare an emergency contact plan?"
          />
          <Button type="submit" disabled={!question.trim() || pending !== null} loading={pending === 'chat'}>
            Ask assistant
          </Button>
        </form>
      </Card>

      <Card className="space-y-3 p-4">
        <h2 className="font-semibold text-ink">Safety tips</h2>
        <label htmlFor="tips-area" className="block text-sm text-ink-muted">
          Optional general area (avoid a home address)
        </label>
        <input
          id="tips-area"
          value={location}
          onChange={(event) => setLocation(event.target.value)}
          maxLength={100}
          className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-ink"
          placeholder="Your city or district"
        />
        <Button type="button" disabled={pending !== null} loading={pending === 'tips'} onClick={() => void getTips()}>
          Get tips
        </Button>
        {tips && (
          <p className="whitespace-pre-wrap text-sm text-ink-muted">
            <strong>AI suggestion:</strong> {tips}
          </p>
        )}
      </Card>

      {error && (
        <p role="alert" className="text-sm text-warn">
          {error}
        </p>
      )}
    </main>
  )
}