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
	"github.com/Ayash-Bera/ophelia/backend/internal/api/handlers"
	"github.com/Ayash-Bera/ophelia/backend/internal/config"
	"github.com/Ayash-Bera/ophelia/backend/internal/database"
	"github.com/Ayash-Bera/ophelia/backend/internal/health"
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

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Validate Alchemyst configuration
	if err := cfg.ValidateAlchemyst(); err != nil {
		logger.WithError(err).Fatal("Alchemyst configuration validation failed")
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
	if err := dbManager.Migrate(); err != nil {
		logger.WithError(err).Fatal("Failed to run migrations")
	}

	// Initialize repositories
	repoManager := repository.NewRepositoryManager(dbManager.DB)

	// Initialize cache
	cache := database.NewCache(dbManager.Redis, logger)

	// Initialize Alchemyst client and service
	alchemystClient := alchemyst.NewClient(cfg.Alchemyst.BaseURL, cfg.Alchemyst.APIKey, logger)
	alchemystService := alchemyst.NewService(alchemystClient, logger)

	// Initialize search service
	searchService := services.NewSearchService(alchemystService, repoManager, logger)

	// Initialize health checker
	healthChecker := health.NewHealthChecker(dbManager, repoManager.SystemHealth, logger, cfg.Alchemyst.BaseURL)

	// Initialize handlers
	searchHandler := handlers.NewSearchHandler(searchService, repoManager, cache, logger)

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin router
	r := gin.New()

	// Add middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Session-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Request logging middleware
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		
		logger.WithFields(map[string]interface{}{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"status":      c.Writer.Status(),
			"duration":    time.Since(start).Milliseconds(),
			"ip":          c.ClientIP(),
			"user_agent":  c.GetHeader("User-Agent"),
		}).Info("Request processed")
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

	// Statistics endpoints
	r.GET("/stats/db", func(c *gin.Context) {
		sqlDB, err := dbManager.DB.DB()
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get database stats", err)
			return
		}

		stats := sqlDB.Stats()
		utils.SuccessResponse(c, http.StatusOK, "Database statistics", map[string]interface{}{
			"open_connections":       stats.OpenConnections,
			"in_use":                stats.InUse,
			"idle":                  stats.Idle,
			"wait_count":            stats.WaitCount,
			"wait_duration":         stats.WaitDuration.String(),
			"max_idle_closed":       stats.MaxIdleClosed,
			"max_idle_time_closed":  stats.MaxIdleTimeClosed,
			"max_lifetime_closed":   stats.MaxLifetimeClosed,
		})
	})

	r.GET("/stats/cache", func(c *gin.Context) {
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
		// Search endpoints
		api.POST("/search", searchHandler.HandleSearch)
		api.POST("/feedback", searchHandler.HandleFeedback)
		api.GET("/suggestions", searchHandler.HandleSearchSuggestions)

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

		api.GET("/analytics/feedback", func(c *gin.Context) {
			feedback, err := repoManager.UserFeedback.GetRecentFeedback(20)
			if err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get recent feedback", err)
				return
			}
			utils.SuccessResponse(c, http.StatusOK, "Recent feedback retrieved", feedback)
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

		// Admin endpoints (in a real app, these would be protected)
		admin := api.Group("/admin")
		{
			admin.GET("/health-history", func(c *gin.Context) {
				healthData, err := repoManager.SystemHealth.GetAllServicesHealth()
				if err != nil {
					utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get health history", err)
					return
				}
				utils.SuccessResponse(c, http.StatusOK, "Health history retrieved", healthData)
			})

			admin.POST("/cache/clear", func(c *gin.Context) {
				if err := cache.ClearAllCache(c.Request.Context()); err != nil {
					utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to clear cache", err)
					return
				}
				utils.SuccessResponse(c, http.StatusOK, "Cache cleared successfully", nil)
			})
		}
	}

	// Start periodic health checks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go healthChecker.PeriodicHealthCheck(ctx, 30*time.Second)

	// Setup graceful shutdown
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
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