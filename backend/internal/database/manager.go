package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Ayash-Bera/ophelia/backend/internal/models"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database connection manager
type Manager struct {
	DB     *gorm.DB
	Redis  *redis.Client
	logger *logrus.Logger
}

// Database configuration
type Config struct {
	DatabaseURL string
	RedisURL    string
	LogLevel    string
}

// NewManager creates a new database manager with connection pooling
func NewManager(config *Config, logger *logrus.Logger) (*Manager, error) {
	// Configure GORM logger
	var gormLogger logger.Interface
	switch config.LogLevel {
	case "debug":
		gormLogger = logger.New(
			logger.NewGormLogger(logger),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		)
	default:
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	// Open database connection with pooling
	db, err := gorm.Open(postgres.Open(config.DatabaseURL), &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true, // Improve performance
		PrepareStmt:            true, // Cache prepared statements
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)                // Maximum idle connections
	sqlDB.SetMaxOpenConns(100)               // Maximum open connections
	sqlDB.SetConnMaxLifetime(time.Hour)      // Connection lifetime
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // Maximum idle time

	// Test database connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Connect to Redis
	redisOpts, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Configure Redis connection pool
	redisOpts.PoolSize = 20
	redisOpts.MinIdleConns = 5
	redisOpts.MaxConnAge = time.Hour
	redisOpts.IdleTimeout = 30 * time.Minute
	redisOpts.IdleCheckFrequency = 30 * time.Second

	redisClient := redis.NewClient(redisOpts)

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Database and Redis connections established successfully")

	return &Manager{
		DB:     db,
		Redis:  redisClient,
		logger: logger,
	}, nil
}

// Migrate runs database migrations
func (m *Manager) Migrate() error {
	m.logger.Info("Running database migrations...")
	
	return m.DB.AutoMigrate(
		&models.SearchQuery{},
		&models.UserFeedback{},
		&models.ContentMetadata{},
		&models.WikiSection{},
		&models.SearchAnalytics{},
		&models.PopularQuery{},
		&models.SystemHealth{},
	)
}

// Close closes all database connections
func (m *Manager) Close() error {
	if m.Redis != nil {
		if err := m.Redis.Close(); err != nil {
			m.logger.WithError(err).Error("Failed to close Redis connection")
		}
	}

	if m.DB != nil {
		sqlDB, err := m.DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}

	return nil
}

// Health check methods
func (m *Manager) PingDatabase() error {
	sqlDB, err := m.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func (m *Manager) PingRedis() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.Redis.Ping(ctx).Err()
}

// Cache implementation
type Cache struct {
	client *redis.Client
	logger *logrus.Logger
}

func NewCache(client *redis.Client, logger *logrus.Logger) *Cache {
	return &Cache{
		client: client,
		logger: logger,
	}
}

// Cache key constants
const (
	SearchResultsKey    = "search:results:%s"
	ContentMetadataKey  = "content:metadata:%s"
	PopularQueriesKey   = "popular:queries"
	SystemHealthKey     = "system:health"
)

// CacheSearchResults caches search results for a query
func (c *Cache) CacheSearchResults(ctx context.Context, query string, results interface{}, expiration time.Duration) error {
	key := fmt.Sprintf(SearchResultsKey, query)
	
	data, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("failed to marshal search results: %w", err)
	}

	return c.client.Set(ctx, key, data, expiration).Err()
}

// GetCachedSearchResults retrieves cached search results
func (c *Cache) GetCachedSearchResults(ctx context.Context, query string, result interface{}) error {
	key := fmt.Sprintf(SearchResultsKey, query)
	
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(data), result)
}

// CacheContentMetadata caches content metadata
func (c *Cache) CacheContentMetadata(ctx context.Context, title string, metadata *models.ContentMetadata, expiration time.Duration) error {
	key := fmt.Sprintf(ContentMetadataKey, title)
	
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal content metadata: %w", err)
	}

	return c.client.Set(ctx, key, data, expiration).Err()
}

// GetCachedContentMetadata retrieves cached content metadata
func (c *Cache) GetCachedContentMetadata(ctx context.Context, title string) (*models.ContentMetadata, error) {
	key := fmt.Sprintf(ContentMetadataKey, title)
	
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var metadata models.ContentMetadata
	err = json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

// CachePopularQueries caches popular queries list
func (c *Cache) CachePopularQueries(ctx context.Context, queries []models.PopularQuery, expiration time.Duration) error {
	data, err := json.Marshal(queries)
	if err != nil {
		return fmt.Errorf("failed to marshal popular queries: %w", err)
	}

	return c.client.Set(ctx, PopularQueriesKey, data, expiration).Err()
}

// GetCachedPopularQueries retrieves cached popular queries
func (c *Cache) GetCachedPopularQueries(ctx context.Context) ([]models.PopularQuery, error) {
	data, err := c.client.Get(ctx, PopularQueriesKey).Result()
	if err != nil {
		return nil, err
	}

	var queries []models.PopularQuery
	err = json.Unmarshal([]byte(data), &queries)
	return queries, err
}

// CacheSystemHealth caches system health status
func (c *Cache) CacheSystemHealth(ctx context.Context, health []models.SystemHealth, expiration time.Duration) error {
	data, err := json.Marshal(health)
	if err != nil {
		return fmt.Errorf("failed to marshal system health: %w", err)
	}

	return c.client.Set(ctx, SystemHealthKey, data, expiration).Err()
}

// GetCachedSystemHealth retrieves cached system health
func (c *Cache) GetCachedSystemHealth(ctx context.Context) ([]models.SystemHealth, error) {
	data, err := c.client.Get(ctx, SystemHealthKey).Result()
	if err != nil {
		return nil, err
	}

	var health []models.SystemHealth
	err = json.Unmarshal([]byte(data), &health)
	return health, err
}

// InvalidateSearchCache removes search result cache for a query
func (c *Cache) InvalidateSearchCache(ctx context.Context, query string) error {
	key := fmt.Sprintf(SearchResultsKey, query)
	return c.client.Del(ctx, key).Err()
}

// InvalidateContentCache removes content metadata cache
func (c *Cache) InvalidateContentCache(ctx context.Context, title string) error {
	key := fmt.Sprintf(ContentMetadataKey, title)
	return c.client.Del(ctx, key).Err()
}

// ClearAllCache clears all cache data
func (c *Cache) ClearAllCache(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

// Cache statistics
func (c *Cache) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	info := c.client.Info(ctx, "stats").Val()
	
	stats := map[string]interface{}{
		"keyspace_hits":   c.extractStat(info, "keyspace_hits"),
		"keyspace_misses": c.extractStat(info, "keyspace_misses"),
		"used_memory":     c.extractStat(info, "used_memory"),
		"connected_clients": c.extractStat(info, "connected_clients"),
	}

	return stats, nil
}

func (c *Cache) extractStat(info, key string) string {
	// Simple stat extraction - could be improved with regex
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, key+":") {
			return strings.TrimPrefix(line, key+":")
		}
	}
	return "0"
}