import { useEffect, useState, type ReactNode } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import {
  BarChart3,
  Bot,
  FileText,
  FolderKanban,
  LayoutDashboard,
  LogOut,
  Map as MapIcon,
  Megaphone,
  Menu,
  Shield,
  ShieldAlert,
  UserCircle,
  Users,
  X,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { useAuth } from '@/auth/AuthContext'
import { fullName, initials } from '@/lib/format'
import { Logo } from '@/components/brand/Logo'
import { NotificationBell } from './NotificationBell'
import type { Role } from '@/types/api'

interface NavItem {
  to: string
  label: string
  icon: ReactNode
  /** Not built yet — rendered disabled with an explanation, never a dead link. */
  soon?: boolean
}

const NAV: Record<Role, NavItem[]> = {
  citizen: [
    { to: '/', label: 'Home', icon: <LayoutDashboard className="size-4" /> },
    { to: '/sos', label: 'Emergency SOS', icon: <ShieldAlert className="size-4" /> },
    { to: '/report', label: 'Report', icon: <FileText className="size-4" /> },
    { to: '/alerts', label: 'Alerts', icon: <Megaphone className="size-4" /> },
    { to: '/community', label: 'Community', icon: <Users className="size-4" /> },
    { to: '/assistant', label: 'Safety assistant', icon: <Bot className="size-4" /> },
    { to: '/appeals', label: 'Appeals', icon: <FileText className="size-4" /> },
    { to: '/map', label: 'Safety map', icon: <MapIcon className="size-4" /> },
  ],
  officer: [
    { to: '/assistant', label: 'Safety assistant', icon: <Bot className="size-4" /> },
    { to: '/officer/queue', label: 'Case queue', icon: <FolderKanban className="size-4" /> },
    { to: '/map', label: 'Operations map', icon: <MapIcon className="size-4" /> },
    { to: '/officer/comms', label: 'Comms', icon: <Users className="size-4" />, soon: true },
    { to: '/appeals', label: 'Appeals', icon: <FileText className="size-4" /> },
  ],
  unit_admin: [
    { to: '/assistant', label: 'Safety assistant', icon: <Bot className="size-4" /> },
    { to: '/admin/cases', label: 'Case review', icon: <FolderKanban className="size-4" /> },
    { to: '/admin/community', label: 'Community', icon: <Users className="size-4" /> },
    { to: '/admin/transfers', label: 'Transfers', icon: <FileText className="size-4" /> },
    { to: '/admin/overview', label: 'Overview', icon: <LayoutDashboard className="size-4" /> },
    { to: '/admin/officers', label: 'Units', icon: <Users className="size-4" /> },
    { to: '/admin/analytics', label: 'Analytics', icon: <BarChart3 className="size-4" /> },
    { to: '/admin/finance', label: 'Finance demo', icon: <FileText className="size-4" /> },
    { to: '/admin/settings', label: 'Unit settings', icon: <Shield className="size-4" /> },
    { to: '/admin/unit-policy', label: 'Governance policy', icon: <Shield className="size-4" /> },
    { to: '/admin/governance-audit', label: 'Governance overview', icon: <Shield className="size-4" /> },
    { to: '/appeals', label: 'Appeals', icon: <FileText className="size-4" /> },
    { to: '/map', label: 'Operations map', icon: <MapIcon className="size-4" /> },
  ],
  // A super admin sees the citizen actions, the officer queue and the platform
  // consoles — every surface a role below can reach, plus the admin routes.
  // Icon set is whatever the blocks above already use; `Megaphone` stands in for
  // `/notifications` because `Bell` is not imported here.
  super_admin: [
    { to: '/assistant', label: 'Safety assistant', icon: <Bot className="size-4" /> },
    { to: '/', label: 'Home', icon: <LayoutDashboard className="size-4" /> },
    { to: '/sos', label: 'Emergency SOS', icon: <ShieldAlert className="size-4" /> },
    { to: '/report', label: 'Report', icon: <FileText className="size-4" /> },
    { to: '/alerts', label: 'Alerts', icon: <Megaphone className="size-4" /> },
    { to: '/community', label: 'Community', icon: <Users className="size-4" /> },
    { to: '/admin/community', label: 'Community admin', icon: <Users className="size-4" /> },
    { to: '/map', label: 'Safety map', icon: <MapIcon className="size-4" /> },
    { to: '/officer/queue', label: 'Case queue', icon: <FolderKanban className="size-4" /> },
    { to: '/notifications', label: 'Notifications', icon: <Megaphone className="size-4" /> },

    { to: '/admin/cases', label: 'Case review', icon: <FolderKanban className="size-4" /> },
{ to: '/admin/transfers', label: 'Transfers', icon: <FileText className="size-4" /> },
    { to: '/admin/officers', label: 'Unit rosters', icon: <Users className="size-4" /> },
    { to: '/super/overview', label: 'Governance', icon: <ShieldAlert className="size-4" /> },
    { to: '/super/units', label: 'Units', icon: <Shield className="size-4" /> },
    { to: '/appeals', label: 'Appeals', icon: <FileText className="size-4" /> },
    { to: '/super/users', label: 'Users', icon: <Users className="size-4" /> },
    { to: '/super/audit', label: 'Audit', icon: <FileText className="size-4" /> },
    { to: '/super/analytics', label: 'Analytics', icon: <BarChart3 className="size-4" /> },
    { to: '/super/settings', label: 'Settings', icon: <FileText className="size-4" /> },
  ],
}

const ROLE_LABEL: Record<Role, string> = {
  citizen: 'Citizen',
  officer: 'Officer',
  unit_admin: 'Unit admin',
  super_admin: 'Super admin',
}

function NavItems({
  items,
  variant,
  onNavigate,
}: {
  items: NavItem[]
  variant: 'sidebar' | 'bottom'
  /** Drawer-only: close the mobile sidebar after a tap. Never wired to the
   *  bottom nav, which must keep behaving exactly as it does today. */
  onNavigate?: () => void
}) {
  return (
    <>
      {items.filter((item) => variant !== 'bottom' || item.to !== '/sos').map((item) => {
        if (item.soon) {
          return (
            <span
              key={item.to}
              aria-disabled="true"
              title="Coming in a later milestone"
              className={cn(
                'flex cursor-not-allowed items-center gap-3 rounded-lg text-ink-faint',
                variant === 'sidebar' ? 'px-3 py-2 text-sm' : 'flex min-w-[72px] flex-col items-center justify-center gap-0.5 px-2 py-2 text-[10px]',
              )}
            >
              {item.icon}
              <span className="truncate">{item.label}</span>
              {variant === 'sidebar' ? (
                <span className="ml-auto rounded bg-surface-hi px-1.5 py-0.5 text-[9px] uppercase tracking-wide">
                  Soon
                </span>
              ) : null}
            </span>
          )
        }
        return (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === '/'}
            onClick={() => {
              if (variant === 'sidebar') onNavigate?.()
            }}
            className={({ isActive }) =>
              cn(
                'flex items-center rounded-lg transition-colors',
                variant === 'sidebar' ? 'gap-3 px-3 py-2 text-sm' : 'flex min-w-[72px] flex-col items-center justify-center gap-0.5 px-2 py-2 text-[10px]',
                isActive
                  ? 'bg-signal/10 text-signal'
                  : 'text-ink-muted hover:bg-surface-hi hover:text-ink',
              )
            }
          >
            {item.icon}
            <span className="truncate">{item.label}</span>
          </NavLink>
        )
      })}
    </>
  )
}

