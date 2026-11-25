import { describe, expect, it } from 'vitest'
import { getEligibleAgencyUnits, validateAgencyObserverForm, type AgencyObserverFormValues, type AgencyUnitCandidate } from './agency'

const valid: AgencyObserverFormValues = {
  firstName: 'Ada',
  lastName: 'Okafor',
  email: 'ada@example.test',
  phone: '08012345678',
  password: 'temporary-123',
}

describe('validateAgencyObserverForm', () => {
  it('accepts a complete observer account', () => {
    expect(validateAgencyObserverForm(valid)).toBeNull()
  })

  it('requires both names', () => {
    expect(validateAgencyObserverForm({ ...valid, firstName: ' ' })).toContain('first and last')
  })

  it('rejects an invalid email address', () => {
    expect(validateAgencyObserverForm({ ...valid, email: 'not-an-email' })).toContain('valid email')
  })

  it('requires a phone number', () => {
    expect(validateAgencyObserverForm({ ...valid, phone: ' ' })).toContain('phone')
  })

  it('requires a temporary password of at least six characters', () => {
    expect(validateAgencyObserverForm({ ...valid, password: '12345' })).toContain('at least 6')
  })
})



const candidates: AgencyUnitCandidate[] = [
  { id: 'matching', name: 'Ikeja Police', type: 'POLICE', state: 'Lagos', lga: 'Ikeja', status: 'active', isVerified: true },
  { id: 'wrong-lga', name: 'Surulere Police', type: 'police', state: 'Lagos', lga: 'Surulere', status: 'active', isVerified: true },
  { id: 'wrong-type', name: 'Ikeja Fire', type: 'fire', state: 'Lagos', lga: 'Ikeja', status: 'active', isVerified: true },
  { id: 'pending', name: 'Pending Police', type: 'police', state: 'Lagos', lga: 'Ikeja', status: 'pending', isVerified: false },
]

describe('getEligibleAgencyUnits', () => {
  const scope = { agencyType: 'police', state: 'Lagos', lga: 'Ikeja' }

  it('matches agency type and exact jurisdiction case-insensitively', () => {
    expect(getEligibleAgencyUnits(candidates, scope, []).map((unit) => unit.id)).toEqual(['matching'])
  })

  it('excludes units already linked to another division', () => {
    expect(getEligibleAgencyUnits(candidates, scope, ['matching'])).toEqual([])
  })

  it('excludes units that are not active and verified', () => {
    expect(getEligibleAgencyUnits(candidates, scope, []).some((unit) => unit.id === 'pending')).toBe(false)
  })

  it('returns no options until a division is selected', () => {
    expect(getEligibleAgencyUnits(candidates, undefined, [])).toEqual([])
  })
})
