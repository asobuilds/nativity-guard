package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var rateLimitTestIPCounter int64

func nextRateLimitTestIP() string {
	n := atomic.AddInt64(&rateLimitTestIPCounter, 1)
	return fmt.Sprintf("198.18.%d.%d:1234", (n/250)%250, n%250)
}

func newRateLimitTestServer(middleware gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID := c.GetHeader("X-Test-User"); userID != "" {
			c.Set("user_id", userID)
		}
		c.Next()
	})
	r.GET("/test", middleware, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func doRateLimitTestRequest(t *testing.T, server http.Handler, userID, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = remoteAddr
	if userID != "" {
		req.Header.Set("X-Test-User", userID)
	}
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	return w
}

func requireRateLimitStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()

	if w.Code != want {
		t.Fatalf("status = %d, want %d. body=%s", w.Code, want, w.Body.String())
	}
}

func TestRateLimitAIEnforcesLimitAndRetryAfter(t *testing.T) {
	server := newRateLimitTestServer(RateLimitAI())
	userID := "user-" + uuid.NewString()
	ip := nextRateLimitTestIP()

	for i := 0; i < aiLimiter.limit; i++ {
		w := doRateLimitTestRequest(t, server, userID, ip)
		requireRateLimitStatus(t, w, http.StatusOK)
	}

	w := doRateLimitTestRequest(t, server, userID, ip)
	requireRateLimitStatus(t, w, http.StatusTooManyRequests)
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestRateLimitAIKeysByUserWithIPFallback(t *testing.T) {
	server := newRateLimitTestServer(RateLimitAI())
	userA := "user-a-" + uuid.NewString()
	userB := "user-b-" + uuid.NewString()
	ipA := nextRateLimitTestIP()
	ipB := nextRateLimitTestIP()

	for i := 0; i < aiLimiter.limit; i++ {
		w := doRateLimitTestRequest(t, server, userA, ipA)
		requireRateLimitStatus(t, w, http.StatusOK)
	}

	w := doRateLimitTestRequest(t, server, userA, ipB)
	requireRateLimitStatus(t, w, http.StatusTooManyRequests)

	w = doRateLimitTestRequest(t, server, userB, ipB)
	requireRateLimitStatus(t, w, http.StatusOK)

	ipC := nextRateLimitTestIP()
	ipD := nextRateLimitTestIP()

	for i := 0; i < aiLimiter.limit; i++ {
		w := doRateLimitTestRequest(t, server, "", ipC)
		requireRateLimitStatus(t, w, http.StatusOK)
	}

	w = doRateLimitTestRequest(t, server, "", ipC)
	requireRateLimitStatus(t, w, http.StatusTooManyRequests)

	w = doRateLimitTestRequest(t, server, "", ipD)
	requireRateLimitStatus(t, w, http.StatusOK)
}

func TestRateLimitGeneralEnforcesIPLimit(t *testing.T) {
	server := newRateLimitTestServer(RateLimitGeneral())
	ip := nextRateLimitTestIP()

	for i := 0; i < generalLimiter.limit; i++ {
		w := doRateLimitTestRequest(t, server, "", ip)
		requireRateLimitStatus(t, w, http.StatusOK)
	}

	w := doRateLimitTestRequest(t, server, "", ip)
	requireRateLimitStatus(t, w, http.StatusTooManyRequests)
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}

	w = doRateLimitTestRequest(t, server, "", nextRateLimitTestIP())
	requireRateLimitStatus(t, w, http.StatusOK)
}
