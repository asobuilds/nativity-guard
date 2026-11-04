import { describe, expect, it, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { LandingServiceNotice } from './LandingServiceNotice'

const mock = vi.hoisted(() => ({ useHealth: vi.fn() }))
vi.mock('@/hooks/usePublicServiceAvailability', () => ({
  usePublicServiceAvailability: mock.useHealth,
}))

describe('public service availability notice', () => {
  it('remains hidden until the health endpoint confirms connection trouble', () => {
    mock.useHealth.mockReturnValue({ data: undefined, refetch: vi.fn(), isFetching: true })
    expect(renderToStaticMarkup(<LandingServiceNotice />)).toBe('')
    mock.useHealth.mockReturnValue({ data: 'unknown', refetch: vi.fn(), isFetching: false })
    expect(renderToStaticMarkup(<LandingServiceNotice />)).toBe('')
    mock.useHealth.mockReturnValue({ data: 'available', refetch: vi.fn(), isFetching: false })
    expect(renderToStaticMarkup(<LandingServiceNotice />)).toBe('')
  })

  it('discloses limitations and links to 112 without suggesting live SOS dispatch', () => {
    mock.useHealth.mockReturnValue({ data: 'unavailable', refetch: vi.fn(), isFetching: false })
    const html = renderToStaticMarkup(<LandingServiceNotice />)
    expect(html).toContain('Live services may be unavailable')
    expect(html).toContain('Sign-in, incident reports, case tracking')
    expect(html).toContain('href="tel:112"')
    expect(html).toContain('Check again')
    expect(html).not.toContain('Emergency response is on its way')
  })
})
