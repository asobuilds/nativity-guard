/**
 * Cover image vertical focus anchor. Persisted server-side in
 * `users.cover_position` and read from the authenticated user's record,
 * so the choice follows the account across devices.
 *
 * Pure helpers only — no storage. The current value lives on the User
 * object returned by `/auth/profile`.
 */
export type CoverPosition = 'top' | 'center' | 'bottom'

/** Coerce an unknown value from the API into a valid CoverPosition. */
export function normalizeCoverPosition(value: unknown): CoverPosition {
  if (value === 'center' || value === 'bottom') return value
  return 'top'
}

/** CSS `object-position` value for the given focus. */
export function coverObjectPosition(position: CoverPosition): string {
  if (position === 'top') return '50% 0%'
  if (position === 'bottom') return '50% 100%'
  return '50% 50%'
}