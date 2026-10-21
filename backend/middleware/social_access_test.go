package middleware

import "testing"

func TestLimitedSocialAccountsCannotAccessOperationalRoutes(t *testing.T) {
 for _,route:=range []string{
  "/api/v1/sos/send",
  "/api/v1/sos/media/upload",
  "/api/v1/evidence/case/:caseId/file",
  "/api/v1/cases",
  "/api/v1/super/users",
  "/api/v1/identity/document",
  "/api/v1/settings/onboarding",
 } {
  if limitedSocialRouteAllowed(route){t.Fatalf("limited account allowed on %s",route)}
 }
}

func TestLimitedSocialAccountsKeepEssentialAccountControls(t *testing.T) {
 for _,route:=range []string{
  "/api/v1/auth/profile",
  "/api/v1/auth/complete-profile",
  "/api/v1/auth/logout",
  "/api/v1/auth/account",
 } {
  if !limitedSocialRouteAllowed(route){t.Fatalf("should allow essential route %s",route)}
 }
}
