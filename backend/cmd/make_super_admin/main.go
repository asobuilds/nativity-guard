package main

import (
	"errors"
	"log"
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

// make_super_admin creates (or promotes) a super-admin account.
//
// Reads the password from the SUPER_ADMIN_PASSWORD environment variable so
// it is never committed to source. Idempotent: rerunning with the same
// email promotes the existing user instead of failing.
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("database connection failed: ", err)
	}
	config.DB = db

	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	normalizedPhone := strings.TrimSpace(phone)

	var user models.User
	err = db.Where("LOWER(email) = ?", normalizedEmail).First(&user).Error

	if err == nil {
		log.Printf("user exists (id=%s, current role=%s) — promoting", user.ID, user.Role)
		if err := db.Model(&user).Updates(map[string]interface{}{
			"role":            "super_admin",
			"is_super_admin":  true,
			"status":          "active",
		}).Error; err != nil {
			log.Fatal("failed to promote: ", err)
		}
		if user.Phone == "" && normalizedPhone != "" {
			_ = db.Model(&user).Update("phone", normalizedPhone)
		}
		ensureAcceptances(db, user.ID)
		log.Println("done — user promoted to super_admin")
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Fatal("query failed: ", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("bcrypt: ", err)
	}

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
	ensureAcceptances(db, user.ID)

	log.Printf("created super_admin %s (%s)", user.Email, user.ID)
	log.Println("done")
}

// ensureAcceptances records the current terms + privacy acceptance for a
// user that was created by this command. Every account on the platform
// must have these rows so the audit trail is complete.
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