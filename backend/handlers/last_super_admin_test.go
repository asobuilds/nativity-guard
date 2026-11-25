package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

func TestIsActiveSuperAdmin(t *testing.T) {
	cases := []struct {
		name string
		user *models.User
		want bool
	}{
		{
			name: "nil user",
			user: nil,
			want: false,
		},
		{
			name: "active role super_admin",
			user: &models.User{Role: models.RoleSuperAdmin, Status: "active"},
			want: true,
		},
		{
			name: "active is_super_admin flag",
			user: &models.User{Role: models.RoleCitizen, IsSuperAdmin: true, Status: "active"},
			want: true,
		},
		{
			name: "suspended super_admin",
			user: &models.User{Role: models.RoleSuperAdmin, IsSuperAdmin: true, Status: "suspended"},
			want: false,
		},
		{
			name: "banned super_admin",
			user: &models.User{Role: models.RoleSuperAdmin, IsSuperAdmin: true, Status: "banned"},
			want: false,
		},
		{
			name: "pending super_admin",
			user: &models.User{Role: models.RoleSuperAdmin, IsSuperAdmin: true, Status: "pending"},
			want: false,
		},
		{
			name: "active unit_admin",
			user: &models.User{Role: models.RoleUnitAdmin, IsSuperAdmin: false, Status: "active"},
			want: false,
		},
		{
			name: "active citizen",
			user: &models.User{Role: models.RoleCitizen, IsSuperAdmin: false, Status: "active"},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isActiveSuperAdmin(tc.user); got != tc.want {
				t.Fatalf("isActiveSuperAdmin() = %v, want %v", got, tc.want)
			}
		})
	}
}

func makeLastAdminCtx(method, path string, targetID uuid.UUID, actor *models.User, payload any) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: targetID.String()}}
	if actor != nil {
		c.Set("user", actor)
	}
	return c, w
}

func makePersistedSuperAdmin(t *testing.T, status string, byFlag bool) *models.User {
	t.Helper()
	role := models.RoleSuperAdmin
	if byFlag {
		role = models.RoleCitizen
	}
	u := testutil.MakeUser(t, role)
	updates := map[string]any{
		"status":         status,
		"is_super_admin": byFlag || role == models.RoleSuperAdmin,
	}
	if err := config.DB.Model(u).Updates(updates).Error; err != nil {
		t.Fatalf("update super admin: %v", err)
	}
	u.Status = status
	u.IsSuperAdmin = byFlag || role == models.RoleSuperAdmin
	return u
}

func syntheticSuperAdminActor() *models.User {
	return &models.User{
		ID:           uuid.New(),
		Role:         models.RoleSuperAdmin,
		IsSuperAdmin: true,
		Status:       "active",
	}
}

