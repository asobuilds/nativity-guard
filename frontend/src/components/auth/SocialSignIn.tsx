import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, ApiError } from '@/lib/apiClient'
import { useAuth } from '@/auth/AuthContext'

type Provider = 'google' | 'apple'

type ProviderConfig = {
  googleClientId: string
  appleServiceId: string
  appleRedirectUri: string
}
type Challenge = { nonce: string; expiresInSeconds: number }
type GoogleCredential = { credential: string }
type GoogleSDK = {
  accounts: { id: {
    initialize: (config: { client_id: string; nonce: string; callback: (result: GoogleCredential) => void }) => void
    renderButton: (element: HTMLElement, options: Record<string, string>) => void
  } }
}
type AppleSDK = {
  auth: {
    init: (config: { clientId: string; scope: string; redirectURI: string; state: string; nonce: string; usePopup: boolean }) => void
    signIn: () => Promise<{ authorization: { id_token?: string } }>
  }
}

declare global {
  interface Window { google?: GoogleSDK; AppleID?: AppleSDK }
}

const sdkPromises = new Map<string, Promise<void>>()
function loadProviderScript(src: string): Promise<void> {
  const existing = sdkPromises.get(src)
  if (existing) return existing
  const result = new Promise<void>((resolve, reject) => {
    const script = document.createElement('script')
    script.src = src
    script.async = true
    script.onload = () => resolve()
    script.onerror = () => {
      sdkPromises.delete(src)
      script.remove()
      reject(new Error('Could not load the identity provider. Check your connection.'))
    }
    document.head.append(script)
  })
  sdkPromises.set(src, result)
  return result
}

/**
 * Social providers are shown ONLY when the backend has an approved client ID.
 * The frontend never trusts Google/Apple profile data directly. The signed ID
 * token and a one-time nonce must be verified by the Go backend.
 */
export function SocialSignIn({ onComplete }: { onComplete: () => void }) {
  const { loginWithSocial } = useAuth()
  const [providers, setProviders] = useState<ProviderConfig | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const googleButton = useRef<HTMLDivElement>(null)
  const appleChallenge = useRef<string | null>(null)
  const googleNonce = useRef<string | null>(null)

  useEffect(() => {
    let mounted = true
    api.get<ProviderConfig>('/auth/social/providers').then(config => {
      if (mounted) setProviders(config)
    }).catch(() => {
      if (mounted) setError('Additional sign-in methods are temporarily unavailable.')
    })
    return () => { mounted = false }
  }, [])

  useEffect(() => {
    if (!providers?.googleClientId || !googleButton.current) return
    let disposed = false
    const target = googleButton.current
    async function initialize() {
      const challenge = await api.postAnonymous<Challenge>('/auth/social/challenge', { provider: 'google' })
      if (disposed) return
      googleNonce.current = challenge.nonce
      await loadProviderScript('https://accounts.google.com/gsi/client')
      if (disposed || !window.google) return
      window.google.accounts.id.initialize({
        client_id: providers!.googleClientId,
        nonce: challenge.nonce,
        callback: result => { void finish('google', result.credential, googleNonce.current) },
      })
      target.replaceChildren()
      window.google.accounts.id.renderButton(target, {
        theme: 'outline', size: 'large', shape: 'rectangular', width: '300',
        text: 'continue_with',
      })
    }
    void initialize().catch(() => { if (!disposed) setError('Could not prepare Google sign-in.') })
    return () => { disposed = true; target.replaceChildren() }
    // loginWithSocial is exposed by AuthContext and stays stable across renders.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [providers?.googleClientId])

  useEffect(() => {
    if (!providers?.appleServiceId) return
    let cancelled = false
    async function prepare() {
      const challenge = await api.postAnonymous<Challenge>('/auth/social/challenge', { provider: 'apple' })
      if (cancelled) return
      appleChallenge.current = challenge.nonce
      await loadProviderScript('https://appleid.cdn-apple.com/appleauth/static/jsapi/appleid/1/en_US/appleid.auth.js')
    }
    void prepare().catch(() => { if (!cancelled) setError('Apple sign-in is not ready on this device.') })
    return () => { cancelled = true }
  }, [providers?.appleServiceId])

  async function finish(provider: Provider, idToken: string, nonce: string | null) {
    if (!nonce || !idToken || busy) {
      setError('This sign-in request has expired. Please reload and try again.')
      return
    }
    setBusy(true)
    setError(null)
    try {
      await loginWithSocial(provider, idToken, nonce)
      onComplete()
    } catch (cause) {
      setError(cause instanceof ApiError ? cause.message : 'Sign-in could not be verified. Try again.')
    } finally {
      setBusy(false)
    }
  }

  async function loginWithApple() {
    if (!providers?.appleServiceId || !providers.appleRedirectUri ||
      !appleChallenge.current || !window.AppleID || busy) {
      setError('Apple sign-in is not ready. Reload the page and try again.')
      return
    }
    setError(null)
    try {
      const nonce = appleChallenge.current
      window.AppleID.auth.init({
        clientId: providers.appleServiceId,
        scope: 'name email',
        redirectURI: providers.appleRedirectUri,
        state: nonce,
        nonce,
        usePopup: true,
      })
      const result = await window.AppleID.auth.signIn()
      await finish('apple', result.authorization?.id_token ?? '', nonce)
    } catch {
      setError('Apple sign-in was cancelled or unavailable. You can try again.')
    }
  }

  if (!providers?.googleClientId && !providers?.appleServiceId) return null
  return (
    <div className="mt-5 space-y-3" aria-label="Simplified sign-in options">
      <div className="flex items-center gap-3 text-xs text-ink-muted">
        <span className="h-px flex-1 bg-border" />
        Or continue using
        <span className="h-px flex-1 bg-border" />
      </div>
      {providers.googleClientId ? (
        <div className="flex min-h-11 justify-center" ref={googleButton} aria-label="Continue with Google" />
      ) : null}
      {providers.appleServiceId ? (
        <button
          type="button"
          disabled={busy}
          onClick={() => void loginWithApple()}
          className="flex min-h-12 w-full items-center justify-center rounded-lg border border-border-hi bg-ink px-3 text-sm font-semibold text-base disabled:opacity-60"
        >
          Continue with Apple
        </button>
      ) : null}
      <p className="text-xs leading-relaxed text-ink-muted">
        New accounts start with limited access. You can explore first, then confirm
        age eligibility, consent and contact details before protected services unlock.
        Read our <Link to="/terms" className="text-signal underline">terms and privacy information</Link>.
      </p>
      {busy ? <p role="status" className="text-xs text-ink-muted">Verifying your account…</p> : null}
      {error ? <p role="alert" className="text-xs text-warn">{error}</p> : null}
    </div>
  )
}
