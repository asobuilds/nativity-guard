package handlers

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"security-solution/config"
	"security-solution/content"
	"security-solution/models"
	"security-solution/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: services.NewAuthService(),
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input struct {
		Email                 string `json:"email" binding:"required,email"`
		Phone                 string `json:"phone" binding:"required"`
		FirstName             string `json:"firstName" binding:"required"`
		LastName              string `json:"lastName" binding:"required"`
		Password              string `json:"password" binding:"required,min=6"`
		Role                  string `json:"role"`
		DateOfBirth           string `json:"dateOfBirth" binding:"required"`
		TermsVersion          string `json:"termsVersion"`
		ConsentTerms          *bool  `json:"consentTerms"`
		ConsentDataProcessing *bool  `json:"consentDataProcessing"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email, phone, firstName, lastName, and password are required"})
		return
	}

	// Terms consent is enforced unless TERMS_CONSENT_ENFORCED is explicitly
	// set to "false". Absent variable = enforced (safe default).
	enforced := !strings.EqualFold(os.Getenv("TERMS_CONSENT_ENFORCED"), "false")

	var acceptanceRows []models.TermsAcceptance
	if enforced {
		// Gate 1 — both consents must be present and true.
		if input.ConsentTerms == nil || !*input.ConsentTerms {
			c.JSON(http.StatusBadRequest, gin.H{"error": "terms_consent_required"})
			return
		}
		if input.ConsentDataProcessing == nil || !*input.ConsentDataProcessing {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data_processing_consent_required"})
			return
		}

		// Gate 2 — version must be present.
		if strings.TrimSpace(input.TermsVersion) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "terms_version_required"})
			return
		}

		// Gate 3 — version must match the currently-published version.
		if input.TermsVersion != content.TermsVersion {
			c.JSON(http.StatusConflict, gin.H{
				"error":          "terms_version_stale",
				"currentVersion": content.TermsVersion,
			})
			return
		}
	}

	parsedDOB, minorStatus, err := validateDOB(input.DateOfBirth)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Uniqueness check — same behavior as before.
	normalizedEmail := strings.ToLower(strings.TrimSpace(input.Email))
	normalizedPhone := strings.TrimSpace(input.Phone)
	var dupCount int64
	if err := config.DB.Model(&models.User{}).
		Where("LOWER(email) = ? OR phone = ?", normalizedEmail, normalizedPhone).
		Count(&dupCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify account uniqueness"})
		return
	}
	if dupCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "An account with that email or phone number already exists"})
		return
	}
	if normalizedPhone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone number is required"})
		return
	}

	if enforced {
		// Gate 4 — both active documents must exist for the current version.
		// Any miss = fail closed: no user, no acceptance rows, 500.
		var docs []models.TermsDocument
		if err := config.DB.
			Where("version = ? AND is_active = ?", content.TermsVersion, true).
			Where("(kind = ? AND role = ?) OR (kind = ? AND role = ?)",
				"terms", "citizen", "privacy", "all").
			Find(&docs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "terms_unavailable"})
			return
		}
		var termsDoc, privacyDoc *models.TermsDocument
		for i := range docs {
			switch {
			case docs[i].Kind == "terms" && docs[i].Role == "citizen":
				termsDoc = &docs[i]
			case docs[i].Kind == "privacy" && docs[i].Role == "all":
				privacyDoc = &docs[i]
			}
		}
		if termsDoc == nil || privacyDoc == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "terms_unavailable"})
			return
		}

		ip := clientIPForConsent(c)
		ua := clampStr(c.Request.UserAgent(), 400)
		now := time.Now().UTC()

		acceptanceRows = []models.TermsAcceptance{
			{
				DocumentID:      termsDoc.ID,
				DocumentKind:    "terms",
				DocumentRole:    "citizen",
				DocumentVersion: content.TermsVersion,
				AcceptedAt:      now,
				IPAddress:       clampStr(ip, 64),
				UserAgent:       ua,
			},
			{
				DocumentID:      privacyDoc.ID,
				DocumentKind:    "privacy",
				DocumentRole:    "all",
				DocumentVersion: content.TermsVersion,
				AcceptedAt:      now,
				IPAddress:       clampStr(ip, 64),
				UserAgent:       ua,
			},
		}
	}

	// Public registration always creates citizens.
	// Privileged roles should be assigned by an administrator.
	user := &models.User{
		Email:       normalizedEmail,
		Phone:       normalizedPhone,
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		Password:    input.Password,
		Role:        "citizen",
		DateOfBirth: &parsedDOB,
		MinorStatus: minorStatus,
	}

	createdUser, err := h.authService.Register(user, acceptanceRows...)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user": gin.H{
			"id":        createdUser.ID,
			"email":     createdUser.Email,
			"firstName": createdUser.FirstName,
			"lastName":  createdUser.LastName,
			"role":      createdUser.Role,
		},
	})
}

// clientIPForConsent returns the client's IP, preferring the first hop of
// X-Forwarded-For when present (Render and most proxies set this).
func clientIPForConsent(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	return c.ClientIP()
}

// clampStr bounds a string to n bytes.
func clampStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input struct {
		Identifier string `json:"identifier"`
		Email      string `json:"email"`
		Password   string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Accept either "identifier" or legacy "email" field.
	ident := strings.TrimSpace(input.Identifier)
	if ident == "" {
		ident = strings.TrimSpace(input.Email)
	}
	if ident == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "identifier (email or phone) is required"})
		return
	}

	token, jti, user, err := h.authService.LoginWithJTI(ident, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	sessionSvc := services.NewSessionService()
	session, err := sessionSvc.Create(
		user.ID,
		jti,
		"",
		c.GetHeader("User-Agent"),
		c.ClientIP(),
		expiresAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	refreshSvc := services.NewRefreshTokenService()
	refreshRaw, _, err := refreshSvc.Issue(user.ID, &session.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to issue refresh token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":        token,
		"refreshToken": refreshRaw,
		"user": gin.H{
			"id":        user.ID,
			"email":     user.Email,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
			"role":      user.Role,
		},
	})
}

// validateDOB parses and validates the registration date-of-birth, reporting
// the exact age-threshold errors required by the age gate. Returns the parsed
// time, the computed minorStatus ("minor" | "adult"), and an error.
func validateDOB(dobStr string) (time.Time, string, error) {
	parsed, err := time.Parse("2006-01-02", dobStr)
	if err != nil {
		return time.Time{}, "", errors.New("Invalid date of birth")
	}
	now := time.Now().UTC()
	if parsed.After(now) {
		return time.Time{}, "", errors.New("Date of birth cannot be in the future")
	}
	age := services.YearsBetween(parsed, now)
	if age > 120 {
		return time.Time{}, "", errors.New("Date of birth is not valid")
	}
	if age < 16 {
		return time.Time{}, "", errors.New("Registration refused: users under 16 are not permitted")
	}
	minorStatus := "adult"
	if age < 18 {
		minorStatus = "minor"
	}
	return parsed, minorStatus, nil
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userInterface, userExists := c.Get("user")
	jtiVal, jtiExists := c.Get("jti")
	expVal, expExists := c.Get("token_exp")

	if userExists && jtiExists && expExists {
		userObj, userOk := userInterface.(*models.User)
		jti, jtiOk := jtiVal.(string)
		expFloat, expOk := expVal.(float64)

		if userOk && jtiOk && expOk && jti != "" && expFloat > 0 {
			tokenSvc := services.NewTokenService()
			expiresAt := time.Unix(int64(expFloat), 0).UTC()
			_ = tokenSvc.Revoke(jti, userObj.ID, expiresAt, "logout")
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	user, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data"})
		return
	}

	var freshUser models.User
	if err := config.DB.First(&freshUser, "id = ?", user.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         freshUser.ID,
			"email":      freshUser.Email,
			"phone":      freshUser.Phone,
			"firstName":  freshUser.FirstName,
			"lastName":   freshUser.LastName,
			"role":       freshUser.Role,
			"unitId":     freshUser.UnitID,
			"status":     freshUser.Status,
			"createdAt":  freshUser.CreatedAt,
			"updatedAt":  freshUser.UpdatedAt,
			"avatarPath": freshUser.AvatarPath,
			"coverPath":  freshUser.CoverPath,
			"photoUrl":   freshUser.AvatarPath,
		},
	})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	var input struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authService.ChangePassword(userObj.ID, input.OldPassword, input.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed. All existing sessions have been revoked.",
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc := services.NewRefreshTokenService()
	newRaw, record, err := svc.Rotate(input.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authService.GetUserByID(record.UserID.String())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	newAccess, err := h.authService.GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to issue access token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":        newAccess,
		"refreshToken": newRaw,
	})
}

func (h *AuthHandler) ListSessions(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	svc := services.NewSessionService()
	sessions, err := svc.ListActive(userObj.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

func (h *AuthHandler) RevokeSession(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	jti := c.Param("jti")
	if jti == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jti is required"})
		return
	}

	var session models.UserSession
	if err := config.DB.Where("user_id = ? AND jti = ?", userObj.ID, jti).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	tx := config.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	now := time.Now().UTC()
	reason := "user_revoked_device"

	if err := tx.Model(&models.UserSession{}).
		Where("id = ?", session.ID).
		Updates(map[string]interface{}{
			"revoked_at":     now,
			"revoked_reason": reason,
		}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke session"})
		return
	}

	revokedToken := models.RevokedToken{
		JTI:       jti,
		UserID:    userObj.ID,
		ExpiresAt: session.ExpiresAt,
		Reason:    reason,
	}
	if err := tx.Create(&revokedToken).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
		return
	}

	if err := tx.Model(&models.RefreshToken{}).
		Where("session_id = ? AND revoked_at IS NULL", session.ID).
		Update("revoked_at", now).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke refresh token"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize session revocation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session revoked"})
}

func (h *AuthHandler) RevokeAllSessions(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	now := time.Now().UTC()
	reason := "user_revoked_all"

	tx := config.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var sessions []models.UserSession
	if err := tx.Where("user_id = ? AND revoked_at IS NULL", userObj.ID).Find(&sessions).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sessions"})
		return
	}

	if err := tx.Model(&models.UserSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userObj.ID).
		Updates(map[string]interface{}{
			"revoked_at":     now,
			"revoked_reason": reason,
		}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke sessions"})
		return
	}

	for _, s := range sessions {
		revokedToken := models.RevokedToken{
			JTI:       s.JTI,
			UserID:    userObj.ID,
			ExpiresAt: s.ExpiresAt,
			Reason:    reason,
		}
		if err := tx.Create(&revokedToken).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke tokens"})
			return
		}
	}

	if err := tx.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userObj.ID).
		Update("revoked_at", now).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke refresh tokens"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize revocation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All sessions revoked"})
}

// ForgotPassword initiates a password reset. The endpoint is public and
// rate-limited. It always returns 200 with a generic message so callers
// cannot enumerate which identifiers map to real accounts.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var input struct {
		Identifier string `json:"identifier" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc := services.NewPasswordResetService()
	_, rawToken, err := svc.RequestReset(input.Identifier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	// TODO: email delivery is not yet wired. When the identifier is an
	// email, the token is currently sent to the resolved user's PHONE
	// number (Correction 1). Replace this SMS branch with an email
	// dispatch once an email service is added.
	_ = rawToken

	c.JSON(http.StatusOK, gin.H{
		"message": "If an account with that email or phone number exists, we've sent a password reset link.",
	})
}

// ResetPassword validates a reset token and applies the new password.
// On success it marks the token used and revokes every existing session
// and refresh token so the user must re-authenticate on all devices.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var input struct {
		Token       string `json:"token" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc := services.NewPasswordResetService()
	if err := svc.ResetWithToken(input.Token, input.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successful. You can now log in with your new password.",
	})
}

// DeleteAccount soft-deletes the authenticated user's account and schedules
// a hard delete after a 30-day grace period. The user can cancel within that
// window via CancelDeletion.
func (h *AuthHandler) DeleteAccount(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	now := time.Now().UTC()
	if err := config.DB.Model(&models.User{}).
		Where("id = ?", userObj.ID).
		Updates(map[string]interface{}{
			"deleted_at":            now,
			"deletion_requested_at": now,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to schedule account deletion"})
		return
	}

	// Revoke all existing sessions so the user cannot keep using the account
	// while it is pending deletion.
	tokenSvc := services.NewTokenService()
	_ = tokenSvc.RevokeAllForUser(userObj.ID, "account_deletion")
	refreshSvc := services.NewRefreshTokenService()
	_ = refreshSvc.RevokeAllForUser(userObj.ID)
	sessionSvc := services.NewSessionService()
	_ = sessionSvc.RevokeAll(userObj.ID, "account_deletion")

	c.JSON(http.StatusOK, gin.H{
		"message":             "Account scheduled for deletion. You can cancel within 30 days.",
		"deletionScheduledAt": now,
	})
}

// CancelDeletion restores an account that was scheduled for deletion,
// provided the 30-day grace period has not yet elapsed.
func (h *AuthHandler) CancelDeletion(c *gin.Context) {
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userObj, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user"})
		return
	}

	var user models.User
	if err := config.DB.Unscoped().First(&user, "id = ?", userObj.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.DeletionRequestedAt == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No pending deletion to cancel"})
		return
	}

	graceEnd := user.DeletionRequestedAt.AddDate(0, 0, 30)
	if time.Now().UTC().After(graceEnd) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Grace period has expired; account cannot be restored"})
		return
	}

	if err := config.DB.Unscoped().Model(&models.User{}).
		Where("id = ?", userObj.ID).
		Updates(map[string]interface{}{
			"deleted_at":            nil,
			"deletion_requested_at": nil,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel deletion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Account deletion cancelled. Your account has been restored.",
	})
}