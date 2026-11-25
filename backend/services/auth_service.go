package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/models"
)

type AuthService struct{}

var (
	ErrGoogleAccountNotFound = errors.New("google account not found")
	ErrGoogleAccountConflict = errors.New("google account is already linked")
)

const googleTokenInfoEndpoint = "https://oauth2.googleapis.com/tokeninfo"

type GoogleIdentity struct {
	Subject   string
	Email     string
	FirstName string
	LastName  string
}

type googleTokenInfo struct {
	Audience      string          `json:"aud"`
	Issuer        string          `json:"iss"`
	ExpiresAt     json.RawMessage `json:"exp"`
	Subject       string          `json:"sub"`
	Email         string          `json:"email"`
	EmailVerified json.RawMessage `json:"email_verified"`
	GivenName     string          `json:"given_name"`
	FamilyName    string          `json:"family_name"`
	Name          string          `json:"name"`
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

// VerifyGoogleIDToken resolves a Google ID token through Google's token-info
// endpoint, then independently enforces the configured audience, issuer,
// expiration and verified-email claims. Development mocks are accepted only
// when explicitly enabled outside production and use a self-contained,
// base64url-encoded token payload prefixed with "mock:".
func (s *AuthService) VerifyGoogleIDToken(ctx context.Context, idToken string) (*GoogleIdentity, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return nil, errors.New("google ID token is required")
	}
	if len(idToken) > 16*1024 {
		return nil, errors.New("invalid Google ID token")
	}

	if strings.HasPrefix(idToken, "mock:") {
		return verifyMockGoogleToken(idToken)
	}

	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	if clientID == "" {
		return nil, errors.New("google authentication is not configured")
	}

	endpoint := googleTokenInfoEndpoint
	if override := strings.TrimSpace(os.Getenv("GOOGLE_TOKENINFO_URL")); override != "" && googleDevelopmentEnvironment() {
		endpoint = override
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.New("google authentication is not configured")
	}
	query := u.Query()
	query.Set("id_token", idToken)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.New("failed to verify Google ID token")
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("failed to verify Google ID token")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, errors.New("invalid Google ID token")
	}

	var claims googleTokenInfo
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 64*1024))
	if err := decoder.Decode(&claims); err != nil {
		return nil, errors.New("invalid Google ID token")
	}
	return validateGoogleClaims(claims, clientID, time.Now().UTC())
}

func verifyMockGoogleToken(idToken string) (*GoogleIdentity, error) {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("GOOGLE_AUTH_MOCK_ENABLED")), "true") {
		return nil, errors.New("invalid Google ID token")
	}
	if !googleDevelopmentEnvironment() {
		return nil, errors.New("invalid Google ID token")
	}

	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(idToken, "mock:"))
	if err != nil {
		return nil, errors.New("invalid Google mock token")
	}
	var claims googleTokenInfo
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, errors.New("invalid Google mock token")
	}
	return validateGoogleClaims(claims, "", time.Now().UTC())
}

func googleDevelopmentEnvironment() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV"))) {
	case "development", "dev", "test", "local":
		return true
	default:
		return false
	}
}

func validateGoogleClaims(claims googleTokenInfo, expectedAudience string, now time.Time) (*GoogleIdentity, error) {
	if expectedAudience != "" && claims.Audience != expectedAudience {
		return nil, errors.New("invalid Google token audience")
	}
	if expectedAudience != "" && claims.Issuer != "accounts.google.com" && claims.Issuer != "https://accounts.google.com" {
		return nil, errors.New("invalid Google token issuer")
	}
	if expectedAudience != "" {
		exp, err := googleInt64Claim(claims.ExpiresAt)
		if err != nil || exp <= now.Unix() {
			return nil, errors.New("Google ID token has expired")
		}
	}
	verified, err := googleBoolClaim(claims.EmailVerified)
	if err != nil || !verified {
		return nil, errors.New("Google email is not verified")
	}

	subject := strings.TrimSpace(claims.Subject)
	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if subject == "" || email == "" || !strings.Contains(email, "@") {
		return nil, errors.New("Google token is missing required identity claims")
	}

	firstName := strings.TrimSpace(claims.GivenName)
	lastName := strings.TrimSpace(claims.FamilyName)
	if firstName == "" && lastName == "" {
		parts := strings.Fields(claims.Name)
		if len(parts) > 0 {
			firstName = parts[0]
			lastName = strings.Join(parts[1:], " ")
		}
	}
	if firstName == "" {
		firstName = "Google"
	}
	if lastName == "" {
		lastName = "User"
	}

	return &GoogleIdentity{
		Subject:   subject,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
	}, nil
}

