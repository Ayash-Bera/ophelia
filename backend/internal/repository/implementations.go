package repository

import (
	"time"

	"github.com/Ayash-Bera/ophelia/backend/internal/models"
	"gorm.io/gorm"
)

// SearchQueryRepositoryImpl implements SearchQueryRepository
type SearchQueryRepositoryImpl struct {
	db *gorm.DB
}

func NewSearchQueryRepository(db *gorm.DB) models.SearchQueryRepository {
	return &SearchQueryRepositoryImpl{db: db}
}

func (r *SearchQueryRepositoryImpl) Create(query *models.SearchQuery) error {
	return r.db.Create(query).Error
}

func (r *SearchQueryRepositoryImpl) GetByID(id uint) (*models.SearchQuery, error) {
	var query models.SearchQuery
	err := r.db.Preload("Feedback").First(&query, id).Error
	if err != nil {
		return nil, err
	}
	return &query, nil
}

func (r *SearchQueryRepositoryImpl) GetBySession(session string) ([]models.SearchQuery, error) {
	var queries []models.SearchQuery
	err := r.db.Where("user_session = ?", session).
		Order("search_timestamp DESC").
		Find(&queries).Error
	return queries, err
}

func (r *SearchQueryRepositoryImpl) GetRecentSearches(limit int) ([]models.SearchQuery, error) {
	var queries []models.SearchQuery
	err := r.db.Order("search_timestamp DESC").
		Limit(limit).
		Find(&queries).Error
	return queries, err
}

func (r *SearchQueryRepositoryImpl) UpdateClickedResult(id uint, resultID string) error {
	return r.db.Model(&models.SearchQuery{}).
		Where("id = ?", id).
		Update("clicked_result_id", resultID).Error
}

func (r *SearchQueryRepositoryImpl) GetSearchAnalytics(from, to time.Time) ([]models.SearchAnalytics, error) {
	var analytics []models.SearchAnalytics
	err := r.db.Where("date_hour BETWEEN ? AND ?", from, to).
		Order("date_hour").
		Find(&analytics).Error
	return analytics, err
}

// ContentMetadataRepositoryImpl implements ContentMetadataRepository
type ContentMetadataRepositoryImpl struct {
	db *gorm.DB
}

func NewContentMetadataRepository(db *gorm.DB) models.ContentMetadataRepository {
	return &ContentMetadataRepositoryImpl{db: db}
}

func (r *ContentMetadataRepositoryImpl) Create(content *models.ContentMetadata) error {
	return r.db.Create(content).Error
}

func (r *ContentMetadataRepositoryImpl) GetByID(id uint) (*models.ContentMetadata, error) {
	var content models.ContentMetadata
	err := r.db.Preload("Sections").First(&content, id).Error
	if err != nil {
		return nil, err
	}
	return &content, nil
}

func (r *ContentMetadataRepositoryImpl) GetByTitle(title string) (*models.ContentMetadata, error) {
	var content models.ContentMetadata
	err := r.db.Preload("Sections").
		Where("wiki_page_title = ?", title).
		First(&content).Error
	if err != nil {
		return nil, err
	}
	return &content, nil
}

func (r *ContentMetadataRepositoryImpl) GetAll() ([]models.ContentMetadata, error) {
	var contents []models.ContentMetadata
	err := r.db.Preload("Sections").Find(&contents).Error
	return contents, err
}

func (r *ContentMetadataRepositoryImpl) GetActive() ([]models.ContentMetadata, error) {
	var contents []models.ContentMetadata
	err := r.db.Where("is_active = ?", true).
		Preload("Sections").
		Find(&contents).Error
	return contents, err
}

func (r *ContentMetadataRepositoryImpl) Update(content *models.ContentMetadata) error {
	return r.db.Save(content).Error
}

func (r *ContentMetadataRepositoryImpl) UpdateCrawlStatus(id uint, status string) error {
	return r.db.Model(&models.ContentMetadata{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"crawl_status":  status,
			"last_crawled": time.Now(),
		}).Error
}

func (r *ContentMetadataRepositoryImpl) GetByCrawlStatus(status string) ([]models.ContentMetadata, error) {
	var contents []models.ContentMetadata
	err := r.db.Where("crawl_status = ?", status).
		Find(&contents).Error
	return contents, err
}

func (r *ContentMetadataRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.ContentMetadata{}, id).Error
}

// UserFeedbackRepositoryImpl implements UserFeedbackRepository
type UserFeedbackRepositoryImpl struct {
	db *gorm.DB
}

func NewUserFeedbackRepository(db *gorm.DB) models.UserFeedbackRepository {
	return &UserFeedbackRepositoryImpl{db: db}
}

func (r *UserFeedbackRepositoryImpl) Create(feedback *models.UserFeedback) error {
	return r.db.Create(feedback).Error
}

