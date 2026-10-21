import { describe, expect, it } from 'vitest'
import { HASH_TO_PANEL, PANEL_TO_HASH, PANEL_TITLES, panelFromHash } from './landingPanels'
import { LANDING_FEATURES } from './LandingNavbar'

describe('landing feature pop-up navigation', () => {
  it('opens every public feature by its documented hash', () => {
    for (const feature of LANDING_FEATURES) {
      expect(panelFromHash('#' + PANEL_TO_HASH[feature.panel])).toBe(feature.panel)
      expect(PANEL_TITLES[feature.panel]).toBeTruthy()
    }
  })

  it('preserves old bookmarks for tracking and other legacy sections', () => {
    expect(panelFromHash('#case-lifecycle')).toBe('track')
    expect(panelFromHash('#track')).toBe('track')
    expect(panelFromHash('#sos-simulator')).toBe('sos-simulator')
    expect(Object.keys(HASH_TO_PANEL).length).toBeGreaterThanOrEqual(LANDING_FEATURES.length)
  })

  it('ignores unknown and malformed URL fragments without crashing', () => {
    expect(panelFromHash('#%E0%A4%A')).toBeNull()
    expect(panelFromHash('#untrusted')).toBeNull()
    expect(panelFromHash('')).toBeNull()
  })
})
