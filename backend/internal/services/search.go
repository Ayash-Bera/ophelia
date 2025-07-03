// backend/internal/services/search.go
package services

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Ayash-Bera/ophelia/backend/internal/alchemyst"
	"github.com/Ayash-Bera/ophelia/backend/internal/models"
	"github.com/Ayash-Bera/ophelia/backend/internal/repository"
	"github.com/sirupsen/logrus"
)

type SearchService struct {
	alchemystService *alchemyst.Service
	repoManager      *repository.RepositoryManager
	logger           *logrus.Logger
}

var (
	// Compiled regex patterns for better performance
	timestampPattern  = regexp.MustCompile(`-\d{8}-\d{6}-\d+$`)
	simplePattern     = regexp.MustCompile(`-\d+-\d+$`)
	digitsOnlyPattern = regexp.MustCompile(`^\d+$`)
)

func NewSearchService(
	alchemystService *alchemyst.Service,
	repoManager *repository.RepositoryManager,
	logger *logrus.Logger,
) *SearchService {
	return &SearchService{
		alchemystService: alchemystService,
		repoManager:      repoManager,
		logger:           logger,
	}
}

// SearchForSolution searches for solutions to the given error query
func (s *SearchService) SearchForSolution(ctx context.Context, errorQuery string) ([]models.SearchResult, error) {
	s.logger.WithField("query", errorQuery).Debug("Starting search for solution")

	// Preprocess the query
	processedQuery := s.preprocessQuery(errorQuery)

	// Search using Alchemyst Context API
	alchemystResults, err := s.alchemystService.SearchForSolution(ctx, processedQuery)
	if err != nil {
		s.logger.WithError(err).Error("Alchemyst search failed")
		return nil, fmt.Errorf("search service unavailable: %w", err)
	}

	s.logger.WithField("raw_results", len(alchemystResults)).Debug("Received results from Alchemyst")

	// Convert and enhance results
	searchResults := s.convertAlchemystResults(alchemystResults)

	s.logger.WithField("original_query", errorQuery).Info("Original query")
	s.logger.WithField("processed_query", processedQuery).Info("Processed query")
	s.logger.WithField("alchemyst_raw_count", len(alchemystResults)).Info("Raw Alchemyst results")
	s.logger.WithField("converted_count", len(searchResults)).Info("After conversion")

	// TODO: Add result filtering and ranking in future iterations
	// Limit results to top 10
	if len(searchResults) > 10 {
		searchResults = searchResults[:10]
	}

	s.logger.WithField("final_results", len(searchResults)).Debug("Search completed")

	return searchResults, nil
}

// preprocessQuery cleans and enhances the search query
func (s *SearchService) preprocessQuery(query string) string {
	// Remove common noise words that don't help with error searching
	noiseWords := []string{
		"please", "help", "how", "do", "i", "can", "you", "me", "my", "the", "a", "an",
		"is", "are", "was", "were", "be", "been", "being", "have", "has", "had", "will", "would",
		"could", "should", "may", "might", "must", "shall", "does", "did", "don't", "doesn't",
		"won't", "wouldn't", "couldn't", "shouldn't", "mustn't", "shan't", "didn't",
	}

	words := strings.Fields(strings.ToLower(query))
	var filteredWords []string

	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) > 2 && !s.contains(noiseWords, word) {
			filteredWords = append(filteredWords, word)
		}
	}

	processed := strings.Join(filteredWords, " ")

	// If filtering removed too much, use original
	if len(processed) < len(query)/3 {
		return query
	}

	return processed
}

// contains checks if a slice contains a string
func (s *SearchService) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// convertAlchemystResults converts Alchemyst results to our SearchResult format
func (s *SearchService) convertAlchemystResults(alchemystResults []alchemyst.SearchResult) []models.SearchResult {
	var results []models.SearchResult
	for _, result := range alchemystResults {
		// Extract page name from filename (format: "PageName-timestamp-random.txt")
		pageName := s.extractPageNameFromFilename(result.Metadata.FileName)

		title := fmt.Sprintf("Arch Wiki - %s", strings.ReplaceAll(pageName, "_", " "))
		wikiURL := fmt.Sprintf("https://wiki.archlinux.org/title/%s", url.QueryEscape(pageName))

		searchResult := models.SearchResult{
			ContextID: result.ID.OID,
			Title:     title,
			Content:   result.Text,
			URL:       wikiURL,
			Score:     result.Score,
			Relevance: s.determineRelevance(result.Score),
		}
		results = append(results, searchResult)
	}
	return results
}

// extractPageNameFromFilename extracts the page name from Alchemyst filename
// Format: "PageName-timestamp-random.txt" -> "PageName"
func (s *SearchService) extractPageNameFromFilename(filename string) string {
	// Remove .txt extension
	filename = strings.TrimSuffix(filename, ".txt")

	// Remove timestamp and random number suffix using compiled regex
	pageName := timestampPattern.ReplaceAllString(filename, "")

	// If no timestamp pattern found, try simpler patterns
	if pageName == filename {
		// Try to remove any trailing -numbers-numbers pattern
		pageName = simplePattern.ReplaceAllString(filename, "")
	}

	// If still no change, try to remove just the last timestamp-like component
	if pageName == filename {
		parts := strings.Split(filename, "-")
		if len(parts) > 2 {
			// Keep everything except the last 2-3 parts which might be timestamp
			for i := len(parts) - 1; i >= 0; i-- {
				if digitsOnlyPattern.MatchString(parts[i]) {
					// This part is all digits, likely part of timestamp
					parts = parts[:i]
				} else {
					break
				}
			}
			pageName = strings.Join(parts, "-")
		}
	}

	// Fallback - if we couldn't extract properly, return the original without .txt
	if pageName == "" {
		pageName = strings.TrimSuffix(filename, ".txt")
	}

	return pageName
}

// determineRelevance converts numeric score to text relevance
func (s *SearchService) determineRelevance(score float64) string {
	if score >= 0.8 {
		return "high"
	} else if score >= 0.6 {
		return "medium"
	} else {
		return "low"
	}
}
