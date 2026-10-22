import { describe, expect, it } from 'vitest'
import { formatDistance, formatDuration } from './format'

describe('formatDistance', () => {
  it('uses metres below a kilometre and kilometres above', () => {
    expect(formatDistance(0)).toBe('0 m')
    expect(formatDistance(850)).toBe('850 m')
    expect(formatDistance(1000)).toBe('1.0 km')
    expect(formatDistance(12500)).toBe('12.5 km')
  })

  it('returns a dash for missing or invalid values', () => {
    expect(formatDistance(undefined)).toBe('—')
    expect(formatDistance(Number.NaN)).toBe('—')
    expect(formatDistance(-1)).toBe('—')
  })
})

describe('formatDuration', () => {
  it('formats minutes and hours', () => {
    expect(formatDuration(30)).toBe('less than a minute')
    expect(formatDuration(8 * 60)).toBe('8 min')
    expect(formatDuration(60 * 60)).toBe('1 h')
    expect(formatDuration(72 * 60)).toBe('1 h 12 min')
  })

  it('returns a dash for missing or invalid values', () => {
    expect(formatDuration(undefined)).toBe('—')
    expect(formatDuration(-1)).toBe('—')
  })
})
