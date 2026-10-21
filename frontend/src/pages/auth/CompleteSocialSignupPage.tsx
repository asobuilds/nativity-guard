import { useEffect, useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { ShieldCheck } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { useAuth } from '@/auth/AuthContext'
import { api, ApiError } from '@/lib/apiClient'
import { latestEligibleDob, MIN_SIGNUP_AGE } from '@/lib/signup'

type TermsResponse = { version: string }

export function CompleteSocialSignupPage() {
  const { status, user, refreshProfile } = useAuth()
  const navigate = useNavigate()
  const [firstName, setFirstName] = useState(user?.firstName === 'Community' ? '' : user?.firstName ?? '')
  const [lastName, setLastName] = useState(user?.lastName === 'Member' ? '' : user?.lastName ?? '')
  const [phone, setPhone] = useState('')
  const [dateOfBirth, setDateOfBirth] = useState('')
  const [termsVersion, setTermsVersion] = useState('')
  const [termsAccepted, setTermsAccepted] = useState(false)
  const [privacyAccepted, setPrivacyAccepted] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    let live = true
    Promise.all([
      api.get<TermsResponse>('/terms/citizen'),
      api.get<TermsResponse>('/terms/citizen?kind=privacy'),
    ]).then(([terms, privacy]) => {
      if (live && terms.version === privacy.version) setTermsVersion(terms.version)
    }).catch(() => { if (live) setError('Unable to load current terms. Please reload.') })
    return () => { live = false }
  }, [])

  if (status === 'loading') return <p role="status" className="p-8">Restoring your account…</p>
  if (status === 'anonymous') return <Navigate to="/auth/login" replace />
  if (user?.accountAccessLevel !== 'limited') return <Navigate to="/" replace />

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (!termsVersion || !termsAccepted || !privacyAccepted) {
      setError('Read and accept the terms and privacy notice to continue.')
      return
    }
    setError('')
    setLoading(true)
    try {
      await api.post('/auth/complete-profile', {
        firstName: firstName.trim(), lastName: lastName.trim(), phone: phone.trim(),
        dateOfBirth, termsVersion, consentTerms: termsAccepted,
        consentDataProcessing: privacyAccepted,
      })
      await refreshProfile()
      navigate('/', { replace: true })
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : 'We could not complete your profile.')
    } finally { setLoading(false) }
  }

  return (
    <main className="ng-public-scene min-h-screen bg-base px-5 py-10">
      <div className="ng-public-glass mx-auto max-w-md rounded-panel border border-border p-6 sm:p-8">
        <div className="flex items-center gap-2 text-signal">
          <ShieldCheck className="size-7" aria-hidden />
          <span className="font-semibold">NATIVITY GUARD</span>
        </div>
        <h1 className="mt-5 text-2xl font-semibold text-ink">Complete your safety profile</h1>
        <p className="mt-2 text-sm leading-relaxed text-ink-muted">
          Your Google or Apple account is connected. Before accessing reports,
          emergency SOS or private community information, we need to confirm
          age eligibility, a contact number and your agreement to the current rules.
        </p>
        <p className="mt-3 rounded-lg border border-border bg-surface-hi p-3 text-xs text-ink-muted">
          Until this is finished, protected features stay locked. Eligibility
          rules still apply, including the minimum registration age of {MIN_SIGNUP_AGE}.
        </p>
        <form onSubmit={submit} className="mt-6 space-y-4">
          <Field label="First name" required>
            {props => <Input {...props} value={firstName} onChange={event => setFirstName(event.target.value)} autoComplete="given-name" required maxLength={100} />}
          </Field>
          <Field label="Last name" required>
            {props => <Input {...props} value={lastName} onChange={event => setLastName(event.target.value)} autoComplete="family-name" required maxLength={100} />}
          </Field>
          <Field label="Phone number" required>
            {props => <Input {...props} type="tel" value={phone} onChange={event => setPhone(event.target.value)} autoComplete="tel" placeholder="+234..." required />}
          </Field>
          <Field label="Date of birth" required>
            {props => <Input {...props} type="date" value={dateOfBirth} onChange={event => setDateOfBirth(event.target.value)} max={latestEligibleDob()} required />}
          </Field>
          <fieldset className="space-y-3 rounded-lg border border-border p-3 text-sm text-ink-muted">
            <legend className="px-1 text-xs font-medium">Before activating your account</legend>
            <label className="flex gap-3">
              <input type="checkbox" checked={termsAccepted} onChange={e => setTermsAccepted(e.target.checked)} className="mt-1 size-4 accent-signal" />
              <span>I agree to the <Link to="/terms" target="_blank" rel="noopener noreferrer" className="text-signal underline">Terms of Service</Link>.</span>
            </label>
            <label className="flex gap-3">
              <input type="checkbox" checked={privacyAccepted} onChange={e => setPrivacyAccepted(e.target.checked)} className="mt-1 size-4 accent-signal" />
              <span>I consent to processing as described in the <Link to="/terms" target="_blank" rel="noopener noreferrer" className="text-signal underline">Privacy Notice</Link>.</span>
            </label>
          </fieldset>
          {error ? <p role="alert" className="text-sm text-warn">{error}</p> : null}
          <Button type="submit" block variant="primary" size="lg" loading={loading} disabled={loading || !termsVersion}>
            Activate citizen access
          </Button>
        </form>
        <Link to="/explore" className="mt-5 block text-center text-sm text-signal underline">
          Explore public features for now
        </Link>
      </div>
    </main>
  )
}
