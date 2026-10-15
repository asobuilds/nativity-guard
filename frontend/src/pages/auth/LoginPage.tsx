
import { useState, type FormEvent } from 'react'
import { Link, Navigate, useLocation } from 'react-router-dom'
import { CheckCircle2, Shield } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { useToast } from '@/components/ui/Toast'
import { useAuth } from '@/auth/AuthContext'
import { homePathForRole } from '@/auth/RequireRole'
import { UnrecognisedRoleScreen } from '@/auth/UnrecognisedRoleScreen'
import { ApiError } from '@/lib/apiClient'
import { MOCK_ACCOUNTS, USE_MOCKS } from '@/mocks/config'

interface LoginHandoff {
  from?: string
  email?: string
  registered?: boolean
  reset?: boolean
}

export function LoginPage() {
  const { login, status, role, rawRole, logout } = useAuth()
  const location = useLocation()
  const { notify } = useToast()

  const handoff = (location.state as LoginHandoff | null) ?? null

  const [email, setEmail] = useState(handoff?.email ?? '')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (status === 'authenticated') {
    const destination = handoff?.from || homePathForRole(role)

    if (!destination) {
      return <UnrecognisedRoleScreen rawRole={rawRole} onSignOut={logout} />
    }

    return <Navigate to={destination} replace />
  }

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError(null)

    try {
      await login(email.trim(), password)
      notify('Signed in.', 'success')
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? cause.message
          : 'Could not sign in. Check your details and try again.',
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
            Nativity Guard connects citizens, security units and administrators
            on one operational record — from the first SOS to the final report.
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

          <h2 className="text-xl font-semibold text-ink">Sign in</h2>
          <p className="mt-1 text-sm text-ink-muted">
            Use your service email address and password.
          </p>

          <Link
            to="/terms"
            className="mt-2 inline-block text-xs text-signal hover:underline"
          >
            Read user rules
          </Link>

          {handoff?.registered ? (
            <p
              role="status"
              className="mt-4 flex items-start gap-2 rounded-lg border border-ok/30 bg-ok/10 px-3 py-2 text-xs text-ok"
            >
              <CheckCircle2 className="mt-px size-4 shrink-0" aria-hidden />
              <span>
                Account created. Sign in with the password you just chose.
              </span>
            </p>
          ) : null}

          {handoff?.reset ? (
            <p
              role="status"
              className="mt-4 flex items-start gap-2 rounded-lg border border-ok/30 bg-ok/10 px-3 py-2 text-xs text-ok"
            >
              <CheckCircle2 className="mt-px size-4 shrink-0" aria-hidden />
              <span>
                Password changed. Every device on the account was signed out —
                sign in with the new password.
              </span>
            </p>
          ) : null}

          <form
            className="mt-6 flex flex-col gap-4"
            onSubmit={onSubmit}
            noValidate
          >
            <Field label="Email" required>
              {(props) => (
                <Input
                  {...props}
                  type="email"
                  autoComplete="email"
                  autoFocus
                  required
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                />
              )}
            </Field>

            <Field label="Password" required error={error ?? undefined}>
              {(props) => (
                <Input
                  {...props}
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              )}
            </Field>

            <Button
              type="submit"
              variant="primary"
              size="lg"
              block
              loading={submitting}
            >
              Sign in
            </Button>

            <Link
              to="/auth/forgot-password"
              className="-mt-2 self-end text-xs text-signal hover:text-signal-ink"
            >
              Forgot password?
            </Link>
          </form>

          <div className="mt-5 border-t border-border pt-4 text-center">
            <p className="text-xs text-ink-muted">
              Prefer not to use your password?
            </p>
            <Link
              to="/auth/request-sign-in-link"
              className="mt-1 inline-block text-sm font-medium text-signal hover:text-signal-ink hover:underline"
            >
              Email me a one-time sign-in link
            </Link>
          </div>

          {USE_MOCKS ? (
            <div className="mt-8 rounded-panel border border-border-hi bg-surface-hi p-3">
              <p className="text-xs font-semibold text-ink">Demo accounts</p>
              <p className="mt-0.5 text-[11px] text-ink-muted">
                Running on mocks (<code>VITE_USE_MOCKS</code>) — tap a role to
                fill the form.
              </p>
              <div className="mt-2 flex flex-wrap gap-2">
                {MOCK_ACCOUNTS.map((account) => (
                  <Button
                    key={account.email}
                    size="sm"
                    variant="secondary"
                    onClick={() => {
                      setEmail(account.email)
                      setPassword(account.password)
                      setError(null)
                    }}
                  >
                    {account.role}
                  </Button>
                ))}
              </div>
            </div>
          ) : null}

          <p className="mt-8 text-sm text-ink-muted">
            No account yet?{' '}
            <Link
              to="/auth/signup"
              className="text-signal hover:text-signal-ink"
            >
              Create one
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
