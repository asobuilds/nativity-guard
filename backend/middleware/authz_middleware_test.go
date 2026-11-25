package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuthMiddleware_RejectsMissingInvalidAndQueryTokenOnHTTP(t *testing.T) {
	r := gin.New()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	cases := []struct {
		name   string
		target string
		header string
	}{
		{name: "missing auth header", target: "/protected"},
		{name: "invalid scheme", target: "/protected", header: "Basic abc"},
		{name: "empty bearer", target: "/protected", header: "Bearer"},
		{name: "query token on plain HTTP", target: "/protected?token=abc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401. body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestRoleMiddleware_EnforcesRolesAndAccountStatus(t *testing.T) {
	makeRouter := func(user *models.User, role string, allowed ...string) *gin.Engine {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			if user != nil {
				c.Set("user", user)
			}
			if role != "" {
				c.Set("role", role)
			}
			c.Next()
		})
		r.GET("/check", RoleMiddleware(allowed...), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		return r
	}

	tests := []struct {
		name       string
		user       *models.User
		role       string
		allowed    []string
		wantStatus int
	}{
		{
			name:       "allowed role passes",
			user:       &models.User{Role: models.RoleUnitAdmin, Status: "active"},
			role:       models.RoleUnitAdmin,
			allowed:    []string{models.RoleSuperAdmin, models.RoleUnitAdmin},
			wantStatus: http.StatusOK,
		},
		{
			name:       "disallowed role rejected",
			user:       &models.User{Role: models.RoleCitizen, Status: "active"},
			role:       models.RoleCitizen,
			allowed:    []string{models.RoleSuperAdmin, models.RoleUnitAdmin},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "missing role rejected",
			user:       nil,
			role:       "",
			allowed:    []string{models.RoleCitizen},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "isSuperAdmin flag allowed for super_admin",
			user:       &models.User{Role: models.RoleUnitAdmin, IsSuperAdmin: true, Status: "active"},
			role:       models.RoleUnitAdmin,
			allowed:    []string{models.RoleSuperAdmin},
			wantStatus: http.StatusOK,
		},
		{
			name:       "banned user rejected even with matching role",
			user:       &models.User{Role: models.RoleSuperAdmin, Status: "banned"},
			role:       models.RoleSuperAdmin,
			allowed:    []string{models.RoleSuperAdmin},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "suspended user rejected even with matching role",
			user:       &models.User{Role: models.RoleOfficer, Status: "suspended"},
			role:       models.RoleOfficer,
			allowed:    []string{models.RoleOfficer},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := makeRouter(tc.user, tc.role, tc.allowed...)
			req := httptest.NewRequest(http.MethodGet, "/check", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d. body=%s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestRequireRole_EnforcesRolesAndAccountStatus(t *testing.T) {
	makeRouter := func(user *models.User, roles ...string) *gin.Engine {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			if user != nil {
				c.Set("user", user)
			}
			c.Next()
		})
		r.GET("/check", RequireRole(roles...), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		return r
	}

	if w := httptest.NewRecorder(); func() int {
		makeRouter(nil, models.RoleOfficer).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/check", nil))
		return w.Code
	}() != http.StatusUnauthorized {
		t.Fatal("expected 401 when user missing from context")
	}

	if w := httptest.NewRecorder(); func() int {
		makeRouter(&models.User{Role: models.RoleSuperAdmin, Status: "banned"}, models.RoleSuperAdmin).
			ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/check", nil))
		return w.Code
	}() != http.StatusForbidden {
		t.Fatal("expected 403 for banned user")
	}

	if w := httptest.NewRecorder(); func() int {
		makeRouter(&models.User{Role: models.RoleCitizen, IsSuperAdmin: true, Status: "active"}, models.RoleSuperAdmin).
			ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/check", nil))
		return w.Code
	}() != http.StatusOK {
		t.Fatal("expected 200 when IsSuperAdmin is true for super_admin requirement")
	}
}

func TestDenyAgencyAdminCaseMutation(t *testing.T) {
	tests := []struct {
		name       string
		user       *models.User
		accessLvl  string
		wantStatus int
	}{
		{
			name:       "agency_division_admin role blocked",
			user:       &models.User{Role: models.RoleAgencyDivisionAdmin},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "agency_admin case_access_level blocked",
			user:       &models.User{Role: models.RoleOfficer},
			accessLvl:  "agency_admin",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "assigned officer allowed",
			user:       &models.User{Role: models.RoleOfficer},
			accessLvl:  "officer_primary",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) {
				if tc.user != nil {
					c.Set("user", tc.user)
				}
				if tc.accessLvl != "" {
					c.Set("case_access_level", tc.accessLvl)
				}
				c.Next()
			})
			r.POST("/mutate", DenyAgencyAdminCaseMutation, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/mutate", nil))
			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestCanAccessUnit(t *testing.T) {
	unitA := uuid.New()
	unitB := uuid.New()

	makeRouter := func(user *models.User, pathPattern string) *gin.Engine {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			if user != nil {
				c.Set("user", user)
			}
			c.Next()
		})
		r.GET(pathPattern, CanAccessUnit, func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		return r
	}

	tests := []struct {
		name       string
		pattern    string
		path       string
		user       *models.User
		wantStatus int
	}{
		{
			name:       "invalid uuid returns 400",
			pattern:    "/units/:unitId",
			path:       "/units/not-a-uuid",
			user:       &models.User{Role: models.RoleSuperAdmin, Status: "active"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "banned user returns 403",
			pattern:    "/units/:unitId",
			path:       "/units/" + unitA.String(),
			user:       &models.User{Role: models.RoleUnitAdmin, UnitID: &unitA, Status: "banned"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "super_admin accesses any unit",
			pattern:    "/units/:unitId",
			path:       "/units/" + unitB.String(),
			user:       &models.User{Role: models.RoleSuperAdmin, UnitID: &unitA, Status: "active"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "matching unit via :id param allowed",
			pattern:    "/units/:id",
			path:       "/units/" + unitA.String(),
			user:       &models.User{Role: models.RoleUnitAdmin, UnitID: &unitA, Status: "active"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "different unit rejected",
			pattern:    "/units/:unitId",
			path:       "/units/" + unitB.String(),
			user:       &models.User{Role: models.RoleUnitAdmin, UnitID: &unitA, Status: "active"},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := makeRouter(tc.user, tc.pattern)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d. body=%s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}
