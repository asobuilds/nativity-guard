package routes_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"security-solution/routes"
)

func TestProtectedRoutesRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	routes.SetupRoutes(router)

	protected := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/auth/profile"},
		{http.MethodGet, "/api/v1/admin/users"},
		{http.MethodPost, "/api/v1/admin/users/00000000-0000-0000-0000-000000000001/ban"},
		{http.MethodPost, "/api/v1/admin/users/00000000-0000-0000-0000-000000000001/restrict"},
		{http.MethodGet, "/api/v1/super/agency-tenants"},
		{http.MethodPost, "/api/v1/agency/observers"},
		{http.MethodPost, "/api/v1/officer/panic"},
		{http.MethodGet, "/api/v1/observer/incidents"},
		{http.MethodGet, "/api/v1/audit/activities"},
		{http.MethodGet, "/api/v1/audit/logs"},
		{http.MethodGet, "/api/v1/audit/health"},
		{http.MethodGet, "/api/v1/audit/notifications"},
		{http.MethodGet, "/api/v1/analytics/units/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/api/v1/analytics/user-activity"},
		{http.MethodPost, "/api/v1/peacebuilding/committees"},
		{http.MethodPost, "/api/v1/video/cameras"},
		{http.MethodGet, "/api/v1/settings/some_key"},
		{http.MethodPut, "/api/v1/settings/some_key"},
		{http.MethodPost, "/api/v1/evidence/case/00000000-0000-0000-0000-000000000001/presign"},
		{http.MethodGet, "/metrics"},
	}

	for _, tc := range protected {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s status = %d, want 401. body=%s", tc.method, tc.path, w.Code, w.Body.String())
			}
		})
	}
}
