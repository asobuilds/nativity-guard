import type { Role } from '@/types/api'

/**
 * Keep mobile bottom navigation small enough to fit narrow iPhones.
 * All other role-appropriate destinations remain in the existing full menu.
 * SOS already has a dedicated emergency action above the bottom bar.
 */
export const MOBILE_PRIMARY_ROUTES: Record<Role, readonly string[]> = {
  citizen: ['/', '/report', '/map'],
  officer: ['/officer/queue', '/map', '/assistant'],
  unit_admin: ['/admin/cases', '/admin/overview', '/map'],
  super_admin: ['/super/overview', '/super/units', '/map'],
  agency_observer: ['/observer/dashboard', '/map', '/units'],
  agency_division_admin: ['/observer/dashboard', '/map', '/units'],
}
