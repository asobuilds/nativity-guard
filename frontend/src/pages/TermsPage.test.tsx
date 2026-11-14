import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import { TermsPage } from './TermsPage'

describe('Public Terms and Privacy page', () => {
  it('renders separate legal-document headings even before either API returns', () => {
    const html = renderToStaticMarkup(<MemoryRouter><TermsPage /></MemoryRouter>)
    expect(html).toContain('id="terms"')
    expect(html).toContain('id="privacy"')
    expect(html).toContain('Terms of Service')
    expect(html).toContain('Privacy Notice')
    expect((html.match(/Loading published document/g) ?? [])).toHaveLength(2)
  })

  it('exposes document anchors and an accurate non-certifying rights explanation', () => {
    const html = renderToStaticMarkup(<MemoryRouter><TermsPage /></MemoryRouter>)
    expect(html).toContain('href="#terms"')
    expect(html).toContain('href="#privacy"')
    expect(html).toContain('does not certify legal compliance')
    expect(html).toContain('mailto:support@nativityguard.com')
    expect(html).not.toContain('Government approved')
  })
})
