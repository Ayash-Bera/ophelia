// backend/internal/api/handlers/search.go
package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Ayash-Bera/ophelia/backend/internal/database"
	"github.com/Ayash-Bera/ophelia/backend/internal/models"
	"github.com/Ayash-Bera/ophelia/backend/internal/repository"
	"github.com/Ayash-Bera/ophelia/backend/internal/services"
	"github.com/Ayash-Bera/ophelia/backend/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type SearchHandler struct {
	searchService   *services.SearchService
	repoManager     *repository.RepositoryManager
	cache           *database.Cache
	logger          *logrus.Logger
}

func NewSearchHandler(
	searchService *services.SearchService,
	repoManager *repository.RepositoryManager,
	cache *database.Cache,
	logger *logrus.Logger,
) *SearchHandler {
	return &SearchHandler{
		searchService: searchService,
		repoManager:   repoManager,
		cache:         cache,
		logger:        logger,
	}
}

// HandleSearch processes search requests
func (h *SearchHandler) HandleSearch(c *gin.Context) {
	startTime := time.Now()
	
	var req models.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid search request")
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// Validate and sanitize query
	query := strings.TrimSpace(req.Query)
	if query == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Query cannot be empty", nil)
		return
	}

	if len(query) > 2000 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Query too long (max 2000 characters)", nil)
		return
	}

	// Get user session for analytics
	userSession := h.getUserSession(c)
	
	h.logger.WithFields(logrus.Fields{
		"query":        query,
		"user_session": userSession,
		"user_agent":   c.GetHeader("User-Agent"),
		"ip_address":   c.ClientIP(),
	}).Info("Processing search request")

	// Check cache first
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var results []models.SearchResult
	// var err error
	
	cacheKey := h.generateCacheKey(query)
	cached := &models.SearchResponse{}
	
	if err := h.cache.GetCachedSearchResults(ctx, cacheKey, cached); err == nil {
		h.logger.Debug("Search results served from cache")
		results = cached.Results
	} else {
		// Cache miss - perform search
		h.logger.Debug("Cache miss - performing search")
		results, err = h.searchService.SearchForSolution(ctx, query)
		if err != nil {
			h.logger.WithError(err).Error("Search failed")
			h.trackSearchQuery(userSession, query, 0, time.Since(startTime), c)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Search failed", err)
			return
		}

		// Cache results for 5 minutes
		searchResp := &models.SearchResponse{
			Results:      results,
			Total:        len(results),
			ResponseTime: int(time.Since(startTime).Milliseconds()),
		}
		
		if err := h.cache.CacheSearchResults(ctx, cacheKey, searchResp, 5*time.Minute); err != nil {
			h.logger.WithError(err).Warn("Failed to cache search results")
		}
	}

	responseTime := time.Since(startTime)
	
	// Track analytics
	go h.trackSearchQuery(userSession, query, len(results), responseTime, c)
	go h.updatePopularQueries(query, len(results), responseTime)

	response := models.SearchResponse{
		Results:      results,
		Total:        len(results),
		ResponseTime: int(responseTime.Milliseconds()),
	}

	h.logger.WithFields(logrus.Fields{
		"results_count": len(results),
		"response_time": responseTime.Milliseconds(),
	}).Info("Search completed successfully")

	utils.SuccessResponse(c, http.StatusOK, "Search completed", response)
}

// HandleFeedback processes user feedback on search results
func (h *SearchHandler) HandleFeedback(c *gin.Context) {
	var req models.FeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid feedback format", err)
		return
	}

	// Validate feedback type
	validTypes := map[string]bool{
		"helpful":           true,
		"not_helpful":       true,
		"partially_helpful": true,
	}
	
	if !validTypes[req.FeedbackType] {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid feedback type", nil)
		return
	}

	// Create feedback record
	feedback := &models.UserFeedback{
		QueryID:      req.QueryID,
		FeedbackType: req.FeedbackType,
		FeedbackText: req.FeedbackText,
		UserSession:  h.getUserSession(c),
	}

	if err := h.repoManager.UserFeedback.Create(feedback); err != nil {
		h.logger.WithError(err).Error("Failed to save feedback")
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to save feedback", err)
		return
	}

	h.logger.WithFields(logrus.Fields{
		"query_id":      req.QueryID,
		"feedback_type": req.FeedbackType,
		"user_session":  feedback.UserSession,
	}).Info("Feedback recorded")

	utils.SuccessResponse(c, http.StatusCreated, "Feedback recorded", nil)
}

// HandleSearchSuggestions returns search suggestions
func (h *SearchHandler) HandleSearchSuggestions(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Query parameter 'q' is required", nil)
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	if limit > 10 {
		limit = 10
	}

	suggestions, err := h.repoManager.PopularQuery.GetTop(limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get search suggestions")
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get suggestions", err)
		return
	}

	// Filter suggestions that contain the query
	filtered := make([]models.PopularQuery, 0)
	queryLower := strings.ToLower(query)
	
	for _, suggestion := range suggestions {
		if strings.Contains(strings.ToLower(suggestion.QueryText), queryLower) {
			filtered = append(filtered, suggestion)
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "Suggestions retrieved", filtered)
}

// Helper methods

func (h *SearchHandler) getUserSession(c *gin.Context) string {
	// Try to get session from header first
	if session := c.GetHeader("X-Session-ID"); session != "" {
		return session
	}
	
	// Generate session based on IP + User-Agent (basic fingerprinting)
	userAgent := c.GetHeader("User-Agent")
	clientIP := c.ClientIP()
	
	// Create a simple session identifier
	sessionID := utils.GenerateSessionID(clientIP + userAgent)
	return sessionID
}

func (h *SearchHandler) generateCacheKey(query string) string {
	// Use MD5 hash of query for cache key
	return utils.MD5Hash(strings.ToLower(strings.TrimSpace(query)))
}

func (h *SearchHandler) trackSearchQuery(userSession, query string, resultsCount int, responseTime time.Duration, c *gin.Context) {
	searchQuery := &models.SearchQuery{
		QueryText:       query,
		UserSession:     userSession,
		ResultsCount:    resultsCount,
		SearchTimestamp: time.Now(),
		ResponseTimeMs:  int(responseTime.Milliseconds()),
		UserAgent:       c.GetHeader("User-Agent"),
		IPAddress:       c.ClientIP(),
	}

	if err := h.repoManager.SearchQuery.Create(searchQuery); err != nil {
		h.logger.WithError(err).Error("Failed to track search query")
	}
}

func (h *SearchHandler) updatePopularQueries(query string, resultsCount int, responseTime time.Duration) {
	if err := h.repoManager.PopularQuery.IncrementCount(query); err != nil {
		h.logger.WithError(err).Error("Failed to update popular queries")
		return
	}

	if err := h.repoManager.PopularQuery.UpdateStats(query, float64(resultsCount), int(responseTime.Milliseconds())); err != nil {
		h.logger.WithError(err).Error("Failed to update query stats")
	}
}