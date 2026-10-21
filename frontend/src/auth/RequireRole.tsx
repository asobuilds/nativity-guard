import { Navigate, useLocation } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useAuth } from './AuthContext'
import { UnrecognisedRoleScreen } from './UnrecognisedRoleScreen'
import type { Role } from '@/types/api'
import { FullPageSpinner } from '@/components/ui/States'

/**
 * Route guard. Redirects anonymous users to login (preserving the intended
 * destination) and authenticated users who lack the required role to their own
 * home. Role is the coarse gate; server-side unit/resource checks still apply.
 */
export function RequireRole({
  roles,
  children,
}: {
  roles: Role[]
  children: ReactNode
}) {
  const { status, role, rawRole, user, logout } = useAuth()
  const location = useLocation()

  if (status === 'loading') return <FullPageSpinner label="Restoring session…" />
  if (status === 'anonymous') {
    return <Navigate to="/auth/login" replace state={{ from: location.pathname }} />
  }
  if (user?.accountAccessLevel === 'limited') {
    return <Navigate to="/auth/complete-profile" replace />
  }

  // Signed in, but the role is not one this build can name. Show it and stop —
  // never guess a console, never redirect. The old `default: '/'` here is what
  // sent the app to `/`, had `HomeRoute` send it back, and blanked the screen.
  if (!role) return <UnrecognisedRoleScreen rawRole={rawRole} onSignOut={logout} />

  if (!roles.includes(role)) {
    const home = homePathForRole(role)
    if (!home) return <UnrecognisedRoleScreen rawRole={rawRole} onSignOut={logout} />
    return <Navigate to={home} replace />
  }

  return <>{children}</>
}

/**
 * The landing route for a role — or `null`, which is the point of the signature.
 *
 * Answering `'/'` for an unknown role (the previous `default`) let this function
 * claim a home for a role that has none, and `'/'` is a *real* destination that
 * bounces every non-citizen straight back here. Returning `null` forces each caller
 * to decide what to show instead, which makes the loop unrepresentable.
 */
export function homePathForRole(role: Role | null | undefined): string | null {
  switch (role) {
    case 'citizen':
      return '/'
    case 'officer':
      return '/officer/queue'
    case 'unit_admin':
      // Land on the screen that works today, not the one that is still a stub.
      return '/admin/cases'
    case 'super_admin':
      return '/super/overview'
    default:
      return null
  }
}
