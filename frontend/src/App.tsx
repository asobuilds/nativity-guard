import type { ReactNode } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { AuthProvider, useAuth } from '@/auth/AuthContext'
import { RequireRole, homePathForRole } from '@/auth/RequireRole'
import { AppShell } from '@/components/layout/AppShell'
import { LoadingBanner } from '@/components/layout/LoadingBanner'
import { FullPageSpinner } from '@/components/ui/States'
import { ToastProvider } from '@/components/ui/Toast'
import { queryClient } from '@/lib/queryClient'
import { useRouteLoading } from '@/hooks/useRouteLoading'
import { LandingPage } from '@/pages/LandingPage'
import { LoginPage } from '@/pages/auth/LoginPage'
import { SignupPage } from '@/pages/auth/SignupPage'
import { ForgotPasswordPage } from '@/pages/auth/ForgotPasswordPage'
import { ResetPasswordPage } from '@/pages/auth/ResetPasswordPage'
import { CitizenHomePage } from '@/pages/citizen/CitizenHomePage'
import { CitizenCasePage } from '@/pages/citizen/CitizenCasePage'
import { SosPage } from '@/pages/citizen/SosPage'
import { ReportIncidentPage } from '@/pages/citizen/ReportIncidentPage'
import { OfficerQueuePage } from '@/pages/officer/OfficerQueuePage'
import { OfficerCasePage } from '@/pages/officer/OfficerCasePage'
import { MapPage } from '@/pages/MapPage'
import { AdminCaseQueuePage } from '@/pages/admin/AdminCaseQueuePage'
import { AdminCaseReviewPage } from '@/pages/admin/AdminCaseReviewPage'
import { TransfersPage } from '@/pages/admin/TransfersPage'
import { AdminOfficersPage } from '@/pages/admin/AdminOfficersPage'
import { ComingSoonPage } from '@/pages/ComingSoonPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { NotificationsPage } from '@/pages/NotificationsPage'
import { AlertsPage, AlertDetailPage, NewsPage, SubscriptionsPage } from '@/pages/AwarenessPage'
import { CommunityPage } from '@/pages/CommunityPage'
import { AdminDemoPage } from '@/pages/AdminDemoPage'
import { SuperOverviewPage } from '@/pages/super/SuperOverviewPage'
import { SuperUsersPage } from '@/pages/super/SuperUsersPage'
import { SuperAuditPage } from '@/pages/super/SuperAuditPage'
import { UnitsRegistryPage } from '@/pages/super/UnitsRegistryPage'
import { UnitRegistrationPage } from '@/pages/super/UnitRegistrationPage'
import { ProfilePage } from '@/pages/ProfilePage'
import { UnitPolicyPage } from '@/pages/admin/UnitPolicyPage'
import { GovernanceAuditPage } from '@/pages/admin/GovernanceAuditPage'
import { InvitationsPage, InviteLandingPage } from '@/pages/InvitationsPage'
import { SessionsPage } from '@/pages/SessionsPage'
import { AppealsPage } from '@/pages/AppealsPage'
import { AiAssistantPage } from '@/pages/AiAssistantPage'
import type { Role } from '@/types/api'

const ALL_ROLES: Role[] = ['citizen', 'officer', 'unit_admin', 'super_admin']
const STAFF: Role[] = ['officer', 'unit_admin', 'super_admin']
const ADMIN_ROLES: Role[] = ['unit_admin', 'super_admin']

/** Signed-in area: session required, role-aware chrome, nested routes. */
function ProtectedShell({ children }: { children?: ReactNode }) {
  return (
    <RequireRole roles={ALL_ROLES}>
      <AppShell>{children}</AppShell>
    </RequireRole>
  )
}

/**
 * `/` — the only address that means two different things.
 *
 * A stranger must land on the front door, and a signed-in citizen must land on
 * their home. That decision cannot be made by the router, because `RequireRole`
 * answers "anonymous" by sending you to a login form — which is exactly the
 * no-front-door problem `frontagent` §11 exists to fix. So `/` sits outside the
 * guard and picks a side here, rendering the console through the same shell every
 * other signed-in route uses, so nothing about the signed-in experience changes.
 */
function RootRoute() {
  const { status } = useAuth()
  if (status === 'loading') return <FullPageSpinner label="Restoring session…" />
  if (status === 'anonymous') return <LandingPage />
  return (
    <ProtectedShell>
      <HomeRoute />
    </ProtectedShell>
  )
}

