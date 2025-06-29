package alchemyst

import (
	"context"
	"fmt"

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
	req := AddContextRequest{
		Documents: []Document{{
			Content:  content,
			FileName: title + ".txt",
			FileType: "text/plain",
		}},
		Source:      fmt.Sprintf("arch-wiki/%s", title),
		ContextType: "resource",
		Scope:       "internal",
		Metadata: map[string]interface{}{
			"wiki_title": title,
			"wiki_url":   url,
			"source":     "arch_linux_wiki",
		},
	}

	return s.client.AddContextWithRetry(ctx, req)
}

func (s *Service) SearchForSolution(ctx context.Context, errorQuery string) ([]SearchResult, error) {
	req := SearchRequest{
		Query:                      errorQuery,
		SimilarityThreshold:        0.8,
		MinimumSimilarityThreshold: 0.3,
		Scope:                      "internal",
	}

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
	}

	return s.client.DeleteContext(req)
}