func googleBoolClaim(raw json.RawMessage) (bool, error) {
	var value bool
	if err := json.Unmarshal(raw, &value); err == nil {
		return value, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return false, err
	}
	return strconv.ParseBool(text)
}

func googleInt64Claim(raw json.RawMessage) (int64, error) {
	var number json.Number
	if err := json.Unmarshal(raw, &number); err == nil {
		if value, parseErr := number.Int64(); parseErr == nil {
			return value, nil
		}
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, err
	}
	return strconv.ParseInt(text, 10, 64)
}

// FindOrLinkGoogleUser authenticates by Google's stable subject. If no subject
// match exists, a verified email may link an existing local account exactly
// once. It never replaces a different existing Google link.
func (s *AuthService) FindOrLinkGoogleUser(identity *GoogleIdentity) (*models.User, bool, error) {
	if identity == nil || strings.TrimSpace(identity.Subject) == "" || strings.TrimSpace(identity.Email) == "" {
		return nil, false, errors.New("verified Google identity is required")
	}

	var user models.User
	err := config.DB.Where("google_id = ?", identity.Subject).First(&user).Error
	if err == nil {
		if err := canStartSession(&user); err != nil {
			return nil, false, err
		}
		return &user, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	err = config.DB.Where("LOWER(email) = ?", strings.ToLower(strings.TrimSpace(identity.Email))).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, ErrGoogleAccountNotFound
	}
	if err != nil {
		return nil, false, err
	}
	if err := canStartSession(&user); err != nil {
		return nil, false, err
	}
	if user.GoogleID != nil && *user.GoogleID != identity.Subject {
		return nil, false, ErrGoogleAccountConflict
	}
	if user.GoogleID == nil {
		result := config.DB.Model(&models.User{}).
			Where("id = ? AND google_id IS NULL", user.ID).
			Update("google_id", identity.Subject)
		if result.Error != nil {
			return nil, false, ErrGoogleAccountConflict
		}
		if result.RowsAffected != 1 {
			return nil, false, ErrGoogleAccountConflict
		}
		linked := identity.Subject
		user.GoogleID = &linked
	}
	return &user, true, nil
}

func (s *AuthService) RegisterGoogleUser(identity *GoogleIdentity, phone string, dateOfBirth *time.Time, minorStatus string, consents ...models.TermsAcceptance) (*models.User, error) {
	if identity == nil {
		return nil, errors.New("verified Google identity is required")
	}
	randomPassword := make([]byte, 32)
	if _, err := rand.Read(randomPassword); err != nil {
		return nil, err
	}
	subject := identity.Subject
	user := &models.User{
		Email:       identity.Email,
		GoogleID:    &subject,
		Phone:       phone,
		FirstName:   identity.FirstName,
		LastName:    identity.LastName,
		Password:    fmt.Sprintf("google-%x", randomPassword),
		Role:        models.RoleCitizen,
		Status:      "pending",
		DateOfBirth: dateOfBirth,
		MinorStatus: minorStatus,
	}
	return s.Register(user, consents...)
}

func canStartSession(user *models.User) error {
	if user.Status == "banned" || user.Status == "suspended" {
		return errors.New("account is not active")
	}
	return nil
}

