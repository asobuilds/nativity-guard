package routes_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"

	"security-solution/routes"
)

var routeRateLimitIPCounter int64

func TestPublicFinancialRouteRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	routes.SetupRoutes(router)

	ip := fmt.Sprintf("203.0.113.%d:1234", atomic.AddInt64(&routeRateLimitIPCounter, 1))
	const path = "/api/v1/public/platform/donation-info"

	for i := 0; i < 200; i++ {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("rate limited too early on request %d", i+1)
		}
	}

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.RemoteAddr = ip
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429. body=%s", w.Code, w.Body.String())
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}
