import { lazy, Suspense, type ReactNode } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter, Navigate, Route, Routes, useLocation } from 'react-router-dom'
import { AuthProvider, useAuth } from '@/auth/AuthContext'
import { RequireRole, homePathForRole } from '@/auth/RequireRole'
import { AppShell } from '@/components/layout/AppShell'
import { LoadingBanner } from '@/components/layout/LoadingBanner'
import { FullPageSpinner } from '@/components/ui/States'
import { ToastProvider } from '@/components/ui/Toast'
import { OfflineBanner } from '@/components/layout/OfflineBanner'
import { queryClient } from '@/lib/queryClient'
import { useRouteLoading } from '@/hooks/useRouteLoading'
import { LandingPage } from '@/pages/LandingPage'
const LoginPage = lazy(() => import('@/pages/auth/LoginPage').then(m => ({ default: m.LoginPage })))
const SignupPage = lazy(() => import('@/pages/auth/SignupPage').then(m => ({ default: m.SignupPage })))
const OnboardingPage = lazy(() => import('@/pages/OnboardingPage').then(m => ({ default: m.OnboardingPage })))
const ForgotPasswordPage = lazy(() => import('@/pages/auth/ForgotPasswordPage').then(m => ({ default: m.ForgotPasswordPage })))
const ResetPasswordPage = lazy(() => import('@/pages/auth/ResetPasswordPage').then(m => ({ default: m.ResetPasswordPage })))
const CitizenHomePage = lazy(() => import('@/pages/citizen/CitizenHomePage').then(m => ({ default: m.CitizenHomePage })))
const CitizenCasePage = lazy(() => import('@/pages/citizen/CitizenCasePage').then(m => ({ default: m.CitizenCasePage })))
const SosPage = lazy(() => import('@/pages/citizen/SosPage').then(m => ({ default: m.SosPage })))
const SosDetailPage = lazy(() => import('@/pages/citizen/SosDetailPage').then(m => ({ default: m.SosDetailPage })))
const ReportIncidentPage = lazy(() => import('@/pages/citizen/ReportIncidentPage').then(m => ({ default: m.ReportIncidentPage })))
const OfficerQueuePage = lazy(() => import('@/pages/officer/OfficerQueuePage').then(m => ({ default: m.OfficerQueuePage })))
const OfficerCasePage = lazy(() => import('@/pages/officer/OfficerCasePage').then(m => ({ default: m.OfficerCasePage })))
const MapPage = lazy(() => import('@/pages/MapPage').then(m => ({ default: m.MapPage })))
const UnitsPage = lazy(() => import('@/pages/UnitsPage').then(m => ({ default: m.UnitsPage })))
const UnitDetailPage = lazy(() => import('@/pages/UnitDetailPage').then(m => ({ default: m.UnitDetailPage })))
const AdminCaseQueuePage = lazy(() => import('@/pages/admin/AdminCaseQueuePage').then(m => ({ default: m.AdminCaseQueuePage })))
const AdminCaseReviewPage = lazy(() => import('@/pages/admin/AdminCaseReviewPage').then(m => ({ default: m.AdminCaseReviewPage })))
const TransfersPage = lazy(() => import('@/pages/admin/TransfersPage').then(m => ({ default: m.TransfersPage })))
const AdminOfficersPage = lazy(() => import('@/pages/admin/AdminOfficersPage').then(m => ({ default: m.AdminOfficersPage })))
const AdminOverviewPage = lazy(() => import('@/pages/admin/AdminOverviewPage').then(m => ({ default: m.AdminOverviewPage })))
const AdminFinancePage = lazy(() => import('@/pages/admin/AdminFinancePage').then(m => ({ default: m.AdminFinancePage })))
const ComingSoonPage = lazy(() => import('@/pages/ComingSoonPage').then(m => ({ default: m.ComingSoonPage })))
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage').then(m => ({ default: m.NotFoundPage })))
const NotificationsPage = lazy(() => import('@/pages/NotificationsPage').then(m => ({ default: m.NotificationsPage })))
const AlertsPage = lazy(() => import('@/pages/AwarenessPage').then(m => ({ default: m.AlertsPage })))
const AlertDetailPage = lazy(() => import('@/pages/AwarenessPage').then(m => ({ default: m.AlertDetailPage })))
const NewsPage = lazy(() => import('@/pages/AwarenessPage').then(m => ({ default: m.NewsPage })))
const SubscriptionsPage = lazy(() => import('@/pages/SubscriptionsPage').then(m => ({ default: m.SubscriptionsPage })))
const CommunityPage = lazy(() => import('@/pages/CommunityPage').then(m => ({ default: m.CommunityPage })))
const SuperOverviewPage = lazy(() => import('@/pages/super/SuperOverviewPage').then(m => ({ default: m.SuperOverviewPage })))
const SuperUsersPage = lazy(() => import('@/pages/super/SuperUsersPage').then(m => ({ default: m.SuperUsersPage })))
const SuperAuditPage = lazy(() => import('@/pages/super/SuperAuditPage').then(m => ({ default: m.SuperAuditPage })))
const SuperAnalyticsPage = lazy(() => import('@/pages/super/SuperAnalyticsPage').then(m => ({ default: m.SuperAnalyticsPage })))
const SuperSettingsPage = lazy(() => import('@/pages/super/SuperSettingsPage').then(m => ({ default: m.SuperSettingsPage })))
const UnitsRegistryPage = lazy(() => import('@/pages/super/UnitsRegistryPage').then(m => ({ default: m.UnitsRegistryPage })))
const UnitRegistrationPage = lazy(() => import('@/pages/super/UnitRegistrationPage').then(m => ({ default: m.UnitRegistrationPage })))
const ProfilePage = lazy(() => import('@/pages/ProfilePage').then(m => ({ default: m.ProfilePage })))
const SettingsPage = lazy(() => import('@/pages/SettingsPage').then(m => ({ default: m.SettingsPage })))
const TermsPage = lazy(() => import('@/pages/TermsPage').then(m => ({ default: m.TermsPage })))
const UnitPolicyPage = lazy(() => import('@/pages/admin/UnitPolicyPage').then(m => ({ default: m.UnitPolicyPage })))
const GovernanceAuditPage = lazy(() => import('@/pages/admin/GovernanceAuditPage').then(m => ({ default: m.GovernanceAuditPage })))
const InvitationsPage = lazy(() => import('@/pages/InvitationsPage').then(m => ({ default: m.InvitationsPage })))
const InviteLandingPage = lazy(() => import('@/pages/InvitationsPage').then(m => ({ default: m.InviteLandingPage })))
const SessionsPage = lazy(() => import('@/pages/SessionsPage').then(m => ({ default: m.SessionsPage })))
const AppealsPage = lazy(() => import('@/pages/AppealsPage').then(m => ({ default: m.AppealsPage })))
const AiAssistantPage = lazy(() => import('@/pages/AiAssistantPage').then(m => ({ default: m.AiAssistantPage })))
const VerifyIdentityPage = lazy(() => import('@/pages/VerifyIdentityPage').then(m => ({ default: m.VerifyIdentityPage })))
const IdentityVerificationQueuePage = lazy(() => import('@/pages/admin/IdentityVerificationQueuePage').then(m => ({ default: m.IdentityVerificationQueuePage })))
const UnitVerificationQueuePage = lazy(() => import('@/pages/admin/UnitVerificationQueuePage').then(m => ({ default: m.UnitVerificationQueuePage })))
import type { Role } from '@/types/api'

