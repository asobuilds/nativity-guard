import { describe, expect, it } from 'vitest'
import { MOBILE_PRIMARY_ROUTES } from './mobileNav'

describe('Mobile primary navigation', () => {
  it('fits three clear destinations plus a More button on every role', () => {
    for (const routes of Object.values(MOBILE_PRIMARY_ROUTES)) {
      expect(routes).toHaveLength(3)
      expect(new Set(routes).size).toBe(3)
      expect(routes).not.toContain('/sos')
    }
  })
  it('keeps citizen reporting and map access one tap away', () => {
    expect(MOBILE_PRIMARY_ROUTES.citizen).toContain('/report')
    expect(MOBILE_PRIMARY_ROUTES.citizen).toContain('/map')
  })
})
