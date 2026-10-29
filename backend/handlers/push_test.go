//go:build integration

package handlers_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"security-solution/config"
	"security-solution/internal/testutil"
	"security-solution/models"
)

func TestPushTestEndpoint(t *testing.T) {
	server := FreshServer(t)
	superAdmin := testutil.MakeUser(t, "super_admin")
	superAdminToken := testutil.AuthHeader(t, superAdmin)

	// 1. Without push subscriptions -> expect 404
	w := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/admin/push/test", superAdminToken, nil)
	requireAgencyStatus(t, w, http.StatusNotFound)

	// 2. Create a push subscription for the super admin
	sub := models.PushSubscription{
		UserID:   superAdmin.ID,
		Endpoint: "https://push.example.com/test-" + uuid.New().String(),
		P256dh:   "test-p256dh",
		Auth:     "test-auth",
	}
	if err := config.DB.Create(&sub).Error; err != nil {
		t.Fatalf("failed to create push subscription: %v", err)
	}

	// 3. With push subscription -> expect 200
	w2 := agencyJSONRequest(t, server, http.MethodPost, "/api/v1/admin/push/test", superAdminToken, nil)
	requireAgencyStatus(t, w2, http.StatusOK)
}