// GenerateJWT is used when a new token needs to be generated
// for an authenticated user, including controlled impersonation.
func (s *AuthService) GenerateJWT(user *models.User) (string, error) {
	return s.generateJWT(user)
}

func (s *AuthService) GenerateJWTWithJTI(user *models.User) (string, string, error) {
	return s.generateJWTWithJTI(user)
}

// Register creates a user. When consents are supplied, the user row and
// every acceptance row are inserted inside a single transaction — either
// all land, or none do. Callers that pass no consents get the original
// single-insert behavior (used by legacy call sites and unit tests).
func (s *AuthService) Register(user *models.User, consents ...models.TermsAcceptance) (*models.User, error) {
	// Normalize email to lowercase so the DB unique constraint is
	// case-insensitive at the application layer. Phone is left as-is
	// (no formatting rules yet); uniqueness is checked by the caller.
	user.Email = strings.ToLower(strings.TrimSpace(user.Email))

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(user.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}
	user.Password = string(hashedPassword)

	// No consents requested — single insert. Preserves legacy behavior.
	if len(consents) == 0 {
		if err := config.DB.Create(user).Error; err != nil {
			return nil, err
		}
		return user, nil
	}

	// Atomic path — user and all acceptance rows commit together or not at all.
	tx := config.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := tx.Create(user).Error; err != nil {
		return nil, err
	}

	for i := range consents {
		consents[i].UserID = user.ID
		if consents[i].AcceptedAt.IsZero() {
			consents[i].AcceptedAt = time.Now().UTC()
		}
		if err := tx.Create(&consents[i]).Error; err != nil {
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	committed = true
	return user, nil
}

func (s *AuthService) LoginWithJTI(identifier, password string) (string, string, *models.User, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", "", nil, errors.New("identifier is required")
	}

	var user models.User
	q := config.DB

	if strings.Contains(identifier, "@") {
		q = q.Where("LOWER(email) = ?", strings.ToLower(identifier))
	} else {
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, identifier)
		q = q.Where("phone = ?", digits)
	}

	if err := q.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", nil, errors.New("invalid credentials")
		}
		return "", "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(password),
	); err != nil {
		return "", "", nil, errors.New("invalid credentials")
	}

	// Banned and suspended accounts cannot start a new session. Pending
	// accounts are still allowed through so they can complete onboarding
	// (e.g. OTP verification), which requires an authenticated token.
	if err := canStartSession(&user); err != nil {
		return "", "", nil, err
	}

	token, jti, err := s.generateJWTWithJTI(&user)
	if err != nil {
		return "", "", nil, err
	}

	return token, jti, &user, nil
}

func (s *AuthService) Login(identifier, password string) (string, *models.User, error) {
	t, _, u, err := s.LoginWithJTI(identifier, password)
	return t, u, err
}

func (s *AuthService) generateJWT(user *models.User) (string, error) {
	token, _, err := s.generateJWTWithJTI(user)
	return token, err
}

func (s *AuthService) generateJWTWithJTI(user *models.User) (string, string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", "", errors.New("JWT_SECRET is not set")
	}

	now := time.Now()
	jti := uuid.NewString()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"role":    user.Role,
		"jti":     jti,
		"iat":     now.Unix(),
		"exp":     now.Add(24 * time.Hour).Unix(),
	})

	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}

	return signed, jti, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET is not set")
	}

	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
}

func (s *AuthService) GetUserByID(id string) (*models.User, error) {
	var user models.User

	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *AuthService) ChangePassword(userID uuid.UUID, oldPassword string, newPassword string) error {
	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		return errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashed)
	if err := config.DB.Save(&user).Error; err != nil {
		return err
	}

	tokenSvc := NewTokenService()
	_ = tokenSvc.RevokeAllForUser(userID, "password_change")
	_ = NewRefreshTokenService().RevokeAllForUser(userID)

	return nil
}