const ALL_ROLES: Role[] = ['citizen', 'officer', 'unit_admin', 'super_admin']
const STAFF: Role[] = ['officer', 'unit_admin', 'super_admin']
const ADMIN_ROLES: Role[] = ['unit_admin', 'super_admin']

function ProtectedShell({ children }: { children?: ReactNode }) {
  return (
    <RequireRole roles={ALL_ROLES}>
      <AppShell>{children}</AppShell>
    </RequireRole>
  )
}

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

function HomeRoute() {
  const { role } = useAuth()
  if (role && role !== 'citizen') {
    const home = homePathForRole(role)
    if (home) return <Navigate to={home} replace />
  }
  return <CitizenHomePage />
}

function LoginAlias() {
  const location = useLocation()
  return <Navigate to="/auth/login" replace state={location.state} />
}

function RouteLoadingBanner() {
  const { loading, message } = useRouteLoading()
  return <LoadingBanner show={loading} message={message} />
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <OfflineBanner />
        <BrowserRouter>
          <RouteLoadingBanner />
          <AuthProvider>
            <Suspense fallback={<FullPageSpinner />}>
              <Routes>
                <Route path="/auth/login" element={<LoginPage />} />
                <Route path="/auth/signup" element={<SignupPage />} />
                <Route path="/terms" element={<TermsPage />} />
                <Route path="/invite" element={<InviteLandingPage />} />

                <Route path="/auth/forgot-password" element={<ForgotPasswordPage />} />
                <Route path="/auth/reset-password" element={<ResetPasswordPage />} />

                <Route path="/login" element={<LoginAlias />} />

                <Route path="/" element={<RootRoute />} />

                <Route element={<ProtectedShell />}>
                  <Route path="/onboarding" element={<OnboardingPage />} />
                  <Route path="/notifications" element={<NotificationsPage />} />
                  <Route path="/profile" element={<ProfilePage />} />
                  <Route path="/verify-identity" element={<VerifyIdentityPage />} />
                  <Route path="/settings" element={<SettingsPage />} />
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
                  <Route path="/sos" element={<SosPage />} />
                  <Route path="/sos/:id" element={<SosDetailPage />} />
                  <Route
                    path="/cases/:id"
                    element={
                      <RequireRole roles={['citizen']}>
                        <CitizenCasePage />
                      </RequireRole>
                    }
                  />

                  <Route path="/report" element={<ReportIncidentPage />} />

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

                  <Route path="/map" element={<MapPage />} />
                  <Route path="/units" element={<UnitsPage />} />
                  <Route path="/units/:id" element={<UnitDetailPage />} />

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
                  <Route path="/admin/overview" element={<RequireRole roles={ADMIN_ROLES}><AdminOverviewPage /></RequireRole>} />
                  <Route path="/admin/finance" element={<RequireRole roles={ADMIN_ROLES}><AdminFinancePage /></RequireRole>} />
                  <Route path="/admin/analytics" element={<RequireRole roles={ADMIN_ROLES}><ComingSoonPage title="Unit analytics" description="Under construction." milestone="M7" /></RequireRole>} />
                  <Route path="/admin/settings" element={<RequireRole roles={ADMIN_ROLES}><ComingSoonPage title="Unit settings" description="Under construction." milestone="M7" /></RequireRole>} />

                  <Route
                    path="/admin/identity"
                    element={
                      <RequireRole roles={['super_admin']}>
                        <IdentityVerificationQueuePage />
                      </RequireRole>
                    }
                  />
                  <Route
                    path="/admin/units-verification"
                    element={
                      <RequireRole roles={['super_admin']}>
                        <UnitVerificationQueuePage />
                      </RequireRole>
                    }
                  />

                  <Route path="/super/overview" element={<RequireRole roles={['super_admin']}><SuperOverviewPage /></RequireRole>} />
                  <Route path="/super/users" element={<RequireRole roles={['super_admin']}><SuperUsersPage /></RequireRole>} />
                  <Route path="/super/audit" element={<RequireRole roles={['super_admin']}><SuperAuditPage /></RequireRole>} />
                  <Route path="/super/analytics" element={<RequireRole roles={['super_admin']}><SuperAnalyticsPage /></RequireRole>} />
                  <Route path="/super/settings" element={<RequireRole roles={['super_admin']}><SuperSettingsPage /></RequireRole>} />
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
                          description="The single-unit record is designed but not yet implemented."
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
                          description="Officer management, unit analytics and settings for unit administrators are designed but not yet implemented."
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
                          description="Cross-unit oversight, user administration and the audit trail are designed but not yet implemented."
                          milestone="M7"
                        />
                      </RequireRole>
                    }
                  />

                  <Route path="*" element={<NotFoundPage />} />
                </Route>
              </Routes>
            </Suspense>
          </AuthProvider>
        </BrowserRouter>
      </ToastProvider>
    </QueryClientProvider>
  )
}