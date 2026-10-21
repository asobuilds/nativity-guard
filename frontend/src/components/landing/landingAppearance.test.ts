import { describe, expect, it } from 'vitest'
import { DEFAULT_LANDING_APPEARANCE, normalizeLandingAppearance } from './landingAppearance'

describe('Landing appearance settings', () => {
  it('defaults to the approved City Gate forest design', () => {
    expect(normalizeLandingAppearance(null)).toEqual(DEFAULT_LANDING_APPEARANCE)
  })
  it('accepts defined palette and wallpaper choices', () => {
    expect(normalizeLandingAppearance({ palette: 'ocean', wallpaper: 'plain' })).toEqual({
      palette: 'ocean', wallpaper: 'plain',
    })
  })
  it('rejects invalid or unexpected stored settings', () => {
    expect(normalizeLandingAppearance({ palette: 'broken', wallpaper: 'custom-url' })).toEqual(DEFAULT_LANDING_APPEARANCE)
    expect(normalizeLandingAppearance('unexpected string')).toEqual(DEFAULT_LANDING_APPEARANCE)
  })
})
