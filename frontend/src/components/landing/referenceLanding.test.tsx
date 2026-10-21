import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { LandingHero } from './LandingHero'
import { LandingNavbar } from './LandingNavbar'

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
})
