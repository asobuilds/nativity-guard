export type LandingPalette = 'forest' | 'midnight' | 'ocean' | 'sand'
export type LandingWallpaper = 'gate' | 'illustration' | 'plain'

export interface LandingAppearance {
  palette: LandingPalette
  wallpaper: LandingWallpaper
}

const STORAGE_KEY = 'nativity-guard-landing-appearance-v1'

export const DEFAULT_LANDING_APPEARANCE: LandingAppearance = {
  palette: 'forest',
  wallpaper: 'gate',
}

export function normalizeLandingAppearance(value: unknown): LandingAppearance {
  const input = value && typeof value === 'object' ? value as Partial<LandingAppearance> : {}
  const palette = input.palette
  const wallpaper = input.wallpaper
  return {
    palette: palette === 'forest' || palette === 'midnight' || palette === 'ocean' || palette === 'sand' ? palette : 'forest',
    wallpaper: wallpaper === 'gate' || wallpaper === 'illustration' || wallpaper === 'plain' ? wallpaper : 'gate',
  }
}

export function readLandingAppearance(): LandingAppearance {
  try {
    return normalizeLandingAppearance(JSON.parse(window.localStorage.getItem(STORAGE_KEY) ?? '{}'))
  } catch {
    return { ...DEFAULT_LANDING_APPEARANCE }
  }
}

export function saveLandingAppearance(value: LandingAppearance): LandingAppearance {
  const normalized = normalizeLandingAppearance(value)
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(normalized))
  } catch {
    // Private browsing or restricted storage: changes still work for this visit.
  }
  return normalized
}