func (r *UserFeedbackRepositoryImpl) GetByQueryID(queryID uint) ([]models.UserFeedback, error) {
	var feedback []models.UserFeedback
	err := r.db.Where("query_id = ?", queryID).
		Find(&feedback).Error
	return feedback, err
}

func (r *UserFeedbackRepositoryImpl) GetByType(feedbackType string) ([]models.UserFeedback, error) {
	var feedback []models.UserFeedback
	err := r.db.Where("feedback_type = ?", feedbackType).
		Preload("Query").
		Find(&feedback).Error
	return feedback, err
}

func (r *UserFeedbackRepositoryImpl) GetRecentFeedback(limit int) ([]models.UserFeedback, error) {
	var feedback []models.UserFeedback
	err := r.db.Order("created_at DESC").
		Limit(limit).
		Preload("Query").
		Find(&feedback).Error
	return feedback, err
}

// PopularQueryRepositoryImpl implements PopularQueryRepository
type PopularQueryRepositoryImpl struct {
	db *gorm.DB
}

func NewPopularQueryRepository(db *gorm.DB) models.PopularQueryRepository {
	return &PopularQueryRepositoryImpl{db: db}
}

func (r *PopularQueryRepositoryImpl) IncrementCount(queryText string) error {
	return r.db.Exec(`
		INSERT INTO popular_queries (query_text, search_count, last_searched, created_at, updated_at)
		VALUES (?, 1, NOW(), NOW(), NOW())
		ON CONFLICT (query_text) 
		DO UPDATE SET 
			search_count = popular_queries.search_count + 1,
			last_searched = NOW(),
			updated_at = NOW()
	`, queryText).Error
}

func (r *PopularQueryRepositoryImpl) GetTop(limit int) ([]models.PopularQuery, error) {
	var queries []models.PopularQuery
	err := r.db.Order("search_count DESC").
		Limit(limit).
		Find(&queries).Error
	return queries, err
}

func (r *PopularQueryRepositoryImpl) UpdateStats(queryText string, resultsCount float64, responseTime int) error {
	return r.db.Exec(`
		UPDATE popular_queries 
		SET 
			avg_results_count = (avg_results_count * (search_count - 1) + ?) / search_count,
			avg_response_time_ms = (avg_response_time_ms * (search_count - 1) + ?) / search_count,
			updated_at = NOW()
		WHERE query_text = ?
	`, resultsCount, responseTime, queryText).Error
}

// SystemHealthRepositoryImpl implements SystemHealthRepository
type SystemHealthRepositoryImpl struct {
	db *gorm.DB
}

func NewSystemHealthRepository(db *gorm.DB) models.SystemHealthRepository {
	return &SystemHealthRepositoryImpl{db: db}
}

func (r *SystemHealthRepositoryImpl) UpdateServiceHealth(serviceName, status string, responseTime int, errorMsg string) error {
	return r.db.Exec(`
		INSERT INTO system_health (service_name, status, response_time_ms, error_message, checked_at)
		VALUES (?, ?, ?, ?, NOW())
	`, serviceName, status, responseTime, errorMsg).Error
}

func (r *SystemHealthRepositoryImpl) GetServiceHealth(serviceName string) (*models.SystemHealth, error) {
	var health models.SystemHealth
	err := r.db.Where("service_name = ?", serviceName).
		Order("checked_at DESC").
		First(&health).Error
	if err != nil {
		return nil, err
	}
	return &health, nil
}

func (r *SystemHealthRepositoryImpl) GetAllServicesHealth() ([]models.SystemHealth, error) {
	var health []models.SystemHealth
	err := r.db.Raw(`
		SELECT DISTINCT ON (service_name) *
		FROM system_health
		ORDER BY service_name, checked_at DESC
	`).Scan(&health).Error
	return health, err
}

func (r *SystemHealthRepositoryImpl) GetUnhealthyServices() ([]models.SystemHealth, error) {
	var health []models.SystemHealth
	err := r.db.Raw(`
		SELECT DISTINCT ON (service_name) *
		FROM system_health
		WHERE status != 'healthy'
		ORDER BY service_name, checked_at DESC
	`).Scan(&health).Error
	return health, err
}

// RepositoryManager bundles all repositories
type RepositoryManager struct {
	SearchQuery     models.SearchQueryRepository
	ContentMetadata models.ContentMetadataRepository
	UserFeedback    models.UserFeedbackRepository
	PopularQuery    models.PopularQueryRepository
	SystemHealth    models.SystemHealthRepository
}

func NewRepositoryManager(db *gorm.DB) *RepositoryManager {
	return &RepositoryManager{
		SearchQuery:     NewSearchQueryRepository(db),
		ContentMetadata: NewContentMetadataRepository(db),
		UserFeedback:    NewUserFeedbackRepository(db),
		PopularQuery:    NewPopularQueryRepository(db),
		SystemHealth:    NewSystemHealthRepository(db),
	}
}