/**
 * Registration rules, as pure functions.
 *
 * These mirror what `handlers/auth_handler.go` actually enforces — the
 * `binding:"required"` tags on `Register`, and `validateDOB`'s age gate — so the
 * form refuses for the same reasons the server would, and in the user's words
 * rather than Gin's. The date arithmetic is pure and takes `today` as an argument
 * so the age boundary is testable rather than dependent on when the suite runs.
 */

/** The backend refuses registration under this age (`validateDOB`). */
export const MIN_SIGNUP_AGE = 16

/** `Register` binds `password` with `min=6`. */
export const MIN_PASSWORD_LENGTH = 6

export interface SignupDraft {
  firstName: string
  lastName: string
  email: string
  phone: string
  dateOfBirth: string
  password: string
}

export type SignupField = keyof SignupDraft

export type SignupMode = 'password' | 'google'

export type SignupStep = 1 | 2

export type SignupErrors = Partial<Record<SignupField, string>>

export const EMPTY_SIGNUP: SignupDraft = {
  firstName: '',
  lastName: '',
  email: '',
  phone: '',
  dateOfBirth: '',
  password: '',
}

/** The order fields are presented and first checked in. */
export const SIGNUP_FIELDS: SignupField[] = [
  'email',
  'phone',
  'firstName',
  'lastName',
  'dateOfBirth',
  'password',
]

export const ACCOUNT_SIGNUP_FIELDS: SignupField[] = ['email', 'firstName', 'lastName', 'password']

export const PROFILE_SIGNUP_FIELDS: SignupField[] = ['phone', 'dateOfBirth']

export interface PasswordStrength {
  score: 0 | 1 | 2 | 3 | 4
  label: 'Add a password' | 'Weak' | 'Fair' | 'Good' | 'Strong'
  checks: {
    minimum: boolean
    long: boolean
    mixedCase: boolean
    number: boolean
    symbol: boolean
  }
}

/**
 * Completed years between an ISO `YYYY-MM-DD` date and `today`, or `null` when the
 * string is not a real calendar date.
 *
 * The round-trip check matters: `new Date('2026-02-31')` silently rolls over to
 * March 3rd, so a bare parse would accept an impossible date of birth and compute
 * an age from a day that does not exist.
 */
export function ageOn(iso: string, today: Date = new Date()): number | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(iso.trim())
  if (!match) return null

  const year = Number(match[1])
  const month = Number(match[2])
  const day = Number(match[3])

  const dob = new Date(Date.UTC(year, month - 1, day))
  if (
    dob.getUTCFullYear() !== year ||
    dob.getUTCMonth() !== month - 1 ||
    dob.getUTCDate() !== day
  ) {
    return null
  }

  let age = today.getUTCFullYear() - year
  const hadBirthdayThisYear =
    today.getUTCMonth() > month - 1 ||
    (today.getUTCMonth() === month - 1 && today.getUTCDate() >= day)
  if (!hadBirthdayThisYear) age -= 1
  return age
}

/**
 * The latest date of birth that still clears the age floor — the `max` for the
 * date input, so the picker stops a plainly ineligible date before it is typed.
 */
export function latestEligibleDob(today: Date = new Date()): string {
  const cutoff = new Date(
    Date.UTC(today.getUTCFullYear() - MIN_SIGNUP_AGE, today.getUTCMonth(), today.getUTCDate()),
  )
  return cutoff.toISOString().slice(0, 10)
}

export function passwordStrength(password: string): PasswordStrength {
  const checks = {
    minimum: password.length >= MIN_PASSWORD_LENGTH,
    long: password.length >= 10,
    mixedCase: /[a-z]/.test(password) && /[A-Z]/.test(password),
    number: /\d/.test(password),
    symbol: /[^A-Za-z0-9]/.test(password),
  }
  if (!password) return { score: 0, label: 'Add a password', checks }

  const met = Object.values(checks).filter(Boolean).length
  const score = Math.min(4, Math.max(1, met)) as 1 | 2 | 3 | 4
  const labels: Record<1 | 2 | 3 | 4, PasswordStrength['label']> = {
    1: 'Weak',
    2: 'Fair',
    3: 'Good',
    4: 'Strong',
  }
  return { score, label: labels[score], checks }
}

export function fieldsForSignupStep(step: SignupStep, mode: SignupMode = 'password'): SignupField[] {
  if (step === 2) return PROFILE_SIGNUP_FIELDS
  return mode === 'google'
    ? ACCOUNT_SIGNUP_FIELDS.filter((field) => field !== 'password')
    : ACCOUNT_SIGNUP_FIELDS
}

export function signupStepComplete(
  draft: SignupDraft,
  step: SignupStep,
  mode: SignupMode = 'password',
  today: Date = new Date(),
): boolean {
  const errors = validateSignup(draft, today, mode)
  return fieldsForSignupStep(step, mode).every((field) => !errors[field])
}

export function validateSignup(
  draft: SignupDraft,
  today: Date = new Date(),
  mode: SignupMode = 'password',
): SignupErrors {
  const errors: SignupErrors = {}

  if (!draft.firstName.trim()) errors.firstName = 'Enter your first name.'
  if (!draft.lastName.trim()) errors.lastName = 'Enter your last name.'

  const email = draft.email.trim()
  if (!email) errors.email = 'Enter your email address.'
  else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    errors.email = 'That does not look like an email address.'
  }

  const phone = draft.phone.trim()
  if (!phone) errors.phone = 'Enter your phone number.'
  else if (!/^(?:\+?234|0)[789][01]\d{8}$/.test(phone)) {
    errors.phone = 'Enter a valid Nigerian phone number.'
  }

  if (!draft.dateOfBirth.trim()) {
    errors.dateOfBirth = 'Enter your date of birth.'
  } else {
    const age = ageOn(draft.dateOfBirth, today)
    if (age === null) errors.dateOfBirth = 'Enter a real date.'
    else if (age < 0) errors.dateOfBirth = 'That date is in the future.'
    else if (age > 120) errors.dateOfBirth = 'That date does not look right.'
    else if (age < MIN_SIGNUP_AGE) {
      errors.dateOfBirth = `You must be ${MIN_SIGNUP_AGE} or older to register.`
    }
  }

  if (mode === 'password') {
    if (!draft.password) errors.password = 'Choose a password.'
    else if (draft.password.length < MIN_PASSWORD_LENGTH) {
      errors.password = `Use at least ${MIN_PASSWORD_LENGTH} characters.`
    }
  }

  return errors
}
