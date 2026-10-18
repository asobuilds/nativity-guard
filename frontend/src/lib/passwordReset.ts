
export const MIN_RESET_PASSWORD_LENGTH = 8
export const RESET_CODE_TTL_MINUTES = 60

/**
 * Normalise a phone number or digit-only value.
 * Kept for existing phone-number matching in the mock handlers.
 */
export function normaliseResetCode(value: string): string {
  return value.replace(/\D/g, '')
}

/**
 * Validate an email address for password-reset requests.
 */
export function validateEmail(value: string): string | undefined {
  const email = value.trim()

  if (!email) {
    return 'Enter your email address.'
  }

  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    return 'Enter a valid email address.'
  }

  return undefined
}

/**
 * Validate the new password used by the password-reset endpoint.
 */
export function validateNewPassword(value: string): string | undefined {
  if (!value) {
    return 'Enter a new password.'
  }

  if (value.length < MIN_RESET_PASSWORD_LENGTH) {
    return `Password must be at least ${MIN_RESET_PASSWORD_LENGTH} characters.`
  }

  return undefined
}

/**
 * Mask an email address or phone number for display.
 */
export function maskIdentifier(identifier: string): string {
  const value = identifier.trim()

  if (value.includes('@')) {
    const [localPart, domain] = value.split('@')

    if (!localPart || !domain) {
      return value
    }

    const visible = localPart.slice(0, Math.min(2, localPart.length))
    const hiddenLength = Math.max(3, localPart.length - visible.length)

    return `${visible}${'•'.repeat(hiddenLength)}@${domain}`
  }

  const digits = normaliseResetCode(value)

  if (digits.length < 5) {
    return value
  }

  return `${digits.slice(0, 3)}${'•'.repeat(Math.max(3, digits.length - 5))}${digits.slice(-2)}`
}
