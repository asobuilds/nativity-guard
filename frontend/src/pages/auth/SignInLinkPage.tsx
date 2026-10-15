import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { Shield } from 'lucide-react'
import { useAuth } from '@/auth/AuthContext'
import { ApiError } from '@/lib/apiClient'

export function SignInLinkPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { loginWithLinkToken } = useAuth()
  const started = useRef(false)
  const [message, setMessage] = useState(
    'Verifying your sign-in link…',
  )
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (started.current) return
    started.current = true

    const token = searchParams.get('token')

    if (!token) {
      setMessage('This sign-in link is missing its token.')
      setFailed(true)
      return
    }

    let cancelled = false

    async function redeem() {
      try {
        await loginWithLinkToken(token!)

        if (!cancelled) {
          navigate('/', { replace: true })
        }
      } catch (cause) {
        if (!cancelled) {
          setMessage(
            cause instanceof ApiError
              ? 'This sign-in link is invalid, expired, or already used. Request a new link.'
              : 'We could not verify this link. Check your connection and request a new link.',
          )
          setFailed(true)
        }
      }
    }

    void redeem()

    return () => {
      cancelled = true
    }
  }, [searchParams, loginWithLinkToken, navigate])

  return (
    <main className="flex min-h-screen items-center justify-center bg-base p-6">
      <section className="w-full max-w-md rounded-panel border border-border-hi bg-surface p-8">
        <div className="mb-5 flex items-center gap-2">
          <Shield className="size-7 text-signal" aria-hidden />
          <span className="font-bold tracking-wide text-ink">
            NATIVITY GUARD
          </span>
        </div>

        <h1 className="text-xl font-semibold text-ink">
          Email sign-in
        </h1>

        <p className="mt-3 text-sm text-ink-muted" role="status">
          {message}
        </p>

        {failed && (
          <Link
            to="/auth/login"
            className="mt-6 inline-flex text-sm font-semibold text-signal hover:text-signal-ink"
          >
            Return to sign in
          </Link>
        )}
      </section>
    </main>
  )
}