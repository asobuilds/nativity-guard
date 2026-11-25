//go:build integration

package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"security-solution/internal/testutil"
	"security-solution/models"
)

func auditWriteRowCounts(t *testing.T) (int64, int64, int64) {
	t.Helper()

	db := testutil.GetDB()
	var activities int64
	var auditLogs int64
	var healthRecords int64
	if err := db.Model(&models.ActivityLog{}).Count(&activities).Error; err != nil {
		t.Fatalf("count activity logs: %v", err)
	}
	if err := db.Model(&models.AuditLog{}).Count(&auditLogs).Error; err != nil {
		t.Fatalf("count audit logs: %v", err)
	}
	if err := db.Model(&models.SystemHealth{}).Count(&healthRecords).Error; err != nil {
		t.Fatalf("count system health records: %v", err)
	}
	return activities, auditLogs, healthRecords
}

func TestAuditWriteEndpoints_RejectCitizen(t *testing.T) {
	testutil.TruncateAll(t)
	defer testutil.TruncateAll(t)

	citizen := testutil.MakeUser(t, "citizen")
	token := testutil.AuthHeader(t, citizen)
	server := FreshServer(t)

	cases := []struct {
		name    string
		path    string
		payload string
	}{
		{
			name:    "activity",
			path:    "/api/v1/audit/activity",
			payload: `{"activityType":"forged","description":"forged activity","device":"forged","location":"forged"}`,
		},
		{
			name:    "logs",
			path:    "/api/v1/audit/logs",
			payload: `{"action":"forged","entityType":"user","entityId":"forged","oldValue":"old","newValue":"new"}`,
		},
		{
			name:    "health",
			path:    "/api/v1/audit/health",
			payload: `{"cpuUsage":99,"memoryUsage":99,"diskUsage":99,"serverStatus":"healthy","databaseStatus":"healthy"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", token)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound && w.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 404 or 403. body=%s", w.Code, w.Body.String())
			}
		})
	}

	activities, auditLogs, healthRecords := auditWriteRowCounts(t)
	if activities != 0 || auditLogs != 0 || healthRecords != 0 {
		t.Fatalf("citizen wrote audit data: activities=%d auditLogs=%d healthRecords=%d",
			activities, auditLogs, healthRecords)
	}
}
