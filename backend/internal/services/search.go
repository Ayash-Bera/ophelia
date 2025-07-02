// backend/internal/services/search.go
package services

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
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

	// In SearchForSolution, add these logs:
	s.logger.WithField("original_query", errorQuery).Info("Original query")
	s.logger.WithField("processed_query", processedQuery).Info("Processed query")
	s.logger.WithField("alchemyst_raw_count", len(alchemystResults)).Info("Raw Alchemyst results")
	s.logger.WithField("converted_count", len(searchResults)).Info("After conversion")

	// Filter and rank results
	// filteredResults := s.filterAndRankResults(searchResults, errorQuery)
	filteredResults := searchResults // Skip filtering temporarily

	// Limit results to top 10
	if len(filteredResults) > 10 {
		filteredResults = filteredResults[:10]
	}

	s.logger.WithField("final_results", len(filteredResults)).Debug("Search completed")

	return filteredResults, nil
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
	var filtered []string

	for _, word := range words {
		// Remove punctuation except for useful characters in error messages
		cleaned := regexp.MustCompile(`[^\w\-\.\/\:]`).ReplaceAllString(word, "")
		if cleaned == "" {
			continue
		}

		// Keep the word if it's not a noise word or if it looks like an error code/command
		isNoiseWord := false
		for _, noise := range noiseWords {
			if cleaned == noise {
				isNoiseWord = true
				break
			}
		}

		// Keep technical terms, error codes, and command names
		if !isNoiseWord || s.isTechnicalTerm(cleaned) {
			filtered = append(filtered, cleaned)
		}
	}

	result := strings.Join(filtered, " ")
	s.logger.WithFields(logrus.Fields{
		"original":  query,
		"processed": result,
	}).Debug("Query preprocessed")

	return result
}

// isTechnicalTerm checks if a word is likely a technical term worth keeping
func (s *SearchService) isTechnicalTerm(word string) bool {
	// Common technical patterns in Arch Linux
	technicalPatterns := []string{
		"pacman", "systemd", "grub", "xorg", "wayland", "networkmanager",
		"bluetooth", "audio", "pulseaudio", "alsa", "nvidia", "amd",
		"kernel", "module", "service", "unit", "mount", "fstab",
		"aur", "makepkg", "pkgbuild", "dependency", "conflict",
	}

	for _, pattern := range technicalPatterns {
		if strings.Contains(word, pattern) {
			return true
		}
	}

	// Check for error code patterns (numbers, version numbers, etc.)
	if matched, _ := regexp.MatchString(`\d`, word); matched {
		return true
	}

	// Check for command-like patterns
	if matched, _ := regexp.MatchString(`^[a-z]+(-[a-z]+)*$`, word); matched && len(word) > 2 {
		return true
	}

	return false
}

// convertAlchemystResults converts Alchemyst results to our SearchResult format
func (s *SearchService) convertAlchemystResults(alchemystResults []alchemyst.SearchResult) []models.SearchResult {
	var results []models.SearchResult

	for _, result := range alchemystResults {
		// Extract metadata from context data
		title, content, wikiURL := s.parseContextData(result.ContextData, result.ContextID)

		searchResult := models.SearchResult{
			ContextID: result.ContextID,
			Title:     title,
			Content:   content,
			URL:       wikiURL,
			Score:     s.calculateRelevanceScore(result.ContextData),
			Relevance: s.determineRelevance(s.calculateRelevanceScore(result.ContextData)),
		}

		results = append(results, searchResult)
	}

	return results
}

// parseContextData extracts title, content, and URL from Alchemyst context data
func (s *SearchService) parseContextData(contextData, contextID string) (title, content, wikiURL string) {
	// Try to extract wiki page title from context ID or data
	if strings.Contains(contextID, "arch-wiki/") {
		title = strings.TrimPrefix(contextID, "arch-wiki/")
		title = strings.ReplaceAll(title, "_", " ")
	} else {
		title = "Arch Linux Documentation"
	}

	// Clean and truncate content
	content = strings.TrimSpace(contextData)
	if len(content) > 500 {
		content = content[:500] + "..."
	}

	// Generate wiki URL
	if strings.Contains(contextID, "arch-wiki/") {
		pageName := strings.TrimPrefix(contextID, "arch-wiki/")
		wikiURL = fmt.Sprintf("https://wiki.archlinux.org/title/%s", url.QueryEscape(pageName))
	} else {
		wikiURL = "https://wiki.archlinux.org/"
	}

	return title, content, wikiURL
}

// calculateRelevanceScore calculates a relevance score for the result
func (s *SearchService) calculateRelevanceScore(contextData string) float64 {
	score := 0.5 // Base score

	contextLower := strings.ToLower(contextData)

	// Boost score for error-related content
	errorTerms := []string{
		"error", "failed", "failure", "problem", "issue", "trouble",
		"cannot", "can't", "unable", "not working", "broken",
		"fix", "solve", "solution", "troubleshoot", "debug",
	}

	for _, term := range errorTerms {
		if strings.Contains(contextLower, term) {
			score += 0.1
		}
	}

	// Boost score for solution-related content
	solutionTerms := []string{
		"install", "configure", "setup", "enable", "disable",
		"restart", "reload", "update", "upgrade", "downgrade",
		"edit", "modify", "change", "add", "remove",
	}

	for _, term := range solutionTerms {
		if strings.Contains(contextLower, term) {
			score += 0.1
		}
	}

	// Boost score for command-related content
	if strings.Contains(contextLower, "sudo") ||
		strings.Contains(contextLower, "pacman") ||
		strings.Contains(contextLower, "systemctl") {
		score += 0.2
	}

	// Cap the score at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
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

// filterAndRankResults filters out low-quality results and ranks by relevance
func (s *SearchService) filterAndRankResults(results []models.SearchResult, originalQuery string) []models.SearchResult {
	var filtered []models.SearchResult

	// Filter out results that are too short or seem irrelevant
	for _, result := range results {
		if len(result.Content) < 50 {
			continue // Skip very short results
		}

		if result.Score < 0.3 {
			continue // Skip low-relevance results
		}

		// Check if result contains keywords from original query
		if s.containsQueryKeywords(result, originalQuery) {
			filtered = append(filtered, result)
		}
	}

	// Sort by score (descending)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Score > filtered[j].Score
	})

	return filtered
}

// containsQueryKeywords checks if the result contains important keywords from the query
func (s *SearchService) containsQueryKeywords(result models.SearchResult, query string) bool {
	// Extract important words from query (skip common words)
	queryWords := s.extractImportantWords(query)
	if len(queryWords) == 0 {
		return true // If no important words, include the result
	}

	resultText := strings.ToLower(result.Title + " " + result.Content)

	// Check if at least one important word appears in the result
	for _, word := range queryWords {
		if strings.Contains(resultText, strings.ToLower(word)) {
			return true
		}
	}

	return false
}

// extractImportantWords extracts important words from a query
func (s *SearchService) extractImportantWords(query string) []string {
	// Split into words and filter
	words := regexp.MustCompile(`\W+`).Split(query, -1)
	var important []string

	for _, word := range words {
		word = strings.TrimSpace(strings.ToLower(word))
		if len(word) > 2 && s.isTechnicalTerm(word) {
			important = append(important, word)
		}
	}

	return important
}
