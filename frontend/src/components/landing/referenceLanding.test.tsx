import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { LandingHero } from './LandingHero'
import { LandingNavbar } from './LandingNavbar'
import { LandingTrustFooter } from './LandingTrustFooter'
import { LandingQuickPanel } from './LandingQuickPanel'

describe('Reference landing layout', () => {
  it('keeps exactly the three primary actions and the separate emergency entry', () => {
    const html = renderToStaticMarkup(
      <QueryClientProvider client={new QueryClient()}>
        <LandingHero onActionTrigger={vi.fn()} onOpen={vi.fn()} />
      </QueryClientProvider>,
    )
    expect(html).toContain('Your Community.')
    expect(html).toContain('Your Safety.')
    expect(html).toContain('Your Voice.')
    expect(html).toContain('Report an Incident')
    expect(html).toContain('Track My Report')
    expect(html).toContain('Explore Safety Map')
    expect(html).toContain('Emergency SOS')
    expect((html.match(/class="ng-ref-action(?: ng-ref-action-report)?"/g) ?? [])).toHaveLength(3)
    expect(html).not.toContain('4 mins')
    expect(html).not.toContain('States Covered')
    expect(html).toContain('National emergency line')
  })

  it('keeps the reference navigation actions and login/signup destinations', () => {
    const html = renderToStaticMarkup(
      <MemoryRouter>
        <LandingNavbar onOpen={vi.fn()} onReport={vi.fn()} appearance={{ palette: 'forest', wallpaper: 'gate' }} onAppearanceChange={vi.fn()} />
      </MemoryRouter>,
    )
    for (const label of ['Home', 'Safety Map', 'Report', 'Track Case', 'Explore Features', 'About']) {
      expect(html).toContain(label)
    }
    expect(html).toContain('href="/auth/login"')
    expect(html).toContain('href="/auth/signup"')
    expect(html).toContain('Search case reports')
    expect(html).toContain('href="/explore"')
    expect(html).toContain('Change landing page colours and wallpaper')
  })
  it('makes the published policies, privacy rights, and emergency limitations visible', () => {
    const html = renderToStaticMarkup(
      <MemoryRouter><LandingTrustFooter /></MemoryRouter>,
    )
    expect(html).toContain('About &amp; accountability')
    expect(html).toContain('href="/about"')
    expect(html).toContain('href="/terms#terms"')
    expect(html).toContain('href="/terms#privacy"')
    expect(html).toContain('Privacy notice &amp; your rights')
    expect(html).toContain('not a police force')
    expect(html).toContain('deletion of personal data')
    expect(html).toContain('href="tel:112"')
    expect(html).not.toContain('Government approved')
  })

  it('labels its public quick-view dialog for assistive technology', () => {
    const html = renderToStaticMarkup(
      <LandingQuickPanel title="Safety Map" onClose={vi.fn()}>
        <p>Map content</p>
      </LandingQuickPanel>,
    )
    expect(html).toContain('role="dialog"')
    expect(html).toContain('aria-modal="true"')
    expect(html).toContain('aria-labelledby=')
    expect(html).toContain('Close Safety Map')
    expect(html).toContain('Map content')
  })
})
