package health

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Ayash-Bera/ophelia/backend/internal/database"
	"github.com/Ayash-Bera/ophelia/backend/internal/models"
	"github.com/Ayash-Bera/ophelia/backend/internal/repository"
	"github.com/sirupsen/logrus"
)

// HealthChecker manages health checks for all services
type HealthChecker struct {
	dbManager    *database.Manager
	cache        *database.Cache
	healthRepo   models.SystemHealthRepository
	logger       *logrus.Logger
	alchemystURL string
}

func NewHealthChecker(dbManager *database.Manager, healthRepo models.SystemHealthRepository, logger *logrus.Logger, alchemystURL string) *HealthChecker {
	return &HealthChecker{
		dbManager:    dbManager,
		cache:        database.NewCache(dbManager.Redis, logger),
		healthRepo:   healthRepo,
		logger:       logger,
		alchemystURL: alchemystURL,
	}
}

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	ResponseTime int    `json:"response_time_ms"`
	Error        string `json:"error,omitempty"`
	LastChecked  string `json:"last_checked"`
}

// OverallHealth represents the overall system health
type OverallHealth struct {
	Status   string          `json:"status"`
	Services []ServiceHealth `json:"services"`
	Uptime   string          `json:"uptime"`
}

// CheckPostgreSQL checks PostgreSQL database health
func (h *HealthChecker) CheckPostgreSQL() ServiceHealth {
	start := time.Now()
	err := h.dbManager.PingDatabase()
	responseTime := int(time.Since(start).Milliseconds())

	status := "healthy"
	errorMsg := ""
	if err != nil {
		status = "unhealthy"
		errorMsg = err.Error()
		h.logger.WithError(err).Error("PostgreSQL health check failed")
	}

	// Update health status in database
	h.healthRepo.UpdateServiceHealth("postgresql", status, responseTime, errorMsg)

	return ServiceHealth{
		Name:         "postgresql",
		Status:       status,
		ResponseTime: responseTime,
		Error:        errorMsg,
		LastChecked:  time.Now().Format(time.RFC3339),
	}
}

// CheckRedis checks Redis cache health
func (h *HealthChecker) CheckRedis() ServiceHealth {
	start := time.Now()
	err := h.dbManager.PingRedis()
	responseTime := int(time.Since(start).Milliseconds())

	status := "healthy"
	errorMsg := ""
	if err != nil {
		status = "unhealthy"
		errorMsg = err.Error()
		h.logger.WithError(err).Error("Redis health check failed")
	}

	// Update health status in database
	h.healthRepo.UpdateServiceHealth("redis", status, responseTime, errorMsg)

	return ServiceHealth{
		Name:         "redis",
		Status:       status,
		ResponseTime: responseTime,
		Error:        errorMsg,
		LastChecked:  time.Now().Format(time.RFC3339),
	}
}

// CheckAlchemyst checks Alchemyst API health
func (h *HealthChecker) CheckAlchemyst() ServiceHealth {
	start := time.Now()
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(h.alchemystURL + "/health")
	
	responseTime := int(time.Since(start).Milliseconds())
	status := "healthy"
	errorMsg := ""

	if err != nil {
		status = "unhealthy"
		errorMsg = err.Error()
	} else {
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			status = "unhealthy"
			errorMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}

	if status != "healthy" {
		h.logger.WithError(err).Error("Alchemyst health check failed")
	}

	// Update health status in database
	h.healthRepo.UpdateServiceHealth("alchemyst", status, responseTime, errorMsg)

	return ServiceHealth{
		Name:         "alchemyst",
		Status:       status,
		ResponseTime: responseTime,
		Error:        errorMsg,
		LastChecked:  time.Now().Format(time.RFC3339),
	}
}

// CheckAll performs health checks on all services
func (h *HealthChecker) CheckAll() OverallHealth {
	services := []ServiceHealth{
		h.CheckPostgreSQL(),
		h.CheckRedis(),
		h.CheckAlchemyst(),
	}

	overallStatus := "healthy"
	for _, service := range services {
		if service.Status == "unhealthy" {
			overallStatus = "unhealthy"
			break
		}
		if service.Status == "degraded" && overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}

	return OverallHealth{
		Status:   overallStatus,
		Services: services,
		Uptime:   h.getUptime(),
	}
}

