import { describe, expect, it } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import { LandingTrustFooter } from './LandingTrustFooter'
import { LandingQuickPanel } from './LandingQuickPanel'
import { panelFromHash, PANEL_TITLES, PANEL_TO_HASH } from './landingPanels'

describe('Landing quick views and public trust information', () => {
  it('maps all public feature hashes and handles malformed URLs safely', () => {
    for (const [panel, hash] of Object.entries(PANEL_TO_HASH)) {
      expect(panelFromHash('#' + hash)).toBe(panel)
    }
    expect(panelFromHash('#map')).toBe('map')
    expect(panelFromHash('#track')).toBe('track')
    expect(panelFromHash('#unknown')).toBeNull()
    expect(panelFromHash('#%broken')).toBeNull()
  })

  it('gives quick-view panels an accessible dialog title and close action', () => {
    const html = renderToStaticMarkup(
      <LandingQuickPanel title={PANEL_TITLES.governance} onClose={() => {}}>
        <p>How oversight works</p>
      </LandingQuickPanel>,
    )
    expect(html).toContain('role="dialog"')
    expect(html).toContain('aria-modal="true"')
    expect(html).toContain('aria-labelledby=')
    expect(html).toContain('Close Governance')
  })

  it('links to published about and legal pages without false certification', () => {
    const html = renderToStaticMarkup(<MemoryRouter><LandingTrustFooter /></MemoryRouter>)
    expect(html).toContain('href="/terms#terms"')
    expect(html).toContain('href="/terms#privacy"')
    expect(html).toContain('href="/about"')
    expect(html).toContain('tel:112')
    expect(html).toContain('not a police force or emergency response service')
    expect(html).toContain('No government endorsement')
    expect(html).not.toContain('legally auditable')
  })
})
