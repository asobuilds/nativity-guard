import { describe, expect, it } from 'vitest'
import { DEFAULT_LANDING_APPEARANCE, normalizeLandingAppearance } from './landingAppearance'
import { validateWallpaperFile } from './useLandingAppearance'

describe('Landing appearance settings', () => {
  it('defaults to the approved City Gate forest design', () => {
    expect(normalizeLandingAppearance(null)).toEqual(DEFAULT_LANDING_APPEARANCE)
  })
  it('accepts defined palette and wallpaper choices', () => {
    expect(normalizeLandingAppearance({ palette: 'ocean', wallpaper: 'plain' })).toEqual({
      palette: 'ocean', wallpaper: 'plain',
    })
  })
  it('recognizes a saved custom wallpaper preference', () => {
    expect(normalizeLandingAppearance({ palette: 'forest', wallpaper: 'custom' })).toEqual({
      palette: 'forest', wallpaper: 'custom',
    })
  })
  it('validates local photo type and upload size', () => {
    expect(validateWallpaperFile({ type: 'image/jpeg', size: 100_000 })).toBeNull()
    expect(validateWallpaperFile({ type: 'image/png', size: 8 * 1024 * 1024 })).toBeNull()
    expect(validateWallpaperFile({ type: 'image/svg+xml', size: 50_000 })).toMatch(/JPG/)
    expect(validateWallpaperFile({ type: 'image/jpeg', size: 8 * 1024 * 1024 + 1 })).toMatch(/8 MB/)
    expect(validateWallpaperFile({ type: 'image/webp', size: 0 })).toMatch(/8 MB/)
  })
  it('rejects invalid or unexpected stored settings', () => {
    expect(normalizeLandingAppearance({ palette: 'broken', wallpaper: 'custom-url' })).toEqual(DEFAULT_LANDING_APPEARANCE)
    expect(normalizeLandingAppearance('unexpected string')).toEqual(DEFAULT_LANDING_APPEARANCE)
  })
})
