package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ayash-Bera/ophelia/backend/internal/config"
	"github.com/Ayash-Bera/ophelia/backend/internal/database"
	"github.com/Ayash-Bera/ophelia/backend/internal/health"
	"github.com/Ayash-Bera/ophelia/backend/internal/migration"
	"github.com/Ayash-Bera/ophelia/backend/internal/repository"
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

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Initialize database manager
	dbConfig := &database.Config{
		DatabaseURL: cfg.Database.URL,
		RedisURL:    cfg.Redis.URL,
		LogLevel:    os.Getenv("LOG_LEVEL"),
	}

	dbManager, err := database.NewManager(dbConfig, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize database manager")
	}
	defer dbManager.Close()

	// Run migrations
	migrationRunner := migration.NewRunner(dbManager, logger)
	if err := migrationRunner.RunMigrations("./migrations"); err != nil {
		logger.WithError(err).Fatal("Failed to run migrations")
	}

	// Initialize repositories
	repoManager := repository.NewRepositoryManager(dbManager.DB)

	// Initialize health checker
	healthChecker := health.NewHealthChecker(dbManager, repoManager.SystemHealth, logger, cfg.Alchemyst.BaseURL)

	// Verify Day 3 setup
	if err := health.VerifyDay3Setup(dbManager, repoManager, logger); err != nil {
		logger.WithError(err).Fatal("Day 3 setup verification failed")
	}

	// Initialize Gin router
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoints
	r.GET("/health", func(c *gin.Context) {
		health := healthChecker.CheckAll()
		statusCode := http.StatusOK
		if health.Status != "healthy" {
			statusCode = http.StatusServiceUnavailable
		}
		c.JSON(statusCode, health)
	})

	r.GET("/health/cached", func(c *gin.Context) {
		health, err := healthChecker.CheckCached(c.Request.Context())
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get cached health status", err)
			return
		}
		statusCode := http.StatusOK
		if health.Status != "healthy" {
			statusCode = http.StatusServiceUnavailable
		}
		c.JSON(statusCode, health)
	})

	r.GET("/health/:service", func(c *gin.Context) {
		service := c.Param("service")
		var serviceHealth health.ServiceHealth

		switch service {
		case "postgresql":
			serviceHealth = healthChecker.CheckPostgreSQL()
		case "redis":
			serviceHealth = healthChecker.CheckRedis()
		case "alchemyst":
			serviceHealth = healthChecker.CheckAlchemyst()
		default:
			utils.ErrorResponse(c, http.StatusNotFound, "Service not found", nil)
			return
		}

		statusCode := http.StatusOK
		if serviceHealth.Status != "healthy" {
			statusCode = http.StatusServiceUnavailable
		}
		c.JSON(statusCode, serviceHealth)
	})

	// Database statistics endpoint
	r.GET("/stats/db", func(c *gin.Context) {
		sqlDB, err := dbManager.DB.DB()
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get database stats", err)
			return
		}

		stats := sqlDB.Stats()
		utils.SuccessResponse(c, http.StatusOK, "Database statistics", map[string]interface{}{
			"open_connections":     stats.OpenConnections,
			"in_use":               stats.InUse,
			"idle":                 stats.Idle,
			"wait_count":           stats.WaitCount,
			"wait_duration":        stats.WaitDuration.String(),
			"max_idle_closed":      stats.MaxIdleClosed,
			"max_idle_time_closed": stats.MaxIdleTimeClosed,
			"max_lifetime_closed":  stats.MaxLifetimeClosed,
		})
	})

	// Cache statistics endpoint
	r.GET("/stats/cache", func(c *gin.Context) {
		cache := database.NewCache(dbManager.Redis, logger)
		stats, err := cache.GetCacheStats(c.Request.Context())
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get cache stats", err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Cache statistics", stats)
	})

	// API routes group
	api := r.Group("/api/v1")
	{
		// Search endpoint (placeholder for Day 4)
		api.POST("/search", func(c *gin.Context) {
			utils.SuccessResponse(c, http.StatusOK, "Search endpoint ready - implementation coming in Day 4", nil)
		})

		// Analytics endpoints
		api.GET("/analytics/popular", func(c *gin.Context) {
			queries, err := repoManager.PopularQuery.GetTop(10)
			if err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get popular queries", err)
				return
			}
			utils.SuccessResponse(c, http.StatusOK, "Popular queries retrieved", queries)
		})

		api.GET("/analytics/recent", func(c *gin.Context) {
			searches, err := repoManager.SearchQuery.GetRecentSearches(20)
			if err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get recent searches", err)
				return
			}
			utils.SuccessResponse(c, http.StatusOK, "Recent searches retrieved", searches)
		})

		// Content management endpoints
		api.GET("/content", func(c *gin.Context) {
			content, err := repoManager.ContentMetadata.GetActive()
			if err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get content", err)
				return
			}
			utils.SuccessResponse(c, http.StatusOK, "Content retrieved", content)
		})

		api.GET("/content/:title", func(c *gin.Context) {
			title := c.Param("title")
			content, err := repoManager.ContentMetadata.GetByTitle(title)
			if err != nil {
				utils.ErrorResponse(c, http.StatusNotFound, "Content not found", err)
				return
			}
			utils.SuccessResponse(c, http.StatusOK, "Content retrieved", content)
		})
	}

	// Start periodic health checks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go healthChecker.PeriodicHealthCheck(ctx, 30*time.Second)

	// Setup graceful shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("port", cfg.Server.Port).Info("Server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Fatal("Server forced to shutdown")
	}

	logger.Info("Server exited")
}
