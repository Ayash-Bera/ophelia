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
	contentSize := int64(len(content))
	now := time.Now()
	timestamp := now.Format("20060102-150405")
	
	// Use unique filename with timestamp + random component
	fileName := fmt.Sprintf("%s-%s-%d.txt", title, timestamp, now.UnixNano()%1000)
	
	// Simple deletion attempt (don't fail if it doesn't work)
	source := fmt.Sprintf("arch-wiki/%s", title)
	deleteReq := DeleteContextRequest{
		Source: source,
		ByDoc:  true,
	}
	
	if err := s.client.DeleteContext(deleteReq); err != nil {
		s.logger.WithError(err).Debug("Delete failed, continuing with unique filename")
	} else {
		// Wait briefly for deletion to propagate
		time.Sleep(3 * time.Second)
	}
	
	req := AddContextRequest{
		Documents: []Document{{
			Content:      content,
			FileName:     fileName,
			FileType:     "text/plain",
			FileSize:     contentSize,
			LastModified: now.Format(time.RFC3339),
		}},
		Source:      source,
		ContextType: "resource",
		Scope:       "internal",
		Metadata: map[string]interface{}{
			"fileName":     fileName,
			"fileSize":     contentSize,
			"fileType":     "text/plain",
			"lastModified": now.Format(time.RFC3339),
			"modalities":   []string{"text"},
		},
	}

	return s.client.AddContextWithRetry(ctx, req)
}


func (s *Service) SearchForSolution(ctx context.Context, errorQuery string) ([]SearchResult, error) {
	req := SearchRequest{
		Query:                      errorQuery,
		SimilarityThreshold:        0.7,
		MinimumSimilarityThreshold: 0.3,
		Scope:                      "internal",
		// Remove metadata - match your working curl exactly
	}

	s.logger.WithFields(logrus.Fields{
		"query": errorQuery,
		"scope": req.Scope,
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