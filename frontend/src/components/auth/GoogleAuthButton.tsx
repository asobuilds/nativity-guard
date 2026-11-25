import { useState } from 'react'
import { Loader2 } from 'lucide-react'
import { cn } from '@/lib/cn'

interface GoogleCredentialResponse {
  credential?: string
}

interface GooglePromptNotification {
  isNotDisplayed?: () => boolean
  isSkippedMoment?: () => boolean
  isDismissedMoment?: () => boolean
}

interface GoogleIdentityServices {
  accounts: {
    id: {
      initialize: (options: {
        client_id: string
        callback: (response: GoogleCredentialResponse) => void
        auto_select?: boolean
        cancel_on_tap_outside?: boolean
      }) => void
      prompt: (callback?: (notification: GooglePromptNotification) => void) => void
    }
  }
}

declare global {
  interface Window {
    google?: GoogleIdentityServices
  }
}

interface GoogleAuthButtonProps {
  onCredential: (credential: string) => Promise<void> | void
  onMockSignIn?: () => Promise<void> | void
  mockEmail?: string
  mockMode?: boolean
  disabled?: boolean
  className?: string
  onError?: (message: string) => void
}

let googleScriptPromise: Promise<void> | null = null

function loadGoogleIdentityServices(): Promise<void> {
  if (window.google?.accounts.id) return Promise.resolve()
  if (googleScriptPromise) return googleScriptPromise

  const pending = new Promise<void>((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>('script[data-ng-google-identity]')
    if (existing) {
      // A prior failed load leaves a script element behind but no future load
      // event to await. Remove that stale attempt before creating a retry.
      existing.remove()
    }

    const script = document.createElement('script')
    script.src = 'https://accounts.google.com/gsi/client'
    script.async = true
    script.defer = true
    script.dataset.ngGoogleIdentity = 'true'
    script.onload = () => resolve()
    script.onerror = () => {
      script.remove()
      reject(new Error('Google sign-in could not load.'))
    }
    document.head.appendChild(script)
  })
  const tracked = pending.catch((error: unknown) => {
    googleScriptPromise = null
    throw error
  })
  googleScriptPromise = tracked
  return tracked
}

function base64UrlEncode(value: string): string {
  const bytes = new TextEncoder().encode(value)
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary).replaceAll('+', '-').replaceAll('/', '_').replace(/=+$/, '')
}

/** Development-only payload understood by the backend's explicitly gated mock verifier. */
export function createGoogleMockToken(email: string): string {
  const normalisedEmail = email.trim().toLowerCase() || 'citizen@shield.ng'
  const localPart = normalisedEmail.split('@')[0] || 'citizen'
  const displayName = localPart
    .split(/[._-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ') || 'Citizen'
  const claims = {
    sub: `development-${normalisedEmail}`,
    email: normalisedEmail,
    email_verified: true,
    name: displayName,
  }
  return `mock:${base64UrlEncode(JSON.stringify(claims))}`
}

function GoogleMark() {
  return (
    <svg viewBox="0 0 24 24" className="size-5" aria-hidden>
      <path
        fill="#4285F4"
        d="M21.6 12.23c0-.71-.06-1.4-.18-2.07H12v3.91h5.38a4.6 4.6 0 0 1-2 3.02v2.54h3.24c1.9-1.75 2.98-4.32 2.98-7.4Z"
      />
      <path
        fill="#34A853"
        d="M12 22c2.7 0 4.98-.9 6.63-2.43l-3.24-2.54c-.9.6-2.05.96-3.39.96-2.61 0-4.82-1.77-5.61-4.14H3.04v2.62A10 10 0 0 0 12 22Z"
      />
      <path
        fill="#FBBC05"
        d="M6.39 13.85A6.01 6.01 0 0 1 6.08 12c0-.64.11-1.27.31-1.85V7.53H3.04A10 10 0 0 0 2 12c0 1.61.38 3.14 1.04 4.47l3.35-2.62Z"
      />
      <path
        fill="#EA4335"
        d="M12 6.01c1.47 0 2.79.5 3.83 1.5l2.87-2.87A9.64 9.64 0 0 0 12 2a10 10 0 0 0-8.96 5.53l3.35 2.62C7.18 7.78 9.39 6.01 12 6.01Z"
      />
    </svg>
  )
}

export function GoogleAuthButton({
  onCredential,
  onMockSignIn,
  mockEmail = '',
  mockMode = false,
  disabled = false,
  className,
  onError,
}: GoogleAuthButtonProps) {
  const [loading, setLoading] = useState(false)

  async function startGoogleSignIn() {
    if (disabled || loading) return
    setLoading(true)
    onError?.('')

    try {
      if (mockMode && onMockSignIn) {
        await onMockSignIn()
        return
      }

      const mockBackendEnabled =
        import.meta.env.DEV && import.meta.env.VITE_GOOGLE_AUTH_MOCK_ENABLED === 'true'
      if (mockBackendEnabled) {
        await onCredential(createGoogleMockToken(mockEmail))
        return
      }

      const clientId = (import.meta.env.VITE_GOOGLE_CLIENT_ID ?? '').trim()
      if (!clientId) {
        throw new Error('Google sign-in is not configured for this environment.')
      }

      await loadGoogleIdentityServices()
      if (!window.google?.accounts.id) throw new Error('Google sign-in is unavailable right now.')

      await new Promise<void>((resolve, reject) => {
        window.google?.accounts.id.initialize({
          client_id: clientId,
          auto_select: false,
          cancel_on_tap_outside: true,
          callback: (response) => {
            if (!response.credential) {
              reject(new Error('Google did not return a sign-in credential.'))
              return
            }
            Promise.resolve(onCredential(response.credential)).then(resolve, reject)
          },
        })
        window.google?.accounts.id.prompt((notification) => {
          if (
            notification.isNotDisplayed?.() ||
            notification.isSkippedMoment?.() ||
            notification.isDismissedMoment?.()
          ) {
            reject(new Error('Google sign-in was not completed. Please try again.'))
          }
        })
      })
    } catch (cause) {
      onError?.(cause instanceof Error ? cause.message : 'Google sign-in could not be completed.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <button
      type="button"
      onClick={() => void startGoogleSignIn()}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={cn(
        'flex h-12 w-full items-center justify-center gap-3 rounded-xl border border-border-hi',
        'bg-surface-hi px-4 text-sm font-semibold text-ink shadow-sm transition-all',
        'hover:-translate-y-0.5 hover:border-signal/40 hover:bg-surface focus-visible:outline-none',
        'focus-visible:ring-2 focus-visible:ring-signal disabled:cursor-not-allowed disabled:opacity-60',
        className,
      )}
    >
      {loading ? <Loader2 className="size-5 animate-spin text-signal" aria-hidden /> : <GoogleMark />}
      {loading ? 'Connecting to Google…' : mockMode ? 'Continue with Google (demo)' : 'Continue with Google'}
    </button>
  )
}
