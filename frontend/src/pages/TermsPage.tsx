import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Card } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { api } from '@/lib/apiClient'
import {
  formatLegalEffectiveDate,
  validatePublicLegalDocument,
  type PublicLegalDocument,
  type PublicLegalKind,
} from '@/lib/publicLegalDocuments'

interface DocumentState {
  document: PublicLegalDocument | null
  error: boolean
}

const INITIAL: DocumentState = { document: null, error: false }

/**
 * Public access to the *actual* backend Terms and Privacy documents.
 * Each request settles independently so a broken Terms endpoint cannot
 * hide the Privacy Notice (or vice versa). Never invent fallback policies.
 */
export function TermsPage() {
  const [terms, setTerms] = useState<DocumentState>(INITIAL)
  const [privacy, setPrivacy] = useState<DocumentState>(INITIAL)
  const [loading, setLoading] = useState(true)
  const [retryToken, setRetryToken] = useState(0)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    const load = async (kind: PublicLegalKind): Promise<DocumentState> => {
      try {
        const path = kind === 'terms' ? '/terms/citizen' : '/terms/citizen?kind=privacy'
        const response = await api.get<unknown>(path)
        return { document: validatePublicLegalDocument(response, kind), error: false }
      } catch {
        return { document: null, error: true }
      }
    }

    void Promise.all([load('terms'), load('privacy')]).then(([termsResult, privacyResult]) => {
      if (cancelled) return
      setTerms(termsResult)
      setPrivacy(privacyResult)
      setLoading(false)
    })

    return () => { cancelled = true }
  }, [retryToken])

  useEffect(() => {
    if (loading) return
    const fragment = window.location.hash.slice(1)
    if (fragment === 'terms' || fragment === 'privacy') {
      document.getElementById(fragment)?.scrollIntoView({ block: 'start' })
    }
  }, [loading])

  const showDocument = (kind: PublicLegalKind, state: DocumentState) => {
    const isTerms = kind === 'terms'
    return (
      <section id={kind} aria-labelledby={kind + '-heading'} className="scroll-mt-6">
        <Card className="space-y-3 p-5">
          <h2 id={kind + '-heading'} className="text-lg font-semibold text-ink">
            {isTerms ? 'Terms of Service' : 'Privacy Notice'}
          </h2>
          {loading ? (
            <p role="status" className="text-sm text-ink-muted">Loading published document…</p>
          ) : state.document ? (
            <>
              <p className="text-xs text-ink-faint">
                {state.document.title} · Version {state.document.version}
                {' '}· Effective {formatLegalEffectiveDate(state.document.effectiveAt)}
              </p>
              <div className="whitespace-pre-wrap break-words text-sm leading-relaxed text-ink-muted">
                {state.document.content}
              </div>
            </>
          ) : (
            <div role="alert" className="space-y-2">
              <p className="text-sm text-warn">
                {state.error
                  ? `The ${isTerms ? 'Terms of Service' : 'Privacy Notice'} could not be loaded or validated. We cannot display an unverified substitute.`
                  : 'This document is not available.'}
              </p>
              <p className="text-xs text-ink-muted">
                Do not rely on an outdated screenshot or an unofficial copy.
              </p>
              <Button variant="secondary" size="sm" onClick={() => setRetryToken((value) => value + 1)}>
                Retry loading documents
              </Button>
            </div>
          )}
        </Card>
      </section>
    )
  }

  return (
    <main className="mx-auto w-full max-w-3xl space-y-5 p-4 sm:p-6">
      <header>
        <Link to="/" className="text-sm text-signal hover:underline">
          ← Nativity Guard
        </Link>
        <h1 className="mt-3 text-2xl font-semibold text-ink">Terms &amp; Privacy</h1>
        <p className="mt-2 text-sm text-ink-muted">
          Read the currently published documents before creating an account or sharing
          sensitive information. This page does not certify legal compliance.
        </p>
      </header>

      <nav aria-label="Legal document sections" className="flex flex-wrap gap-4 text-sm">
        <a href="#terms" className="text-signal underline-offset-2 hover:underline">Terms of Service</a>
        <a href="#privacy" className="text-signal underline-offset-2 hover:underline">Privacy Notice</a>
      </nav>

      {showDocument('terms', terms)}
      {showDocument('privacy', privacy)}

      <Card className="space-y-3 p-5">
        <h2 className="text-base font-semibold text-ink">Privacy questions and your rights</h2>
        <p className="text-sm leading-relaxed text-ink-muted">
          Nigeria's data protection framework provides rights including information, access,
          correction, and certain rights to deletion, restriction or objection depending on
          the circumstances. These are legal rights, not a certification of this platform.
        </p>
        <p className="text-sm text-ink-muted">
          For questions or privacy requests, the platform currently lists{' '}
          <a href="mailto:support@nativityguard.com" className="text-signal hover:underline">
            support@nativityguard.com
          </a>. Its mailbox delivery and the named data controller have not been independently
          verified in this review.
        </p>
        <a
          href="https://ndpc.gov.ng/"
          target="_blank"
          rel="noopener noreferrer"
          className="inline-block text-sm text-signal hover:underline"
        >
          Learn about your rights from the Nigeria Data Protection Commission ↗
        </a>
      </Card>
    </main>
  )
}
