// backend/cmd/server/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ayash-Bera/ophelia/backend/internal/alchemyst"
	// "github.com/Ayash-Bera/ophelia/backend/internal/api/handlers"
	"github.com/Ayash-Bera/ophelia/backend/internal/api/handlers"
	"github.com/Ayash-Bera/ophelia/backend/internal/config"
	"github.com/Ayash-Bera/ophelia/backend/internal/database"
	"github.com/Ayash-Bera/ophelia/backend/internal/health"
	"github.com/Ayash-Bera/ophelia/backend/internal/middleware"
	"github.com/Ayash-Bera/ophelia/backend/internal/migration"
	"github.com/Ayash-Bera/ophelia/backend/internal/repository"

	"github.com/Ayash-Bera/ophelia/backend/internal/services"
	"github.com/Ayash-Bera/ophelia/backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	// Initialize logger
	logger := utils.GetLogger()
	logger.Info("Starting Arch Search API Server...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Initialize database
	dbConfig := &database.Config{
		DatabaseURL: cfg.Database.URL,
		RedisURL:    cfg.Redis.URL,
		LogLevel:    os.Getenv("LOG_LEVEL"),
	}

	dbManager, err := database.NewManager(dbConfig, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize database manager")
	}
	defer func() {
		if err := dbManager.Close(); err != nil {
			logger.WithError(err).Error("Failed to close database connections")
		}
	}()

	// Run migrations
	migrationRunner := migration.NewRunner(dbManager, logger)
	if err := migrationRunner.RunMigrations("migrations"); err != nil {
		logger.WithError(err).Fatal("Failed to run database migrations")
	}

	// Initialize repositories
	repoManager := repository.NewRepositoryManager(dbManager.DB)

	// Initialize Alchemyst client and service
	alchemystClient := alchemyst.NewClient(cfg.Alchemyst.BaseURL, cfg.Alchemyst.APIKey, logger)
	alchemystService := alchemyst.NewService(alchemystClient, logger)

	// Initialize services
	searchService := services.NewSearchService(alchemystService, repoManager, logger)

	// Initialize cache
	cache := database.NewCache(dbManager.Redis, logger)

	// Initialize handlers
	searchHandler := handlers.NewSearchHandler(searchService, repoManager, cache, logger)

	// Initialize health checker
	healthChecker := health.NewHealthChecker(dbManager, repoManager.SystemHealth, logger, cfg.Alchemyst.BaseURL)

	// Set up Gin router
	if gin.Mode() == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.RequestID())

	// Rate limiting
	rateLimiter := middleware.NewRateLimiter(100) // 100 requests per minute
	router.Use(rateLimiter.RateLimit())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoints
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// Try to get cached health first
		health, err := healthChecker.CheckCached(ctx)
		if err != nil {
			// If no cached health, run full check
			fullHealth := healthChecker.CheckAll()
			health = &fullHealth
		}

		status := http.StatusOK
		if health.Status != "healthy" {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, health)
	})

	router.GET("/health/detailed", func(c *gin.Context) {
		health := healthChecker.CheckAll()
		status := http.StatusOK
		if health.Status != "healthy" {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, health)
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Search endpoints
		v1.POST("/search", searchHandler.HandleSearch)
		v1.POST("/feedback", searchHandler.HandleFeedback)
		v1.GET("/suggestions", searchHandler.HandleSearchSuggestions)

		// Analytics endpoints (basic)
		v1.GET("/analytics", func(c *gin.Context) {
			// Simple analytics endpoint
			recentQueries, err := repoManager.SearchQuery.GetRecentSearches(10)
			if err != nil {
				logger.WithError(err).Error("Failed to get recent searches")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve analytics"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"recent_queries": recentQueries,
				"server_time":    time.Now(),
			})
		})
	}

	// Start server
	port := cfg.Server.Port
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("port", port).Info("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.WithError(err).Fatal("Server forced to shutdown")
	}

	logger.Info("Server exited gracefully")
}
