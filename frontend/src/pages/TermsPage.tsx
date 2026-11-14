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

type DocumentStatus = 'loading' | 'ready' | 'error'

interface DocumentState {
  document: PublicLegalDocument | null
  status: DocumentStatus
  slow: boolean
}

const INITIAL: DocumentState = { document: null, status: 'loading', slow: false }
const SLOW_NOTICE_MS = 8_000
const DOCUMENT_TIMEOUT_MS = 30_000

/**
 * The public page never substitutes fictional policies.
 * Both documents load separately; a slow or failed request cannot hide the
 * successful one or leave visitors facing an indefinite unlabelled spinner.
 */
export function TermsPage() {
  const [terms, setTerms] = useState<DocumentState>(INITIAL)
  const [privacy, setPrivacy] = useState<DocumentState>(INITIAL)
  const [retryToken, setRetryToken] = useState(0)

  useEffect(() => {
    let cancelled = false
    const controllers = [new AbortController(), new AbortController()]
    const timers: Array<ReturnType<typeof setTimeout>> = []
    setTerms(INITIAL)
    setPrivacy(INITIAL)

    const load = (kind: PublicLegalKind, controller: AbortController) => {
      const update = kind === 'terms' ? setTerms : setPrivacy
      const path = kind === 'terms' ? '/terms/citizen' : '/terms/citizen?kind=privacy'

      const slowTimer = setTimeout(() => {
        if (!cancelled) {
          update(current => current.status === 'loading'
            ? { ...current, slow: true }
            : current)
        }
      }, SLOW_NOTICE_MS)
      const timeoutTimer = setTimeout(() => controller.abort(), DOCUMENT_TIMEOUT_MS)
      timers.push(slowTimer, timeoutTimer)

      void api.get<unknown>(path, controller.signal)
        .then(response => {
          const document = validatePublicLegalDocument(response, kind)
          if (!cancelled) update({ document, status: 'ready', slow: false })
        })
        .catch(() => {
          if (!cancelled) update({ document: null, status: 'error', slow: false })
        })
        .finally(() => {
          clearTimeout(slowTimer)
          clearTimeout(timeoutTimer)
        })
    }

    load('terms', controllers[0])
    load('privacy', controllers[1])

    return () => {
      cancelled = true
      controllers.forEach(controller => controller.abort())
      timers.forEach(timer => clearTimeout(timer))
    }
  }, [retryToken])

  useEffect(() => {
    const fragment = window.location.hash.slice(1)
    if (fragment === 'terms' || fragment === 'privacy') {
      document.getElementById(fragment)?.scrollIntoView({ block: 'start' })
    }
  }, [terms.status, privacy.status])

  const retry = () => setRetryToken(value => value + 1)

  const showDocument = (kind: PublicLegalKind, state: DocumentState) => {
    const label = kind === 'terms' ? 'Terms of Service' : 'Privacy Notice'

    return (
      <section id={kind} aria-labelledby={kind + '-heading'} className="scroll-mt-6">
        <Card className="space-y-3 p-5">
          <h2 id={kind + '-heading'} className="text-lg font-semibold text-ink">{label}</h2>
          {state.status === 'ready' && state.document ? (
            <>
              <p className="text-xs text-ink-faint">
                {state.document.title} · Version {state.document.version}
                {' '}· Effective {formatLegalEffectiveDate(state.document.effectiveAt)}
              </p>
              <div className="whitespace-pre-wrap break-words text-sm leading-relaxed text-ink-muted">
                {state.document.content}
              </div>
            </>
          ) : state.status === 'loading' ? (
            <div className="space-y-2">
              <p role="status" className="text-sm text-ink-muted">
                {state.slow
                  ? 'Still connecting to the document service. This may take longer than expected.'
                  : 'Loading published document…'}
              </p>
              {state.slow ? (
                <Button variant="secondary" size="sm" onClick={retry}>Try connecting again</Button>
              ) : null}
            </div>
          ) : (
            <div role="alert" className="space-y-2">
              <p className="text-sm text-warn">
                The {label} could not be loaded or validated. The service may be temporarily unavailable.
                No unverified substitute is displayed.
              </p>
              <p className="text-xs text-ink-muted">
                Do not rely on an outdated screenshot or unofficial copy.
              </p>
              <Button variant="secondary" size="sm" onClick={retry}>Retry loading documents</Button>
            </div>
          )}
        </Card>
      </section>
    )
  }

  return (
    <main className="mx-auto w-full max-w-3xl space-y-5 p-4 sm:p-6">
      <header>
        <Link to="/" className="text-sm text-signal hover:underline">← Nativity Guard</Link>
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
          the circumstances. These rights are not a certification of this platform.
        </p>
        <p className="text-sm text-ink-muted">
          For questions or privacy requests, the platform currently lists{' '}
          <a href="mailto:support@nativityguard.com" className="text-signal hover:underline">
            support@nativityguard.com
          </a>. Its mailbox delivery and the named data controller have not been independently
          verified in this review.
        </p>
        <a href="https://ndpc.gov.ng/" target="_blank" rel="noopener noreferrer"
          className="inline-block text-sm text-signal hover:underline">
          Learn about your rights from the Nigeria Data Protection Commission ↗
        </a>
      </Card>
    </main>
  )
}
