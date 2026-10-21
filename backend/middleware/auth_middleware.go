package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"security-solution/services"
	"time"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		parts := strings.Fields(authHeader)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		authService := services.NewAuthService()

		token, err := authService.ValidateToken(tokenString)
		if err != nil || token == nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token claims",
			})
			c.Abort()
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token claims",
			})
			c.Abort()
			return
		}

		user, err := authService.GetUserByID(userID)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not found",
			})
			c.Abort()
			return
		}

		// The database user is the source of truth for authorization.
		c.Set("user", user)
		c.Set("user_id", userID)
		c.Set("role", user.Role)

		if jti, ok := claims["jti"].(string); ok {
			c.Set("jti", jti)
			services.NewSessionService().Touch(jti)
		}
		if expVal, ok := claims["exp"].(float64); ok {
			c.Set("token_exp", expVal)
		}

		// JWT revocation check
		tokenSvc := services.NewTokenService()

		if jti, ok := claims["jti"].(string); ok && jti != "" {
			if tokenSvc.IsRevoked(jti) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
				c.Abort()
				return
			}
		}

		if iatFloat, ok := claims["iat"].(float64); ok {
			issuedAt := time.Unix(int64(iatFloat), 0).UTC()
			if tokenSvc.IsUserRevokedAfter(user.ID, issuedAt) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Token has been revoked"})
				c.Abort()
				return
			}
		}

		// Provider-created accounts begin with unverified age and no consent.
		// Never grant reporting, SOS, identity upload, or any authenticated API
		// before they complete the same eligibility rules as password signup.
		if user.AccountAccessLevel == "limited" && !limitedSocialRouteAllowed(c.FullPath()) {
			c.JSON(http.StatusForbidden, gin.H{"error": "profile_completion_required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied",
			})
			c.Abort()
			return
		}

		role, ok := roleValue.(string)
		if !ok || role == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Invalid role",
			})
			c.Abort()
			return
		}

		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error": "Insufficient permissions",
		})
		c.Abort()
	}
}

func limitedSocialRouteAllowed(path string) bool {
	switch path {
	case "/api/v1/auth/profile", "/api/v1/auth/complete-profile",
		"/api/v1/auth/logout", "/api/v1/auth/account":
		return true
	default:
		return false
	}
}
