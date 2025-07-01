package alchemyst

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type Service struct {
	client *Client
	logger *logrus.Logger
}

func NewService(client *Client, logger *logrus.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

func (s *Service) AddWikiContent(ctx context.Context, title, content, url string) error {
	// Calculate content size
	contentSize := int64(len(content))

	// Get current timestamp for lastModified
	now := time.Now().Format(time.RFC3339)

	// Delete existing content first
	source := fmt.Sprintf("arch-wiki/%s", title)
	deleteReq := DeleteContextRequest{
		Source: source,
		ByDoc:  true,
		ByID:   false,
	}
	
	// Ignore delete errors (content might not exist)
	_ = s.client.DeleteContext(deleteReq)
	
	req := AddContextRequest{
		Documents: []Document{{
			Content:      content,
			FileName:     title + ".txt",
			FileType:     "text/plain",
			FileSize:     contentSize,
			LastModified: now,
		}},
		Source:      source,
		ContextType: "resource",
		Scope:       "internal",
		Chained:     false,
		Metadata: map[string]interface{}{
			// Required fields that the API expects
			"fileName":     title + ".txt",
			"fileSize":     contentSize,
			"fileType":     "text/plain",
			"lastModified": now,
			// Additional metadata
			"file_name":  title + ".txt",
			"doc_type":   "text/plain",
			"modalities": []string{"text"},
			"size":       contentSize,
			"wiki_title": title,
			"wiki_url":   url,
			"source":     "arch_linux_wiki",
			"extracted":  now,
		},
	}

	s.logger.WithFields(logrus.Fields{
		"title":        title,
		"content_size": contentSize,
		"url":          url,
		"source":       req.Source,
	}).Debug("Preparing Alchemyst request")

	return s.client.AddContextWithRetry(ctx, req)
}

func (s *Service) SearchForSolution(ctx context.Context, errorQuery string) ([]SearchResult, error) {
	req := SearchRequest{
		Query:                      errorQuery,
		SimilarityThreshold:        0.8,
		MinimumSimilarityThreshold: 0.3,
		Scope:                      "internal",
		Metadata: map[string]interface{}{
			// Required fields for search
			"size":       int64(len(errorQuery)),
			"doc_type":   "text/plain",
			"file_name":  "search_query.txt",
			// Additional metadata
			"search_type": "error_query",
			"source":      "arch_search_system",
			"modalities":  []string{"text"},
		},
	}

	s.logger.WithFields(logrus.Fields{
		"query":                        errorQuery,
		"similarity_threshold":         req.SimilarityThreshold,
		"minimum_similarity_threshold": req.MinimumSimilarityThreshold,
	}).Debug("Searching Alchemyst context")

	response, err := s.client.SearchContextWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}

	return response.Results, nil
}

func (s *Service) DeleteWikiContent(ctx context.Context, title string) error {
	req := DeleteContextRequest{
		Source: fmt.Sprintf("arch-wiki/%s", title),
		ByDoc:  true,
		ByID:   false,
	}

	s.logger.WithFields(logrus.Fields{
		"title":  title,
		"source": req.Source,
	}).Debug("Deleting from Alchemyst context")

	return s.client.DeleteContext(req)
}