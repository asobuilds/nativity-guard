import type { CSSProperties } from 'react'

/**
 * Cover photo focus values. Stored server-side on the user record so the
 * choice follows the account across devices.
 *
 *   x, y — object-position percentage, 0–100
 *   zoom — 100 = 1.0×, 300 = 3.0×
 */
export interface CoverAdjust {
  x: number
  y: number
  zoom: number
}

/**
 * Sensible default for a portrait selfie: horizontally centered, vertically
 * at 35% so the whole head is visible. The previous default (y=50) cropped
 * the top of the head off on typical portrait photos.
 */
export const DEFAULT_COVER_ADJUST: CoverAdjust = { x: 50, y: 35, zoom: 100 }

/**
 * Coerce unknown values from the API into a valid CoverAdjust.
 *
 * Upgrade path: any value matching a previous default that we know crops
 * faces badly (y=20 from an older release, y=50 from the immediate previous
 * release) with no other customization is replaced by the new default.
 * Users who moved the image or adjusted zoom keep their choice.
 */
export function normalizeCoverAdjust(
  input:
    | {
        coverPositionX?: unknown
        coverPositionY?: unknown
        coverZoom?: unknown
      }
    | undefined,
): CoverAdjust {
  const x = clampNumber(input?.coverPositionX, 0, 100, DEFAULT_COVER_ADJUST.x)
  const y = clampNumber(input?.coverPositionY, 0, 100, DEFAULT_COVER_ADJUST.y)
  const zoom = clampNumber(input?.coverZoom, 100, 300, DEFAULT_COVER_ADJUST.zoom)

  // Old defaults we shipped and now consider broken for face framing.
  const isOldBadDefault = x === 50 && (y === 20 || y === 50) && zoom === 100
  if (isOldBadDefault) return DEFAULT_COVER_ADJUST

  return { x, y, zoom }
}

function clampNumber(value: unknown, lo: number, hi: number, fallback: number): number {
  const n = typeof value === 'number' ? value : Number(value)
  if (!Number.isFinite(n)) return fallback
  return Math.min(hi, Math.max(lo, Math.round(n)))
}

/**
 * CSS for an <img> that fills its container, showing the region of the
 * source selected by `adjust`. Apply inside a wrapper with overflow hidden
 * and a fixed height.
 */
export function coverImageStyle(adjust: CoverAdjust): CSSProperties {
  return {
    width: '100%',
    height: '100%',
    objectFit: 'cover',
    objectPosition: `${adjust.x}% ${adjust.y}%`,
    transform: `scale(${adjust.zoom / 100})`,
    transformOrigin: `${adjust.x}% ${adjust.y}%`,
  }
}