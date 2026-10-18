package main

import (
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"security-solution/config"
	"security-solution/middleware"
	"security-solution/routes"
	"security-solution/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	config.EnforceRequiredEnv()
	config.ConnectDatabase()
	defer config.CloseDatabase()

	if err := config.AutoMigrateAll(); err != nil {
		log.Fatalf("auto-migration failed: %v", err)
	}

	// Seed the current terms documents.
	services.EnsureTermsSeeded()

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(os.Getenv("GIN_MODE"))
	}

	router := gin.Default()

	if err := router.SetTrustedProxies([]string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}); err != nil {
		log.Fatalf("failed to set trusted proxies: %v", err)
	}

	router.Use(middleware.SecurityHeaders())

	// Structured request logging.
	router.Use(middleware.StructuredLogger())

	// Audit middleware.
	router.Use(middleware.AuditMiddleware())

	// Recover from handler panics.
	router.Use(middleware.PanicRecovery())

	router.Use(cors.New(cors.Config{
		AllowOrigins:    parseAllowedOrigins(os.Getenv("ALLOWED_ORIGINS")),
		AllowOriginFunc: allowDebugLocalhost,
		AllowMethods: []string{
			"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin", "Content-Type", "Accept", "Authorization",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	routes.SetupRoutes(router)

	// Start background scheduler.
	scheduler := services.NewSchedulerService()
	scheduler.Start(1 * time.Hour)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on http://localhost:%s", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// parseAllowedOrigins splits ALLOWED_ORIGINS on commas,
// trims each entry, and drops blanks.
func parseAllowedOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}

	return out
}

// allowDebugLocalhost permits loopback origins only in debug mode.
func allowDebugLocalhost(origin string) bool {
	if gin.Mode() != gin.DebugMode {
		return false
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}

	return false
}
