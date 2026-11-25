import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '@/auth/AuthContext'
import { AuthLayout } from '@/components/auth/AuthLayout'
import { createGoogleMockToken, GoogleAuthButton } from '@/components/auth/GoogleAuthButton'
import { ToastProvider } from '@/components/ui/Toast'
import { LoginPage } from './LoginPage'

function renderLogin(initialState?: Record<string, unknown>) {
  return renderToStaticMarkup(
    <MemoryRouter initialEntries={[{ pathname: '/auth/login', state: initialState }]}>
      <ToastProvider>
        <AuthProvider>
          <LoginPage />
        </AuthProvider>
      </ToastProvider>
    </MemoryRouter>,
  )
}

describe('LoginPage redesign', () => {
  it('renders the responsive auth scene and core sign-in paths', () => {
    const html = renderLogin()
    expect(html).toContain('ng-public-scene')
    expect(html).toContain('lg:grid-cols-')
    expect(html).toContain('Welcome back')
    expect(html).toContain('type="email"')
    expect(html).toContain('autoComplete="email"')
    expect(html).toContain('type="password"')
    expect(html).toContain('autoComplete="current-password"')
    expect(html).toContain('Sign in securely')
    expect(html).toContain('Continue with Google')
    expect(html).toContain('href="/auth/forgot-password"')
    expect(html).toContain('href="/auth/signup"')
    expect(html).toContain('href="/terms"')
  })

  it('exposes an accessible password visibility control', () => {
    const html = renderLogin()
    expect(html).toContain('aria-label="Show password"')
    expect(html).toContain('aria-pressed="false"')
    expect(html).toContain('type="button"')
  })

  it('preserves registration and password-reset handoff guidance', () => {
    const registered = renderLogin({ email: 'new@example.com', registered: true })
    expect(registered).toContain('value="new@example.com"')
    expect(registered).toContain('Account created. Sign in with the password you just chose.')
    expect(registered).toContain('role="status"')

    const reset = renderLogin({ reset: true })
    expect(reset).toContain('Every device was signed out')
  })

  it('keeps the layout theme-aware, mobile-branded and emergency-safe', () => {
    const html = renderToStaticMarkup(
      <AuthLayout title="Sign in" subtitle="Continue safely">
        <p>Form content</p>
      </AuthLayout>,
    )
    expect(html).toContain('ng-public-scene')
    expect(html).toContain('ng-public-glass')
    expect(html).toContain('lg:hidden')
    expect(html).toContain('NATIVITY GUARD')
    expect(html).toContain('href="tel:112"')
    expect(html).toContain('Access checks apply to every protected request.')
  })

  it('renders the branded Google action as a non-submit button', () => {
    const html = renderToStaticMarkup(
      <GoogleAuthButton onCredential={() => {}} />,
    )
    expect(html).toContain('Continue with Google')
    expect(html).toContain('type="button"')
    expect(html).toContain('#4285F4')
    expect(html).toContain('#34A853')
    expect(html).toContain('#FBBC05')
    expect(html).toContain('#EA4335')
  })

  it('builds only the backend-compatible, verified development mock payload', () => {
    const token = createGoogleMockToken(' ADA.OKAFOR@Example.COM ')
    expect(token.startsWith('mock:')).toBe(true)
    const encoded = token.slice('mock:'.length).replaceAll('-', '+').replaceAll('_', '/')
    const padded = encoded.padEnd(Math.ceil(encoded.length / 4) * 4, '=')
    const claims = JSON.parse(atob(padded)) as Record<string, unknown>
    expect(claims).toMatchObject({
      sub: 'development-ada.okafor@example.com',
      email: 'ada.okafor@example.com',
      email_verified: true,
      name: 'Ada Okafor',
    })
    expect(claims).not.toHaveProperty('password')
  })
})