/**
 * Application shell: desktop sidebar + top bar, mobile bottom nav.
 * Role-aware, and honest about what isn't built yet.
 *
 * Below `md` the same sidebar becomes an off-canvas drawer: hidden by
 * default, opened from the top bar, dismissed by the backdrop, the close
 * button, Escape, or any nav link. At `md` and up it is the permanent
 * sidebar it always was, and the drawer affordances are all `md:hidden`.
 */
export function AppShell({ children }: { children?: ReactNode } = {}) {
  const { user, role, logout } = useAuth()
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const items = role ? NAV[role] : []
  const [mobileNavOpen, setMobileNavOpen] = useState(false)

  function closeMobileNav() {
    setMobileNavOpen(false)
  }

  // A route change dismisses the drawer. Plain state only — never scroll
  // here, or the browser's own scroll restoration fights it on mobile.
  useEffect(() => {
    setMobileNavOpen(false)
  }, [pathname])

  // Escape closes the drawer while it is open; the listener only exists
  // for as long as it is.
  useEffect(() => {
    if (!mobileNavOpen) return
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') setMobileNavOpen(false)
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [mobileNavOpen])

  function handleLogout() {
    logout()
    navigate('/auth/login', { replace: true })
  }

  return (
    <div className="flex min-h-screen bg-base">
      {/* Scrim behind the drawer. `md:hidden` keeps it off desktop entirely. */}
      {mobileNavOpen ? (
        <div
          className="fixed inset-0 z-40 bg-black/60 md:hidden"
          onClick={closeMobileNav}
          aria-hidden="true"
        />
      ) : null}

      {/* Sidebar: off-canvas drawer below md, permanent rail at md and up. */}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 flex w-72 max-w-[80vw] shrink-0 flex-col border-r border-border bg-surface transition-transform duration-200',
          mobileNavOpen ? 'translate-x-0' : '-translate-x-full',
          'md:static md:z-auto md:w-60 md:translate-x-0',
        )}
      >
        <div className="flex items-center gap-2 border-b border-border px-4 py-4">
          <Logo size={32} variant="icon" theme="dark" />
          <div>
            <p className="text-sm font-bold tracking-wide text-ink">NATIVITY GUARD</p>
            <p className="text-[11px] text-ink-muted">{role ? ROLE_LABEL[role] : ''} console</p>
          </div>
          <button
            type="button"
            className="ml-auto rounded-md p-1.5 text-ink-muted transition-colors hover:bg-surface-hi hover:text-ink md:hidden"
            aria-label="Close navigation"
            onClick={closeMobileNav}
          >
            <X className="size-5" aria-hidden />
          </button>
        </div>

        <nav className="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto p-3" aria-label="Primary">
          <NavItems items={items} variant="sidebar" onNavigate={closeMobileNav} />
        </nav>

        <div className="border-t border-border p-3">
          <NavLink to="/profile" onClick={closeMobileNav} className="mb-3 flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-ink-muted hover:text-signal"><UserCircle className="size-4" /> Profile</NavLink>
          <div className="flex items-center gap-2">
            <span className="grid size-8 shrink-0 place-items-center rounded-full bg-surface-hi text-xs font-semibold text-ink">
              {initials(user?.firstName, user?.lastName)}
            </span>
            <div className="min-w-0 flex-1">
              <p className="truncate text-xs font-medium text-ink">
                {fullName(user?.firstName, user?.lastName)}
              </p>
              <p className="truncate text-[10px] text-ink-muted">{user?.email}</p>
            </div>
            <button
              type="button"
              onClick={handleLogout}
              aria-label="Log out"
              title="Log out"
              className="rounded-md p-1.5 text-ink-muted transition-colors hover:bg-surface-hi hover:text-ink"
            >
              <LogOut className="size-4" aria-hidden />
            </button>
          </div>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        {/* Top bar */}
        <header className="sticky top-0 z-30 flex items-center justify-between gap-3 border-b border-border bg-base/90 px-4 py-3 backdrop-blur">
          <div className="flex items-center gap-2 md:hidden">
            <button
              type="button"
              className="rounded-md p-2 text-ink-muted transition-colors hover:bg-surface-hi hover:text-ink"
              aria-label="Open navigation"
              aria-expanded={mobileNavOpen}
              onClick={() => setMobileNavOpen(true)}
            >
              <Menu className="size-5" aria-hidden />
            </button>
            <Logo size={28} variant="icon" theme="dark" />
            <span className="text-sm font-bold tracking-wide text-ink">NGS</span>
          </div>
          <div className="hidden md:flex items-center gap-2">
            <Logo size={28} variant="icon" theme="dark" />
            <span className="text-sm font-bold tracking-wide text-ink hidden lg:inline">
              NATIVITY GUARD SYSTEM
            </span>
          </div>
          <div className="flex items-center gap-2">
            <NotificationBell />
            <NavLink to="/profile" aria-label="Profile" className="grid size-9 place-items-center rounded-lg border border-border-hi bg-surface-hi text-ink-muted md:hidden">
              <UserCircle className="size-4" aria-hidden />
            </NavLink>
          </div>
        </header>

        <main className="min-w-0 flex-1 pb-20 md:pb-0">
          {/* Narrow boundary: a page that throws takes out the content area only,
              so the header and navigation survive and the user can walk away from
              the broken screen instead of losing the whole console. */}
          <ErrorBoundary title="This page failed to render">
            {/* `children` when a caller renders the shell directly (`/` does, since
                it has to decide between the landing page and the console before
                the router picks a child); `<Outlet/>` for every nested route. */}
            {children ?? <Outlet />}
          </ErrorBoundary>
        </main>

        {role !== null ? (
          <NavLink
            to="/sos"
            aria-label="Open emergency SOS"
            className="fixed bottom-16 right-4 z-30 flex min-h-12 items-center gap-2 rounded-full border-2 border-white bg-emergency px-4 font-semibold text-white shadow-panel focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white md:bottom-5"
          >
            <ShieldAlert className="size-5" aria-hidden /> SOS
          </NavLink>
        ) : null}

        {/* Mobile bottom nav — horizontally scrollable so every role's items are
            reachable. super_admins and unit_admins carry more entries than fit a
            phone, so the bar scrolls instead of truncating. */}
        <nav
          aria-label="Primary"
          className="mobile-nav-scroll fixed inset-x-0 bottom-0 z-30 flex items-stretch gap-1 overflow-x-auto border-t bg-surface/95 px-1 py-1.5 backdrop-blur md:hidden"
          style={{ scrollbarWidth: 'none', msOverflowStyle: 'none' }}
        >
          <NavItems items={items} variant="bottom" />
        </nav>
      </div>
    </div>
  )
}
