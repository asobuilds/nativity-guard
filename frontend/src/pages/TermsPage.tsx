import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Card } from '@/components/ui/Card'
import { api } from '@/lib/apiClient'

interface TermsDocument {
  id: string
  kind: string
  role: string
  version: string
  title: string
  content: string
  summary?: string
  effectiveAt: string
  isActive: boolean
}

interface TermsResponse {
  document: TermsDocument
  version: string
}

/**
 * `/terms` — the public page a prospective user reads before agreeing.
 *
 * Fetched live from the backend so the version rendered here and the
 * version recorded when the user ticks a checkbox can never drift. The
 * page is public; no auth is required to read it.
 */
export function TermsPage() {
  const [terms, setTerms] = useState<TermsDocument | null>(null)
  const [privacy, setPrivacy] = useState<TermsDocument | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    Promise.all([
      api.get<TermsResponse>('/terms/citizen'),
      api.get<TermsResponse>('/terms/citizen?kind=privacy'),
    ])
      .then(([t, p]) => {
        if (cancelled) return
        setTerms(t.document)
        setPrivacy(p.document)
        setLoading(false)
      })
      .catch(() => {
        if (cancelled) return
        setError('Could not load the terms. Check your connection and reload the page.')
        setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  if (loading) {
    return (
      <main className="mx-auto w-full max-w-3xl space-y-5 p-4 sm:p-6">
        <header>
          <Link to="/" className="text-sm text-signal hover:underline">
            ← Nativity Guard
          </Link>
          <h1 className="mt-3 text-2xl font-semibold text-ink">Terms & Privacy</h1>
        </header>
        <Card className="p-5">
          <p className="text-sm text-ink-muted">Loading…</p>
        </Card>
      </main>
    )
  }

  if (error || !terms || !privacy) {
    return (
      <main className="mx-auto w-full max-w-3xl space-y-5 p-4 sm:p-6">
        <header>
          <Link to="/" className="text-sm text-signal hover:underline">
            ← Nativity Guard
          </Link>
          <h1 className="mt-3 text-2xl font-semibold text-ink">Terms & Privacy</h1>
        </header>
        <Card className="p-5">
          <p className="text-sm text-warn">
            {error ?? 'Terms are unavailable right now.'}
          </p>
        </Card>
      </main>
    )
  }

  return (
    <main className="mx-auto w-full max-w-3xl space-y-5 p-4 sm:p-6">
      <header>
        <Link to="/" className="text-sm text-signal hover:underline">
          ← Nativity Guard
        </Link>
        <h1 className="mt-3 text-2xl font-semibold text-ink">Terms & Privacy</h1>
        <p className="mt-2 text-sm text-ink-muted">
          These are the documents you agree to when you create an account. Both are required.
        </p>
      </header>

      <Card className="space-y-2 p-5">
        <h2 className="text-lg font-semibold text-ink">{terms.title}</h2>
        <p className="text-xs text-ink-faint">
          Version {terms.version} · Effective{' '}
          {new Date(terms.effectiveAt).toLocaleDateString()}
        </p>
        <div className="whitespace-pre-wrap text-sm leading-relaxed text-ink-muted">
          {terms.content}
        </div>
      </Card>

      <Card className="space-y-2 p-5">
        <h2 className="text-lg font-semibold text-ink">{privacy.title}</h2>
        <p className="text-xs text-ink-faint">
          Version {privacy.version} · Effective{' '}
          {new Date(privacy.effectiveAt).toLocaleDateString()}
        </p>
        <div className="whitespace-pre-wrap text-sm leading-relaxed text-ink-muted">
          {privacy.content}
        </div>
      </Card>
    </main>
  )
}