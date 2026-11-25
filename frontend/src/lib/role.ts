import type { Role } from '@/types/api'

/**
 * The role vocabulary.
 *
 * Declared as one list, in the same shape as `CASE_STATUS` in `types/api.ts`, so a
 * contract correction is a single edit rather than a hunt. These are the strings the
 * backend actually compares against — underscore, verified 2026-09-24 across
 * `middleware/permission_middleware.go` and `handlers/*_handler.go`.
 *
 * `head_admin` is deliberately absent: it is `models.UnitMembership.IsHeadAdmin`, a
 * boolean on a membership, not a role a user account carries.
 */
export const ROLES = [
  'citizen',
  'officer',
  'unit_admin',
  'super_admin',
  'agency_division_admin',
  'agency_observer',
] as const

export function isKnownRole(value: unknown): value is Role {
  return typeof value === 'string' && (ROLES as readonly string[]).includes(value)
}

/**
 * Normalise a role the API returned onto the canonical union.
 *
 * A role the frontend could not name used to fall through `homePathForRole`'s
 * `default` to `'/'`, which `HomeRoute` sent straight back — an infinite
 * self-redirect that React killed with "Maximum update depth exceeded", and with no
 * error boundary above it the app simply went black. Accepting the known spelling
 * variants costs one line and closes that whole class of outage; anything else
 * returns `null` so the caller can *show* the problem instead of guessing at which
 * console to open.
 */
export function normaliseRole(raw: string | null | undefined): Role | null {
  if (typeof raw !== 'string') return null
  const canonical = raw.trim().toLowerCase().replace(/-/g, '_')
  return isKnownRole(canonical) ? canonical : null
}