func TestLastSuperAdminProtection_DeleteUser(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)

	sole := makePersistedSuperAdmin(t, "active", false)
	actor := syntheticSuperAdminActor()

	c, w := makeLastAdminCtx(http.MethodDelete, "/admin/users/"+sole.ID.String(), sole.ID, actor, nil)
	DeleteUser(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("DeleteUser sole super_admin status = %d, want %d; body=%s", w.Code, http.StatusConflict, w.Body.String())
	}

	// Suspended super_admin may still be deleted while sole active remains.
	suspended := makePersistedSuperAdmin(t, "suspended", true)
	c, w = makeLastAdminCtx(http.MethodDelete, "/admin/users/"+suspended.ID.String(), suspended.ID, actor, nil)
	DeleteUser(c)
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteUser suspended super_admin status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	// Once a second active super_admin exists, deleting the first succeeds.
	second := makePersistedSuperAdmin(t, "active", true)
	c, w = makeLastAdminCtx(http.MethodDelete, "/admin/users/"+sole.ID.String(), sole.ID, second, nil)
	DeleteUser(c)
	if w.Code != http.StatusOK {
		t.Fatalf("DeleteUser with second active super_admin status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestLastSuperAdminProtection_UpdateUser(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)

	sole := makePersistedSuperAdmin(t, "active", false)
	actor := syntheticSuperAdminActor()

	// Deactivating via status change must be rejected.
	c, w := makeLastAdminCtx(http.MethodPut, "/admin/users/"+sole.ID.String(), sole.ID, actor, map[string]any{
		"status": "suspended",
	})
	UpdateUser(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("UpdateUser deactivate sole super_admin status = %d, want %d; body=%s", w.Code, http.StatusConflict, w.Body.String())
	}

	// Updating non-status profile fields of the sole active super_admin is allowed.
	c, w = makeLastAdminCtx(http.MethodPut, "/admin/users/"+sole.ID.String(), sole.ID, actor, map[string]any{
		"firstName": "Updated",
	})
	UpdateUser(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateUser profile fields of sole super_admin status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	// With a second active super_admin, deactivating the first is allowed.
	second := makePersistedSuperAdmin(t, "active", false)
	c, w = makeLastAdminCtx(http.MethodPut, "/admin/users/"+sole.ID.String(), sole.ID, second, map[string]any{
		"status": "suspended",
	})
	UpdateUser(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateUser deactivate with second active super_admin status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestLastSuperAdminProtection_UpdateUserRole(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)

	sole := makePersistedSuperAdmin(t, "active", false)
	actor := syntheticSuperAdminActor()

	c, w := makeLastAdminCtx(http.MethodPut, "/admin/users/"+sole.ID.String()+"/role", sole.ID, actor, map[string]any{
		"role": models.RoleCitizen,
	})
	UpdateUserRole(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("UpdateUserRole demote sole super_admin status = %d, want %d; body=%s", w.Code, http.StatusConflict, w.Body.String())
	}

	second := makePersistedSuperAdmin(t, "active", false)
	c, w = makeLastAdminCtx(http.MethodPut, "/admin/users/"+sole.ID.String()+"/role", sole.ID, second, map[string]any{
		"role": models.RoleUnitAdmin,
	})
	UpdateUserRole(c)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateUserRole demote with second active super_admin status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestLastSuperAdminProtection_SuspendUser(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)

	sole := makePersistedSuperAdmin(t, "active", false)
	actor := syntheticSuperAdminActor()

	c, w := makeLastAdminCtx(http.MethodPost, "/admin/users/"+sole.ID.String()+"/suspend", sole.ID, actor, nil)
	SuspendUser(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("SuspendUser sole super_admin status = %d, want %d; body=%s", w.Code, http.StatusConflict, w.Body.String())
	}

	second := makePersistedSuperAdmin(t, "active", false)
	c, w = makeLastAdminCtx(http.MethodPost, "/admin/users/"+sole.ID.String()+"/suspend", sole.ID, second, nil)
	SuspendUser(c)
	if w.Code != http.StatusOK {
		t.Fatalf("SuspendUser with second active super_admin status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestLastSuperAdminProtection_BanUser(t *testing.T) {
	skipWithoutDB(t)
	testutil.TruncateAll(t)

	sole := makePersistedSuperAdmin(t, "active", true)
	actor := syntheticSuperAdminActor()

	c, w := makeLastAdminCtx(http.MethodPost, "/admin/users/"+sole.ID.String()+"/ban", sole.ID, actor, nil)
	BanUser(c)
	if w.Code != http.StatusConflict {
		t.Fatalf("BanUser sole super_admin status = %d, want %d; body=%s", w.Code, http.StatusConflict, w.Body.String())
	}

	second := makePersistedSuperAdmin(t, "active", false)
	c, w = makeLastAdminCtx(http.MethodPost, "/admin/users/"+sole.ID.String()+"/ban", sole.ID, second, nil)
	BanUser(c)
	if w.Code != http.StatusOK {
		t.Fatalf("BanUser with second active super_admin status = %d, want %d; body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}
