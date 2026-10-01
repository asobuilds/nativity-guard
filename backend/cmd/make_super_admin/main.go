package main

import (
	"errors"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"security-solution/config"
	"security-solution/content"
	"security-solution/models"
)

// make_super_admin creates (or upserts) a super-admin account.
//
// Reads the password from SUPER_ADMIN_PASSWORD. Always resets the password
// to that value — so rerunning the command makes the account's password
// deterministic instead of leaving whatever was there before.
//
// Run once:
//   $env:SUPER_ADMIN_PASSWORD = "the-password"
//   go run ./cmd/make_super_admin
//   Remove-Item Env:\SUPER_ADMIN_PASSWORD
func main() {
	const (
		email     = "eyungemmanuel@gmail.com"
		phone     = "08128162561"
		firstName = "Emmanuel"
		lastName  = "Eyung"
	)

	if err := godotenv.Load(".env"); err != nil {
		log.Println("using environment variables")
	}

	password := os.Getenv("SUPER_ADMIN_PASSWORD")
	if password == "" {
		log.Fatal("SUPER_ADMIN_PASSWORD is not set")
	}
	if len(password) < 8 {
		log.Fatal("SUPER_ADMIN_PASSWORD must be at least 8 characters")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	// Show which host we're about to write to, with the password masked.
	log.Printf("connecting to: %s", maskDSN(dsn))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("database connection failed: ", err)
	}
	config.DB = db

	log.Println("running auto-migration for the models this command touches...")
	if err := db.AutoMigrate(
		&models.User{},
		&models.TermsDocument{},
		&models.TermsAcceptance{},
		&models.IdentityVerification{},
	); err != nil {
		log.Fatal("auto-migration failed: ", err)
	}
	log.Println("auto-migration complete")

	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	normalizedPhone := strings.TrimSpace(phone)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("bcrypt: ", err)
	}

	var user models.User
	err = db.Unscoped().Where("LOWER(email) = ?", normalizedEmail).First(&user).Error

	switch {
	case err == nil:
		log.Printf("user exists (id=%s, role=%s, status=%s) — updating", user.ID, user.Role, user.Status)
		updates := map[string]interface{}{
			"first_name":     firstName,
			"last_name":      lastName,
			"password":       string(hash), // always reset the password
			"role":           "super_admin",
			"is_super_admin": true,
			"status":         "active",
			"deleted_at":     nil, // undelete if it was soft-deleted
		}
		if user.Phone == "" && normalizedPhone != "" {
			updates["phone"] = normalizedPhone
		}
		if err := db.Unscoped().Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
			log.Fatal("failed to update user: ", err)
		}

	case errors.Is(err, gorm.ErrRecordNotFound):
		user = models.User{
			ID:           uuid.New(),
			Email:        normalizedEmail,
			Phone:        normalizedPhone,
			FirstName:    firstName,
			LastName:     lastName,
			Password:     string(hash),
			Role:         "super_admin",
			Status:       "active",
			IsSuperAdmin: true,
		}
		if err := db.Create(&user).Error; err != nil {
			log.Fatal("create user: ", err)
		}
		log.Printf("created super_admin %s (%s)", user.Email, user.ID)

	default:
		log.Fatal("query failed: ", err)
	}

	// Reload and print the final DB state so we can see exactly what landed.
	var final models.User
	if err := db.Unscoped().Where("LOWER(email) = ?", normalizedEmail).First(&final).Error; err != nil {
		log.Fatal("could not reload user: ", err)
	}

	ensureAcceptances(db, final.ID)

	passwordOK := bcrypt.CompareHashAndPassword([]byte(final.Password), []byte(password)) == nil
	log.Printf("---- FINAL DB STATE ----")
	log.Printf("id            = %s", final.ID)
	log.Printf("email         = %s", final.Email)
	log.Printf("phone         = %s", final.Phone)
	log.Printf("role          = %s", final.Role)
	log.Printf("status        = %s", final.Status)
	log.Printf("isSuperAdmin  = %v", final.IsSuperAdmin)
	log.Printf("deleted_at    = %v", final.DeletedAt)
	log.Printf("password hash verifies against the SUPPLIED password: %v", passwordOK)
	log.Println("done")
}

// maskDSN hides the password portion of a postgres URL.
func maskDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return "(unparseable)"
	}
	if u.User != nil {
		u.User = url.UserPassword(u.User.Username(), "****")
	}
	return u.String()
}

// ensureAcceptances records the current terms + privacy acceptance for the
// user. Every account on the platform must have these rows.
func ensureAcceptances(db *gorm.DB, userID uuid.UUID) {
	var docs []models.TermsDocument
	err := db.
		Where("version = ? AND is_active = ?", content.TermsVersion, true).
		Where("(kind = ? AND role = ?) OR (kind = ? AND role = ?)",
			"terms", "super_admin", "privacy", "all").
		Find(&docs).Error
	if err != nil {
		log.Printf("could not load terms documents: %v", err)
		return
	}

	now := time.Now().UTC()
	for _, d := range docs {
		var count int64
		db.Model(&models.TermsAcceptance{}).
			Where("user_id = ? AND document_id = ?", userID, d.ID).
			Count(&count)
		if count > 0 {
			continue
		}

		acc := models.TermsAcceptance{
			ID:              uuid.New(),
			UserID:          userID,
			DocumentID:      d.ID,
			DocumentKind:    d.Kind,
			DocumentRole:    d.Role,
			DocumentVersion: d.Version,
			AcceptedAt:      now,
			IPAddress:       "system",
			UserAgent:       "make_super_admin",
		}
		if err := db.Create(&acc).Error; err != nil {
			log.Printf("failed to record acceptance %s/%s: %v", d.Kind, d.Role, err)
		} else {
			log.Printf("recorded acceptance: %s/%s v%s", d.Kind, d.Role, d.Version)
		}
	}
}