/** `/` means different things to different roles. */
function HomeRoute() {
  const { role } = useAuth()
  // `/` *is* the citizen home, so it must never be a `Navigate` target from here —
  // redirecting to the route we are already on is exactly the loop this guards
  // against. A role with no home is screened by `RequireRole` above; the check
  // below is the type-level half of the same rule, so a missing destination falls
  // through to the citizen page rather than navigating somewhere invented.
  if (role && role !== 'citizen') {
    const home = homePathForRole(role)
    if (home) return <Navigate to={home} replace />
  }
  return <CitizenHomePage />
}

/**
 * `/login` → `/auth/login`.
 *
 * The brief and the deployed site both point people at `/login`, and the console
 * answers at `/auth/login`. A redirect rather than a second route keeps exactly one
 * sign-in screen in the app — but it carries `location.state` across, so a guard
 * that bounced you here with a `from` still returns you where you were headed once
 * you are signed in. A bare `<Navigate>` would silently drop that.
 */
function LoginAlias() {
  const location = useLocation()
  return <Navigate to="/auth/login" replace state={location.state} />
}

/**
 * The route-loading indicator, hoisted to the top of the router so it overlays
 * the signed-in shell and can render on every route — including the guarded
 * ones that decide whether the shell is shown at all. `useRouteLoading` reads
 * react-router's location, so it must live inside `<BrowserRouter>`.
 */
