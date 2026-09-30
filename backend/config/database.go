package config

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDatabase() {
	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	var err error

	DB, err = gorm.Open(
		postgres.Open(dsn),
		&gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		},
	)
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("Error getting database instance:", err)
	}

        sqlDB.SetMaxIdleConns(5)
        sqlDB.SetMaxOpenConns(20)
        sqlDB.SetConnMaxLifetime(30 * time.Minute)
        sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Error pinging database:", err)
	}

	log.Println("Database connected successfully")
}

func CloseDatabase() {
	if DB == nil {
		return
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Println("Error getting database instance:", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		log.Println("Error closing database:", err)
	}
}
