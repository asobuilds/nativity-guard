package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"security-solution/services"
)

const signInLinkMessage = "If an account with that email exists, a sign-in link will be sent."

// RequestSignInLink emails a short-lived, single-use sign-in link.
// The response does not reveal whether the email belongs to an account.
func (h *AuthHandler) RequestSignInLink(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A valid email address is required"})
		return
	}

	frontendURL := strings.TrimRight(strings.TrimSpace(os.Getenv("FRONTEND_URL")), "/")
	if frontendURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sign-in email is not configured"})
		return
	}

	rawToken, err := services.NewAuthLinkService().CreateSignInLink(input.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process sign-in request"})
		return
	}

	// An empty token means no matching account was found. Return the same
	// generic response without disclosing whether the account exists.
	if rawToken != "" {
		signInURL := frontendURL + "/auth/sign-in-link?token=" + url.QueryEscape(rawToken)
		body := fmt.Sprintf(
			"Use this link to sign in to Nativity Guard:\n\n%s\n\n"+
				"This link can be used once and expires in 15 minutes. "+
				"If you did not request it, you can ignore this email.",
			signInURL,
		)

		if err := SendEmail(
			input.Email,
			"Your Nativity Guard sign-in link",
			body,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send sign-in email"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": signInLinkMessage})
}

// RedeemSignInLink consumes a valid sign-in link and creates a normal
// authenticated session, including access and refresh tokens.
func (h *AuthHandler) RedeemSignInLink(c *gin.Context) {
	var input struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A sign-in token is required"})
		return
	}

	user, err := services.NewAuthLinkService().RedeemSignInLink(input.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired sign-in link"})
		return
	}

	accessToken, jti, err := h.authService.GenerateJWTWithJTI(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	now := time.Now().UTC()
	sessionExpiry := now.Add(24 * time.Hour)

	session, err := services.NewSessionService().Create(
		user.ID,
		jti,
		"email-link",
		c.Request.UserAgent(),
		c.ClientIP(),
		sessionExpiry,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	refreshToken, _, err := services.NewRefreshTokenService().Issue(user.ID, &session.ID)
	if err != nil {
		_ = services.NewSessionService().RevokeAll(user.ID, "sign_in_link_error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":        accessToken,
		"refreshToken": refreshToken,
		"user": gin.H{
			"id":        user.ID,
			"email":     user.Email,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
			"role":      user.Role,
		},
	})
}
