import { describe, expect, it } from 'vitest'
import { validateAgencyObserverForm, type AgencyObserverFormValues } from './agency'

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
