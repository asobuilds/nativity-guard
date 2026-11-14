import { describe, expect, it } from 'vitest'
import { publishedUnitCount } from './publicLandingStats'

describe('published unit statistics', () => {
  const units = [
    { status: 'active', isVerified: true },
    { status: 'active', isVerified: false },
    { status: 'inactive', isVerified: true },
    { status: 'active', isVerified: true },
  ] as const

  it('counts only active units with platform verification', () => {
    expect(publishedUnitCount(units, false, false)).toEqual({
      value: '2', label: 'Active platform-verified units',
    })
  })

  it('shows a real zero when the API succeeds with no published units', () => {
    expect(publishedUnitCount([], false, false)).toEqual({
      value: '0', label: 'Active platform-verified units',
    })
  })

  it('does not turn a failed or loading API response into a zero count', () => {
    expect(publishedUnitCount(undefined, true, false).label).toBe('Loading unit count')
    expect(publishedUnitCount([], false, true)).toEqual({
      value: '—', label: 'Unit count unavailable',
    })
  })
})
