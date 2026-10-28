package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateBucket struct {
	count   int
	resetAt time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	limit   int
	window  time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		buckets: make(map[string]*rateBucket),
		limit:   limit,
		window:  window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for k, b := range rl.buckets {
			if now.After(b.resetAt) {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(key string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok || now.After(b.resetAt) {
		rl.buckets[key] = &rateBucket{count: 1, resetAt: now.Add(rl.window)}
		return true, 0
	}
	if b.count >= rl.limit {
		return false, time.Until(b.resetAt)
	}
	b.count++
	return true, 0
}

// pre-built limiters for different route sensitivities
var (
	authLimiter    = newRateLimiter(10, 1*time.Minute)  // 10 req/min per IP
	otpLimiter     = newRateLimiter(5, 5*time.Minute)   // 5 req/5min per IP
	voteLimiter    = newRateLimiter(20, 1*time.Minute)  // 20 votes/min per user
	inviteLimiter  = newRateLimiter(10, 1*time.Hour)    // 10 invites/hour per user
	generalLimiter = newRateLimiter(200, 1*time.Minute) // 200 req/min per IP
	mapLimiter     = newRateLimiter(60, 1*time.Minute)  // 60 req/min per IP
	aiLimiter      = newRateLimiter(20, 1*time.Minute)  // 20 req/min per user or IP
)

// makeKey returns the identity used for limiting: user_id if present, otherwise client IP.
func makeKey(c *gin.Context) string {
	if v, ok := c.Get("user_id"); ok {
		if s, ok := v.(string); ok && s != "" {
			return "u:" + s
		}
	}
	return "ip:" + c.ClientIP()
}

func reject(c *gin.Context, retryAfter time.Duration) {
	c.Header("Retry-After", retryAfter.Round(time.Second).String())
	c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down"})
	c.Abort()
}

// RateLimitAuth applies strict limit to authentication endpoints.
func RateLimitAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "auth:" + c.ClientIP()
		if ok, wait := authLimiter.allow(key); !ok {
			reject(c, wait)
			return
		}
		c.Next()
	}
}

// RateLimitOTP applies strict limit to OTP endpoints.
func RateLimitOTP() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "otp:" + c.ClientIP()
		if ok, wait := otpLimiter.allow(key); !ok {
			reject(c, wait)
			return
		}
		c.Next()
	}
}

// RateLimitVote applies per-identity limit to voting endpoints.
func RateLimitVote() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "vote:" + makeKey(c)
		if ok, wait := voteLimiter.allow(key); !ok {
			reject(c, wait)
			return
		}
		c.Next()
	}
}

// RateLimitInvite applies per-identity limit to invite creation.
func RateLimitInvite() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "invite:" + makeKey(c)
		if ok, wait := inviteLimiter.allow(key); !ok {
			reject(c, wait)
			return
		}
		c.Next()
	}
}

// RateLimitGeneral applies a global per-IP throttle to all routes.
func RateLimitGeneral() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "gen:" + c.ClientIP()
		if ok, wait := generalLimiter.allow(key); !ok {
			reject(c, wait)
			return
		}
		c.Next()
	}
}

// RateLimitMap applies a per-IP throttle to map and location endpoints.
func RateLimitMap() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "map:" + c.ClientIP()
		if ok, wait := mapLimiter.allow(key); !ok {
			reject(c, wait)
			return
		}
		c.Next()
	}
}

// RateLimitAI applies a tight per-identity throttle to AI endpoints.
func RateLimitAI() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "ai:" + makeKey(c)
		if ok, wait := aiLimiter.allow(key); !ok {
			reject(c, wait)
			return
		}
		c.Next()
	}
}
