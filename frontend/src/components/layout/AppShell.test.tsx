import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { MOBILE_PRIMARY_ROUTES } from '@/lib/mobileNav'
import type { Role } from '@/types/api'
import {
  AppShell,
  NAV,
  NAV_GROUPS,
  findActiveGroupIds,
  flattenRoleNav,
  isNavItemActive,
} from './AppShell'

vi.mock('@/auth/AuthContext', () => ({
  useAuth: () => ({
    user: {
      id: 'user-1',
      firstName: 'Ada',
      lastName: 'Okafor',
      email: 'ada@example.com',
      role: 'citizen',
    },
    role: 'citizen' as Role,
    rawRole: 'citizen',
    status: 'authenticated',
    login: vi.fn(),
    logout: vi.fn(),
    refreshProfile: vi.fn(),
  }),
}))

vi.mock('./NotificationBell', () => ({
  NotificationBell: () => <button type="button" aria-label="Notifications" />,
}))

const ALL_ROLES: Role[] = [
  'citizen',
  'officer',
  'unit_admin',
  'super_admin',
  'agency_observer',
  'agency_division_admin',
]

describe('AppShell navigation configuration', () => {
  it('has no duplicate routes for any role in NAV or NAV_GROUPS', () => {
    for (const role of ALL_ROLES) {
      const items = NAV[role]
      const routes = items.map((item) => item.to)
      expect(new Set(routes).size, `duplicate route found for role ${role}`).toBe(routes.length)
      expect(items).toEqual(flattenRoleNav(NAV_GROUPS[role]))
    }
  })

  it('removes duplicate /community/chat and /messages entries from citizen navigation', () => {
    const citizenRoutes = NAV.citizen.map((item) => item.to)
    expect(citizenRoutes.filter((route) => route === '/community/chat')).toHaveLength(1)
    expect(citizenRoutes.filter((route) => route === '/messages')).toHaveLength(1)
  })

  it('resolves every MOBILE_PRIMARY_ROUTES entry for every role', () => {
    for (const role of ALL_ROLES) {
      const routes = new Set(NAV[role].map((item) => item.to))
      for (const primaryRoute of MOBILE_PRIMARY_ROUTES[role]) {
        expect(routes.has(primaryRoute), `${primaryRoute} missing for ${role}`).toBe(true)
      }
    }
  })

  it('organizes citizen navigation into pinned items and the expected collapsible groups', () => {
    expect(NAV_GROUPS.citizen.pinned.map((item) => item.label)).toEqual(['Home', 'Emergency SOS'])
    expect(NAV_GROUPS.citizen.groups.map((group) => group.label)).toEqual([
      'Safety & Reporting',
      'Community & Messaging',
      'Units & Bookings',
      'Support & Appeals',
    ])
  })

  it('keeps agency roles flat without collapsible groups', () => {
    expect(NAV_GROUPS.agency_observer.groups).toEqual([])
    expect(NAV_GROUPS.agency_division_admin.groups).toEqual([])
    expect(NAV_GROUPS.agency_observer.pinned).toHaveLength(3)
  })
})

describe('AppShell active-route and group auto-expansion logic', () => {
  it('matches root and nested routes accurately', () => {
    expect(isNavItemActive('/', '/')).toBe(true)
    expect(isNavItemActive('/', '/report')).toBe(false)
    expect(isNavItemActive('/bookings', '/bookings')).toBe(true)
    expect(isNavItemActive('/bookings', '/bookings/new')).toBe(true)
    expect(isNavItemActive('/bookings', '/alerts')).toBe(false)
  })

  it('identifies active groups to auto-expand on pathname change', () => {
    expect(findActiveGroupIds(NAV_GROUPS.citizen.groups, '/')).toEqual([])
    expect(findActiveGroupIds(NAV_GROUPS.citizen.groups, '/community/chat')).toEqual([
      'citizen-community',
    ])
    expect(findActiveGroupIds(NAV_GROUPS.citizen.groups, '/bookings/new')).toEqual([
      'citizen-units',
    ])
    expect(findActiveGroupIds(NAV_GROUPS.super_admin.groups, '/admin/cases/42')).toEqual([
      'super-cases',
    ])
  })

  it('renders collapsible groups with aria-expanded and auto-expands the active group', () => {
    const html = renderToStaticMarkup(
      <MemoryRouter initialEntries={['/bookings/new']}>
        <AppShell>
          <div>Content</div>
        </AppShell>
      </MemoryRouter>,
    )

    expect(html).toContain('aria-controls="nav-group-citizen-units"')
    expect(html).toContain('id="nav-group-citizen-units"')
    expect(html).toContain('aria-expanded="true"')
    expect(html).toContain('aria-expanded="false"')
    expect(html).toContain('aria-label="Mobile primary navigation"')
    expect(html).toContain('aria-label="Close navigation"')
    expect(html).toContain('aria-label="Open navigation"')
  })
})
