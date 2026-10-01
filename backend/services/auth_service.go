package services

import (
	"errors"
	"os"
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

func NewAuthService() *AuthService {
	return &AuthService{}
}

// GenerateJWT is used when a new token needs to be generated
// for an authenticated user, including controlled impersonation.
func (s *AuthService) GenerateJWT(user *models.User) (string, error) {
	return s.generateJWT(user)
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