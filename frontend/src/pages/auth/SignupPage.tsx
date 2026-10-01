import { useEffect, useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { AlertCircle, Shield, X } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { api, ApiError } from '@/lib/apiClient'
import {
  EMPTY_SIGNUP,
  MIN_PASSWORD_LENGTH,
  MIN_SIGNUP_AGE,
  SIGNUP_FIELDS,
  latestEligibleDob,
  validateSignup,
  type SignupDraft,
  type SignupField,
} from '@/lib/signup'
import type { RegisterResponse } from '@/types/api'
import { useAuth } from '@/auth/AuthContext'

const DRAFT_KEY = 'cs.signup.draft'

interface PersistedDraft {
  firstName: string
  lastName: string
  email: string
  phone: string
  dateOfBirth: string
  consentTerms: boolean
  consentDataProcessing: boolean
}

function loadPersisted(): PersistedDraft | null {
  try {
    const raw = sessionStorage.getItem(DRAFT_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<PersistedDraft>
    if (!parsed || typeof parsed !== 'object') return null
    return {
      firstName: typeof parsed.firstName === 'string' ? parsed.firstName : '',
      lastName: typeof parsed.lastName === 'string' ? parsed.lastName : '',
      email: typeof parsed.email === 'string' ? parsed.email : '',
      phone: typeof parsed.phone === 'string' ? parsed.phone : '',
      dateOfBirth: typeof parsed.dateOfBirth === 'string' ? parsed.dateOfBirth : '',
      consentTerms: parsed.consentTerms === true,
      consentDataProcessing: parsed.consentDataProcessing === true,
    }
  } catch {
    return null
  }
}

function savePersisted(state: PersistedDraft): void {
  try {
    sessionStorage.setItem(DRAFT_KEY, JSON.stringify(state))
  } catch {
    /* storage unavailable (private mode) - draft stays in memory only */
  }
}

function clearPersisted(): void {
  try {
    sessionStorage.removeItem(DRAFT_KEY)
  } catch {
    /* ignore */
  }
}

interface TermsDocument {
  id: string
  kind: string
  role: string
  version: string
  title: string
  content: string
  effectiveAt: string
}

interface TermsResponse {
  document: TermsDocument
  version: string
}

type ModalKind = 'terms' | 'privacy'

/**
 * Create an account.
 *
 * Terms and Privacy open in modals over this page - never navigate away.
 * That guarantees the form is preserved exactly where the user left it,
 * regardless of browser back button, tab behavior, or mobile quirks.
 *
 * The form draft is also persisted to `sessionStorage` (except password)
 * so a manual close-and-reopen of the tab doesn't lose progress either.
 * The draft is cleared on successful signup.
 */
export function SignupPage() {
  const navigate = useNavigate()
  const { login } = useAuth()

  const [draft, setDraft] = useState<SignupDraft>(() => {
    const p = loadPersisted()
    return p
      ? {
          ...EMPTY_SIGNUP,
          firstName: p.firstName,
          lastName: p.lastName,
          email: p.email,
          phone: p.phone,
          dateOfBirth: p.dateOfBirth,
        }
      : EMPTY_SIGNUP
  })
  const [consentTerms, setConsentTerms] = useState<boolean>(
    () => loadPersisted()?.consentTerms ?? false,
  )
  const [consentDataProcessing, setConsentDataProcessing] = useState<boolean>(
    () => loadPersisted()?.consentDataProcessing ?? false,
  )

  const [touched, setTouched] = useState<ReadonlySet<SignupField>>(new Set())
  const [submitting, setSubmitting] = useState(false)
  const [refusal, setRefusal] = useState<string | null>(null)

  const [termsVersion, setTermsVersion] = useState<string | null>(null)
  const [termsDoc, setTermsDoc] = useState<TermsDocument | null>(null)
  const [privacyDoc, setPrivacyDoc] = useState<TermsDocument | null>(null)
  const [termsError, setTermsError] = useState<string | null>(null)
  const [consentTouched, setConsentTouched] = useState(false)

  const [openModal, setOpenModal] = useState<ModalKind | null>(null)

  // Fetch BOTH documents once on mount so opening the modal is instant.
  useEffect(() => {
    let cancelled = false
    Promise.all([
      api.get<TermsResponse>('/terms/citizen'),
      api.get<TermsResponse>('/terms/citizen?kind=privacy'),
    ])
      .then(([t, p]) => {
        if (cancelled) return
        setTermsDoc(t.document)
        setPrivacyDoc(p.document)
        setTermsVersion(t.version)
      })
      .catch(() => {
        if (cancelled) return
        setTermsError('Could not load the terms. Check your connection and reload the page.')
      })
    return () => {
      cancelled = true
    }
  }, [])

  // Persist the draft on every change.
  useEffect(() => {
    savePersisted({
      firstName: draft.firstName,
      lastName: draft.lastName,
      email: draft.email,
      phone: draft.phone,
      dateOfBirth: draft.dateOfBirth,
      consentTerms,
      consentDataProcessing,
    })
  }, [draft, consentTerms, consentDataProcessing])

  // Esc closes the modal.
  useEffect(() => {
    if (!openModal) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpenModal(null)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [openModal])

  const errors = validateSignup(draft)
  const formComplete = SIGNUP_FIELDS.every((field) => !errors[field])
  const consentsGiven = consentTerms && consentDataProcessing
  const canSubmit = formComplete && consentsGiven && !!termsVersion && !termsError

  const errorFor = (field: SignupField) => (touched.has(field) ? errors[field] : undefined)

  function update(field: SignupField, value: string) {
    setDraft((current) => ({ ...current, [field]: value }))
    setRefusal(null)
  }

  function markTouched(field: SignupField) {
    setTouched((current) => new Set(current).add(field))
  }

  function friendlyError(message: string): string {
    const map: Record<string, string> = {
      terms_consent_required: 'You must accept the Terms of Service to create an account.',
      data_processing_consent_required:
        'You must consent to data processing to create an account.',
      terms_version_required: 'Terms version is missing. Please reload the page and try again.',
      terms_version_stale:
        'The terms have been updated since this page loaded. Please reload the page and try again.',
      terms_unavailable: 'Terms are temporarily unavailable. Please try again in a moment.',
    }
    return map[message] ?? message
  }

  function agreeFromModal(kind: ModalKind) {
    if (kind === 'terms') setConsentTerms(true)
    if (kind === 'privacy') setConsentDataProcessing(true)
    setOpenModal(null)
    setRefusal(null)
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setTouched(new Set(SIGNUP_FIELDS))
    setConsentTouched(true)
    setRefusal(null)
    if (!canSubmit || !termsVersion) return

    setSubmitting(true)
    let registered = false
    try {
      await api.postAnonymous<RegisterResponse>('/auth/register', {
        email: draft.email.trim(),
        phone: draft.phone.trim(),
        firstName: draft.firstName.trim(),
        lastName: draft.lastName.trim(),
        dateOfBirth: draft.dateOfBirth,
        password: draft.password,
        termsVersion,
        consentTerms,
        consentDataProcessing,
      })
      registered = true
      clearPersisted()
      await login(draft.email.trim(), draft.password)
      // Send new users to complete their profile.
      navigate('/onboarding', { replace: true })
    } catch (cause) {
      if (registered) {
        navigate('/auth/login', {
          replace: true,
          state: { email: draft.email.trim(), registered: true },
        })
        return
      }
      const raw =
        cause instanceof ApiError
          ? cause.message
          : 'Could not create your account. Check your connection and try again.'
      setRefusal(friendlyError(raw))
    } finally {
      setSubmitting(false)
    }
  }

  const activeModalDoc = openModal === 'terms' ? termsDoc : openModal === 'privacy' ? privacyDoc : null
  const activeModalTitle =
    openModal === 'terms' ? 'Terms of Service' : openModal === 'privacy' ? 'Privacy Notice' : ''

  return (
    <div className="grid min-h-screen bg-base lg:grid-cols-2">
      {/* Brand panel */}
      <div className="hidden flex-col justify-between bg-surface p-10 lg:flex">
        <div className="flex items-center gap-2">
          <Shield className="size-7 text-signal" aria-hidden />
          <span className="text-base font-bold tracking-wide text-ink">NATIVITY GUARD</span>
        </div>
        <div>
          <h1 className="max-w-md text-3xl font-semibold leading-tight text-ink">
            Every report answered. Every case accounted for.
          </h1>
          <p className="mt-3 max-w-md text-sm text-ink-muted">
            Create an account to report an incident, follow what happens to it, and see the cases
            your community has raised.
          </p>
        </div>
        <p className="text-xs text-ink-faint">
          Your reports are visible to the unit handling them. Personal details are not shown to
          other citizens.
        </p>
      </div>

      {/* Form panel */}
      <div className="flex items-center justify-center p-6">
        <div className="w-full max-w-sm py-6">
          <div className="mb-6 flex items-center gap-2 lg:hidden">
            <Shield className="size-6 text-signal" aria-hidden />
            <span className="text-sm font-bold tracking-wide text-ink">NATIVITY GUARD</span>
          </div>

          <h2 className="text-xl font-semibold text-ink">Create your account</h2>
          <p className="mt-1 text-sm text-ink-muted">
            This creates a citizen account - you can report incidents and follow them. Joining a
            unit as an officer or administrator is arranged separately, by that unit.
          </p>

          <form className="mt-6 flex flex-col gap-4" onSubmit={onSubmit} noValidate>
            <Field label="Email" required error={errorFor('email')}>
              {(props) => (
                <Input
                  {...props}
                  type="email"
                  autoComplete="email"
                  autoFocus
                  required
                  value={draft.email}
                  onChange={(event) => update('email', event.target.value)}
                  onBlur={() => markTouched('email')}
                />
              )}
            </Field>

            <Field label="Phone number" required error={errorFor('phone')}>
              {(props) => (
                <Input
                  {...props}
                  type="tel"
                  autoComplete="tel"
                  required
                  value={draft.phone}
                  onChange={(event) => update('phone', event.target.value)}
                  onBlur={() => markTouched('phone')}
                />
              )}
            </Field>

            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="First name" required error={errorFor('firstName')}>
                {(props) => (
                  <Input
                    {...props}
                    autoComplete="given-name"
                    required
                    value={draft.firstName}
                    onChange={(event) => update('firstName', event.target.value)}
                    onBlur={() => markTouched('firstName')}
                  />
                )}
              </Field>

              <Field label="Last name" required error={errorFor('lastName')}>
                {(props) => (
                  <Input
                    {...props}
                    autoComplete="family-name"
                    required
                    value={draft.lastName}
                    onChange={(event) => update('lastName', event.target.value)}
                    onBlur={() => markTouched('lastName')}
                  />
                )}
              </Field>
            </div>

            <Field
              label="Date of birth"
              required
              hint={`You must be ${MIN_SIGNUP_AGE} or older to register.`}
              error={errorFor('dateOfBirth')}
            >
              {(props) => (
                <Input
                  {...props}
                  type="date"
                  autoComplete="bday"
                  required
                  max={latestEligibleDob()}
                  value={draft.dateOfBirth}
                  onChange={(event) => update('dateOfBirth', event.target.value)}
                  onBlur={() => markTouched('dateOfBirth')}
                />
              )}
            </Field>

            <Field
              label="Password"
              required
              hint={`At least ${MIN_PASSWORD_LENGTH} characters.`}
              error={errorFor('password')}
            >
              {(props) => (
                <Input
                  {...props}
                  type="password"
                  autoComplete="new-password"
                  required
                  value={draft.password}
                  onChange={(event) => update('password', event.target.value)}
                  onBlur={() => markTouched('password')}
                />
              )}
            </Field>

            {/* Consent block. Links open modals, never navigate away. */}
            <fieldset className="mt-2 flex flex-col gap-3 rounded-lg border border-border p-3">
              <legend className="px-1 text-xs font-medium text-ink-muted">
                Before you continue
              </legend>

              <label className="flex items-start gap-3 text-sm text-ink-muted">
                <input
                  type="checkbox"
                  checked={consentTerms}
                  onChange={(event) => {
                    setConsentTerms(event.target.checked)
                    setRefusal(null)
                  }}
                  className="mt-0.5 size-4 shrink-0 accent-signal"
                />
                <span>
                  I have read and agree to the{' '}
                  <button
                    type="button"
                    onClick={() => setOpenModal('terms')}
                    className="cursor-pointer text-signal hover:underline"
                  >
                    Terms of Service
                  </button>
                  .
                </span>
              </label>

              <label className="flex items-start gap-3 text-sm text-ink-muted">
                <input
                  type="checkbox"
                  checked={consentDataProcessing}
                  onChange={(event) => {
                    setConsentDataProcessing(event.target.checked)
                    setRefusal(null)
                  }}
                  className="mt-0.5 size-4 shrink-0 accent-signal"
                />
                <span>
                  I consent to my personal data being processed as described in the{' '}
                  <button
                    type="button"
                    onClick={() => setOpenModal('privacy')}
                    className="cursor-pointer text-signal hover:underline"
                  >
                    Privacy Notice
                  </button>
                  .
                </span>
              </label>

              {consentTouched && !consentsGiven ? (
                <p className="text-xs text-warn">
                  Both boxes must be ticked before your account can be created.
                </p>
              ) : null}

              {termsError ? <p className="text-xs text-warn">{termsError}</p> : null}

              {!termsError && !termsVersion ? (
                <p className="text-xs text-ink-faint">Loading the current terms...</p>
              ) : null}
            </fieldset>

            {refusal ? (
              <p
                role="alert"
                className="flex items-start gap-2 rounded-lg border border-warn/30 bg-warn/10 px-3 py-2 text-xs text-warn"
              >
                <AlertCircle className="mt-px size-4 shrink-0" aria-hidden />
                <span>{refusal}</span>
              </p>
            ) : null}

            <Button
              type="submit"
              variant="primary"
              size="lg"
              block
              loading={submitting}
              disabled={!canSubmit || submitting}
            >
              Create account
            </Button>
          </form>

          <p className="mt-6 text-sm text-ink-muted">
            Already have an account?{' '}
            <Link to="/auth/login" className="text-signal hover:text-signal-ink">
              Sign in
            </Link>
          </p>
        </div>
      </div>

      {/* Terms / Privacy modal - renders over the form, never navigates. */}
      {openModal && activeModalDoc ? (
        <div
          role="dialog"
          aria-modal="true"
          aria-label={activeModalTitle}
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
          onClick={(event) => {
            if (event.target === event.currentTarget) setOpenModal(null)
          }}
        >
          <div className="flex max-h-[85vh] w-full max-w-2xl flex-col rounded-lg border border-border bg-base shadow-2xl">
            <header className="flex items-center justify-between border-b border-border p-4">
              <div>
                <h3 className="text-base font-semibold text-ink">{activeModalTitle}</h3>
                <p className="text-xs text-ink-faint">
                  Version {activeModalDoc.version}
                </p>
              </div>
              <button
                type="button"
                aria-label="Close"
                onClick={() => setOpenModal(null)}
                className="grid size-8 place-items-center rounded-md text-ink-muted hover:bg-surface-hi"
              >
                <X className="size-4" aria-hidden />
              </button>
            </header>
            <div className="overflow-y-auto p-5">
              <pre className="whitespace-pre-wrap font-sans text-sm leading-relaxed text-ink-muted">
                {activeModalDoc.content}
              </pre>
            </div>
            <footer className="flex items-center justify-end gap-2 border-t border-border p-4">
              <Button variant="ghost" onClick={() => setOpenModal(null)}>
                Close
              </Button>
              <Button variant="primary" onClick={() => agreeFromModal(openModal)}>
                I agree
              </Button>
            </footer>
          </div>
        </div>
      ) : null}
    </div>
  )
}