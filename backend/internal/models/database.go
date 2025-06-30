package models

// GORM models

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// StringArray for PostgreSQL array support
type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "{}", nil
	}
	return fmt.Sprintf("{%s}", strings.Join(s, ",")), nil
}

func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = StringArray{}
		return nil
	}

	switch v := value.(type) {
	case string:
		if v == "{}" {
			*s = StringArray{}
			return nil
		}
		// Remove curly braces and split
		v = strings.Trim(v, "{}")
		if v == "" {
			*s = StringArray{}
			return nil
		}
		*s = StringArray(strings.Split(v, ","))
	case []byte:
		return s.Scan(string(v))
	default:
		return fmt.Errorf("cannot scan %T into StringArray", value)
	}
	return nil
}

// Base model with common fields
type BaseModel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SearchQuery represents search analytics
type SearchQuery struct {
	BaseModel
	QueryText       string    `json:"query_text" gorm:"not null"`
	UserSession     string    `json:"user_session"`
	ResultsCount    int       `json:"results_count" gorm:"default:0"`
	ClickedResultID *string   `json:"clicked_result_id"`
	SearchTimestamp time.Time `json:"search_timestamp" gorm:"default:NOW()"`
	ResponseTimeMs  int       `json:"response_time_ms"`
	UserAgent       string    `json:"user_agent"`
	IPAddress       string    `json:"ip_address" gorm:"type:inet"`

	// Associations
	Feedback []UserFeedback `json:"feedback" gorm:"foreignKey:QueryID"`
}

// UserFeedback represents user feedback on search results
type UserFeedback struct {
	BaseModel
	QueryID      uint   `json:"query_id" gorm:"not null"`
	FeedbackType string `json:"feedback_type" gorm:"not null;check:feedback_type IN ('helpful','not_helpful','partially_helpful')"`
	FeedbackText string `json:"feedback_text"`
	UserSession  string `json:"user_session"`

	// Associations
	Query SearchQuery `json:"query" gorm:"foreignKey:QueryID"`
}

// ContentMetadata represents cached wiki page metadata
type ContentMetadata struct {
	BaseModel
	WikiPageTitle      string      `json:"wiki_page_title" gorm:"unique;not null"`
	AlchemystContextID *string     `json:"alchemyst_context_id"`
	ErrorPatterns      StringArray `json:"error_patterns" gorm:"type:text[]"`
	ContentHash        string      `json:"content_hash"`
	PageURL            string      `json:"page_url"`
	ContentType        string      `json:"content_type" gorm:"default:'wiki_page'"`
	LastCrawled        *time.Time  `json:"last_crawled"`
	LastUpdated        time.Time   `json:"last_updated" gorm:"default:NOW()"`
	IsActive           bool        `json:"is_active" gorm:"default:true"`
	CrawlStatus        string      `json:"crawl_status" gorm:"default:'pending';check:crawl_status IN ('pending','crawling','completed','failed')"`
	WordCount          int         `json:"word_count"`
	SectionCount       int         `json:"section_count"`

	// Associations
	Sections []WikiSection `json:"sections" gorm:"foreignKey:ContentMetadataID"`
}

// WikiSection represents individual sections of wiki pages
type WikiSection struct {
	BaseModel
	ContentMetadataID  uint        `json:"content_metadata_id" gorm:"not null"`
	SectionTitle       string      `json:"section_title" gorm:"not null"`
	SectionContent     string      `json:"section_content" gorm:"not null"`
	SectionOrder       int         `json:"section_order" gorm:"not null"`
	AlchemystContextID *string     `json:"alchemyst_context_id"`
	ErrorPatterns      StringArray `json:"error_patterns" gorm:"type:text[]"`

	// Associations
	ContentMetadata ContentMetadata `json:"content_metadata" gorm:"foreignKey:ContentMetadataID"`
}

// SearchAnalytics represents hourly search performance metrics
type SearchAnalytics struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	DateHour           time.Time `json:"date_hour" gorm:"unique;not null"`
	TotalSearches      int       `json:"total_searches" gorm:"default:0"`
	AvgResponseTimeMs  int       `json:"avg_response_time_ms" gorm:"default:0"`
	SuccessfulSearches int       `json:"successful_searches" gorm:"default:0"`
	FailedSearches     int       `json:"failed_searches" gorm:"default:0"`
	UniqueSessions     int       `json:"unique_sessions" gorm:"default:0"`
	CreatedAt          time.Time `json:"created_at"`
}

