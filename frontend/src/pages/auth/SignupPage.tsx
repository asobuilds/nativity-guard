import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { AlertCircle, Shield } from 'lucide-react'
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

/**
 * Create an account.
 *
 * Two facts about `POST /auth/register` shape this screen, both read off
 * `handlers/auth_handler.go` and recorded in frontReadme Appendix B:
 *
 *  1. It returns **no token** — `{ message, user }` and nothing else. After
 *     registration, sign in with the submitted credentials to establish a session.
 *  2. It **ignores any `role` you send** and hardcodes `"citizen"`. There is no
 *     role selector here, and the copy says plainly what kind of account this
 *     creates rather than offering a choice that does not exist.
 *
 * The age gate is checked here *and* on the server. That duplication is
 * deliberate: the server's refusal is the authority, but a person should not have
 * to submit a form to be told they are too young to have one.
 */
export function SignupPage() {
  const navigate = useNavigate()
  const { login } = useAuth()

  const [draft, setDraft] = useState<SignupDraft>(EMPTY_SIGNUP)
  const [touched, setTouched] = useState<ReadonlySet<SignupField>>(new Set())
  const [submitting, setSubmitting] = useState(false)
  const [refusal, setRefusal] = useState<string | null>(null)

  const errors = validateSignup(draft)
  const complete = SIGNUP_FIELDS.every((field) => !errors[field])

  /** An error is shown once you have left the field, or once you have tried to submit. */
  const errorFor = (field: SignupField) => (touched.has(field) ? errors[field] : undefined)

  function update(field: SignupField, value: string) {
    setDraft((current) => ({ ...current, [field]: value }))
    setRefusal(null)
  }

  function markTouched(field: SignupField) {
    setTouched((current) => new Set(current).add(field))
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setTouched(new Set(SIGNUP_FIELDS))
    setRefusal(null)
    if (!complete) return

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
        // No `role` — the endpoint ignores it, and sending one would suggest
        // this form can choose a privileged account. It cannot.
      })
      registered = true
      await login(draft.email.trim(), draft.password)
      navigate('/', { replace: true })
    } catch (cause) {
      if (registered) {
        // The account exists even if session creation failed. Never retry registration.
        navigate('/auth/login', {
          replace: true,
          state: { email: draft.email.trim(), registered: true },
        })
        return
      }
      // The server's refusal is repeated verbatim. Its duplicate-account message is
      // deliberately generic — it never says *which* of email or phone matched — and
      // guessing a field here would undo that.
      setRefusal(
        cause instanceof ApiError
          ? cause.message
          : 'Could not create your account. Check your connection and try again.',
      )
    } finally {
      setSubmitting(false)
    }
  }

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
            This creates a citizen account — you can report incidents and follow them. Joining a
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

            {refusal ? (
              <p
                role="alert"
                className="flex items-start gap-2 rounded-lg border border-warn/30 bg-warn/10 px-3 py-2 text-xs text-warn"
              >
                <AlertCircle className="mt-px size-4 shrink-0" aria-hidden />
                <span>{refusal}</span>
              </p>
            ) : null}

            <Button type="submit" variant="primary" size="lg" block loading={submitting}>
              Create account
            </Button>
          </form>

          <p className="mt-6 text-sm text-ink-muted">
            Please read the <Link to="/terms" className="text-signal hover:underline">user rules for your account</Link> before creating an account.
          </p>
          <p className="mt-3 text-sm text-ink-muted">
            Already have an account?{' '}
            <Link to="/auth/login" className="text-signal hover:text-signal-ink">
              Sign in
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
