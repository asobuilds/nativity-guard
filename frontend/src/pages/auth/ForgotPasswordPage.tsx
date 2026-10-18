import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { AlertCircle, Shield } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { api, ApiError } from '@/lib/apiClient'
import { validateEmail } from '@/lib/passwordReset'
import { USE_MOCKS } from '@/mocks/config'
import type { ForgotPasswordResponse } from '@/types/api'

export function ForgotPasswordPage() {
  const navigate = useNavigate()

  const [email, setEmail] = useState('')
  const [touched, setTouched] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [sent, setSent] = useState(false)
  const [refusal, setRefusal] = useState<string | null>(null)

  const normalizedEmail = email.trim().toLowerCase()
  const error = validateEmail(normalizedEmail)

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setTouched(true)
    setRefusal(null)

    if (error) return

    setSubmitting(true)

    try {
      const response = await api.postAnonymous<ForgotPasswordResponse>(
        '/auth/forgot-password',
        { email: normalizedEmail },
      )

      // The mock server returns a simulated token for local testing only.
      // The real backend should send the token through email, not its response.
      if (USE_MOCKS && response.token) {
        navigate(
          `/auth/reset-password?token=${encodeURIComponent(response.token)}`,
          { replace: true },
        )
        return
      }

      setSent(true)
    } catch (cause) {
      setRefusal(
        cause instanceof ApiError
          ? cause.message
          : 'Could not request a password reset. Check your connection and try again.',
      )
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="grid min-h-screen bg-base lg:grid-cols-2">
      <div className="hidden flex-col justify-between bg-surface p-10 lg:flex">
        <div className="flex items-center gap-2">
          <Shield className="size-7 text-signal" aria-hidden />
          <span className="text-base font-bold tracking-wide text-ink">
            NATIVITY GUARD
          </span>
        </div>

        <div>
          <h1 className="max-w-md text-3xl font-semibold leading-tight text-ink">
            Every report answered. Every case accounted for.
          </h1>
          <p className="mt-3 max-w-md text-sm text-ink-muted">
            Recover access to your account using a secure password-reset link.
          </p>
        </div>

        <p className="text-xs text-ink-faint">
          Authorised personnel only. Activity on this system is recorded.
        </p>
      </div>

      <div className="flex items-center justify-center p-6">
        <div className="w-full max-w-sm">
          <div className="mb-6 flex items-center gap-2 lg:hidden">
            <Shield className="size-6 text-signal" aria-hidden />
            <span className="text-sm font-bold tracking-wide text-ink">
              NATIVITY GUARD
            </span>
          </div>

          {sent ? (
            <>
              <h2 className="text-xl font-semibold text-ink">
                Check your email
              </h2>

              <p className="mt-2 text-sm text-ink-muted">
                If an account exists for that email address, a password-reset
                link will be sent. Check your inbox and spam folder.
              </p>

              <p className="mt-4 break-words text-sm text-ink">
                {normalizedEmail}
              </p>

              <p className="mt-8 text-sm text-ink-muted">
                <Link
                  to="/auth/login"
                  className="text-signal hover:text-signal-ink"
                >
                  Back to sign in
                </Link>
              </p>
            </>
          ) : (
            <>
              <h2 className="text-xl font-semibold text-ink">
                Reset your password
              </h2>

              <p className="mt-1 text-sm text-ink-muted">
                Enter the email address associated with your account. If it
                matches an account, we will send a secure reset link.
              </p>

              <form
                className="mt-6 flex flex-col gap-4"
                onSubmit={onSubmit}
                noValidate
              >
                <Field
                  label="Email address"
                  required
                  error={touched ? error : undefined}
                >
                  {(props) => (
                    <Input
                      {...props}
                      type="email"
                      autoComplete="email"
                      autoFocus
                      required
                      value={email}
                      onChange={(event) => {
                        setEmail(event.target.value)
                        setRefusal(null)
                      }}
                      onBlur={() => setTouched(true)}
                    />
                  )}
                </Field>

                {refusal ? (
                  <p
                    role="alert"
                    className="flex items-start gap-2 rounded-lg border border-warn/30 bg-warn/10 px-3 py-2 text-xs text-warn"
                  >
                    <AlertCircle
                      className="mt-px size-4 shrink-0"
                      aria-hidden
                    />
                    <span>{refusal}</span>
                  </p>
                ) : null}

                <Button
                  type="submit"
                  variant="primary"
                  size="lg"
                  block
                  loading={submitting}
                >
                  Send reset link
                </Button>
              </form>

              <p className="mt-8 text-sm text-ink-muted">
                Remembered it?{' '}
                <Link
                  to="/auth/login"
                  className="text-signal hover:text-signal-ink"
                >
                  Back to sign in
                </Link>
              </p>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