// PopularQuery represents frequently searched terms
type PopularQuery struct {
	BaseModel
	QueryText         string    `json:"query_text" gorm:"unique;not null"`
	SearchCount       int       `json:"search_count" gorm:"default:1"`
	AvgResultsCount   float64   `json:"avg_results_count" gorm:"type:decimal(5,2);default:0"`
	AvgResponseTimeMs int       `json:"avg_response_time_ms" gorm:"default:0"`
	LastSearched      time.Time `json:"last_searched" gorm:"default:NOW()"`
}

// SystemHealth represents service health monitoring
type SystemHealth struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	ServiceName    string    `json:"service_name" gorm:"not null"`
	Status         string    `json:"status" gorm:"not null;check:status IN ('healthy','degraded','unhealthy')"`
	ResponseTimeMs int       `json:"response_time_ms"`
	ErrorMessage   string    `json:"error_message"`
	CheckedAt      time.Time `json:"checked_at" gorm:"default:NOW()"`
}

// Database interfaces for repository pattern
type SearchQueryRepository interface {
	Create(query *SearchQuery) error
	GetByID(id uint) (*SearchQuery, error)
	GetBySession(session string) ([]SearchQuery, error)
	GetRecentSearches(limit int) ([]SearchQuery, error)
	UpdateClickedResult(id uint, resultID string) error
	GetSearchAnalytics(from, to time.Time) ([]SearchAnalytics, error)
}

type ContentMetadataRepository interface {
	Create(content *ContentMetadata) error
	GetByID(id uint) (*ContentMetadata, error)
	GetByTitle(title string) (*ContentMetadata, error)
	GetAll() ([]ContentMetadata, error)
	GetActive() ([]ContentMetadata, error)
	Update(content *ContentMetadata) error
	UpdateCrawlStatus(id uint, status string) error
	GetByCrawlStatus(status string) ([]ContentMetadata, error)
	Delete(id uint) error
}

type UserFeedbackRepository interface {
	Create(feedback *UserFeedback) error
	GetByQueryID(queryID uint) ([]UserFeedback, error)
	GetByType(feedbackType string) ([]UserFeedback, error)
	GetRecentFeedback(limit int) ([]UserFeedback, error)
}

type PopularQueryRepository interface {
	IncrementCount(queryText string) error
	GetTop(limit int) ([]PopularQuery, error)
	UpdateStats(queryText string, resultsCount float64, responseTime int) error
}

type SystemHealthRepository interface {
	UpdateServiceHealth(serviceName, status string, responseTime int, errorMsg string) error
	GetServiceHealth(serviceName string) (*SystemHealth, error)
	GetAllServicesHealth() ([]SystemHealth, error)
	GetUnhealthyServices() ([]SystemHealth, error)
}

// TableName methods for custom table names
func (SearchQuery) TableName() string     { return "search_queries" }
func (UserFeedback) TableName() string    { return "user_feedback" }
func (ContentMetadata) TableName() string { return "content_metadata" }
func (WikiSection) TableName() string     { return "wiki_sections" }
func (SearchAnalytics) TableName() string { return "search_analytics" }
func (PopularQuery) TableName() string    { return "popular_queries" }
func (SystemHealth) TableName() string    { return "system_health" }

// Model validation methods
func (sq *SearchQuery) Validate() error {
	if sq.QueryText == "" {
		return fmt.Errorf("query text is required")
	}
	if sq.ResponseTimeMs < 0 {
		return fmt.Errorf("response time cannot be negative")
	}
	return nil
}

func (uf *UserFeedback) Validate() error {
	if uf.QueryID == 0 {
		return fmt.Errorf("query ID is required")
	}
	validTypes := map[string]bool{
		"helpful":           true,
		"not_helpful":       true,
		"partially_helpful": true,
	}
	if !validTypes[uf.FeedbackType] {
		return fmt.Errorf("invalid feedback type: %s", uf.FeedbackType)
	}
	return nil
}

func (cm *ContentMetadata) Validate() error {
	if cm.WikiPageTitle == "" {
		return fmt.Errorf("wiki page title is required")
	}
	validStatuses := map[string]bool{
		"pending":   true,
		"crawling":  true,
		"completed": true,
		"failed":    true,
	}
	if !validStatuses[cm.CrawlStatus] {
		return fmt.Errorf("invalid crawl status: %s", cm.CrawlStatus)
	}
	return nil
}

// GORM hooks
func (sq *SearchQuery) BeforeCreate(tx *gorm.DB) error {
	return sq.Validate()
}

func (uf *UserFeedback) BeforeCreate(tx *gorm.DB) error {
	return uf.Validate()
}

func (cm *ContentMetadata) BeforeCreate(tx *gorm.DB) error {
	return cm.Validate()
}

func (cm *ContentMetadata) BeforeUpdate(tx *gorm.DB) error {
	return cm.Validate()
}
