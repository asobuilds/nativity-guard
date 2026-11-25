import { useEffect, useId, useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import {
  AlertCircle,
  ArrowLeft,
  CalendarDays,
  Check,
  Eye,
  EyeOff,
  LockKeyhole,
  Mail,
  Phone,
  ShieldCheck,
  UserRound,
  X,
} from 'lucide-react'
import { AuthLayout } from '@/components/auth/AuthLayout'
import { GoogleAuthButton } from '@/components/auth/GoogleAuthButton'
import { ProgressiveSignupStep } from '@/components/auth/ProgressiveSignupStep'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { useAuth } from '@/auth/AuthContext'
import { api, ApiError, tokenStore } from '@/lib/apiClient'
import {
  ACCOUNT_SIGNUP_FIELDS,
  EMPTY_SIGNUP,
  MIN_PASSWORD_LENGTH,
  MIN_SIGNUP_AGE,
  SIGNUP_FIELDS,
  ageOn,
  latestEligibleDob,
  passwordStrength,
  signupStepComplete,
  validateSignup,
  type SignupDraft,
  type SignupField,
  type SignupMode,
  type SignupStep,
} from '@/lib/signup'
import type { RegisterResponse } from '@/types/api'

const DRAFT_KEY = 'cs.signup.draft'

interface PersistedDraft {
  firstName: string
  lastName: string
  email: string
  phone: string
  dateOfBirth: string
  termsVersion: string
  consentTerms: boolean
  consentDataProcessing: boolean
}

interface GoogleProfile {
  email: string
  firstName: string
  lastName: string
}

interface SignupHandoff {
  from?: string
  googleProfile?: GoogleProfile
}

interface GoogleAuthResponse {
  token?: string
  refreshToken?: string
  requiresProfileCompletion?: boolean
  profile?: GoogleProfile
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
      termsVersion: typeof parsed.termsVersion === 'string' ? parsed.termsVersion : '',
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
    /* Storage unavailable: the draft remains in memory, and credentials are never persisted. */
  }
}

function clearPersisted(): void {
  try {
    sessionStorage.removeItem(DRAFT_KEY)
  } catch {
    /* ignore */
  }
}

