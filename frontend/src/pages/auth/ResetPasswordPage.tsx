import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { AlertCircle, CheckCircle2, Shield } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { useAuth } from '@/auth/AuthContext'
import { api, ApiError } from '@/lib/apiClient'
import {
  MIN_RESET_PASSWORD_LENGTH,
  validateNewPassword,
} from '@/lib/passwordReset'
import { USE_MOCKS } from '@/mocks/config'
import type { ResetPasswordResponse } from '@/types/api'

export function ResetPasswordPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { logout } = useAuth()

  // Keep the URL token intact. It is an opaque token, not a numeric code.
  const token = searchParams.get('token') ?? ''

  const [password, setPassword] = useState('')
  const [touched, setTouched] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [refusal, setRefusal] = useState<string | null>(null)

  const passwordError = validateNewPassword(password)
  const complete = !passwordError

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setTouched(true)
    setRefusal(null)

    if (!token) {
      setRefusal('This reset link is missing its token. Request a new reset link.')
      return
    }

    if (!complete) return

    setSubmitting(true)

    try {
      await api.postAnonymous<ResetPasswordResponse>(
        '/auth/reset-password',
        {
          token,
          newPassword: password,
        },
      )

      // The backend revokes sessions after a successful password reset.
      logout()

      navigate('/auth/login', {
        replace: true,
        state: { reset: true },
      })
    } catch (cause) {
      setRefusal(
        cause instanceof ApiError
          ? cause.message
          : 'Could not reset your password. Check your connection and try again.',
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
            Choose a new password to secure your account. Resetting your
            password signs out your other sessions.
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

          <h2 className="text-xl font-semibold text-ink">
            Choose a new password
          </h2>

          <p className="mt-1 text-sm text-ink-muted">
            Enter a new password of at least {MIN_RESET_PASSWORD_LENGTH} characters.
          </p>

          {!token ? (
            <p
              role="alert"
              className="mt-4 rounded-lg border border-warn/30 bg-warn/10 px-3 py-2 text-sm text-warn"
            >
              This link has no reset token. Request a new password-reset link.
            </p>
          ) : null}

          <form
            className="mt-6 flex flex-col gap-4"
            onSubmit={onSubmit}
            noValidate
          >
            <Field
              label="New password"
              required
              hint={`At least ${MIN_RESET_PASSWORD_LENGTH} characters.`}
              error={touched ? passwordError : undefined}
            >
              {(props) => (
                <Input
                  {...props}
                  type="password"
                  autoComplete="new-password"
                  autoFocus
                  required
                  value={password}
                  onChange={(event) => {
                    setPassword(event.target.value)
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
              disabled={!token || !complete}
            >
              Set new password
            </Button>
          </form>

          {USE_MOCKS ? (
            <p
              role="status"
              className="mt-4 flex items-start gap-2 rounded-lg border border-border-hi bg-surface-hi px-3 py-2 text-xs text-ink-muted"
            >
              <CheckCircle2
                className="mt-px size-4 shrink-0 text-signal"
                aria-hidden
              />
              <span>
                Mock mode is enabled. No email is sent; the reset link is
                simulated locally.
              </span>
            </p>
          ) : null}

          <p className="mt-8 text-sm text-ink-muted">
            Need a new link?{' '}
            <Link
              to="/auth/forgot-password"
              className="text-signal hover:text-signal-ink"
            >
              Request another
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
