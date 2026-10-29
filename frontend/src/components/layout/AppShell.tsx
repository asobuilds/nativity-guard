import { Suspense, useEffect, useRef, useState, type ReactNode } from 'react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Spinner } from '@/components/ui/States'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import {
  BarChart3,
  Bell,
  Bot,
  Building2,
  FileText,
  FolderKanban,
  LayoutDashboard,
  LogOut,
  Map as MapIcon,
  Megaphone,
  Menu,
  MessageSquare,
  Newspaper,
  Shield,
  ShieldAlert,
  Settings,
  UserCircle,
  Users,
  Wallet,
  X,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { useAuth } from '@/auth/AuthContext'
import { fullName, initials } from '@/lib/format'
import { Logo } from '@/components/brand/Logo'
import { NotificationBell } from './NotificationBell'
import { AppBackground } from './AppBackground'
import type { Role } from '@/types/api'
import { MOBILE_PRIMARY_ROUTES } from '@/lib/mobileNav'


interface NavItem {
  to: string
  label: string
  icon: ReactNode
  soon?: boolean
}

const AGENCY_NAV: NavItem[] = [
  { to: '/observer/dashboard', label: 'Oversight', icon: <Shield className="size-4" /> },
  { to: '/map', label: 'Safety map', icon: <MapIcon className="size-4" /> },
  { to: '/units', label: 'Security units', icon: <Shield className="size-4" /> },
]

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
    { to: '/units', label: 'Security units', icon: <Shield className="size-4" /> },
    { to: '/feedback', label: 'Feedback & support', icon: <MessageSquare className="size-4" /> },
  ],
  officer: [
    { to: '/assistant', label: 'Safety assistant', icon: <Bot className="size-4" /> },
    { to: '/officer/queue', label: 'Case queue', icon: <FolderKanban className="size-4" /> },
    { to: '/lga', label: 'LGA channel', icon: <Building2 className="size-4" /> },
    { to: '/map', label: 'Operations map', icon: <MapIcon className="size-4" /> },
    { to: '/appeals', label: 'Appeals', icon: <FileText className="size-4" /> },
    { to: '/feedback', label: 'Feedback & support', icon: <MessageSquare className="size-4" /> },
  ],
  unit_admin: [
    { to: '/assistant', label: 'Safety assistant', icon: <Bot className="size-4" /> },
    { to: '/admin/cases', label: 'Case review', icon: <FolderKanban className="size-4" /> },
    { to: '/admin/community', label: 'Community', icon: <Users className="size-4" /> },
    { to: '/admin/reports', label: 'Moderation', icon: <ShieldAlert className="size-4" /> },
    { to: '/admin/transfers', label: 'Transfers', icon: <FileText className="size-4" /> },
    { to: '/admin/overview', label: 'Overview', icon: <LayoutDashboard className="size-4" /> },
    { to: '/admin/unit', label: 'Unit dashboard', icon: <LayoutDashboard className="size-4" /> },
    { to: '/unit/finance', label: 'Financial ledger', icon: <Wallet className="size-4" /> },
    { to: '/admin/officers', label: 'Units', icon: <Users className="size-4" /> },
    { to: '/admin/analytics', label: 'Analytics', icon: <BarChart3 className="size-4" /> },
    { to: '/admin/settings', label: 'Unit settings', icon: <Shield className="size-4" /> },
    { to: '/admin/unit-policy', label: 'Governance policy', icon: <Shield className="size-4" /> },
    { to: '/admin/governance-audit', label: 'Governance overview', icon: <Shield className="size-4" /> },
    { to: '/appeals', label: 'Appeals', icon: <FileText className="size-4" /> },
    { to: '/map', label: 'Operations map', icon: <MapIcon className="size-4" /> },
    { to: '/feedback', label: 'Feedback & support', icon: <MessageSquare className="size-4" /> },
  ],
  super_admin: [
    { to: '/super/overview', label: 'Command center', icon: <LayoutDashboard className="size-4" /> },
    { to: '/sos', label: 'Emergency SOS', icon: <ShieldAlert className="size-4" /> },
    { to: '/alerts', label: 'Alerts', icon: <Megaphone className="size-4" /> },
    { to: '/news', label: 'Live news', icon: <Newspaper className="size-4" /> },
    { to: '/subscriptions', label: 'Alert subscriptions', icon: <Bell className="size-4" /> },
    { to: '/map', label: 'Operations map', icon: <MapIcon className="size-4" /> },
    { to: '/admin/cases', label: 'Case review', icon: <FolderKanban className="size-4" /> },
    { to: '/admin/transfers', label: 'Transfers', icon: <FileText className="size-4" /> },
    { to: '/super/units', label: 'Security units', icon: <Shield className="size-4" /> },
    { to: '/unit/finance', label: 'Financial ledger', icon: <Wallet className="size-4" /> },
    { to: '/admin/officers', label: 'Unit rosters', icon: <Users className="size-4" /> },
    { to: '/community', label: 'Community', icon: <Users className="size-4" /> },
    { to: '/admin/community', label: 'Community management', icon: <ShieldAlert className="size-4" /> },
    { to: '/admin/reports', label: 'Moderation', icon: <ShieldAlert className="size-4" /> },
    { to: '/super/users', label: 'Users & roles', icon: <Users className="size-4" /> },
    { to: '/admin/identity', label: 'Identity verification', icon: <Shield className="size-4" /> },
    { to: '/admin/units-verification', label: 'Unit verification', icon: <Shield className="size-4" /> },
    { to: '/admin/feedback', label: 'Feedback & support', icon: <MessageSquare className="size-4" /> },
    { to: '/appeals', label: 'Appeals', icon: <FileText className="size-4" /> },
    { to: '/super/audit', label: 'Audit & oversight', icon: <FileText className="size-4" /> },
    { to: '/super/analytics', label: 'Analytics', icon: <BarChart3 className="size-4" /> },
    { to: '/super/settings', label: 'Platform settings', icon: <Settings className="size-4" /> },
  ],
  agency_observer: AGENCY_NAV,
  agency_division_admin: AGENCY_NAV,
}

