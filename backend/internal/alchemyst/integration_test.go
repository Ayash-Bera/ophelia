//go:build integration

package alchemyst

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestIntegration_RealAPI(t *testing.T) {
	apiKey := os.Getenv("ALCHEMYST_API_KEY")
	baseURL := os.Getenv("ALCHEMYST_BASE_URL")
	
	if apiKey == "" || baseURL == "" {
		t.Skip("ALCHEMYST_API_KEY and ALCHEMYST_BASE_URL required for integration tests")
	}

	client := NewClient(baseURL, apiKey, logrus.New())

	// Test adding context
	addReq := AddContextRequest{
		Documents: []Document{{
			Content:  "Test error: pacman failed to install package",
			FileName: "test.txt",
			FileType: "text/plain",
		}},
		Source:      "integration-test",
		ContextType: "resource",
		Scope:       "internal",
		Metadata: map[string]interface{}{
			"test": true,
		},
	}

	err := client.AddContext(addReq)
	require.NoError(t, err)

	// Test searching
	searchReq := SearchRequest{
		Query:                      "pacman install error",
		SimilarityThreshold:        0.8,
		MinimumSimilarityThreshold: 0.5,
		Scope:                      "internal",
	}

	response, err := client.SearchContext(searchReq)
	require.NoError(t, err)
	require.NotNil(t, response)

	// Cleanup
	deleteReq := DeleteContextRequest{
		Source: "integration-test",
		ByDoc:  true,
	}
	client.DeleteContext(deleteReq)
}