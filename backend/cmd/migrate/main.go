package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"security-solution/config"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("Using environment variables")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("database connection failed:", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer sqlDB.Close()

	log.Println("Connected to security_platform")
	log.Println("Running schema migration...")

	config.DB = db

	if err := config.AutoMigrateAll(); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("Schema migration completed successfully.")
}