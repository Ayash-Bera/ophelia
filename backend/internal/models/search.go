package models

import (
	"time"
)

type SearchQuery struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	QueryText        string    `json:"query_text" gorm:"not null"`
	UserSession      string    `json:"user_session"`
	ResultsCount     int       `json:"results_count"`
	ClickedResultID  string    `json:"clicked_result_id"`
	SearchTimestamp  time.Time `json:"search_timestamp" gorm:"default:now()"`
	ResponseTimeMs   int       `json:"response_time_ms"`
	CreatedAt        time.Time `json:"created_at" gorm:"default:now()"`
}

type UserFeedback struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	QueryID      uint      `json:"query_id"`
	FeedbackType string    `json:"feedback_type"` // 'helpful', 'not_helpful', 'partially_helpful'
	FeedbackText string    `json:"feedback_text"`
	CreatedAt    time.Time `json:"created_at" gorm:"default:now()"`
}

type ContentMetadata struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	WikiPageTitle       string    `json:"wiki_page_title" gorm:"unique;not null"`
	AlchemystContextID  string    `json:"alchemyst_context_id"`
	ErrorPatterns       []string  `json:"error_patterns" gorm:"type:text[]"`
	ContentHash         string    `json:"content_hash"`
	LastUpdated         time.Time `json:"last_updated" gorm:"default:now()"`
	CreatedAt           time.Time `json:"created_at" gorm:"default:now()"`
}