/** Two-stage citizen signup with password or verified Google identity. */
export function SignupPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const { login } = useAuth()
  const handoff = (location.state as SignupHandoff | null) ?? null
  const [persisted] = useState<PersistedDraft | null>(() => loadPersisted())

  const [draft, setDraft] = useState<SignupDraft>(() => ({
    ...EMPTY_SIGNUP,
    firstName: handoff?.googleProfile?.firstName ?? persisted?.firstName ?? '',
    lastName: handoff?.googleProfile?.lastName ?? persisted?.lastName ?? '',
    email: handoff?.googleProfile?.email ?? persisted?.email ?? '',
    phone: persisted?.phone ?? '',
    dateOfBirth: persisted?.dateOfBirth ?? '',
  }))
  const [mode, setMode] = useState<SignupMode>('password')
  const [step, setStep] = useState<SignupStep>(1)
  const [googleCredential, setGoogleCredential] = useState<string | null>(null)
  const [showPassword, setShowPassword] = useState(false)
  const [consentTerms, setConsentTerms] = useState(persisted?.consentTerms ?? false)
  const [consentDataProcessing, setConsentDataProcessing] = useState(
    persisted?.consentDataProcessing ?? false,
  )
  const [touched, setTouched] = useState<ReadonlySet<SignupField>>(new Set())
  const [consentTouched, setConsentTouched] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [googleSubmitting, setGoogleSubmitting] = useState(false)
  const [refusal, setRefusal] = useState<string | null>(null)

  const [termsVersion, setTermsVersion] = useState<string | null>(null)
  const [termsDoc, setTermsDoc] = useState<TermsDocument | null>(null)
  const [privacyDoc, setPrivacyDoc] = useState<TermsDocument | null>(null)
  const [termsError, setTermsError] = useState<string | null>(null)
  const [openModal, setOpenModal] = useState<ModalKind | null>(null)

  useEffect(() => {
    let cancelled = false
    Promise.all([
      api.get<TermsResponse>('/terms/citizen'),
      api.get<TermsResponse>('/terms/citizen?kind=privacy'),
    ])
      .then(([terms, privacy]) => {
        if (cancelled) return
        if (terms.version !== privacy.version) {
          setTermsError('The legal documents are out of sync. Reload the page before continuing.')
          setTermsVersion(null)
          setConsentTerms(false)
          setConsentDataProcessing(false)
          return
        }
        setTermsDoc(terms.document)
        setPrivacyDoc(privacy.document)
        setTermsVersion(terms.version)
        if (persisted?.termsVersion !== terms.version) {
          setConsentTerms(false)
          setConsentDataProcessing(false)
        }
      })
      .catch(() => {
        if (!cancelled) {
          setTermsError('Could not load the terms. Check your connection and reload the page.')
        }
      })
    return () => {
      cancelled = true
    }
  }, [persisted])

  useEffect(() => {
    savePersisted({
      firstName: draft.firstName,
      lastName: draft.lastName,
      email: draft.email,
      phone: draft.phone,
      dateOfBirth: draft.dateOfBirth,
      termsVersion: termsVersion ?? persisted?.termsVersion ?? '',
      consentTerms,
      consentDataProcessing,
    })
  }, [draft, termsVersion, persisted, consentTerms, consentDataProcessing])

  useEffect(() => {
    if (!openModal) return
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') setOpenModal(null)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [openModal])

  const errors = validateSignup(draft, new Date(), mode)
  const accountComplete = signupStepComplete(draft, 1, mode)
  const profileComplete = signupStepComplete(draft, 2, mode)
  const consentsGiven = consentTerms && consentDataProcessing
  const canSubmit = accountComplete && profileComplete && consentsGiven && !!termsVersion && !termsError
  const strength = passwordStrength(draft.password)
  const currentAge = ageOn(draft.dateOfBirth)

  const errorFor = (field: SignupField) => (touched.has(field) ? errors[field] : undefined)

  function update(field: SignupField, value: string) {
    setDraft((current) => ({ ...current, [field]: value }))
    setRefusal(null)
  }

  function markTouched(field: SignupField) {
    setTouched((current) => new Set(current).add(field))
  }

  function touchFields(fields: SignupField[]) {
    setTouched((current) => new Set([...current, ...fields]))
  }

  function friendlyError(message: string): string {
    const map: Record<string, string> = {
      terms_consent_required: 'You must accept the Terms of Service to create an account.',
      data_processing_consent_required: 'You must consent to data processing to create an account.',
      terms_version_required: 'Terms version is missing. Reload the page and try again.',
      terms_version_stale: 'The terms changed while this page was open. Reload and review them again.',
      terms_unavailable: 'Terms are temporarily unavailable. Try again in a moment.',
    }
    return map[message] ?? message
  }

  function goToProfileStep() {
    touchFields(ACCOUNT_SIGNUP_FIELDS)
    setRefusal(null)
    if (!accountComplete) return
    setStep(2)
  }

  function usePasswordSignup() {
    setMode('password')
    setGoogleCredential(null)
    setStep(1)
    setRefusal(null)
  }

  function applyGoogleProfile(profile: GoogleProfile) {
    setDraft((current) => ({
      ...current,
      email: profile.email,
      firstName: profile.firstName,
      lastName: profile.lastName,
      password: '',
    }))
  }

  function finishGoogleSession(result: GoogleAuthResponse) {
    if (!result.token || !result.refreshToken) {
      throw new Error('Google signup returned an incomplete session.')
    }
    clearPersisted()
    tokenStore.set(result.token)
    tokenStore.setRefresh(result.refreshToken)
    window.location.assign('/onboarding')
  }

  async function onGoogleCredential(idToken: string) {
    setGoogleSubmitting(true)
    setRefusal(null)
    try {
      const result = await api.postAnonymous<GoogleAuthResponse>('/auth/google', { idToken })
      if (result.token) {
        finishGoogleSession(result)
        return
      }
      if (!result.requiresProfileCompletion || !result.profile) {
        throw new Error('Google could not prepare your signup profile.')
      }
      applyGoogleProfile(result.profile)
      setGoogleCredential(idToken)
      setMode('google')
      setTouched(new Set())
      setStep(2)
    } catch (cause) {
      setRefusal(
        cause instanceof ApiError
          ? friendlyError(cause.message)
          : cause instanceof Error
            ? cause.message
            : 'Google signup could not be completed.',
      )
    } finally {
      setGoogleSubmitting(false)
    }
  }

  function agreeFromModal(kind: ModalKind) {
    if (kind === 'terms') setConsentTerms(true)
    if (kind === 'privacy') setConsentDataProcessing(true)
    setOpenModal(null)
    setRefusal(null)
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    touchFields(SIGNUP_FIELDS)
    setConsentTouched(true)
    setRefusal(null)
    if (!canSubmit || !termsVersion) return

    setSubmitting(true)
    let registered = false
    try {
      if (mode === 'google') {
        if (!googleCredential) {
          setStep(1)
          throw new Error('Continue with Google again to verify your identity.')
        }
        const result = await api.postAnonymous<GoogleAuthResponse>('/auth/google', {
          idToken: googleCredential,
          phone: draft.phone.trim(),
          dateOfBirth: draft.dateOfBirth,
          termsVersion,
          consentTerms,
          consentDataProcessing,
        })
        finishGoogleSession(result)
        return
      }

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
          : cause instanceof Error
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

  const strengthColours = ['bg-border', 'bg-emergency', 'bg-warn', 'bg-signal', 'bg-ok']

  return (
    <>
      <AuthLayout
        title="Create your citizen account"
        subtitle="Start with your identity, then add the contact and consent details needed to protect your account."
        eyebrow="Join in two short steps"
      >
        <ol className="flex gap-1" aria-label="Signup progress">
          <ProgressiveSignupStep
            number={1}
            title="Account"
            description="Identity and sign-in"
            active={step === 1}
            complete={accountComplete}
            onSelect={step === 2 ? () => setStep(1) : undefined}
          />
          <ProgressiveSignupStep
            number={2}
            title="Profile & consent"
            description="Contact and compliance"
            active={step === 2}
            complete={profileComplete && consentsGiven}
            onSelect={step === 1 && accountComplete ? () => setStep(2) : undefined}
          />
        </ol>

        {step === 1 ? (
          <div className="mt-5">
            {handoff?.googleProfile && mode !== 'google' ? (
              <p className="mb-4 rounded-xl border border-signal/25 bg-signal/10 px-3 py-2.5 text-xs text-ink-muted">
                Google shared your profile details. Continue with Google once more to verify them securely.
              </p>
            ) : null}

            {mode === 'google' ? (
              <div className="mb-4 flex items-start justify-between gap-3 rounded-xl border border-ok/30 bg-ok/10 p-3">
                <div className="flex gap-2">
                  <ShieldCheck className="mt-0.5 size-4 shrink-0 text-ok" aria-hidden />
                  <div>
                    <p className="text-xs font-semibold text-ink">Google identity verified</p>
                    <p className="mt-0.5 text-[11px] text-ink-muted">{draft.email}</p>
                  </div>
                </div>
                <button type="button" onClick={usePasswordSignup} className="text-[11px] font-semibold text-signal hover:underline">
                  Use email instead
                </button>
              </div>
            ) : (
              <>
                <GoogleAuthButton
                  onCredential={onGoogleCredential}
                  mockEmail={draft.email}
                  disabled={submitting || googleSubmitting}
                  onError={(message) => setRefusal(message || null)}
                />
                <div className="my-4 flex items-center gap-3" aria-hidden>
                  <span className="h-px flex-1 bg-border" />
                  <span className="text-[11px] font-medium uppercase tracking-[0.14em] text-ink-faint">or use email</span>
                  <span className="h-px flex-1 bg-border" />
                </div>
              </>
            )}

            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="First name" required error={errorFor('firstName')}>
                {(props) => (
                  <div className="relative">
                    <UserRound className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint" aria-hidden />
                    <Input {...props} className="h-12 pl-10" autoComplete="given-name" required disabled={mode === 'google'} value={draft.firstName} onChange={(event) => update('firstName', event.target.value)} onBlur={() => markTouched('firstName')} />
                  </div>
                )}
              </Field>
              <Field label="Last name" required error={errorFor('lastName')}>
                {(props) => (
                  <Input {...props} className="h-12" autoComplete="family-name" required disabled={mode === 'google'} value={draft.lastName} onChange={(event) => update('lastName', event.target.value)} onBlur={() => markTouched('lastName')} />
                )}
              </Field>
            </div>

            <Field className="mt-4" label="Email address" required error={errorFor('email')}>
              {(props) => (
                <div className="relative">
                  <Mail className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint" aria-hidden />
                  <Input {...props} className="h-12 pl-10" type="email" autoComplete="email" required disabled={mode === 'google'} value={draft.email} onChange={(event) => update('email', event.target.value)} onBlur={() => markTouched('email')} />
                </div>
              )}
            </Field>

            {mode === 'password' ? (
              <Field className="mt-4" label="Password" required hint={`At least ${MIN_PASSWORD_LENGTH} characters.`} error={errorFor('password')}>
                {(props) => (
                  <>
                    <div className="relative">
                      <LockKeyhole className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint" aria-hidden />
                      <Input {...props} className="h-12 px-10" type={showPassword ? 'text' : 'password'} autoComplete="new-password" required value={draft.password} onChange={(event) => update('password', event.target.value)} onBlur={() => markTouched('password')} />
                      <button type="button" onClick={() => setShowPassword((visible) => !visible)} aria-label={showPassword ? 'Hide password' : 'Show password'} aria-pressed={showPassword} className="absolute right-1.5 top-1/2 grid size-9 -translate-y-1/2 place-items-center rounded-lg text-ink-faint hover:bg-surface focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-signal">
                        {showPassword ? <EyeOff className="size-4" aria-hidden /> : <Eye className="size-4" aria-hidden />}
                      </button>
                    </div>
                    <div className="mt-2" aria-live="polite">
                      <div className="flex gap-1" aria-hidden>
                        {[1, 2, 3, 4].map((segment) => (
                          <span key={segment} className={`h-1.5 flex-1 rounded-full ${segment <= strength.score ? strengthColours[strength.score] : 'bg-border'}`} />
                        ))}
                      </div>
                      <div className="mt-1.5 flex items-center justify-between gap-2 text-[11px]">
                        <span className="font-semibold text-ink-muted">Password strength: {strength.label}</span>
                        <span className="text-ink-faint">10+ characters recommended</span>
                      </div>
                    </div>
                  </>
                )}
              </Field>
            ) : null}

            {refusal ? <SignupRefusal message={refusal} /> : null}

            <Button className="mt-5" type="button" variant="primary" size="lg" block onClick={goToProfileStep} disabled={googleSubmitting}>
              Continue to profile
            </Button>
          </div>
        ) : (
          <form className="mt-5 flex flex-col gap-4" onSubmit={onSubmit} noValidate>
            <div className="rounded-xl border border-border bg-surface-hi/60 px-3 py-2.5 text-xs text-ink-muted">
              <span className="font-semibold text-ink">{draft.firstName} {draft.lastName}</span>
              <span className="mx-1.5 text-ink-faint">•</span>
              {draft.email}
            </div>

            <Field label="Nigerian phone number" required hint="Use 080… or +234… format." error={errorFor('phone')}>
              {(props) => (
                <div className="relative">
                  <Phone className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint" aria-hidden />
                  <Input {...props} className="h-12 pl-10" type="tel" inputMode="tel" autoComplete="tel" required placeholder="08031234567" value={draft.phone} onChange={(event) => update('phone', event.target.value)} onBlur={() => markTouched('phone')} />
                </div>
              )}
            </Field>

            <Field label="Date of birth" required hint={`You must be ${MIN_SIGNUP_AGE} or older.`} error={errorFor('dateOfBirth')}>
              {(props) => (
                <>
                  <div className="relative">
                    <CalendarDays className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint" aria-hidden />
                    <Input {...props} className="h-12 pl-10" type="date" autoComplete="bday" required max={latestEligibleDob()} value={draft.dateOfBirth} onChange={(event) => update('dateOfBirth', event.target.value)} onBlur={() => markTouched('dateOfBirth')} />
                  </div>
                  {currentAge !== null && currentAge >= MIN_SIGNUP_AGE && currentAge <= 120 ? (
                    <p className="mt-1.5 flex items-center gap-1.5 text-[11px] text-ok" role="status">
                      <Check className="size-3.5" aria-hidden /> Age check passed ({currentAge} years old).
                    </p>
                  ) : null}
                </>
              )}
            </Field>

            <fieldset className="rounded-xl border border-border p-3.5">
              <legend className="px-1 text-xs font-semibold text-ink-muted">Review and consent</legend>
              <ConsentRow checked={consentTerms} onChange={(checked) => { setConsentTerms(checked); setRefusal(null) }} onOpen={() => setOpenModal('terms')} label="Terms of Service" prefix="I have read and agree to the" />
              <ConsentRow checked={consentDataProcessing} onChange={(checked) => { setConsentDataProcessing(checked); setRefusal(null) }} onOpen={() => setOpenModal('privacy')} label="Privacy Notice" prefix="I consent to data processing described in the" className="mt-3" />
              {consentTouched && !consentsGiven ? <p className="mt-3 text-xs text-warn">Both boxes must be selected before your account can be created.</p> : null}
              {termsError ? <p className="mt-3 text-xs text-warn">{termsError}</p> : null}
              {!termsError && !termsVersion ? <p className="mt-3 text-xs text-ink-faint">Loading the current terms…</p> : null}
            </fieldset>

            {refusal ? <SignupRefusal message={refusal} /> : null}

            <div className="flex gap-3">
              <Button type="button" variant="secondary" size="lg" onClick={() => setStep(1)} icon={<ArrowLeft className="size-4" aria-hidden />}>
                Back
              </Button>
              <Button type="submit" variant="primary" size="lg" block loading={submitting} disabled={submitting || !!termsError || !termsVersion}>
                {mode === 'google' ? 'Create account with Google' : 'Create citizen account'}
              </Button>
            </div>
          </form>
        )}

        <p className="mt-6 text-center text-sm text-ink-muted">
          Already have an account?{' '}
          <Link to="/auth/login" className="font-semibold text-signal hover:text-signal-ink hover:underline">Sign in</Link>
        </p>
      </AuthLayout>

      {openModal && activeModalDoc ? (
        <div role="dialog" aria-modal="true" aria-label={activeModalTitle} className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={(event) => { if (event.target === event.currentTarget) setOpenModal(null) }}>
          <div className="flex max-h-[85vh] w-full max-w-2xl flex-col rounded-2xl border border-border bg-base shadow-2xl">
            <header className="flex items-center justify-between border-b border-border p-4">
              <div>
                <h3 className="text-base font-semibold text-ink">{activeModalTitle}</h3>
                <p className="text-xs text-ink-faint">Version {activeModalDoc.version}</p>
              </div>
              <button type="button" aria-label={`Close ${activeModalTitle}`} onClick={() => setOpenModal(null)} className="grid size-9 place-items-center rounded-lg text-ink-muted hover:bg-surface-hi focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-signal">
                <X className="size-4" aria-hidden />
              </button>
            </header>
            <div className="overflow-y-auto p-5">
              <pre className="whitespace-pre-wrap font-sans text-sm leading-relaxed text-ink-muted">{activeModalDoc.content}</pre>
            </div>
            <footer className="flex items-center justify-end gap-2 border-t border-border p-4">
              <Button variant="ghost" onClick={() => setOpenModal(null)}>Close</Button>
              <Button variant="primary" onClick={() => agreeFromModal(openModal)}>I agree</Button>
            </footer>
          </div>
        </div>
      ) : null}
    </>
  )
}

function SignupRefusal({ message }: { message: string }) {
  return (
    <p role="alert" className="mt-4 flex items-start gap-2 rounded-xl border border-warn/30 bg-warn/10 px-3 py-2.5 text-xs text-warn">
      <AlertCircle className="mt-px size-4 shrink-0" aria-hidden />
      <span>{message}</span>
    </p>
  )
}

function ConsentRow({ checked, onChange, onOpen, label, prefix, className = '' }: { checked: boolean; onChange: (checked: boolean) => void; onOpen: () => void; label: string; prefix: string; className?: string }) {
  const id = useId()
  return (
    <div className={`flex items-start gap-3 text-sm text-ink-muted ${className}`}>
      <input id={id} type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} className="mt-0.5 size-4 shrink-0 accent-signal" />
      <span>
        <label htmlFor={id}>{prefix}</label>{' '}
        <button type="button" onClick={onOpen} className="font-medium text-signal hover:underline">{label}</button>.
      </span>
    </div>
  )
}