const ROLE_LABEL: Record<Role, string> = {
  citizen: 'Citizen',
  officer: 'Officer',
  unit_admin: 'Unit admin',
  super_admin: 'Super admin',
  agency_observer: 'Agency observer',
  agency_division_admin: 'Agency division admin',
}

function NavItems({
  items,
  variant,
  onNavigate,
}: {
  items: NavItem[]
  variant: 'sidebar' | 'bottom'
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
                variant === 'sidebar' ? 'px-3 py-2 text-sm' : 'flex min-w-0 flex-1 flex-col items-center justify-center gap-1 px-1 py-2 text-[11px] leading-tight',
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
                variant === 'sidebar' ? 'gap-3 px-3 py-2 text-sm' : 'flex min-w-0 flex-1 flex-col items-center justify-center gap-1 px-1 py-2 text-[11px] leading-tight',
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

export function AppShell({ children }: { children?: ReactNode } = {}) {
  const { user, role, logout } = useAuth()
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const items = role ? NAV[role] : []
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const mobileDrawerRef = useRef<HTMLElement>(null)
  const mobileDrawerCloseRef = useRef<HTMLButtonElement>(null)
  const bottomItems = role
    ? MOBILE_PRIMARY_ROUTES[role].map(path => items.find(item => item.to === path)).filter((item): item is NavItem => Boolean(item))
    : []

  function closeMobileNav() {
    setMobileNavOpen(false)
  }

  useEffect(() => {
    setMobileNavOpen(false)
  }, [pathname])

  useEffect(() => {
    if (!mobileNavOpen) return
    const previousOverflow = document.body.style.overflow
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    document.body.style.overflow = 'hidden'
    mobileDrawerCloseRef.current?.focus()
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setMobileNavOpen(false)
        return
      }
      if (event.key !== 'Tab' || !mobileDrawerRef.current) return
      const focusable = Array.from(mobileDrawerRef.current.querySelectorAll<HTMLElement>('a[href],button:not([disabled])'))
      if (!focusable.length) return
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault()
        last.focus()
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first.focus()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => {
      window.removeEventListener('keydown', onKeyDown)
      document.body.style.overflow = previousOverflow
      previousFocus?.focus()
    }
  }, [mobileNavOpen])

  function handleLogout() {
    logout()
    navigate('/auth/login', { replace: true })
  }

  return (
    <div className="relative flex min-h-screen">
      <AppBackground />
      {mobileNavOpen ? (
        <div
          className="fixed inset-0 z-40 bg-black/60 lg:hidden"
          onClick={closeMobileNav}
          aria-hidden="true"
        />
      ) : null}

      <aside
        id="app-mobile-drawer"
        ref={mobileDrawerRef}
        className={cn(
          // Below `lg` this is a drawer. It stops 3.5rem short of the right edge so
          // the page it covers stays visible and the backdrop stays tappable,
          // instead of the panel filling 80% of a 360px phone.
          'fixed inset-y-0 left-0 z-50 flex w-[17rem] max-w-[calc(100vw-3.5rem)] shrink-0 flex-col border-r border-border bg-surface transition-transform duration-200',
          'safe-inset-y',
          mobileNavOpen ? 'translate-x-0' : '-translate-x-full',
          'lg:static lg:z-auto lg:w-60 lg:translate-x-0',
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
            className="ml-auto rounded-md p-1.5 text-ink-muted transition-colors hover:bg-surface-hi hover:text-ink lg:hidden"
            aria-label="Close navigation"
            ref={mobileDrawerCloseRef}
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
          <NavLink to="/settings" onClick={closeMobileNav} className="mb-3 flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-ink-muted hover:text-signal"><Settings className="size-4" /> Your settings</NavLink>
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
        <header className="safe-top-inset sticky top-0 z-30 flex items-center justify-between gap-3 border-b border-border bg-base/90 px-4 pb-3 backdrop-blur">
          <div className="flex items-center gap-2 lg:hidden">
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
          <div className="hidden lg:flex items-center gap-2">
            <Logo size={28} variant="icon" theme="dark" />
            <span className="text-sm font-bold tracking-wide text-ink hidden lg:inline">
              NATIVITY GUARD SYSTEM
            </span>
          </div>
          <div className="flex items-center gap-2">
            <NotificationBell />
            <NavLink to="/profile" aria-label="Profile" className="grid size-9 place-items-center rounded-lg border border-border-hi bg-surface-hi text-ink-muted lg:hidden">
              <UserCircle className="size-4" aria-hidden />
            </NavLink>
            <NavLink to="/settings" aria-label="Your settings" className="grid size-9 place-items-center rounded-lg border border-border-hi bg-surface-hi text-ink-muted lg:hidden">
              <Settings className="size-4" aria-hidden />
            </NavLink>
          </div>
        </header>

        <main className="safe-main-inset min-w-0 flex-1">
          <ErrorBoundary title="This page failed to render">
            {/* `children` when a caller renders the shell directly (`/` does, since
                it has to decide between the landing page and the console before
                the router picks a child); `<Outlet/>` for every nested route. */}
            <Suspense fallback={
              <div role="status" aria-live="polite" className="flex min-h-48 items-center justify-center gap-3 text-ink-muted">
                <Spinner /><span>Opening pageâ€¦</span>
              </div>
            }>
              {children ?? <Outlet />}
            </Suspense>
          </ErrorBoundary>
        </main>

        {role !== null ? (
          <NavLink
            to="/sos"
            aria-label="Open emergency SOS"
            className="safe-sos-offset fixed right-4 z-30 flex min-h-12 items-center gap-2 rounded-full border-2 border-white bg-emergency px-4 font-semibold text-white shadow-panel focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white lg:bottom-5"
          >
            <ShieldAlert className="size-5" aria-hidden /> SOS
          </NavLink>
        ) : null}

        <nav
          aria-label="Mobile primary navigation"
          className="safe-bottom-bar fixed inset-x-0 bottom-0 z-30 grid grid-cols-4 items-stretch gap-1 border-t border-border bg-surface/95 px-1 pt-1.5 backdrop-blur lg:hidden"
        >
          <NavItems items={bottomItems} variant="bottom" />
          <button
            type="button"
            onClick={() => setMobileNavOpen(true)}
            aria-label="More navigation options"
            aria-controls="app-mobile-drawer"
            aria-expanded={mobileNavOpen}
            className="flex min-h-12 min-w-0 flex-col items-center justify-center gap-1 rounded-lg px-1 py-2 text-[11px] leading-tight text-ink-muted transition-colors hover:bg-surface-hi hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-signal"
          >
            <Menu className="size-4" aria-hidden />
            <span>More</span>
          </button>
        </nav>
      </div>
    </div>
  )
}