function RouteLoadingBanner() {
  const { loading, message } = useRouteLoading()
  return <LoadingBanner show={loading} message={message} />
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <BrowserRouter>
          <RouteLoadingBanner />
          <AuthProvider>
            <Routes>
              <Route path="/auth/login" element={<LoginPage />} />
              <Route path="/auth/signup" element={<SignupPage />} />
              <Route path="/invite" element={<InviteLandingPage />} />

              {/* Recovery. Two routes rather than one screen with steps, because
                  the code alone is what `/auth/reset-password` needs — so the
                  second half still works after a reload drops the router state
                  that carried the identifier across. */}
              <Route path="/auth/forgot-password" element={<ForgotPasswordPage />} />
              <Route path="/auth/reset-password" element={<ResetPasswordPage />} />

              {/* Outside `ProtectedShell` on purpose: a signed-out visitor is
                  exactly who these aliases are for. */}
              <Route path="/login" element={<LoginAlias />} />

              {/* The front door. Public by design — see `RootRoute`. */}
              <Route path="/" element={<RootRoute />} />

              <Route element={<ProtectedShell />}>
                <Route path="/notifications" element={<NotificationsPage />} />
                <Route path="/profile" element={<ProfilePage />} />
                <Route path="/invites" element={<InvitationsPage />} />
                <Route path="/sessions" element={<SessionsPage />} />
                <Route path="/alerts" element={<RequireRole roles={['citizen']}><AlertsPage /></RequireRole>} />
                <Route path="/alerts/:id" element={<RequireRole roles={['citizen']}><AlertDetailPage /></RequireRole>} />
                <Route path="/news" element={<RequireRole roles={['citizen']}><NewsPage /></RequireRole>} />
                <Route path="/subscriptions" element={<RequireRole roles={['citizen']}><SubscriptionsPage /></RequireRole>} />
                <Route path="/community" element={<RequireRole roles={['citizen']}><CommunityPage /></RequireRole>} />
                <Route path="/admin/community" element={<RequireRole roles={ADMIN_ROLES}><CommunityPage /></RequireRole>} />
                <Route path="/appeals" element={<AppealsPage />} />
                <Route path="/assistant" element={<AiAssistantPage />} />
                {/* SOS is for everyone signed in — an officer in trouble needs the
                    same button a citizen does, so no role gate beyond the shell. */}
                <Route path="/sos" element={<SosPage />} />
                {/* A citizen's own report — the curated detail view, deliberately
                    narrower than the staff case page. Citizen-only, so an officer
                    who lands here is sent to their own queue rather than shown a
                    reporter's rendering of a case they work. */}
                <Route
                  path="/cases/:id"
                  element={
                    <RequireRole roles={['citizen']}>
                      <CitizenCasePage />
                    </RequireRole>
                  }
                />

                {/* Filing a report. Open to every signed-in role: the endpoint
                    creates a case owned by the authenticated reporter, and staff
                    reporting an incident is a real event, not a misuse. */}
                <Route path="/report" element={<ReportIncidentPage />} />

                {/* Officer workspace — the operational core (staff only). */}
                <Route
                  path="/officer/queue"
                  element={
                    <RequireRole roles={STAFF}>
                      <OfficerQueuePage />
                    </RequireRole>
                  }
                />
                <Route
                  path="/officer/cases/:id"
                  element={
                    <RequireRole roles={STAFF}>
                      <OfficerCasePage />
                    </RequireRole>
                  }
                />

                {/* Shared spatial view, used by every role. */}
                <Route path="/map" element={<MapPage />} />

                {/* Unit administration — triage, assignment and case review. */}
                <Route
                  path="/admin/cases"
                  element={
                    <RequireRole roles={ADMIN_ROLES}>
                      <AdminCaseQueuePage />
                    </RequireRole>
                  }
                />
                <Route
                  path="/admin/cases/:id"
                  element={
                    <RequireRole roles={ADMIN_ROLES}>
                      <AdminCaseReviewPage />
                    </RequireRole>
                  }
                />
                <Route path="/admin/transfers" element={<RequireRole roles={ADMIN_ROLES}><TransfersPage /></RequireRole>} />
                <Route path="/admin/unit-policy" element={<RequireRole roles={ADMIN_ROLES}><UnitPolicyPage /></RequireRole>} />
                <Route path="/admin/governance-audit" element={<RequireRole roles={ADMIN_ROLES}><GovernanceAuditPage /></RequireRole>} />

                <Route path="/admin/officers" element={<RequireRole roles={ADMIN_ROLES}><AdminOfficersPage /></RequireRole>} />
                <Route path="/admin/officers/:unitId" element={<RequireRole roles={ADMIN_ROLES}><AdminOfficersPage /></RequireRole>} />
                {/* Remaining admin sections still use demo data. */}
                {['overview','analytics','finance','settings'].map((section) => <Route key={section} path={`/admin/${section}`} element={<RequireRole roles={ADMIN_ROLES}><AdminDemoPage section={section} /></RequireRole>} />)}
                {['overview','audit','analytics','settings'].map((section) => <Route key={section} path={`/super/${section}`} element={<RequireRole roles={['super_admin']}><AdminDemoPage section={section} platform /></RequireRole>} />)}
                {/* The unit registry is built — it left the demo stub and reads real
                    data. Listed before the `/super/*` fallback so the real page
                    wins, and named ahead of the `:id` route so "new" is never read
                    as a unit id. */}
                <Route
                  path="/super/units"
                  element={
                    <RequireRole roles={['super_admin']}>
                      <UnitsRegistryPage />
                    </RequireRole>
                  }
                />
                <Route
                  path="/super/units/new"
                  element={
                    <RequireRole roles={['super_admin']}>
                      <UnitRegistrationPage />
                    </RequireRole>
                  }
                />
                <Route
                  path="/super/units/:id"
                  element={
                    <RequireRole roles={['super_admin']}>
                      <ComingSoonPage
                        title="Unit record"
                        description="The single-unit record — verification, roster and the full registration as submitted — is designed but not yet implemented. The registry list and the registration form are available."
                        milestone="M7"
                      />
                    </RequireRole>
                  }
                />
                <Route
                  path="/admin/*"
                  element={
                    <RequireRole roles={['unit_admin', 'super_admin']}>
                      <ComingSoonPage
                        title="Unit administration"
                        description="Officer management, unit analytics and settings for unit administrators are designed but not yet implemented. Case review and assignment are available."
                        milestone="M5"
                      />
                    </RequireRole>
                  }
                />
                <Route
                  path="/super/*"
                  element={
                    <RequireRole roles={['super_admin']}>
                      <ComingSoonPage
                        title="Platform governance"
                        description="Cross-unit oversight, user administration and the audit trail for super administrators are designed but not yet implemented."
                        milestone="M7"
                      />
                    </RequireRole>
                  }
                />

                <Route path="*" element={<NotFoundPage />} />
              </Route>
            </Routes>
          </AuthProvider>
        </BrowserRouter>
      </ToastProvider>
    </QueryClientProvider>
  )
}
