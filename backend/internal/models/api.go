package models

type SearchRequest struct {
	Query string `json:"query" binding:"required"`
}

type SearchResponse struct {
	Results      []SearchResult `json:"results"`
	Total        int            `json:"total"`
	ResponseTime int            `json:"response_time_ms"`
}

type SearchResult struct {
	ContextID   string  `json:"context_id"`
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	URL         string  `json:"url"`
	Score       float64 `json:"score"`
	Relevance   string  `json:"relevance"`
}

type FeedbackRequest struct {
	QueryID      uint   `json:"query_id" binding:"required"`
	FeedbackType string `json:"feedback_type" binding:"required"`
	FeedbackText string `json:"feedback_text"`
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}