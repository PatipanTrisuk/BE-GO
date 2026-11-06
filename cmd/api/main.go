package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" || jwtSecret == "change-me" || len(jwtSecret) < 32 {
		log.Fatal("JWT_SECRET must be set to a strong value (min 32 chars, not the example default)")
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
	router.Use(middleware.CORS())
	router.Use(middleware.LocaleMiddleware())
	router.Use(middleware.RateLimit(middleware.GlobalRateLimitConfig()))

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	loginAttemptRepo := repository.NewLoginAttemptRepository(db)
	passwordResetRepo := repository.NewPasswordResetRepository(db)
	emailVerificationRepo := repository.NewEmailVerificationRepository(db)
	exportJobRepo := repository.NewExportJobRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)

	// Initialize services
	authService := service.NewAuthService(
		userRepo,
		refreshTokenRepo,
		deviceRepo,
		loginAttemptRepo,
		passwordResetRepo,
		emailVerificationRepo,
		mailer.NewFromEnv(),
		os.Getenv("FRONTEND_URL"),
		oauth.NewDiscordClient(oauth.DiscordConfigFromEnv()),
	)
	notificationService := service.NewNotificationService(notificationRepo)
	exportService := service.NewExportService(exportJobRepo, userRepo, notificationService, os.Getenv("EXPORT_DIR"))
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	exportService.Start(workerCtx, 2)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	adminHandler := handler.NewAdminHandler(authService, userRepo, exportService, db)
	healthHandler := handler.NewHealthHandler(db)

	authRateLimit := middleware.RateLimit(middleware.AuthRateLimitConfig())
	authMiddleware := middleware.AuthMiddleware(jwtSecret)

	// Auth routes
	auth := router.Group("/api/auth")
	{
		auth.POST("/register", authRateLimit, authHandler.Register)
		auth.POST("/login", authRateLimit, authHandler.Login)
		auth.POST("/discord", authRateLimit, authHandler.DiscordLogin)
		auth.POST("/refresh", authRateLimit, authHandler.Refresh)
		auth.POST("/forgot-password", authRateLimit, authHandler.ForgotPassword)
		auth.POST("/reset-password", authRateLimit, authHandler.ResetPassword)
		auth.POST("/verify-email", authRateLimit, authHandler.VerifyEmail)
		auth.POST("/resend-verification", authRateLimit, authHandler.ResendVerification)
		auth.POST("/logout", authMiddleware, authHandler.Logout)
	}

	// Protected routes
	protected := router.Group("/api/users")

	protected.Use(authMiddleware)
	{
		protected.GET("/profile", authHandler.GetProfile)
		protected.PATCH("/profile", authHandler.UpdateProfile)
		protected.POST("/change-password", authHandler.ChangePassword)
		protected.GET("/devices", authHandler.ListDevices)
		protected.DELETE("/devices/:id", authHandler.RevokeDevice)
		protected.POST("/sessions/revoke-all", authHandler.LogoutAllSessions)
		protected.GET("/notifications", notificationHandler.List)
		protected.GET("/notifications/unread-count", notificationHandler.UnreadCount)
		protected.POST("/notifications/read-all", notificationHandler.MarkAllRead)
		protected.POST("/notifications/:id/read", notificationHandler.MarkRead)
	}

	// Admin routes
	admin := router.Group("/api/admin")
	admin.Use(authMiddleware, middleware.RequireAdmin())
	{
		admin.GET("/users", adminHandler.ListUsers)
		admin.POST("/users/export", adminHandler.ExportUsers)
		admin.GET("/users/export/jobs/:id", adminHandler.GetExportJob)
		admin.GET("/users/export/jobs/:id/download", adminHandler.DownloadExportJob)
		admin.POST("/users", adminHandler.CreateUser)
		admin.PATCH("/users/:id", adminHandler.UpdateUser)
		admin.GET("/stats", adminHandler.Stats)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)
		admin.POST("/users/:id/restore", adminHandler.RestoreUser)
		admin.POST("/users/:id/kick", adminHandler.KickUser)
		admin.POST("/users/:id/reset-password", adminHandler.SetPassword)
		admin.POST("/users/:id/send-reset-password", adminHandler.SendResetPassword)
		admin.POST("/users/:id/verify-email", adminHandler.VerifyEmail)
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

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		log.Println("Server running on port:", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed:", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")
	workerCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown: %v", err)
	}
}
