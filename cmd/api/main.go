package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"go-be/config"
	"go-be/internal/handler"
	"go-be/internal/mailer"
	"go-be/internal/middleware"
	"go-be/internal/oauth"
	"go-be/internal/repository"
	"go-be/internal/service"
)

func main() {
	// Load environment
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Connect database
	db, err := config.ConnectDatabase()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	defer db.Close()

	log.Println("PostgreSQL connected successfully!")

	// Initialize Gin
	router := gin.Default()
	router.Use(middleware.LocaleMiddleware())
	router.Use(middleware.RateLimit(middleware.GlobalRateLimitConfig()))

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	loginAttemptRepo := repository.NewLoginAttemptRepository(db)
	passwordResetRepo := repository.NewPasswordResetRepository(db)

	// Initialize services
	authService := service.NewAuthService(
		userRepo,
		refreshTokenRepo,
		deviceRepo,
		loginAttemptRepo,
		passwordResetRepo,
		mailer.NewFromEnv(),
		os.Getenv("FRONTEND_URL"),
		oauth.NewDiscordClient(oauth.DiscordConfigFromEnv()),
	)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	adminHandler := handler.NewAdminHandler(userRepo, db)
	healthHandler := handler.NewHealthHandler(db)

	authRateLimit := middleware.RateLimit(middleware.AuthRateLimitConfig())

	// Auth routes
	auth := router.Group("/api/auth")
	{
		auth.POST("/register", authRateLimit, authHandler.Register)
		auth.POST("/login", authRateLimit, authHandler.Login)
		auth.POST("/discord", authRateLimit, authHandler.DiscordLogin)
		auth.POST("/refresh", authRateLimit, authHandler.Refresh)
		auth.POST("/forgot-password", authRateLimit, authHandler.ForgotPassword)
		auth.POST("/reset-password", authRateLimit, authHandler.ResetPassword)
		auth.POST("/logout", middleware.AuthMiddleware(os.Getenv("JWT_SECRET")), authHandler.Logout)
	}

	// Protected routes
	protected := router.Group("/api/users")

	protected.Use(middleware.AuthMiddleware(os.Getenv("JWT_SECRET")))
	{
		protected.GET("/profile", authHandler.GetProfile)
		protected.GET("/devices", authHandler.ListDevices)
	}

	// Admin routes
	admin := router.Group("/api/admin")
	admin.Use(middleware.AuthMiddleware(os.Getenv("JWT_SECRET")), middleware.RequireAdmin())
	{
		admin.GET("/users", adminHandler.ListUsers)
		admin.GET("/stats", adminHandler.Stats)
	}

	// Health check
	health := router.Group("/api/health")
	{
		health.GET("/", healthHandler.HealthCheck)
		health.GET("/db", healthHandler.DatabaseHealthCheck)
	}

	// Start server
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server running on port:", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Server failed:", err)
	}
}
