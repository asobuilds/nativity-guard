
import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/lib/apiClient'

export function RequestSignInLinkPage() {
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setMessage('')
    setError('')

    const normalizedEmail = email.trim().toLowerCase()

    if (
      !normalizedEmail ||
      !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizedEmail)
    ) {
      setError('Please enter a valid email address.')
      return
    }

    setLoading(true)

    try {
      await api.postAnonymous('/auth/sign-in-link/request', {
        email: normalizedEmail,
      })

      setMessage(
        'If an account with that email exists, a sign-in link will be sent. Check your inbox and spam folder.',
      )
      setEmail('')
    } catch {
      setError(
        'We could not process your request right now. Please try again shortly.',
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="min-h-screen bg-base flex items-center justify-center px-4 py-10">
      <section className="w-full max-w-md rounded-2xl border border-border bg-surface p-6 shadow-xl sm:p-8">
        <div className="mb-8 text-center">
          <h1 className="text-2xl font-bold tracking-tight text-ink">
            Sign in with email
          </h1>
          <p className="mt-3 text-sm text-ink-muted">
            We will email you a secure, one-time sign-in link. You will not
            need to enter your password.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label
              htmlFor="sign-in-email"
              className="mb-2 block text-sm font-medium text-ink"
            >
              Email address
            </label>
            <input
              id="sign-in-email"
              name="email"
              type="email"
              autoComplete="email"
              autoFocus
              required
              maxLength={254}
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="you@example.com"
              disabled={loading}
              className="w-full rounded-xl border border-border bg-base px-4 py-3 text-ink outline-none transition focus:ring-2 focus:ring-signal disabled:cursor-not-allowed disabled:opacity-60"
            />
          </div>

          {error ? (
            <div
              role="alert"
              className="rounded-xl border border-red-500/40 bg-red-500/10 p-3 text-sm text-ink"
            >
              {error}
            </div>
          ) : null}

          {message ? (
            <div
              role="status"
              className="rounded-xl border border-ok/40 bg-ok/10 p-3 text-sm text-ink"
            >
              {message}
            </div>
          ) : null}

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-xl bg-signal px-4 py-3 font-semibold text-signal-ink transition hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {loading ? 'Sending link…' : 'Email me a sign-in link'}
          </button>
        </form>

        <div className="mt-6 space-y-3 text-center text-sm">
          <p>
            <Link
              to="/auth/login"
              className="font-medium text-signal underline underline-offset-4"
            >
              Back to sign in
            </Link>
          </p>

          <p className="text-ink-muted">
            Forgot your password?{' '}
            <Link
              to="/auth/forgot-password"
              className="font-medium text-signal underline underline-offset-4"
            >
              Reset it here
            </Link>
          </p>
        </div>
      </section>
    </main>
  )
}