// CheckCached returns cached health status if available
func (h *HealthChecker) CheckCached(ctx context.Context) (*OverallHealth, error) {
	cachedHealth, err := h.cache.GetCachedSystemHealth(ctx)
	if err != nil {
		return nil, err
	}

	services := make([]ServiceHealth, len(cachedHealth))
	overallStatus := "healthy"
	
	for i, health := range cachedHealth {
		services[i] = ServiceHealth{
			Name:         health.ServiceName,
			Status:       health.Status,
			ResponseTime: health.ResponseTimeMs,
			Error:        health.ErrorMessage,
			LastChecked:  health.CheckedAt.Format(time.RFC3339),
		}

		if health.Status == "unhealthy" {
			overallStatus = "unhealthy"
		} else if health.Status == "degraded" && overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}

	return &OverallHealth{
		Status:   overallStatus,
		Services: services,
		Uptime:   h.getUptime(),
	}, nil
}

var startTime = time.Now()

func (h *HealthChecker) getUptime() string {
	uptime := time.Since(startTime)
	return uptime.String()
}

// PeriodicHealthCheck runs health checks periodically
func (h *HealthChecker) PeriodicHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			health := h.CheckAll()
			
			// Cache the health status
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			healthModels := make([]models.SystemHealth, len(health.Services))
			for i, service := range health.Services {
				checkedAt, _ := time.Parse(time.RFC3339, service.LastChecked)
				healthModels[i] = models.SystemHealth{
					ServiceName:    service.Name,
					Status:         service.Status,
					ResponseTimeMs: service.ResponseTime,
					ErrorMessage:   service.Error,
					CheckedAt:      checkedAt,
				}
			}
			
			if err := h.cache.CacheSystemHealth(cacheCtx, healthModels, 2*interval); err != nil {
				h.logger.WithError(err).Error("Failed to cache health status")
			}
			cancel()

			h.logger.WithField("status", health.Status).Debug("Periodic health check completed")
		}
	}
}

// Migration runner
package migration

import (
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Ayash-Bera/ophelia/backend/internal/database"
	"github.com/sirupsen/logrus"
)

type Runner struct {
	dbManager *database.Manager
	logger    *logrus.Logger
}

func NewRunner(dbManager *database.Manager, logger *logrus.Logger) *Runner {
	return &Runner{
		dbManager: dbManager,
		logger:    logger,
	}
}

// RunMigrations executes all pending migrations
func (r *Runner) RunMigrations(migrationsPath string) error {
	r.logger.Info("Starting database migrations...")

	// First run GORM auto-migrations
	if err := r.dbManager.Migrate(); err != nil {
		return fmt.Errorf("GORM auto-migration failed: %w", err)
	}

	// Then run SQL migrations
	if err := r.runSQLMigrations(migrationsPath); err != nil {
		return fmt.Errorf("SQL migrations failed: %w", err)
	}

	r.logger.Info("Database migrations completed successfully")
	return nil
}

func (r *Runner) runSQLMigrations(migrationsPath string) error {
	files, err := ioutil.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var sqlFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, file.Name())
		}
	}

	sort.Strings(sqlFiles) // Ensure migrations run in order

	for _, fileName := range sqlFiles {
		if err := r.runSQLFile(filepath.Join(migrationsPath, fileName)); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", fileName, err)
		}
		r.logger.WithField("file", fileName).Info("Migration executed successfully")
	}

	return nil
}

func (r *Runner) runSQLFile(filePath string) error {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	return r.dbManager.DB.Exec(string(content)).Error
}

// Day 3 completion checker
func VerifyDay3Setup(dbManager *database.Manager, repoManager *repository.RepositoryManager, logger *logrus.Logger) error {
	logger.Info("Verifying Day 3 setup...")

	// Check database connection
	if err := dbManager.PingDatabase(); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	// Check Redis connection  
	if err := dbManager.PingRedis(); err != nil {
		return fmt.Errorf("Redis connection failed: %w", err)
	}

	// Test repository operations
	testContent := &models.ContentMetadata{
		WikiPageTitle: "test_page",
		ContentHash:   "test_hash",
		IsActive:      true,
		CrawlStatus:   "pending",
	}

	if err := repoManager.ContentMetadata.Create(testContent); err != nil {
		return fmt.Errorf("repository test failed: %w", err)
	}

	// Clean up test data
	repoManager.ContentMetadata.Delete(testContent.ID)

	logger.Info("Day 3 setup verification completed successfully!")
	return nil
}