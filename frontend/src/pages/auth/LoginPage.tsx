import { useState, type FormEvent } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { CheckCircle2, Eye, EyeOff, LockKeyhole, Mail } from 'lucide-react'
import { AuthLayout } from '@/components/auth/AuthLayout'
import { GoogleAuthButton } from '@/components/auth/GoogleAuthButton'
import { Button } from '@/components/ui/Button'
import { Field, Input } from '@/components/ui/Field'
import { useToast } from '@/components/ui/Toast'
import { useAuth } from '@/auth/AuthContext'
import { homePathForRole } from '@/auth/RequireRole'
import { UnrecognisedRoleScreen } from '@/auth/UnrecognisedRoleScreen'
import { api, ApiError, tokenStore } from '@/lib/apiClient'
import { MOCK_ACCOUNTS, USE_MOCKS } from '@/mocks/config'

interface LoginHandoff {
  from?: string
  email?: string
  registered?: boolean
  reset?: boolean
}

interface GoogleAuthResponse {
  token?: string
  refreshToken?: string
  requiresProfileCompletion?: boolean
  profile?: {
    email: string
    firstName: string
    lastName: string
  }
}

/** Sign-in entry for password, live Google Identity Services, and gated development mocks. */
export function LoginPage() {
  const { login, status, role, rawRole, logout } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()
  const { notify } = useToast()
  const handoff = (location.state as LoginHandoff | null) ?? null

  const [email, setEmail] = useState(handoff?.email ?? '')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [googleSubmitting, setGoogleSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (status === 'authenticated') {
    const destination = handoff?.from || homePathForRole(role)
    if (!destination) return <UnrecognisedRoleScreen rawRole={rawRole} onSignOut={logout} />
    return <Navigate to={destination} replace />
  }

  async function onSubmit(event: FormEvent) {
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

  async function onGoogleCredential(idToken: string) {
    setGoogleSubmitting(true)
    setError(null)
    try {
      const result = await api.postAnonymous<GoogleAuthResponse>('/auth/google', { idToken })
      if (result.requiresProfileCompletion) {
        notify('Complete your citizen profile to continue with Google.', 'info')
        navigate('/auth/signup', {
          state: {
            googleProfile: result.profile,
            from: handoff?.from,
          },
        })
        return
      }
      if (!result.token || !result.refreshToken) {
        throw new Error('Google sign-in returned an incomplete session.')
      }
      tokenStore.set(result.token)
      tokenStore.setRefresh(result.refreshToken)
      notify('Signed in with Google.', 'success')
      // AuthProvider restores and verifies the new session on boot. Reloading also
      // avoids mutating its private state from outside the provider contract.
      window.location.reload()
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? cause.message
          : cause instanceof Error
            ? cause.message
            : 'Google sign-in could not be completed.',
      )
    } finally {
      setGoogleSubmitting(false)
    }
  }

  async function onMockGoogleSignIn() {
    const account = MOCK_ACCOUNTS.find((candidate) => candidate.role === 'Citizen') ?? MOCK_ACCOUNTS[0]
    if (!account) throw new Error('No demo account is available.')
    await login(account.email, account.password)
    notify('Signed in with the Google demo.', 'success')
  }

  return (
    <AuthLayout
      title="Welcome back"
      subtitle="Sign in to continue coordinating safer, more accountable communities."
      eyebrow="Return to your workspace"
    >
      {handoff?.registered ? (
        <p
          role="status"
          className="mb-4 flex items-start gap-2 rounded-xl border border-ok/30 bg-ok/10 px-3 py-2.5 text-xs text-ok"
        >
          <CheckCircle2 className="mt-px size-4 shrink-0" aria-hidden />
          <span>Account created. Sign in with the password you just chose.</span>
        </p>
      ) : null}

      {handoff?.reset ? (
        <p
          role="status"
          className="mb-4 flex items-start gap-2 rounded-xl border border-ok/30 bg-ok/10 px-3 py-2.5 text-xs text-ok"
        >
          <CheckCircle2 className="mt-px size-4 shrink-0" aria-hidden />
          <span>Password changed. Every device was signed out — use your new password.</span>
        </p>
      ) : null}

      <form className="flex flex-col gap-4" onSubmit={onSubmit} noValidate>
        <Field label="Email address" required>
          {(props) => (
            <div className="relative">
              <Mail
                className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint"
                aria-hidden
              />
              <Input
                {...props}
                className="h-12 pl-10"
                type="email"
                autoComplete="email"
                autoFocus
                required
                placeholder="you@example.com"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
            </div>
          )}
        </Field>

        <Field label="Password" required>
          {(props) => (
            <div className="relative">
              <LockKeyhole
                className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-ink-faint"
                aria-hidden
              />
              <Input
                {...props}
                className="h-12 px-10"
                type={showPassword ? 'text' : 'password'}
                autoComplete="current-password"
                required
                placeholder="Enter your password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
              <button
                type="button"
                onClick={() => setShowPassword((visible) => !visible)}
                aria-label={showPassword ? 'Hide password' : 'Show password'}
                aria-pressed={showPassword}
                className="absolute right-1.5 top-1/2 grid size-9 -translate-y-1/2 place-items-center rounded-lg text-ink-faint transition-colors hover:bg-surface hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-signal"
              >
                {showPassword ? <EyeOff className="size-4" aria-hidden /> : <Eye className="size-4" aria-hidden />}
              </button>
            </div>
          )}
        </Field>

        <div className="-mt-1 flex items-center justify-between gap-3 text-xs">
          <Link to="/terms" className="text-ink-muted hover:text-ink hover:underline">
            User rules
          </Link>
          <Link
            to="/auth/forgot-password"
            className="font-medium text-signal hover:text-signal-ink hover:underline"
          >
            Forgot password?
          </Link>
        </div>

        {error ? (
          <p role="alert" className="rounded-xl border border-emergency/30 bg-emergency/10 px-3 py-2.5 text-xs text-emergency">
            {error}
          </p>
        ) : null}

        <Button type="submit" variant="primary" size="lg" block loading={submitting} disabled={googleSubmitting}>
          Sign in securely
        </Button>
      </form>

      <div className="my-5 flex items-center gap-3" aria-hidden>
        <span className="h-px flex-1 bg-border" />
        <span className="text-[11px] font-medium uppercase tracking-[0.14em] text-ink-faint">or</span>
        <span className="h-px flex-1 bg-border" />
      </div>

      <GoogleAuthButton
        onCredential={onGoogleCredential}
        onMockSignIn={onMockGoogleSignIn}
        mockEmail={email}
        mockMode={USE_MOCKS}
        disabled={submitting || googleSubmitting}
        onError={(message) => setError(message || null)}
      />

      {USE_MOCKS ? (
        <div className="mt-5 rounded-xl border border-border-hi bg-surface-hi/70 p-3">
          <p className="text-xs font-semibold text-ink">Demo accounts</p>
          <p className="mt-0.5 text-[11px] text-ink-muted">Tap a role to fill the form.</p>
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

      <p className="mt-6 text-center text-sm text-ink-muted">
        New to Nativity Guard?{' '}
        <Link to="/auth/signup" className="font-semibold text-signal hover:text-signal-ink hover:underline">
          Create an account
        </Link>
      </p>
    </AuthLayout>
  )
}
