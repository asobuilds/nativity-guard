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

export const DEFAULT_COVER_ADJUST: CoverAdjust = { x: 50, y: 20, zoom: 100 }

/** Coerce unknown values from the API into a valid CoverAdjust. */
export function normalizeCoverAdjust(input: {
  coverPositionX?: unknown
  coverPositionY?: unknown
  coverZoom?: unknown
} | undefined): CoverAdjust {
  const x = clampNumber(input?.coverPositionX, 0, 100, DEFAULT_COVER_ADJUST.x)
  const y = clampNumber(input?.coverPositionY, 0, 100, DEFAULT_COVER_ADJUST.y)
  const zoom = clampNumber(input?.coverZoom, 100, 300, DEFAULT_COVER_ADJUST.zoom)
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