import { describe, expect, it } from 'vitest'
import {
  EMPTY_SIGNUP,
  MIN_PASSWORD_LENGTH,
  ageOn,
  fieldsForSignupStep,
  latestEligibleDob,
  passwordStrength,
  signupStepComplete,
  validateSignup,
  type SignupDraft,
} from './signup'

/**
 * Registration validation guards the age gate the backend enforces, so the two
 * boundaries that matter are pinned here: exactly-16 is eligible, one day short is
 * not, and an impossible calendar date is never treated as valid. `today` is
 * injected throughout so these do not drift as the clock moves.
 */

const TODAY = new Date(Date.UTC(2026, 8, 24)) // 2026-09-24

const valid: SignupDraft = {
  firstName: 'Amaka',
  lastName: 'Obi',
  email: 'amaka@example.com',
  phone: '08167135998',
  dateOfBirth: '1995-04-02',
  password: 'correct horse',
}

describe('ageOn', () => {
  it('counts completed years, not calendar-year differences', () => {
    expect(ageOn('1995-04-02', TODAY)).toBe(31)
    // Birthday not yet reached this year.
    expect(ageOn('1995-12-31', TODAY)).toBe(30)
  })

  it('treats a birthday falling today as reached', () => {
    expect(ageOn('2010-09-24', TODAY)).toBe(16)
  })

  it('returns null for impossible calendar dates', () => {
    // `new Date('2026-02-31')` would roll over to March 3rd if parsed naively.
    expect(ageOn('2026-02-31', TODAY)).toBeNull()
    expect(ageOn('1995-13-01', TODAY)).toBeNull()
    expect(ageOn('not-a-date', TODAY)).toBeNull()
    expect(ageOn('', TODAY)).toBeNull()
  })

  it('reports a negative age for a future date rather than clamping it', () => {
    expect(ageOn('2030-01-01', TODAY)).toBeLessThan(0)
  })
})

describe('latestEligibleDob', () => {
  it('is the date exactly MIN_SIGNUP_AGE years ago', () => {
    expect(latestEligibleDob(TODAY)).toBe('2010-09-24')
  })
})

describe('validateSignup', () => {
  it('accepts a complete, eligible draft', () => {
    expect(validateSignup(valid, TODAY)).toEqual({})
  })

  it('names every missing field on an empty draft', () => {
    const errors = validateSignup(EMPTY_SIGNUP, TODAY)
    expect(Object.keys(errors).sort()).toEqual(
      ['dateOfBirth', 'email', 'firstName', 'lastName', 'password', 'phone'].sort(),
    )
  })

  it('refuses an applicant one day short of the age floor', () => {
    // 2010-09-25 → 15 on 2026-09-24, the day before their 16th birthday.
    const errors = validateSignup({ ...valid, dateOfBirth: '2010-09-25' }, TODAY)
    expect(errors.dateOfBirth).toMatch(/16 or older/)
  })

  it('accepts an applicant on their sixteenth birthday', () => {
    expect(validateSignup({ ...valid, dateOfBirth: '2010-09-24' }, TODAY)).toEqual({})
  })

  it('refuses a future or impossible date of birth', () => {
    expect(validateSignup({ ...valid, dateOfBirth: '2030-01-01' }, TODAY).dateOfBirth).toMatch(
      /future/,
    )
    expect(
      validateSignup({ ...valid, dateOfBirth: '2026-02-31' }, TODAY).dateOfBirth,
    ).toMatch(/real date/)
  })

  it('holds the password to the length the endpoint binds', () => {
    const short = 'a'.repeat(MIN_PASSWORD_LENGTH - 1)
    expect(validateSignup({ ...valid, password: short }, TODAY).password).toMatch(/at least 6/)
    expect(
      validateSignup({ ...valid, password: 'a'.repeat(MIN_PASSWORD_LENGTH) }, TODAY),
    ).toEqual({})
  })

  it('rejects a malformed email but accepts a plain one', () => {
    expect(validateSignup({ ...valid, email: 'amaka@' }, TODAY).email).toBeDefined()
    expect(validateSignup({ ...valid, email: 'amaka@example.com' }, TODAY).email).toBeUndefined()
  })

  it('treats whitespace-only names as missing', () => {
    const errors = validateSignup({ ...valid, firstName: '   ', lastName: '\t' }, TODAY)
    expect(errors.firstName).toBeDefined()
    expect(errors.lastName).toBeDefined()
  })

  it('requires a Nigerian mobile number in local or international form', () => {
    expect(validateSignup({ ...valid, phone: '08031234567' }, TODAY).phone).toBeUndefined()
    expect(validateSignup({ ...valid, phone: '+2348031234567' }, TODAY).phone).toBeUndefined()
    expect(validateSignup({ ...valid, phone: '2348031234567' }, TODAY).phone).toBeUndefined()
    expect(validateSignup({ ...valid, phone: '+2338031234567' }, TODAY).phone).toMatch(/Nigerian/)
    expect(validateSignup({ ...valid, phone: '8031234567' }, TODAY).phone).toMatch(/Nigerian/)
  })

  it('does not require a local password after a verified Google identity', () => {
    expect(validateSignup({ ...valid, password: '' }, TODAY, 'google')).toEqual({})
    expect(validateSignup({ ...valid, password: '' }, TODAY, 'password').password).toBeDefined()
  })
})

describe('progressive signup steps', () => {
  it('separates account identity from profile and compliance details', () => {
    expect(fieldsForSignupStep(1)).toEqual(['email', 'firstName', 'lastName', 'password'])
    expect(fieldsForSignupStep(1, 'google')).toEqual(['email', 'firstName', 'lastName'])
    expect(fieldsForSignupStep(2)).toEqual(['phone', 'dateOfBirth'])
  })

  it('reports each step independently', () => {
    expect(signupStepComplete(valid, 1, 'password', TODAY)).toBe(true)
    expect(signupStepComplete(valid, 2, 'password', TODAY)).toBe(true)
    expect(signupStepComplete({ ...valid, phone: '' }, 1, 'password', TODAY)).toBe(true)
    expect(signupStepComplete({ ...valid, phone: '' }, 2, 'password', TODAY)).toBe(false)
    expect(signupStepComplete({ ...valid, password: '' }, 1, 'google', TODAY)).toBe(true)
  })
})

describe('passwordStrength', () => {
  it('provides progressive, human-readable strength feedback', () => {
    expect(passwordStrength('')).toMatchObject({ score: 0, label: 'Add a password' })
    expect(passwordStrength('short')).toMatchObject({ score: 1, label: 'Weak' })
    expect(passwordStrength('longerpass')).toMatchObject({ score: 2, label: 'Fair' })
    expect(passwordStrength('LongerPass1!')).toMatchObject({ score: 4, label: 'Strong' })
  })

  it('exposes individual checks for live meter details', () => {
    expect(passwordStrength('LongerPass1!').checks).toEqual({
      minimum: true,
      long: true,
      mixedCase: true,
      number: true,
      symbol: true,
    })
  })
})
